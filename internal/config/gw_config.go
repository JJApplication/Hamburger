package config

import (
	"Hamburger/internal/constant"
	"Hamburger/internal/json"
	"os"
	"strings"
)

type ProxyConfig struct {
	FlushInterval   int64  `yaml:"flush_interval" json:"flush_interval"`
	BufSize         int    `yaml:"buf_size" json:"buf_size"`
	Transport       string `yaml:"transport" json:"transport"`                   // 传统 | fast
	ProxyMode       string `yaml:"proxy_mode" json:"proxy_mode"`                 // 代理模式: http | fasthttp
	MaxConnsPerHost int    `yaml:"max_conns_per_host" json:"max_conns_per_host"` // 每个主机最大连接数
	IdleConnTimeout int    `yaml:"idle_conn_timeout" json:"idle_conn_timeout"`   // 空闲连接超时
}

// ServerConfig 服务器配置结构体
type ServerConfig struct {
	Name           string         `yaml:"name" json:"name"`                         // 服务器名称
	Host           string         `yaml:"host" json:"host"`                         // 监听主机地址
	Port           int            `yaml:"port" json:"port"`                         // 监听端口
	UseHttp2       bool           `yaml:"use_http2" json:"use_http2"`               // 使用HTTP2
	Protocol       string         `yaml:"protocol" json:"protocol"`                 // 协议类型: http, https, http3
	Enabled        bool           `yaml:"enabled" json:"enabled"`                   // 是否启用
	MaxRequestBody int64          `yaml:"max_request_body" json:"max_request_body"` // 最大请求体大小（字节）
	TLS            *TLSConfig     `yaml:"tls,omitempty" json:"tls,omitempty"`       // TLS配置
	DomainConfig   []DomainConfig `yaml:"domains" json:"domains"`                   // 域名绑定配置
	// 后端服务器映射
	ServiceDomains []string `yaml:"service_domains" json:"service_domains"` // 后端服务域名
	Service        string   `yaml:"service" json:"service"`                 // 后端服务
	// 扩展配置
	ReadTimeout       int64 `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout      int64 `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout       int64 `yaml:"idle_timeout" json:"idle_timeout"`
	ReadHeaderTimeout int64 `yaml:"read_header_timeout" json:"read_header_timeout"`
	MaxHeaderBytes    int64 `yaml:"max_header_bytes" json:"max_header_bytes"`
}

// TLSConfig TLS证书配置结构体
type TLSConfig struct {
	CertFile string `yaml:"cert_file" json:"cert_file"` // 证书文件路径
	KeyFile  string `yaml:"key_file" json:"key_file"`   // 私钥文件路径
	AutoTLS  bool   `yaml:"auto_tls" json:"auto_tls"`   // 是否启用自动TLS
}

// DomainConfig 域名配置结构体
type DomainConfig struct {
	Domains        []string `yaml:"domains" json:"domains"`                 // 域名
	UseTLS         bool     `yaml:"use_tls" json:"use_tls"`                 // 监听在https
	AutoRedirect   bool     `yaml:"auto_redirect" json:"auto_redirect"`     // 自动重定向
	UseWebsocket   bool     `yaml:"use_websocket" json:"use_websocket"`     // 开启websocket
	HSTSMaxAge     int      `yaml:"hsts_max_age" json:"hsts_max_age"`       // HSTS最大生存时间（秒），0表示不设置HSTS
	HSTSSubdomains bool     `yaml:"hsts_subdomains" json:"hsts_subdomains"` // HSTS是否包含子域名
	HSTSPreload    bool     `yaml:"hsts_preload" json:"hsts_preload"`       // HSTS是否启用预加载
}

// BreakConfig 熔断配置
type BreakConfig struct {
	Bucket   int `yaml:"bucket" json:"bucket"`       // 桶数量
	MaxError int `yaml:"max_error" json:"max_error"` // 最大允许错误
	Reset    int `yaml:"reset" json:"reset"`         // 重置时间
}

// FlowControlRule 流控规则配置结构体
type FlowControlRule struct {
	Name        string      `yaml:"name" json:"name"`               // 规则名称
	Enabled     bool        `yaml:"enabled" json:"enabled"`         // 是否启用
	Priority    int         `yaml:"priority" json:"priority"`       // 优先级，数字越小优先级越高
	MatchType   string      `yaml:"match_type" json:"match_type"`   // 匹配类型: host, header, ip
	MatchValue  string      `yaml:"match_value" json:"match_value"` // 匹配值
	HeaderKey   string      `yaml:"header_key" json:"header_key"`   // 当match_type为header时的header键名
	Limits      []RateLimit `yaml:"limits" json:"limits"`           // 速率限制配置列表
	Action      string      `yaml:"action" json:"action"`           // 限流动作: block, delay
	Description string      `yaml:"description" json:"description"` // 规则描述
}

// RateLimit 速率限制配置结构体
type RateLimit struct {
	Requests int    `yaml:"requests" json:"requests"` // 允许的请求数
	Window   string `yaml:"window" json:"window"`     // 时间窗口，如 "100s"、"10min"
	Unit     string `yaml:"unit" json:"unit"`         // 时间单位: s, min
	Mode     string `yaml:"mode" json:"mode"`         // 限流模式: fixed, leaky, token, sliding
}

// FlowControlConfig 流控配置结构体
type FlowControlConfig struct {
	Enabled     bool              `yaml:"enabled" json:"enabled"`           // 是否启用流控
	GlobalLimit RateLimit         `yaml:"global_limit" json:"global_limit"` // 全局限流配置
	Rules       []FlowControlRule `yaml:"rules" json:"rules"`               // 流控规则列表
	Recording   FlowRecordConfig  `yaml:"recording" json:"recording"`       // 流控记录配置
}

// FlowRecordConfig 流控记录配置结构体
type FlowRecordConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`                   // 是否启用限流记录
	RecordBlocked   bool   `yaml:"record_blocked" json:"record_blocked"`     // 是否记录被限流的请求
	RecordAllowed   bool   `yaml:"record_allowed" json:"record_allowed"`     // 是否记录通过的请求
	StorageType     string `yaml:"storage_type" json:"storage_type"`         // 存储类型: influx, mongo, file
	RetentionPeriod string `yaml:"retention_period" json:"retention_period"` // 数据保留期
}

// MiddlewareConfig 中间件配置结构体
type MiddlewareConfig struct {
	Name    string                 `yaml:"name" json:"name"`       // 中间件名称
	Enabled bool                   `yaml:"enabled" json:"enabled"` // 是否启用
	Order   int                    `yaml:"order" json:"order"`     // 执行顺序
	Config  map[string]interface{} `yaml:"config" json:"config"`   // 中间件配置参数
}

func (m *MiddlewareConfig) GetInt(key string) int {
	v, ok := m.Config[key]
	if !ok {
		return 0
	}
	value, ok := v.(float64)
	if !ok {
		return 0
	}
	return int(value)
}

func (m *MiddlewareConfig) GetString(key string) string {
	v, ok := m.Config[key]
	if !ok {
		return ""
	}
	value, ok := v.(string)
	if !ok {
		return ""
	}
	return value
}

func (m *MiddlewareConfig) GetBool(key string) bool {
	v, ok := m.Config[key]
	if !ok {
		return false
	}
	value, ok := v.(bool)
	if ok {
		return value
	}
	switch v.(type) {
	case float64:
		return v.(float64) > 0
	case bool:
		return v.(bool)
	case string:
		vs := v.(string)
		if strings.ToLower(vs) == "true" || strings.ToLower(vs) == "yes" {
			return true
		}
		return false
	default:
		return false
	}
}

// FeatureConfig 功能特性配置结构体
type FeatureConfig struct {
	HTTP3        HTTP3Config       `yaml:"http3" json:"http3"`         // HTTP/3配置
	WebSocket    WebSocketConfig   `yaml:"websocket" json:"websocket"` // WebSocket配置
	Cache        CacheConfig       `yaml:"cache" json:"cache"`         // 缓存配置
	Gzip         GzipConfig        `yaml:"gzip" json:"gzip"`           // Gzip压缩配置
	NoCache      bool              `yaml:"no_cache" json:"no_cache"`
	SecureHeader bool              `yaml:"secure_header" json:"secure_header"` // 安全响应头
	Trace        TraceConfig       `yaml:"trace" json:"trace"`                 // 请求跟踪
	AutoCert     AutoCertConfig    `yaml:"auto_cert" json:"auto_cert"`         // 自动证书配置
	GrpcProxy    GrpcProxyConfig   `yaml:"grpc_proxy" json:"grpc_proxy"`       // gRPC代理配置
	FlowControl  FlowControlConfig `yaml:"flow_control" json:"flow_control"`   // 流控配置
	Break        BreakConfig       `yaml:"break" json:"break"`                 // 熔断配置
}

// HTTP3Config HTTP/3协议配置结构体
type HTTP3Config struct {
	Enabled        bool `yaml:"enabled" json:"enabled"`                 // 是否启用HTTP/3
	MaxConnections int  `yaml:"max_connections" json:"max_connections"` // 最大连接数
	IdleTimeout    int  `yaml:"idle_timeout" json:"idle_timeout"`       // 空闲超时时间
	KeepAlive      int  `yaml:"keep_alive" json:"keep_alive"`           // 保活时间
}

// WebSocketConfig WebSocket协议配置结构体
type WebSocketConfig struct {
	Enabled        bool  `yaml:"enabled" json:"enabled"`                   // 是否启用WebSocket
	PingInterval   int   `yaml:"ping_interval" json:"ping_interval"`       // 心跳间隔
	PongTimeout    int   `yaml:"pong_timeout" json:"pong_timeout"`         // 心跳响应超时
	MaxMessageSize int64 `yaml:"max_message_size" json:"max_message_size"` // 最大消息大小
	BufferSize     int   `yaml:"buffer_size" json:"buffer_size"`           // 缓冲区大小
}

// GzipConfig Gzip压缩配置结构体
type GzipConfig struct {
	Enabled   bool     `yaml:"enabled" json:"enabled"`     // 是否启用Gzip压缩
	Level     int      `yaml:"level" json:"level"`         // 压缩级别 1-9
	Types     []string `yaml:"types" json:"types"`         // 压缩的MIME类型列表
	Threshold int      `yaml:"threshold" json:"threshold"` // 开启压缩的阈值
}

// CacheConfig 缓存配置结构体
type CacheConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`   // 是否启用缓存
	Size     int    `yaml:"size" json:"size"`         // 缓存大小
	TTL      int    `yaml:"ttl" json:"ttl"`           // 缓存过期时间
	Strategy string `yaml:"strategy" json:"strategy"` // 缓存策略: lru, lfu, fifo
}

type TraceConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	TraceId string `yaml:"trace_id" json:"trace_id"`
}

// AutoCertConfig 自动证书配置结构体
type AutoCertConfig struct {
	Email   string   `yaml:"email" json:"email"`     // 注册邮箱
	Domains []string `yaml:"domains" json:"domains"` // 域名列表
}

type GrpcProxyConfig struct {
	Enabled    bool     `yaml:"enabled" json:"enabled"`         // 是否启用gRPC代理
	Hosts      []string `yaml:"hosts" json:"hosts"`             // 目标gRPC主机列表
	GrpcHeader string   `yaml:"grpc_header" json:"grpc_header"` // gRPC识别请求头
	GrpcAddr   string   `yaml:"grpc_addr" json:"grpc_addr"`     // 目标gRPC地址
}

// DatabaseConfig 数据库配置结构体
type DatabaseConfig struct {
	Mongo  MongoConfig  `yaml:"mongo" json:"mongo"`   // MongoDB配置
	Influx InfluxConfig `yaml:"influx" json:"influx"` // InfluxDB配置
}

// MongoConfig MongoDB配置结构体
type MongoConfig struct {
	URL      string `yaml:"url" json:"url"`           // MongoDB连接URL
	Database string `yaml:"database" json:"database"` // 数据库名称
	Timeout  int    `yaml:"timeout" json:"timeout"`   // 连接超时时间
}

// InfluxConfig InfluxDB配置结构体
type InfluxConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`   // 是否启用InfluxDB
	URL      string `yaml:"url" json:"url"`           // InfluxDB连接URL
	Token    string `yaml:"token" json:"token"`       // 访问令牌
	Org      string `yaml:"org" json:"org"`           // 组织名称
	Bucket   string `yaml:"bucket" json:"bucket"`     // 存储桶名称
	Password string `yaml:"password" json:"password"` // 密码
}

// MonitorConfig 监控配置结构体
type MonitorConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`       // 是否启用监控
	Port       int    `yaml:"port" json:"port"`             // 监控端口
	Path       string `yaml:"path" json:"path"`             // 监控路径
	Interval   int    `yaml:"interval" json:"interval"`     // 监控间隔
	Prometheus bool   `yaml:"prometheus" json:"prometheus"` // 是否启用Prometheus
}

type cors struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Method  []string `yaml:"method" json:"method"`
	Origin  []string `yaml:"origin" json:"origin"` // 默认*
	Header  []string `yaml:"header" json:"header"`
}

// SecurityConfig 安全配置结构体
type SecurityConfig struct {
	StrictMode bool     `yaml:"strict_mode" json:"strict_mode"` // 严格模式
	AllowIPs   []string `yaml:"allow_ips" json:"allow_ips"`     // 允许的IP列表
	DenyIPs    []string `yaml:"deny_ips" json:"deny_ips"`       // 拒绝的IP列表
	RateLimit  int      `yaml:"rate_limit" json:"rate_limit"`   // 速率限制
	CORS       cors     `yaml:"cors" json:"cors"`               // cors策略

	HSTS             bool         `yaml:"hsts" json:"hsts"`                     // HSTS策略
	HSTSSubdomain    bool         `yaml:"hsts_subdomain" json:"hsts_subdomain"` // 包含子域名
	HSTSPreload      bool         `yaml:"hsts_preload" json:"hsts_preload"`     // 预加载
	XssProtection    bool         `yaml:"xss_protection" json:"xss_protection"` // XSS保护
	IFrameProtection bool         `yaml:"iframe_protection" json:"iframe_protection"`
	SameSite         bool         `yaml:"same_site" json:"same_site"`         // 同源策略
	ImageProtect     ImageProtect `yaml:"image_protect" json:"image_protect"` // 图片防盗链
}

type FrontProxyConfig struct {
	GrpcAddr     string `yaml:"grpc_addr" json:"grpc_addr"`
	FrontendFlag string `yaml:"frontend_flag" json:"frontend_flag"`
	FrontendHost string `yaml:"frontend_host" json:"frontend_host"`
	FrontendPort int    `yaml:"frontend_port" json:"frontend_port"`
}

type ProxyHeader struct {
	TraceId            string `yaml:"trace_id" json:"trace_id"`                         // traceId头
	FrontendHostHeader string `yaml:"frontend_host_header" json:"frontend_host_header"` // 前端服务真实HOST
	BackendHeader      string `yaml:"backend_header" json:"backend_header"`             // 区分后端服务标识
	ProxyApp           string `yaml:"proxy_app" json:"proxy_app"`                       // 要转到的后端服务标识
}

type LogConfig struct {
	LogLevel string `yaml:"log_level" json:"log_level"`
	LogFile  string `yaml:"log_file" json:"log_file"`
	Color    bool   `yaml:"color" json:"color"`
}

type ModuleConfig struct {
	Name string `yaml:"name" json:"name"`
	Path string `yaml:"path" json:"path"`
	Type string `yaml:"type" json:"type"` // mod | pre 可选 会根据Lookup自动匹配
}

type StatConfig struct {
	DBFile       string         `yaml:"db_file" json:"db_file"`       // SQLite数据库文件路径
	UseDB        bool           `yaml:"use_db" json:"use_db"`         // 传统stat是否使用数据库记录
	Compatible   bool           `yaml:"compatible" json:"compatible"` // 兼容加载file到DB中
	Enabled      bool           `yaml:"enabled" json:"enabled"`       // 是否开启服务器 不开启服务器也会统计
	Host         string         `yaml:"host" json:"host"`
	Port         int            `yaml:"port" json:"port"`
	EnableStat   bool           `yaml:"enable_stat" json:"enable_stat"` // 开启统计
	SyncDuration int            `yaml:"sync_duration" json:"sync_duration"`
	SaveDuration int            `yaml:"save_duration" json:"save_duration"`
	SaveFile     string         `json:"save_file"`
	GeoFile      string         `json:"geo_file"`
	DomainFile   string         `json:"domain_file"`
	GeoDB        string         `json:"geo_db"`                   // geo数据库
	Sequence     SequenceConfig `yaml:"sequence" json:"sequence"` // 时序统计配置
}

// SequenceConfig 时序统计配置结构体
// 用于配置是否启用时序统计、数据库文件路径以及统计时间间隔
type SequenceConfig struct {
	Enabled  bool `yaml:"enabled" json:"enabled"`   // 是否启用时序统计
	Interval int  `yaml:"interval" json:"interval"` // 时序间隔，例如"1h"表示每小时一个时序表
}

type ImageProtect struct {
	ImageType    []string `yaml:"image_type" json:"image_type"`       // 过滤的图片类型
	AllowReferer []string `yaml:"allow_referer" json:"allow_referer"` // 允许的请求头
}

type PProf struct {
	Enable bool `yaml:"enable" json:"enable"`
	Port   int  `yaml:"port" json:"port"`
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *AppConfig {
	return &AppConfig{
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
					CertFile: "/path/to/cert.pem",
					KeyFile:  "/path/to/key.pem",
					AutoTLS:  false,
				},
			},
		},
		Middleware: []MiddlewareConfig{
			{
				Name:    "limiter",
				Enabled: true,
				Order:   1,
				Config: map[string]interface{}{
					"limit": 100,
					"reset": 60,
				},
			},
			{
				Name:    "breaker",
				Enabled: true,
				Order:   2,
				Config: map[string]interface{}{
					"limit": 20,
					"reset": 10,
				},
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
			Gzip: GzipConfig{
				Enabled: true,
				Level:   6,
				Types:   []string{"text/html", "text/css", "text/javascript", "application/json"},
			},
			Cache: CacheConfig{
				Enabled:  true,
				Size:     1000,
				TTL:      60,
				Strategy: "lru",
			},
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
			"Proxy-Server":    constant.Sandwich,
			"Proxy-Copyright": constant.Copyright,
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
