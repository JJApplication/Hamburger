package config

type ProxyConfig struct {
	FlushInterval   int64  `yaml:"flush_interval" json:"flush_interval"`
	BufSize         int    `yaml:"buf_size" json:"buf_size"`
	Transport       string `yaml:"transport" json:"transport"`                   // 传统 | fast
	ProxyMode       string `yaml:"proxy_mode" json:"proxy_mode"`                 // 代理模式: http | fasthttp
	NetIO           string `yaml:"net_io" json:"net_io"`                         // fasthttp | nbio | http
	MaxConnsPerHost int    `yaml:"max_conns_per_host" json:"max_conns_per_host"` // 每个主机最大连接数
	IdleConnTimeout int    `yaml:"idle_conn_timeout" json:"idle_conn_timeout"`   // 空闲连接超时
	EnableHTTP3     bool   `yaml:"enable_http3" json:"enable_http3"`             // 是否启用HTTP/3
}

// ProxyErrorConfig 网关错误配置
type ProxyErrorConfig struct {
	ErrorMode       string         `yaml:"error_mode" json:"error_mode"`               // 传统text格式错误页面 | html页面
	ErrorPage       map[int]string `yaml:"error_page" json:"error_page"`               // 不同状态码对应的页面
	EnablePageCache bool           `yaml:"enable_page_cache" json:"enable_page_cache"` // 开启缓存
}

// ServerConfig 服务器配置结构体
type ServerConfig struct {
	Name           string         `yaml:"name" json:"name"`                         // 服务器名称
	Host           string         `yaml:"host" json:"host"`                         // 监听主机地址
	Port           int            `yaml:"port" json:"port"`                         // 监听端口
	UseHttp2       bool           `yaml:"use_http2" json:"use_http2"`               // 使用HTTP2
	Http2          *HTTP2Config   `yaml:"http2" json:"http2"`                       // HTTP2配置
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

// HTTP2Config HTTP2服务器配置结构体
type HTTP2Config struct {
	MaxHandlers                  int   `yaml:"max_handlers" json:"max_handlers"`                                         // 最大处理协程
	MaxConcurrentStreams         int64 `yaml:"max_concurrent_streams" json:"max_concurrent_streams"`                     // 最大并发流
	MaxReadFrameSize             int64 `yaml:"max_read_frame_size" json:"max_read_frame_size"`                           // 最大帧大小
	MaxDecoderHeaderTableSize    int64 `yaml:"max_decoder_header_table_size" json:"max_decoder_header_table_size"`       // 解码端头表大小
	MaxEncoderHeaderTableSize    int64 `yaml:"max_encoder_header_table_size" json:"max_encoder_header_table_size"`       // 编码端头表大小
	MaxUploadBufferPerConnection int64 `yaml:"max_upload_buffer_per_connection" json:"max_upload_buffer_per_connection"` // 连接级上传缓冲
	MaxUploadBufferPerStream     int64 `yaml:"max_upload_buffer_per_stream" json:"max_upload_buffer_per_stream"`         // 流级上传缓冲
	IdleTimeout                  int64 `yaml:"idle_timeout" json:"idle_timeout"`                                         // 空闲超时
	ReadIdleTimeout              int64 `yaml:"read_idle_timeout" json:"read_idle_timeout"`                               // 读空闲超时
	PingTimeout                  int64 `yaml:"ping_timeout" json:"ping_timeout"`                                         // Ping超时
	WriteByteTimeout             int64 `yaml:"write_byte_timeout" json:"write_byte_timeout"`                             // 写超时
	PermitProhibitedCipherSuites bool  `yaml:"permit_prohibited_cipher_suites" json:"permit_prohibited_cipher_suites"`   // 允许弱密码套件
}

type CertConfig struct {
	Domains  []string `yaml:"domains" json:"domains"`     // 域名组
	CertFile string   `yaml:"cert_file" json:"cert_file"` // 证书文件路径
	KeyFile  string   `yaml:"key_file" json:"key_file"`   // 私钥文件路径
}

// TLSConfig TLS证书配置结构体
type TLSConfig struct {
	CertMap    map[string]CertConfig `yaml:"cert_map" json:"cert_map"`
	AutoTLS    bool                  `yaml:"auto_tls" json:"auto_tls"`       // 是否启用自动TLS
	MinVersion string                `yaml:"min_version" json:"min_version"` // 最小TLS版本
}

// DomainConfig 域名配置结构体
type DomainConfig struct {
	Domains        []string `yaml:"domains" json:"domains"`                 // 域名
	WsDomains      []string `yaml:"ws_domains" json:"ws_domains"`           // websocket域名
	UseTLS         bool     `yaml:"use_tls" json:"use_tls"`                 // 监听在https
	AutoRedirect   bool     `yaml:"auto_redirect" json:"auto_redirect"`     // 自动重定向
	UseWebsocket   bool     `yaml:"use_websocket" json:"use_websocket"`     // 开启websocket
	HSTSMaxAge     int      `yaml:"hsts_max_age" json:"hsts_max_age"`       // HSTS最大生存时间（秒），0表示不设置HSTS
	HSTSSubdomains bool     `yaml:"hsts_subdomains" json:"hsts_subdomains"` // HSTS是否包含子域名
	HSTSPreload    bool     `yaml:"hsts_preload" json:"hsts_preload"`       // HSTS是否启用预加载
}

// MiddlewareConfig 中间件配置结构体
type MiddlewareConfig struct {
	Gzip         GzipConfig   `yaml:"gzip" json:"gzip"` // Gzip压缩配置
	NoCache      bool         `yaml:"no_cache" json:"no_cache"`
	SecureHeader bool         `yaml:"secure_header" json:"secure_header"` // 安全响应头
	Trace        TraceConfig  `yaml:"trace" json:"trace"`                 // 请求跟踪
	CORS         CorsConfig   `yaml:"cors" json:"cors"`                   // cors策略
	Sanitizer    Sanitizer    `yaml:"sanitizer" json:"sanitizer"`         // 请求头标准化
	DomainCheck  DomainCheck  `yaml:"domain_check" json:"domain_check"`   // 域名强制校验
	ImageProtect ImageProtect `yaml:"image_protect" json:"image_protect"` // 图片防盗链
}

// FeatureConfig 功能特性配置结构体
type FeatureConfig struct {
	HTTP3       HTTP3Config       `yaml:"http3" json:"http3"`               // HTTP/3配置
	WebSocket   WebSocketConfig   `yaml:"websocket" json:"websocket"`       // WebSocket配置
	Cache       CacheConfig       `yaml:"cache" json:"cache"`               // 缓存配置
	AutoCert    AutoCertConfig    `yaml:"auto_cert" json:"auto_cert"`       // 自动证书配置
	GrpcProxy   GrpcProxyConfig   `yaml:"grpc_proxy" json:"grpc_proxy"`     // gRPC代理配置
	FlowControl FlowControlConfig `yaml:"flow_control" json:"flow_control"` // 流控配置
	Break       BreakConfig       `yaml:"break" json:"break"`               // 熔断配置
	ProxyCache  ProxyCacheConfig  `yaml:"proxy_cache" json:"proxy_cache"`   // 代理缓存
}

// DatabaseConfig 数据库配置结构体
type DatabaseConfig struct {
	Mongo  MongoConfig  `yaml:"mongo" json:"mongo"`   // MongoDB配置
	Influx InfluxConfig `yaml:"influx" json:"influx"` // InfluxDB配置
}

// MonitorConfig 监控配置结构体
type MonitorConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`       // 是否启用监控
	Port       int    `yaml:"port" json:"port"`             // 监控端口
	Path       string `yaml:"path" json:"path"`             // 监控路径
	Interval   int    `yaml:"interval" json:"interval"`     // 监控间隔
	Prometheus bool   `yaml:"prometheus" json:"prometheus"` // 是否启用Prometheus
}

// SecurityConfig 安全配置结构体
type SecurityConfig struct {
	StrictMode bool     `yaml:"strict_mode" json:"strict_mode"` // 严格模式
	AllowIPs   []string `yaml:"allow_ips" json:"allow_ips"`     // 允许的IP列表
	DenyIPs    []string `yaml:"deny_ips" json:"deny_ips"`       // 拒绝的IP列表
	RateLimit  int      `yaml:"rate_limit" json:"rate_limit"`   // 速率限制

	HSTS             bool `yaml:"hsts" json:"hsts"`                     // HSTS策略
	HSTSSubdomain    bool `yaml:"hsts_subdomain" json:"hsts_subdomain"` // 包含子域名
	HSTSPreload      bool `yaml:"hsts_preload" json:"hsts_preload"`     // 预加载
	XssProtection    bool `yaml:"xss_protection" json:"xss_protection"` // XSS保护
	IFrameProtection bool `yaml:"iframe_protection" json:"iframe_protection"`
	SameSite         bool `yaml:"same_site" json:"same_site"` // 同源策略
}

type ProxyHeader struct {
	TraceId            string `yaml:"trace_id" json:"trace_id"`                         // traceId头
	FrontendHostHeader string `yaml:"frontend_host_header" json:"frontend_host_header"` // 前端服务真实HOST
	ForwardHostHeader  string `yaml:"forward_host_header" json:"forward_host_header"`   // 转发真实服务器时携带的原始Host
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

type PProf struct {
	Enable bool `yaml:"enable" json:"enable"`
	Port   int  `yaml:"port" json:"port"`
}
