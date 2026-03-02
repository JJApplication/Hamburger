package wasm_plugin

import (
	"Hamburger/internal/config"
	"Hamburger/internal/json"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	requestHandleFunc  = "request_handle"
	responseHandleFunc = "response_handle"
	allocFuncName      = "alloc"
	mallocFuncName     = "malloc"
	freeFuncName       = "free"
	deallocFuncName    = "dealloc"
)

type manager struct {
	once       sync.Once
	initErr    error
	plugins    []*plugin
	runtime    wazero.Runtime
	pluginRoot string
	enabled    bool
	logger     *zerolog.Logger
}

type plugin struct {
	name       string
	enabled    bool
	params     map[string]interface{}
	module     api.Module
	requestFn  api.Function
	responseFn api.Function
	allocFn    api.Function
	freeFn     api.Function
	memory     api.Memory
}

type requestContext struct {
	Method     string              `json:"method"`
	URL        string              `json:"url"`
	Host       string              `json:"host"`
	RemoteAddr string              `json:"remote_addr"`
	Header     map[string][]string `json:"header"`
}

type responseContext struct {
	StatusCode int                 `json:"status_code"`
	Header     map[string][]string `json:"header"`
}

type requestResult struct {
	Allow     *bool             `json:"allow"`
	Error     string            `json:"error"`
	Status    int               `json:"status"`
	HeaderSet map[string]string `json:"header_set"`
	HeaderAdd map[string]string `json:"header_add"`
	HeaderDel []string          `json:"header_del"`
}

type responseResult struct {
	Status    int               `json:"status"`
	HeaderSet map[string]string `json:"header_set"`
	HeaderAdd map[string]string `json:"header_add"`
	HeaderDel []string          `json:"header_del"`
}

var defaultManager = &manager{}

func Init(cfg *config.Config, logger *zerolog.Logger) error {
	return defaultManager.init(cfg, logger)
}

func HandleRequest(logger *zerolog.Logger, request *http.Request) error {
	return defaultManager.handleRequest(logger, request)
}

func HandleResponse(logger *zerolog.Logger, response *http.Response) error {
	return defaultManager.handleResponse(logger, response)
}

func (m *manager) init(cfg *config.Config, logger *zerolog.Logger) error {
	m.once.Do(func() {
		if cfg == nil {
			return
		}
		m.enabled = cfg.Plugin.Enabled
		if !m.enabled {
			return
		}
		if strings.TrimSpace(cfg.Plugin.Root) == "" {
			m.enabled = false
			return
		}
		m.logger = logger
		m.pluginRoot = cfg.Plugin.Root
		ctx := context.Background()
		m.runtime = wazero.NewRuntime(ctx)
		_, err := wasi_snapshot_preview1.Instantiate(ctx, m.runtime)
		if err != nil {
			m.initErr = err
			return
		}
		entries, err := os.ReadDir(m.pluginRoot)
		if err != nil {
			m.initErr = err
			return
		}
		configMap := buildPluginConfigMap(cfg.Plugin.Plugins)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !strings.EqualFold(filepath.Ext(entry.Name()), ".wasm") {
				continue
			}
			filePath := filepath.Join(m.pluginRoot, entry.Name())
			wasmBytes, err := os.ReadFile(filePath)
			if err != nil {
				m.logErr(err, "read wasm failed", filePath)
				continue
			}
			mod, err := m.runtime.InstantiateWithConfig(ctx, wasmBytes, wazero.NewModuleConfig().WithName(entry.Name()))
			if err != nil {
				m.logErr(err, "instantiate wasm failed", filePath)
				continue
			}
			baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			item, ok := getPluginConfig(configMap, entry.Name(), baseName)
			enabled := true
			var params map[string]interface{}
			if ok {
				enabled = item.Enabled
				params = item.Params
			}
			p := buildPlugin(baseName, mod, enabled, params)
			if p.requestFn == nil && p.responseFn == nil {
				_ = mod.Close(ctx)
				continue
			}
			m.plugins = append(m.plugins, p)
			m.logInfo("load wasm plugin success", p.name)
		}
	})
	return m.initErr
}

func (m *manager) handleRequest(logger *zerolog.Logger, request *http.Request) error {
	if !m.enabled {
		return nil
	}
	if len(m.plugins) == 0 {
		return nil
	}
	payload, err := json.Marshal(requestContext{
		Method:     request.Method,
		URL:        request.URL.String(),
		Host:       request.Host,
		RemoteAddr: request.RemoteAddr,
		Header:     cloneHeader(request.Header),
	})
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, p := range m.plugins {
		if !p.enabled {
			continue
		}
		if p.requestFn == nil {
			continue
		}
		out, err := p.call(ctx, p.requestFn, payload)
		if err != nil {
			m.logPluginErr(logger, err, p.name, "request")
			return err
		}
		if len(out) == 0 {
			continue
		}
		var result requestResult
		if err := json.Unmarshal(out, &result); err != nil {
			m.logPluginErr(logger, err, p.name, "request")
			return err
		}
		applyHeaderOps(request.Header, result.HeaderSet, result.HeaderAdd, result.HeaderDel)
		if !isAllowed(result.Allow) || strings.TrimSpace(result.Error) != "" {
			if result.Error != "" {
				return errors.New(result.Error)
			}
			return errors.New("wasm request plugin rejected")
		}
		if result.Status > 0 {
			request.Header.Set("X-Wasm-Status", fmt.Sprintf("%d", result.Status))
		}
	}
	return nil
}

func (m *manager) handleResponse(logger *zerolog.Logger, response *http.Response) error {
	if !m.enabled {
		return nil
	}
	if len(m.plugins) == 0 {
		return nil
	}
	payload, err := json.Marshal(responseContext{
		StatusCode: response.StatusCode,
		Header:     cloneHeader(response.Header),
	})
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, p := range m.plugins {
		if !p.enabled {
			continue
		}
		if p.responseFn == nil {
			continue
		}
		out, err := p.call(ctx, p.responseFn, payload)
		if err != nil {
			m.logPluginErr(logger, err, p.name, "response")
			continue
		}
		if len(out) == 0 {
			continue
		}
		var result responseResult
		if err := json.Unmarshal(out, &result); err != nil {
			m.logPluginErr(logger, err, p.name, "response")
			continue
		}
		if result.Status > 0 && result.Status != response.StatusCode {
			response.StatusCode = result.Status
			response.Status = fmt.Sprintf("%d %s", result.Status, http.StatusText(result.Status))
		}
		applyHeaderOps(response.Header, result.HeaderSet, result.HeaderAdd, result.HeaderDel)
	}
	return nil
}

func buildPlugin(name string, mod api.Module, enabled bool, params map[string]interface{}) *plugin {
	reqFn := mod.ExportedFunction(requestHandleFunc)
	respFn := mod.ExportedFunction(responseHandleFunc)
	allocFn := mod.ExportedFunction(allocFuncName)
	if allocFn == nil {
		allocFn = mod.ExportedFunction(mallocFuncName)
	}
	freeFn := mod.ExportedFunction(deallocFuncName)
	if freeFn == nil {
		freeFn = mod.ExportedFunction(freeFuncName)
	}
	return &plugin{
		name:       name,
		enabled:    enabled,
		params:     params,
		module:     mod,
		requestFn:  reqFn,
		responseFn: respFn,
		allocFn:    allocFn,
		freeFn:     freeFn,
		memory:     mod.Memory(),
	}
}

func buildPluginConfigMap(items []config.PluginItem) map[string]config.PluginItem {
	out := make(map[string]config.PluginItem, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		out[strings.ToLower(item.Name)] = item
	}
	return out
}

func getPluginConfig(items map[string]config.PluginItem, names ...string) (config.PluginItem, bool) {
	for _, name := range names {
		if name == "" {
			continue
		}
		if item, ok := items[strings.ToLower(name)]; ok {
			return item, true
		}
	}
	return config.PluginItem{}, false
}

func (p *plugin) call(ctx context.Context, fn api.Function, payload []byte) ([]byte, error) {
	if p.allocFn == nil || p.memory == nil {
		return nil, errors.New("wasm plugin missing alloc or memory")
	}
	ptrRes, err := p.allocFn.Call(ctx, uint64(len(payload)))
	if err != nil {
		return nil, err
	}
	ptr := uint32(ptrRes[0])
	if ok := p.memory.Write(ptr, payload); !ok {
		return nil, errors.New("wasm write memory failed")
	}
	outRes, err := fn.Call(ctx, uint64(ptr), uint64(len(payload)))
	if err != nil {
		return nil, err
	}
	outPtr, outLen := splitPtrLen(outRes[0])
	if outLen == 0 {
		return nil, nil
	}
	out, ok := p.memory.Read(outPtr, outLen)
	if !ok {
		return nil, errors.New("wasm read memory failed")
	}
	if p.freeFn != nil {
		_, _ = p.freeFn.Call(ctx, uint64(outPtr), uint64(outLen))
	}
	return out, nil
}

func splitPtrLen(v uint64) (uint32, uint32) {
	return uint32(v), uint32(v >> 32)
}

func cloneHeader(header http.Header) map[string][]string {
	out := make(map[string][]string, len(header))
	for k, v := range header {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

func applyHeaderOps(header http.Header, set map[string]string, add map[string]string, del []string) {
	for k, v := range set {
		header.Set(k, v)
	}
	for k, v := range add {
		header.Add(k, v)
	}
	for _, k := range del {
		header.Del(k)
	}
}

func isAllowed(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func (m *manager) logErr(err error, msg string, path string) {
	if m.logger == nil {
		return
	}
	m.logger.Error().Err(err).Str("path", path).Msg(msg)
}

func (m *manager) logInfo(msg string, name string) {
	if m.logger == nil {
		return
	}
	m.logger.Info().Str("plugin", name).Msg(msg)
}

func (m *manager) logPluginErr(logger *zerolog.Logger, err error, pluginName string, phase string) {
	l := logger
	if l == nil {
		l = m.logger
	}
	if l == nil {
		return
	}
	l.Error().Err(err).Str("plugin", pluginName).Str("phase", phase).Msg("wasm plugin failed")
}
