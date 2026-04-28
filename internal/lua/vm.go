package lua

import (
	"Hamburger/internal/config"
	"fmt"
	"net/http"
	"sync"

	"github.com/rs/zerolog"
	lua "github.com/yuin/gopher-lua"
)

type manager struct {
	once        sync.Once
	lock        sync.Mutex
	state       *lua.LState
	middlewares []middleware
	enabled     bool
	scriptsRoot string
	initErr     error
	logger      *zerolog.Logger
}

var defaultManager = &manager{}

// Init 初始化全局Lua虚拟机。
func Init(cfg config.LuaConfig, logger *zerolog.Logger) error {
	return defaultManager.init(cfg, logger)
}

// HandleRequest 执行Lua请求处理中间件。
func HandleRequest(request *http.Request) error {
	return defaultManager.handleRequest(request)
}

func (m *manager) init(cfg config.LuaConfig, logger *zerolog.Logger) error {
	m.once.Do(func() {
		m.logger = logger
		m.enabled = cfg.Enabled
		if !cfg.Enabled {
			return
		}
		root, err := resolveScriptsRoot(cfg.ScriptsRoot)
		if err != nil {
			m.initErr = err
			return
		}
		m.scriptsRoot = root
		m.state = lua.NewState()
		registerGlobals(m.state)
		m.middlewares, err = loadScripts(m.state, root)
		if err != nil {
			m.state.Close()
			m.state = nil
			m.initErr = err
			return
		}
		if m.logger != nil {
			m.logger.Info().Str("root", root).Int("middlewares", len(m.middlewares)).Msg("lua vm initialized")
		}
	})
	return m.initErr
}

func luaParam(fn *lua.LFunction) lua.P {
	return lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}
}

func rejectError(message string) error {
	if message == "" {
		message = "lua request middleware rejected"
	}
	return fmt.Errorf(message)
}
