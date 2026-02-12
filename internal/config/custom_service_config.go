package config

type PxyCustomServiceConfig struct {
	Enable        bool                  `yaml:"enable" json:"enable"`
	CustomService []CustomServiceConfig `yaml:"custom_service" json:"custom_service"`
}

// CustomServiceConfig 自定义的域名服务映射关系
// 仅作为后端使用
type CustomServiceConfig struct {
	Domain   string     `yaml:"domain" json:"domain"`
	Upstream []Upstream `yaml:"upstream" json:"upstream"` // 负载均衡
}

type Upstream struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}
