package core_config

// GzipConfig Gzip压缩配置结构体
type GzipConfig struct {
	Enabled   bool     `yaml:"enabled" json:"enabled"`     // 是否启用Gzip压缩
	Level     int      `yaml:"level" json:"level"`         // 压缩级别 1-9
	Types     []string `yaml:"types" json:"types"`         // 压缩的MIME类型列表
	Threshold int      `yaml:"threshold" json:"threshold"` // 开启压缩的阈值
}

type ZstdConfig struct {
	Enabled   bool     `yaml:"enabled" json:"enabled"`
	Level     int      `yaml:"level" json:"level"`
	Types     []string `yaml:"types" json:"types"`
	Threshold int      `yaml:"threshold" json:"threshold"`
}

type TraceConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	TraceId string `yaml:"trace_id" json:"trace_id"`
}

type CorsConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Method  []string `yaml:"method" json:"method"`
	Origin  []string `yaml:"origin" json:"origin"` // 默认*
	Header  []string `yaml:"header" json:"header"`
}

type Sanitizer struct {
	Enabled bool `yaml:"enabled" json:"enabled"` // 是否启用缓存
}

type DomainCheck struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

type ImageProtect struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`             // 是否启用
	ImageType    []string `yaml:"image_type" json:"image_type"`       // 过滤的图片类型
	AllowReferer []string `yaml:"allow_referer" json:"allow_referer"` // 允许的请求头
}

type FailResponse struct {
	Enabled bool  `yaml:"enabled" json:"enabled"` // 是否启用失败响应拦截
	Code    []int `yaml:"code" json:"code"`       // 失败响应状态码
}
