package config

import (
	"Hamburger/internal/constant"
	"Hamburger/internal/json"
	"os"
)

// AppConfig 配置文件格式模型
type AppConfig struct {
	PxyBackendFile  string       `yaml:"pxy_backend_file" json:"pxy_backend_file"` // 配置文件路径
	PxyFrontendFile string       `yaml:"pxy_frontend_file" json:"pxy_frontend_file"`
	DomainMap       string       `yaml:"domain_map" json:"domain_map"` // 域名映射文件
	Plugin          PluginConfig `yaml:"plugin" json:"plugin"`         // 插件配置

	CoreProxy     ProxyConfig       `yaml:"proxy" json:"proxy"` // 核心代理配置
	ErrorConfig   ProxyErrorConfig  `yaml:"error_config" json:"error_config"`
	Servers       []ServerConfig    `yaml:"servers" json:"servers"`             // 服务器配置列表
	Middleware    MiddlewareConfig  `yaml:"middleware" json:"middleware"`       // 中间件配置列表
	Features      FeatureConfig     `yaml:"features" json:"features"`           // 功能特性配置
	GRPC          GRPCConfig        `yaml:"grpc" json:"grpc"`                   // gRPC服务配置
	Database      DatabaseConfig    `yaml:"database" json:"database"`           // 数据库配置
	Security      SecurityConfig    `yaml:"security" json:"security"`           // 安全配置
	ProxyHeader   ProxyHeader       `yaml:"proxy_header" json:"proxy_header"`   // 内置的代理头配置
	Log           LogConfig         `yaml:"log" json:"log"`                     // 日志配置
	Module        []ModuleConfig    `yaml:"module" json:"module"`               // 模块
	Stat          StatConfig        `yaml:"stat" json:"stat"`                   // 状态统计配置
	Latency       LatencyConfig     `yaml:"latency" json:"latency"`             // 延迟统计配置
	CustomHeader  map[string]string `yaml:"custom_header" json:"custom_header"` // 自定义Header
	PreAuthConfig PreAuthConfig     `yaml:"pre_auth_config" json:"pre_auth_config"`
	NotifyConfig  NotifyConfig      `yaml:"notify_config" json:"notify_config"` // 通知系统配置
	Syncer        Syncer            `yaml:"syncer" json:"syncer"` // 定时器时间
	Debug         bool              `yaml:"debug" json:"debug"`   // 调试模式
	PProf         PProf             `yaml:"pprof" json:"pprof"`
	MaxCores      int               `yaml:"max_cores" json:"max_cores"`

	// 第二优先级
	PxyBackend  PxyBackendConfig `yaml:"pxy_backend" json:"pxy_backend"`
	PxyFrontend PxyFrontConfig   `yaml:"pxy_frontend" json:"pxy_frontend"`

	ExpConfig ExpConfig `yaml:"exp_config" json:"exp_config"`
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *AppConfig {
	return &AppConfig{
		Plugin: PluginConfig{
			Enabled: true,
			Root:    "plugins",
			Plugins: []PluginItem{},
		},
		Servers: []ServerConfig{
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
				TLS: &TLSConfig{
					CertMap: map[string]CertConfig{
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
		Middleware: MiddlewareConfig{
			Gzip: GzipConfig{
				Enabled: true,
				Level:   6,
				Types:   []string{"text/html", "text/css", "text/javascript", "application/json"},
			},
		},
		Features: FeatureConfig{
			HTTP3: HTTP3Config{
				Enabled:        false,
				MaxConnections: 1000,
				IdleTimeout:    60,
				KeepAlive:      30,
			},
			WebSocket: WebSocketConfig{
				Enabled:        false,
				PingInterval:   10,
				PongTimeout:    10,
				MaxMessageSize: 1048576, // 1MB
				BufferSize:     1024,
			},
			Cache: CacheConfig{
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
		Database: DatabaseConfig{
			Mongo: MongoConfig{
				URL:      "mongodb://localhost:27017",
				Database: "sandwich",
				Timeout:  10,
			},
			Influx: InfluxConfig{
				Enabled:  false,
				URL:      "http://localhost:8086",
				Token:    "",
				Org:      "sandwich",
				Bucket:   "metrics",
				Password: "",
			},
		},
		Security: SecurityConfig{
			StrictMode: false,
			AllowIPs:   []string{},
			DenyIPs:    []string{},
			RateLimit:  1000,
		},
		ProxyHeader: ProxyHeader{
			TraceId:            "X-Gateway-Trace-Id",
			FrontendHostHeader: "X-Proxy-Internal-Host",
			BackendHeader:      "X-Proxy-Internal-Local",
			ProxyApp:           "X-Proxy-Backend",
		},
		CustomHeader: map[string]string{
			"Proxy-Server":    constant.AppName,
			"Proxy-Copyright": constant.Copyright,
		},
		PreAuthConfig: PreAuthConfig{
			Enabled:           false,
			PassThroughHeader: "X-Gateway-Pre-Auth",
			PassThroughValue:  "verified",
			OAuth2: OAuth2AuthConfig{
				Enabled:        false,
				TokenHeader:    "Authorization",
				TimeoutSeconds: 3,
			},
			Basic: BasicAuthConfig{
				Enabled: false,
				Users:   []BasicUserConfig{},
			},
			JWT: JWTAuthConfig{
				Enabled:        false,
				TokenHeader:    "Authorization",
				AllowedMethods: []string{"HS256"},
			},
		},
		NotifyConfig: NotifyConfig{
			Enabled: false,
			Queue: NotifyQueueConfig{
				Topic:  "gateway.notify",
				Buffer: 128,
			},
			Mail: NotifyMailConfig{
				Provider: "smtp",
				From:     "",
				SMTP: NotifySMTPConfig{
					Host: "",
					Port: 587,
					TLS:  true,
				},
				POP3: NotifyPOP3Config{
					Host: "",
					Port: 995,
					TLS:  true,
				},
				IMAP: NotifyIMAPConfig{
					Host: "",
					Port: 993,
					TLS:  true,
				},
			},
			DefaultRecipients: []string{},
		},
		Stat: StatConfig{
			DBFile: "stat.db",
			Sequence: SequenceConfig{
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
