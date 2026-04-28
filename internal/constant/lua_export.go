package constant

// LuaExport 返回注入到Lua中的常量和运行时变量快照。
func LuaExport() map[string]interface{} {
	out := map[string]interface{}{
		"AppName":            AppName,
		"Copyright":          Copyright,
		"JSON":               JSON,
		"YAML":               YAML,
		"HTTP1":              HTTP1,
		"HTTP2":              HTTP2,
		"HTTP3":              HTTP3,
		"NetIO_NET":          NetIO_NET,
		"NetIO_NBIO":         NetIO_NBIO,
		"ProxyMode_HTTP":     ProxyMode_HTTP,
		"ProxyMode_FastHTTP": ProxyMode_FastHTTP,
		"SchemeSandwich":     SchemeSandwich,
		"SchemeGrpc":         SchemeGrpc,
		"SchemeGrpcWeb":      SchemeGrpcWeb,
		"Localhost":          Localhost,
		"ZeroHost":           ZeroHost,
		"HTTPPort":           HTTPPort,
		"HTTPSPort":          HTTPSPort,
		"Backend":            Backend,
		"Frontend":           Frontend,
		"FrontendFromConf":   FrontendFromConf,
		"BackendFromConf":    BackendFromConf,
		"FrontendType":       FrontendType,
		"BackendType":        BackendType,
		"CustomType":         CustomType,
		"SyncTime":           SyncTime,
		"LIMIT":              LIMIT,
		"RESET":              RESET,
		"BreakerLimit":       BreakerLimit,
		"BreakerReset":       BreakerReset,
	}
	return out
}
