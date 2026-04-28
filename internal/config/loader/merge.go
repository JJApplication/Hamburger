package loader

import "Hamburger/internal/config"

// Merge 合并配置 配置文件中的配置优先级更好
func Merge(appConfig *config.AppConfig) *config.Config {
	if appConfig == nil {
		return nil
	}

	conf := &config.Config{
		CoreProxy:       appConfig.CoreProxy,
		ErrorConfig:     appConfig.ErrorConfig,
		Servers:         appConfig.Servers,
		Middleware:      appConfig.Middleware,
		Features:        appConfig.Features,
		GRPC:            appConfig.GRPC,
		ApiServerConfig: appConfig.ApiServerConfig,
		Database:        appConfig.Database,
		Security:        appConfig.Security,
		ProxyHeader:     appConfig.ProxyHeader,
		Log:             appConfig.Log,
		Module:          appConfig.Module,
		Stat:            appConfig.Stat,
		CustomHeader:    appConfig.CustomHeader,
		Plugin:          appConfig.Plugin,
		Lua:             appConfig.Lua,
		Syncer:          appConfig.Syncer,
		Debug:           appConfig.Debug,
		PProf:           appConfig.PProf,
		MaxCores:        appConfig.MaxCores,
		PxyBackend:      appConfig.PxyBackend,
		PxyFrontend:     appConfig.PxyFrontend,
		ExpConfig:       appConfig.ExpConfig,
		PreAuthConfig:   appConfig.PreAuthConfig,
		NotifyConfig:    appConfig.NotifyConfig,
	}

	if appConfig.PxyFrontendFile != "" {
		fc, err := config.LoadFrontConfig(appConfig.PxyFrontendFile)
		if err == nil {
			conf.PxyFrontend = fc
		}
	}

	if appConfig.PxyBackendFile != "" {
		bc, err := config.LoadBackendConfig(appConfig.PxyBackendFile)
		if err == nil {
			conf.PxyBackend = bc
		}
	}

	return conf
}
