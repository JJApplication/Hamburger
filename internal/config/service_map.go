package config

// 服务配置
//
// 支持JJAPP内置的服务和自定义服务
// JJAPP内置的服务支持内置端口加载

type DomainServiceMap struct {
	DomainService []DomainService `yaml:"domain_service" json:"domain_service"`
	Sevices       []Service       `yaml:"services" json:"services"`
}

type DomainService struct {
	Domain  string `yaml:"domain" json:"domain"` // 本地调试项目domain可以为localhost
	Tag     string `yaml:"tag" json:"tag"`
	Group   string `yaml:"group" json:"group"`     // 标签名称
	Service string `yaml:"service" json:"service"` // 映射的服务
}

// Service 由Helios代理的前端代理转发都在Helios配置
//
// 自定义服务的代理转发都由gw处理在Service中处理
// 同一域名只能映射一个服务
type Service struct {
	Tag         string `yaml:"tag" json:"tag"`
	Group       string `yaml:"group" json:"group"` // 标签名称
	ServiceName string `yaml:"service_name" json:"service_name"`
	ServiceType string `yaml:"service_type" json:"service_type"` // 前端 后端 自定义
	// 自定义服务需要的扩展参数
	ServiceRoot string         `yaml:"service_root" json:"service_root"`
	Host        string         `yaml:"host" json:"host"`
	Port        int            `yaml:"port" json:"port"`
	ProxyPass   []ServiceProxy `yaml:"proxy_pass" json:"proxy_pass"` // 代理转发
}

type ServiceProxy struct {
	API          string       `json:"api" toml:"api"`
	Service      string       `json:"service" toml:"service"`
	UseRewrite   bool         `json:"use_rewrite" toml:"use_rewrite"`
	Rewrite      string       `json:"rewrite" toml:"rewrite"`
	IsStatic     bool         `json:"is_static" toml:"is_static"` // 是否为静态目录
	StaticDirect StaticDirect `json:"static_direct" toml:"static_direct"`
	// 通用转发配置
	ProxyDirect ProxyDirect `json:"proxy_direct" toml:"proxy_direct"` // 后端代理转发
}
