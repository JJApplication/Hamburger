package config

// Merge 合并配置 配置文件中的配置优先级更好
func Merge(appConfig *AppConfig) *Config {
	if appConfig == nil {
		return nil
	}

	conf := &Config{
		CoreProxy:    appConfig.CoreProxy,
		Servers:      appConfig.Servers,
		Middleware:   appConfig.Middleware,
		Features:     appConfig.Features,
		Database:     appConfig.Database,
		Security:     appConfig.Security,
		ProxyHeader:  appConfig.ProxyHeader,
		Log:          appConfig.Log,
		Module:       appConfig.Module,
		Stat:         appConfig.Stat,
		CustomHeader: appConfig.CustomHeader,
		Syncer:       appConfig.Syncer,
		Debug:        appConfig.Debug,
		PProf:        appConfig.PProf,
		MaxCores:     appConfig.MaxCores,
		PxyBackend:   appConfig.PxyBackend,
		PxyFrontend:  appConfig.PxyFrontend,
	}

	if appConfig.PxyFrontendFile != "" {
		fc, err := LoadFrontConfig(appConfig.PxyFrontendFile)
		if err == nil {
			conf.PxyFrontend = fc
		}
	}

	if appConfig.PxyBackendFile != "" {
		bc, err := LoadBackendConfig(appConfig.PxyBackendFile)
		if err == nil {
			conf.PxyBackend = bc
		}
	}

	return conf
}
