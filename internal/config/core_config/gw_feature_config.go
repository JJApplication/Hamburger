package core_config

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

const (
	DefaultBreakMaxError            = 5
	DefaultBreakResetSeconds        = 30
	DefaultBreakBucket              = 16
	DefaultBreakMinRequests         = 10
	DefaultBreakHalfOpenMaxRequests = 3
	DefaultBreakSuccessThreshold    = 3
)

// DefaultBreakTripStatusCodes 是后端响应里计入失败的状态码。
func DefaultBreakTripStatusCodes() []int {
	return []int{500, 502, 503, 504}
}

// BreakConfig 熔断配置。
// Enabled 为空时默认开启；显式 false 才关闭。
type BreakConfig struct {
	Enabled             *bool   `yaml:"enabled" json:"enabled"`                               // 为空默认开启
	Bucket              int     `yaml:"bucket" json:"bucket"`                                 // 内存链路表预分配容量
	MaxError            int     `yaml:"max_error" json:"max_error"`                           // 窗口内连续失败次数阈值
	Reset               int     `yaml:"reset" json:"reset"`                                   // 统计窗口和熔断冷却时间（秒）
	FailureRatio        float64 `yaml:"failure_ratio" json:"failure_ratio"`                   // 窗口失败率阈值，0 表示不启用
	MinRequests         int     `yaml:"min_requests" json:"min_requests"`                     // 失败率判定的最小样本数
	HalfOpenMaxRequests int     `yaml:"half_open_max_requests" json:"half_open_max_requests"` // 半开状态并发探测数
	SuccessThreshold    int     `yaml:"success_threshold" json:"success_threshold"`           // 半开恢复所需连续成功次数
	TripStatusCodes     []int   `yaml:"trip_status_codes" json:"trip_status_codes"`           // 计入失败的后端状态码
}

// IsEnabled 报告熔断是否生效。未配置时默认开启。
func (c BreakConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// Effective 返回补齐默认值并修正非法值后的配置副本。
func (c BreakConfig) Effective() BreakConfig {
	out := c
	if out.MaxError <= 0 {
		out.MaxError = DefaultBreakMaxError
	}
	if out.Reset <= 0 {
		out.Reset = DefaultBreakResetSeconds
	}
	if out.Bucket <= 0 {
		out.Bucket = DefaultBreakBucket
	}
	if out.FailureRatio < 0 {
		out.FailureRatio = 0
	}
	if out.FailureRatio > 1 {
		out.FailureRatio = 1
	}
	if out.MinRequests <= 0 {
		out.MinRequests = DefaultBreakMinRequests
	}
	if out.HalfOpenMaxRequests <= 0 {
		out.HalfOpenMaxRequests = DefaultBreakHalfOpenMaxRequests
	}
	if out.SuccessThreshold <= 0 {
		out.SuccessThreshold = DefaultBreakSuccessThreshold
	}
	if len(c.TripStatusCodes) == 0 {
		out.TripStatusCodes = DefaultBreakTripStatusCodes()
	} else {
		out.TripStatusCodes = append([]int(nil), c.TripStatusCodes...)
	}
	return out
}

// Normalize 就地补齐默认值并修正非法值。
func (c *BreakConfig) Normalize() {
	if c == nil {
		return
	}
	normalized := c.Effective()
	*c = normalized
}

// DefaultBreakConfig 是进程内置的熔断配置，默认开启。
func DefaultBreakConfig() BreakConfig {
	enabled := true
	return (BreakConfig{Enabled: &enabled}).Effective()
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
