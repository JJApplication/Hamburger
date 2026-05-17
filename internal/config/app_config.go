package config

import (
	"Hamburger/internal/config/backproxy_config"
	"Hamburger/internal/config/core_config"
	"Hamburger/internal/config/exp_config"
	"Hamburger/internal/config/frontproxy_config"
	"Hamburger/internal/config/svr_config"
	"os"

	"Hamburger/internal/constant"
	"Hamburger/internal/json"
)

// AppConfig 配置文件格式模型
type AppConfig struct {
	PxyBackendFile  string       `yaml:"pxy_backend_file" json:"pxy_backend_file"` // 配置文件路径
	PxyFrontendFile string       `yaml:"pxy_frontend_file" json:"pxy_frontend_file"`
	DomainMap       string       `yaml:"domain_map" json:"domain_map"` // 域名映射文件
	Plugin          PluginConfig `yaml:"plugin" json:"plugin"`         // 插件配置

	CoreProxy       core_config.ProxyConfig      `yaml:"proxy" json:"proxy"` // 核心代理配置
	ErrorConfig     core_config.ProxyErrorConfig `yaml:"error_config" json:"error_config"`
	Servers         []core_config.ServerConfig   `yaml:"servers" json:"servers"`       // 服务器配置列表
	Middleware      core_config.MiddlewareConfig `yaml:"middleware" json:"middleware"` // 中间件配置列表
	Features        core_config.FeatureConfig    `yaml:"features" json:"features"`     // 功能特性配置
	GRPC            GRPCConfig                   `yaml:"grpc" json:"grpc"`             // gRPC服务配置
	ApiServerConfig svr_config.ApiServerConfig   `yaml:"api_server_config" json:"api_server_config"`
	Database        core_config.DatabaseConfig   `yaml:"database" json:"database"`           // 数据库配置
	Security        core_config.SecurityConfig   `yaml:"security" json:"security"`           // 安全配置
	ProxyHeader     core_config.ProxyHeader      `yaml:"proxy_header" json:"proxy_header"`   // 内置的代理头配置
	Log             core_config.LogConfig        `yaml:"log" json:"log"`                     // 日志配置
	Module          []core_config.ModuleConfig   `yaml:"module" json:"module"`               // 模块
	Stat            svr_config.StatConfig        `yaml:"stat" json:"stat"`                   // 状态统计配置
	Latency         svr_config.LatencyConfig     `yaml:"latency" json:"latency"`             // 延迟统计配置
	CustomHeader    map[string]string            `yaml:"custom_header" json:"custom_header"` // 自定义Header
	PreAuthConfig   svr_config.PreAuthConfig     `yaml:"pre_auth_config" json:"pre_auth_config"`
	NotifyConfig    svr_config.NotifyConfig      `yaml:"notify_config" json:"notify_config"` // 通知系统配置
	Lua             svr_config.LuaConfig         `yaml:"lua" json:"lua"`                     // Lua脚本配置
	Syncer          Syncer                       `yaml:"syncer" json:"syncer"`               // 定时器时间
	Debug           bool                         `yaml:"debug" json:"debug"`                 // 调试模式
	DevMode         bool                         `yaml:"dev_mode" json:"dev_mode"`           // 开发模式
	PProf           core_config.PProf            `yaml:"pprof" json:"pprof"`
	MaxCores        int                          `yaml:"max_cores" json:"max_cores"`

	// 第二优先级
	PxyBackend  backproxy_config.PxyBackendConfig `yaml:"pxy_backend" json:"pxy_backend"`
	PxyFrontend frontproxy_config.PxyFrontConfig  `yaml:"pxy_frontend" json:"pxy_frontend"`

	ExpConfig exp_config.ExpConfig `yaml:"exp_config" json:"exp_config"`
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *AppConfig {
	return &AppConfig{
		Plugin: PluginConfig{
			Enabled: true,
			Root:    "plugins",
			Plugins: []PluginItem{},
		},
		Lua: svr_config.LuaConfig{
			Enabled:     true,
			ScriptsRoot: "lua_scripts",
		},
		Servers: []core_config.ServerConfig{
			{
				Name:           "http-server",
				Host:           "0.0.0.0",
				Port:           80,
				Protocol:       "http",
				Enabled:        true,
				MaxRequestBody: 32 * 1024 * 1024, // 32MB
			},
			{
				Name:           "https-server",
				Host:           "0.0.0.0",
				Port:           443,
				Protocol:       "https",
				Enabled:        false,
				MaxRequestBody: 32 * 1024 * 1024, // 32MB
				TLS: &core_config.TLSConfig{
					CertMap: map[string]core_config.CertConfig{
						"renj.io": {
							[]string{""},
							"/path/to/cert.pem",
							"/path/to/key.pem",
						},
					},
					AutoTLS: false,
				},
			},
		},
		Middleware: core_config.MiddlewareConfig{
			Gzip: core_config.GzipConfig{
				Enabled: true,
				Level:   6,
				Types:   []string{"text/html", "text/css", "text/javascript", "application/json"},
			},
			Zstd: core_config.ZstdConfig{
				Enabled:   false,
				Level:     3,
				Types:     []string{"text/html", "text/css", "text/javascript", "application/json"},
				Threshold: 1024,
			},
			FailResponse: core_config.FailResponse{
				Enabled: false,
				Code:    []int{500},
			},
		},
		Features: core_config.FeatureConfig{
			HTTP3: core_config.HTTP3Config{
				Enabled:        false,
				MaxConnections: 1000,
				IdleTimeout:    60,
				KeepAlive:      30,
			},
			WebSocket: core_config.WebSocketConfig{
				Enabled:        false,
				PingInterval:   10,
				PongTimeout:    10,
				MaxMessageSize: 1048576, // 1MB
				BufferSize:     1024,
			},
			Cache: core_config.CacheConfig{
				Enabled:  true,
				Size:     1000,
				TTL:      60,
				Strategy: "lru",
			},
		},
		GRPC: GRPCConfig{
			Enabled: false,
			Address: "",
		},
		ApiServerConfig: svr_config.ApiServerConfig{
			Enabled: false,
			Host:    "0.0.0.0",
			Port:    8282,
			HTTP2: svr_config.APIHTTP2Config{
				Enabled:  false,
				Insecure: true,
			},
			JWT: svr_config.JWTConfig{
				Enabled:        false,
				TokenHeader:    "Authorization",
				AllowedMethods: []string{"HS256"},
			},
			BBolt: svr_config.APIBBoltConfig{
				Enabled:         true,
				File:            "bblot.db",
				TimeoutSeconds:  1,
				UserBucket:      "api_users",
				DefaultUsername: "admin",
				DefaultPassword: "admin",
			},
		},
		Database: core_config.DatabaseConfig{
			Mongo: core_config.MongoConfig{
				URL:      "mongodb://localhost:27017",
				Database: "sandwich",
				Timeout:  10,
			},
			Influx: core_config.InfluxConfig{
				Enabled:  false,
				URL:      "http://localhost:8086",
				Token:    "",
				Org:      "sandwich",
				Bucket:   "metrics",
				Password: "",
			},
		},
		Security: core_config.SecurityConfig{
			StrictMode: false,
			AllowIPs:   []string{},
			DenyIPs:    []string{},
			RateLimit:  1000,
		},
		ProxyHeader: core_config.ProxyHeader{
			TraceId:            "X-Gateway-Trace-Id",
			FrontendHostHeader: "X-Proxy-Internal-Host",
			BackendHeader:      "X-Proxy-Internal-Local",
			ProxyApp:           "X-Proxy-Backend",
		},
		CustomHeader: map[string]string{
			"Proxy-Server":    constant.AppName,
			"Proxy-Copyright": constant.Copyright,
		},
		PreAuthConfig: svr_config.PreAuthConfig{
			Enabled:           false,
			PassThroughHeader: "X-Gateway-Pre-Auth",
			PassThroughValue:  "verified",
			OAuth2: svr_config.OAuth2AuthConfig{
				Enabled:        false,
				TokenHeader:    "Authorization",
				TimeoutSeconds: 3,
			},
			Basic: svr_config.BasicAuthConfig{
				Enabled: false,
				Users:   []svr_config.BasicUserConfig{},
			},
			JWT: svr_config.JWTAuthConfig{
				Enabled:        false,
				TokenHeader:    "Authorization",
				AllowedMethods: []string{"HS256"},
			},
		},
		NotifyConfig: svr_config.NotifyConfig{
			Enabled: false,
			Queue: svr_config.NotifyQueueConfig{
				Topic:  "gateway.notify",
				Buffer: 128,
			},
			Mail: svr_config.NotifyMailConfig{
				Provider: "smtp",
				From:     "",
				SMTP: svr_config.NotifySMTPConfig{
					Host: "",
					Port: 587,
					TLS:  true,
				},
				POP3: svr_config.NotifyPOP3Config{
					Host: "",
					Port: 995,
					TLS:  true,
				},
				IMAP: svr_config.NotifyIMAPConfig{
					Host: "",
					Port: 993,
					TLS:  true,
				},
			},
			DefaultRecipients: []string{},
		},
		Stat: svr_config.StatConfig{
			DBFile: "stat.db",
			Sequence: svr_config.SequenceConfig{
				Enabled:  true,
				Interval: 3600,
			},
		},
	}
}

// CreateConfig 生成默认配置文件
func CreateConfig() error {
	config := GetDefaultConfig()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("config.default.json", data, 0644)
}
