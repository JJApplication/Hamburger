package grpc_server

import (
	"Hamburger/internal/config/loader"
	"context"
	"encoding/json"
	"fmt"
	"os"
	stdRuntime "runtime"
	"strings"
	"time"

	appgrpc "Hamburger/app/grpc"
	"Hamburger/backend_proxy"
	"Hamburger/exp/any_tls"
	"Hamburger/exp/trojan"
	"Hamburger/exp/vpn_proxy"
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/api"
	"Hamburger/gateway/latency"
	"Hamburger/gateway/manager"
	"Hamburger/gateway/modifier"
	"Hamburger/gateway/prehandler"
	gwRuntime "Hamburger/gateway/runtime"
	"Hamburger/internal/constant"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AppService struct {
	getManager         func() *manager.Manager
	getFrontServer     func() *frontend_proxy.HeliosServer
	getModifierManager func() *modifier.ModifierManager
	getAPIServer       func() *api.Server
	getBackendServer   func() *backend_proxy.BackendProxy
	getLatencyServer   func() *latency.LatencyServer
	getVPNServer       func() *vpn_proxy.VpnServer
	getTrojanServer    func() *trojan.TrojanServer
	getAnyTLSServer    func() *any_tls.AnyTLSServer
	appgrpc.UnimplementedAppServiceServer
}

func NewAppService(getManager func() *manager.Manager, getFrontServer func() *frontend_proxy.HeliosServer, getModifierManager func() *modifier.ModifierManager, getAPIServer func() *api.Server, getBackendServer func() *backend_proxy.BackendProxy, getLatencyServer func() *latency.LatencyServer, getVPNServer func() *vpn_proxy.VpnServer, getTrojanServer func() *trojan.TrojanServer, getAnyTLSServer func() *any_tls.AnyTLSServer) *AppService {
	return &AppService{
		getManager:         getManager,
		getFrontServer:     getFrontServer,
		getModifierManager: getModifierManager,
		getAPIServer:       getAPIServer,
		getBackendServer:   getBackendServer,
		getLatencyServer:   getLatencyServer,
		getVPNServer:       getVPNServer,
		getTrojanServer:    getTrojanServer,
		getAnyTLSServer:    getAnyTLSServer,
	}
}

func (s *AppService) GetGatewayStatus(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.GatewayStatusResponse, error) {
	if s.getManager == nil {
		return nil, status.Error(codes.Unavailable, "gateway manager unavailable")
	}
	managerInstance := s.getManager()
	if managerInstance == nil {
		return nil, status.Error(codes.Unavailable, "gateway manager unavailable")
	}
	cfg := loader.Get()
	resp := &appgrpc.GatewayStatusResponse{}
	gwStatus := managerInstance.GetServerStatus()
	for _, instance := range gwStatus {
		address := fmt.Sprintf("%s:%d", instance.Config.Host, instance.Config.Port)
		resp.Servers = append(resp.Servers, &appgrpc.ServerStatus{
			Name:    instance.Name,
			Running: instance.Started,
			Address: address,
		})
	}
	http3Status := managerInstance.GetHttp3ServerStatus()
	for _, instance := range http3Status {
		resp.Http3Servers = append(resp.Http3Servers, &appgrpc.ServerStatus{
			Name:    instance.Name,
			Running: instance.IsStarted(),
			Address: instance.Address,
		})
	}
	if cfg != nil {
		for _, serverCfg := range cfg.Servers {
			domains := []string{}
			wsDomains := []string{}
			autoRedirect := false
			useWebsocket := false
			for _, dc := range serverCfg.DomainConfig {
				domains = append(domains, dc.Domains...)
				wsDomains = append(wsDomains, dc.WsDomains...)
				autoRedirect = autoRedirect || dc.AutoRedirect
				useWebsocket = useWebsocket || dc.UseWebsocket
			}
			resp.ServerConfigs = append(resp.ServerConfigs, &appgrpc.GatewayServerConfig{
				Name:           serverCfg.Name,
				Host:           serverCfg.Host,
				Port:           int32(serverCfg.Port),
				UseHttp2:       serverCfg.UseHttp2,
				Protocol:       serverCfg.Protocol,
				Enabled:        serverCfg.Enabled,
				MaxRequestBody: int64(serverCfg.MaxRequestBody),
				Domains:        domains,
				AutoRedirect:   autoRedirect,
				UseWebsocket:   useWebsocket,
				WsDomains:      wsDomains,
			})
		}
		m := cfg.Middleware
		resp.Middleware = &appgrpc.MiddlewareStatus{
			GzipEnabled:         m.Gzip.Enabled,
			GzipLevel:           int32(m.Gzip.Level),
			GzipTypes:           m.Gzip.Types,
			NoCache:             m.NoCache,
			SecureHeader:        m.SecureHeader,
			TraceEnabled:        m.Trace.Enabled,
			TraceId:             m.Trace.TraceId,
			CorsEnabled:         m.CORS.Enabled,
			CorsMethod:          m.CORS.Method,
			CorsOrigin:          m.CORS.Origin,
			CorsHeader:          m.CORS.Header,
			SanitizerEnabled:    m.Sanitizer.Enabled,
			DomainCheckEnabled:  m.DomainCheck.Enabled,
			ImageProtectEnabled: m.ImageProtect.Enabled,
			ImageType:           m.ImageProtect.ImageType,
			AllowReferer:        m.ImageProtect.AllowReferer,
		}
	}
	return resp, nil
}

func (s *AppService) GetFrontProxyStatus(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.FrontProxyStatusResponse, error) {
	cfg := loader.Get()
	if cfg == nil {
		return nil, status.Error(codes.Unavailable, "config unavailable")
	}
	frontCfg := cfg.PxyFrontend
	mode := "http"
	if frontCfg.ExpFastConnect.Http3.Enabled {
		mode = "http3"
	} else if frontCfg.ExpFastConnect.Enabled {
		mode = "http2"
	}
	root := ""
	if len(frontCfg.Servers) > 0 {
		root = frontCfg.Servers[0].Root
	}
	running := s.getFrontServer != nil && s.getFrontServer() != nil
	return &appgrpc.FrontProxyStatusResponse{
		Running:      running,
		Mode:         mode,
		Root:         root,
		Host:         frontCfg.Host,
		Port:         int32(frontCfg.Port),
		Balancer:     frontCfg.Balancer,
		ServerCount:  int32(len(frontCfg.Servers)),
		CacheEnabled: frontCfg.Cache.Enable,
	}, nil
}

func (s *AppService) GetModifierManagerInfo(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.ModifierManagerInfoResponse, error) {
	if s.getModifierManager == nil {
		return nil, status.Error(codes.Unavailable, "modifier manager unavailable")
	}
	modManager := s.getModifierManager()
	if modManager == nil {
		return nil, status.Error(codes.Unavailable, "modifier manager unavailable")
	}
	statusMap := modManager.GetStatus()
	var total int32
	switch v := statusMap["total_modifiers"].(type) {
	case int:
		total = int32(v)
	case int32:
		total = v
	case int64:
		total = int32(v)
	case float64:
		total = int32(v)
	}
	modifiers := []*appgrpc.ModifierInfo{}
	if rawList, ok := statusMap["modifiers"]; ok {
		switch list := rawList.(type) {
		case []map[string]interface{}:
			for _, item := range list {
				modifiers = append(modifiers, modifierFromMap(item))
			}
		case []interface{}:
			for _, item := range list {
				if m, ok := item.(map[string]interface{}); ok {
					modifiers = append(modifiers, modifierFromMap(m))
				}
			}
		}
	}
	return &appgrpc.ModifierManagerInfoResponse{
		ModifierCount: total,
		Modifiers:     modifiers,
	}, nil
}

func (s *AppService) GetStatServerConfig(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.StatServerConfigResponse, error) {
	cfg := loader.Get()
	if cfg == nil {
		return nil, status.Error(codes.Unavailable, "config unavailable")
	}
	statCfg := cfg.Stat
	return &appgrpc.StatServerConfigResponse{
		Enabled:          statCfg.Enabled,
		EnableStat:       statCfg.EnableStat,
		SyncDuration:     int32(statCfg.SyncDuration),
		SaveDuration:     int32(statCfg.SaveDuration),
		SaveFile:         statCfg.SaveFile,
		GeoFile:          statCfg.GeoFile,
		DomainFile:       statCfg.DomainFile,
		GeoDb:            statCfg.GeoDB,
		SequenceEnabled:  statCfg.Sequence.Enabled,
		SequenceInterval: int32(statCfg.Sequence.Interval),
	}, nil
}

func (s *AppService) GetRuntime(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.RuntimeResponse, error) {
	var mem stdRuntime.MemStats
	stdRuntime.ReadMemStats(&mem)
	netIoBlock := false
	cfg := loader.Get()
	if cfg != nil {
		netIoBlock = strings.EqualFold(cfg.CoreProxy.NetIO, constant.NetIO_NET)
	}
	return &appgrpc.RuntimeResponse{
		Cpu:         0,
		MemoryBytes: int64(mem.Alloc),
		RssBytes:    int64(mem.Sys),
		Goroutines:  int32(stdRuntime.NumGoroutine()),
		NetIoBlock:  netIoBlock,
	}, nil
}

func (s *AppService) GetDomainMap(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.DomainMapResponse, error) {
	domains, domainMap, frontMap := gwRuntime.GetDomainsSnapshot()
	resp := &appgrpc.DomainMapResponse{
		Domains:        domains,
		DomainsMap:     map[string]string{},
		DomainFrontMap: map[string]string{},
	}
	for domain, m := range domainMap {
		resp.DomainsMap[domain] = m
	}
	for k, v := range frontMap {
		resp.DomainFrontMap[k] = v
	}
	return resp, nil
}

func (s *AppService) GetDomainPorts(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.DomainPortsResponse, error) {
	snapshot := gwRuntime.GetDomainPortsSnapshot()
	resp := &appgrpc.DomainPortsResponse{}
	for domain, ports := range snapshot {
		resp.Ports = append(resp.Ports, &appgrpc.DomainPorts{
			Domain: domain,
			Ports:  toInt32Slice(ports),
		})
	}
	return resp, nil
}

func (s *AppService) ReloadConfig(ctx context.Context, req *appgrpc.ReloadConfigRequest) (*appgrpc.ReloadConfigResponse, error) {
	file := strings.TrimSpace(req.GetFile())
	if err := reloadConfigInPlace(file); err != nil {
		return &appgrpc.ReloadConfigResponse{Success: false, Error: err.Error()}, nil
	}
	return &appgrpc.ReloadConfigResponse{Success: true}, nil
}

func (s *AppService) ReStartFrontServer(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.ActionResponse, error) {
	if s.getFrontServer == nil {
		return nil, status.Error(codes.Unavailable, "front server unavailable")
	}
	frontServer := s.getFrontServer()
	if frontServer == nil {
		return nil, status.Error(codes.Unavailable, "front server unavailable")
	}
	if err := reloadConfigInPlace(""); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	frontServer.Shutdown()
	go func() {
		_ = frontServer.Start()
	}()
	return &appgrpc.ActionResponse{Success: true}, nil
}

func (s *AppService) ReStartGateway(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.ActionResponse, error) {
	if s.getManager == nil {
		return nil, status.Error(codes.Unavailable, "gateway manager unavailable")
	}
	managerInstance := s.getManager()
	if managerInstance == nil {
		return nil, status.Error(codes.Unavailable, "gateway manager unavailable")
	}
	if err := reloadConfigInPlace(""); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := managerInstance.Restart(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	return &appgrpc.ActionResponse{Success: true}, nil
}

func (s *AppService) ReStartAPIServer(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.ActionResponse, error) {
	if s.getAPIServer == nil {
		return nil, status.Error(codes.Unavailable, "api server unavailable")
	}
	apiServer := s.getAPIServer()
	if apiServer == nil {
		return nil, status.Error(codes.Unavailable, "api server unavailable")
	}
	if err := reloadConfigInPlace(""); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := apiServer.Stop(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := apiServer.Start(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	return &appgrpc.ActionResponse{Success: true}, nil
}

func (s *AppService) ReStartBackendServer(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.ActionResponse, error) {
	if s.getBackendServer == nil {
		return nil, status.Error(codes.Unavailable, "backend server unavailable")
	}
	backendServer := s.getBackendServer()
	if backendServer == nil {
		return nil, status.Error(codes.Unavailable, "backend server unavailable")
	}
	if err := reloadConfigInPlace(""); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	backendServer.Stop()
	backendServer.Start()
	return &appgrpc.ActionResponse{Success: true}, nil
}

func (s *AppService) ReStartLatencyServer(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.ActionResponse, error) {
	if s.getLatencyServer == nil {
		return nil, status.Error(codes.Unavailable, "latency server unavailable")
	}
	latencyServer := s.getLatencyServer()
	if latencyServer == nil {
		return nil, status.Error(codes.Unavailable, "latency server unavailable")
	}
	if err := reloadConfigInPlace(""); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := latencyServer.Stop(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := latencyServer.Start(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	return &appgrpc.ActionResponse{Success: true}, nil
}

func (s *AppService) ReStartVPNServer(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.ActionResponse, error) {
	if s.getVPNServer == nil {
		return nil, status.Error(codes.Unavailable, "vpn server unavailable")
	}
	vpnServer := s.getVPNServer()
	if vpnServer == nil {
		return nil, status.Error(codes.Unavailable, "vpn server unavailable")
	}
	if err := reloadConfigInPlace(""); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := vpnServer.Stop(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := vpnServer.Start(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	return &appgrpc.ActionResponse{Success: true}, nil
}

func (s *AppService) ReStartTrojanServer(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.ActionResponse, error) {
	if s.getTrojanServer == nil {
		return nil, status.Error(codes.Unavailable, "trojan server unavailable")
	}
	trojanServer := s.getTrojanServer()
	if trojanServer == nil {
		return nil, status.Error(codes.Unavailable, "trojan server unavailable")
	}
	if err := reloadConfigInPlace(""); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := trojanServer.Stop(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := trojanServer.Start(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	return &appgrpc.ActionResponse{Success: true}, nil
}

func (s *AppService) ReStartAnyTLSServer(ctx context.Context, _ *appgrpc.Empty) (*appgrpc.ActionResponse, error) {
	if s.getAnyTLSServer == nil {
		return nil, status.Error(codes.Unavailable, "anytls server unavailable")
	}
	anyTLSServer := s.getAnyTLSServer()
	if anyTLSServer == nil {
		return nil, status.Error(codes.Unavailable, "anytls server unavailable")
	}
	if err := reloadConfigInPlace(""); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := anyTLSServer.Stop(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := anyTLSServer.Start(); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	return &appgrpc.ActionResponse{Success: true}, nil
}

func (s *AppService) StartDomainService(ctx context.Context, req *appgrpc.DomainServiceRequest) (*appgrpc.ActionResponse, error) {
	domain := strings.TrimSpace(req.GetDomain())
	if err := prehandler.GetServiceManager().StartDomain(domain); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	return &appgrpc.ActionResponse{Success: true}, nil
}

func (s *AppService) StopDomainService(ctx context.Context, req *appgrpc.DomainServiceRequest) (*appgrpc.ActionResponse, error) {
	domain := strings.TrimSpace(req.GetDomain())
	if err := prehandler.GetServiceManager().StopDomain(domain); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	return &appgrpc.ActionResponse{Success: true}, nil
}

func (s *AppService) DumpRuntime(ctx context.Context, req *appgrpc.DumpRuntimeRequest) (*appgrpc.ActionResponse, error) {
	path := strings.TrimSpace(req.GetPath())
	if path == "" {
		return &appgrpc.ActionResponse{Success: false, Message: "path is empty"}, nil
	}
	domains, domainMap, frontMap := gwRuntime.GetDomainsSnapshot()
	domainPorts := gwRuntime.GetDomainPortsSnapshot()
	var mem stdRuntime.MemStats
	stdRuntime.ReadMemStats(&mem)
	payload := map[string]interface{}{
		"timestamp":    time.Now().Format(time.RFC3339),
		"domains":      domains,
		"domain_map":   domainMap,
		"front_map":    frontMap,
		"domain_ports": domainPorts,
		"goroutines":   stdRuntime.NumGoroutine(),
		"memory_bytes": mem.Alloc,
		"rss_bytes":    mem.Sys,
		"config":       loader.Get(),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return &appgrpc.ActionResponse{Success: false, Message: err.Error()}, nil
	}
	return &appgrpc.ActionResponse{Success: true}, nil
}

func modifierFromMap(m map[string]interface{}) *appgrpc.ModifierInfo {
	info := &appgrpc.ModifierInfo{}
	if v, ok := m["name"].(string); ok {
		info.Name = v
	}
	if v, ok := m["enabled"].(bool); ok {
		info.Enabled = v
	}
	return info
}

func toInt32Slice(values []int) []int32 {
	result := make([]int32, 0, len(values))
	for _, v := range values {
		result = append(result, int32(v))
	}
	return result
}

func reloadConfigInPlace(file string) error {
	path := strings.TrimSpace(file)
	if path == "" {
		path = "config/config.json"
	}
	appCfg, err := loader.LoadConfig(path)
	if err != nil {
		return err
	}
	mergedCfg := loader.Merge(appCfg)
	currentCfg := loader.Get()
	if currentCfg == nil {
		loader.Set(mergedCfg)
		return nil
	}
	*currentCfg = *mergedCfg
	loader.Set(currentCfg)
	return nil
}
