package svr_config

// 内置API服务器配置 不开启API服务时
// Stat在开启时也会记录

type ApiServerConfig struct {
	Enabled bool           `yaml:"enabled" json:"enabled"`
	Host    string         `yaml:"host" json:"host"`
	Port    int            `yaml:"port" json:"port"`
	HTTP2   APIHTTP2Config `yaml:"http2" json:"http2"`
	JWT     JWTConfig      `yaml:"jwt" json:"jwt"`
	BBolt   APIBBoltConfig `yaml:"bblot" json:"bblot"`
}

type JWTConfig struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	TokenHeader    string   `yaml:"token_header" json:"token_header"`
	Secret         string   `yaml:"secret" json:"secret"`
	Issuer         string   `yaml:"issuer" json:"issuer"`
	Audience       string   `yaml:"audience" json:"audience"`
	AllowedMethods []string `yaml:"allowed_methods" json:"allowed_methods"`
}

type APIHTTP2Config struct {
	Enabled              bool  `yaml:"enabled" json:"enabled"`
	Insecure             bool  `yaml:"insecure" json:"insecure"`
	MaxHandlers          int   `yaml:"max_handlers" json:"max_handlers"`
	MaxConcurrentStreams int64 `yaml:"max_concurrent_streams" json:"max_concurrent_streams"`
	ReadTimeout          int64 `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout         int64 `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout          int64 `yaml:"idle_timeout" json:"idle_timeout"`
	ReadHeaderTimeout    int64 `yaml:"read_header_timeout" json:"read_header_timeout"`
	MaxHeaderBytes       int64 `yaml:"max_header_bytes" json:"max_header_bytes"`
}

type APIBBoltConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`
	File            string `yaml:"file" json:"file"`
	TimeoutSeconds  int    `yaml:"timeout_seconds" json:"timeout_seconds"`
	UserBucket      string `yaml:"user_bucket" json:"user_bucket"`
	DefaultUsername string `yaml:"default_username" json:"default_username"`
	DefaultPassword string `yaml:"default_password" json:"default_password"`
}
