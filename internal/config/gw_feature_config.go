package config

type ProxyCacheConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	LayerType string `yaml:"layer_type" json:"layer_type"` // 存储层类型 file|db|cache
	CacheSize int    `yaml:"cache_size" json:"cache_size"`
	CacheTTL  int    `yaml:"cache_ttl" json:"cache_ttl"`
	CachePath string `yaml:"cache_path" json:"cache_path"` // 缓存路径
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

// CacheConfig 缓存配置结构体
type CacheConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`   // 是否启用缓存
	Size     int    `yaml:"size" json:"size"`         // 缓存大小
	TTL      int    `yaml:"ttl" json:"ttl"`           // 缓存过期时间
	Strategy string `yaml:"strategy" json:"strategy"` // 缓存策略: lru, lfu, fifo
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
