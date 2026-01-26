package config

// PxyBackendConfig 后端自定义服务器配置
type PxyBackendConfig struct {
	Enabled bool            `json:"enabled" toml:"enabled"` // 是否开启此特性
	Servers []BackendServer `json:"servers" toml:"servers"`
}

type BackendServer struct {
	ServiceName string `yaml:"service_name" json:"service_name"` // 后端服务名
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	// 响应自定义
	Response []Response `yaml:"response" json:"response"`
}

type Response struct {
	Path    string            `yaml:"path" json:"path"`
	Code    int               `yaml:"code" json:"code"`
	Msg     string            `yaml:"msg" json:"msg"`
	Headers map[string]string `yaml:"headers" json:"headers"`
}
