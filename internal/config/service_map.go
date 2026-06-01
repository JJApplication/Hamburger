package config

import "Hamburger/internal/config/frontproxy_config"

// 服务配置
//
// 支持JJAPP内置的服务和自定义服务
// JJAPP内置的服务支持内置端口加载

type DomainServiceMap struct {
	Sevices []Service `yaml:"services" json:"services"`
}

// Service 由Helios代理的前端代理转发都在Helios配置
//
// 自定义服务的代理转发都由gw处理在Service中处理
// 同一域名只能映射一个服务
type Service struct {
	Tag           string `yaml:"tag" json:"tag"`
	Group         string `yaml:"group" json:"group"` // 标签名称
	ServiceName   string `yaml:"service_name" json:"service_name"`
	ServiceType   string `yaml:"service_type" json:"service_type"`     // 前端 后端 自定义
	ServiceDomain string `yaml:"service_domain" json:"service_domain"` // 正则表达式或直接域名配置
	PreCheck      PreCheckConfig `yaml:"pre_check" json:"pre_check"`   // 前置检查（类似 Cloudflare challenge）
	// 自定义服务需要的扩展参数
	ServiceRoot string         `yaml:"service_root" json:"service_root"`
	Host        string         `yaml:"host" json:"host"`
	Port        int            `yaml:"port" json:"port"`
	ProxyPass   []ServiceProxy `yaml:"proxy_pass" json:"proxy_pass"` // 代理转发
}

type PreCheckConfig struct {
	Enabled       bool     `yaml:"enabled" json:"enabled"`
	TTLSeconds    int64    `yaml:"ttl_seconds" json:"ttl_seconds"`         // 通过后免检时长（秒）
	CacheMaxMB    int      `yaml:"cache_max_mb" json:"cache_max_mb"`       // bigcache 最大内存（MB）
	PathPrefix    string   `yaml:"path_prefix" json:"path_prefix"`         // challenge 路由前缀，如 /__precheck
	ExcludePaths  []string `yaml:"exclude_paths" json:"exclude_paths"`     // 不走前置检查的路径前缀
	VerifyTimeout int64    `yaml:"verify_timeout" json:"verify_timeout"`   // 校验流程超时（秒），用于页面提示/重试（可选）
}

type ServiceProxy struct {
	API          string                         `json:"api" toml:"api"`
	Service      string                         `json:"service" toml:"service"`
	UseRewrite   bool                           `json:"use_rewrite" toml:"use_rewrite"`
	Rewrite      string                         `json:"rewrite" toml:"rewrite"`
	StaticDirect frontproxy_config.StaticDirect `json:"static_direct" toml:"static_direct"` // 静态文件转发
	// 通用转发配置
	ProxyDirect frontproxy_config.ProxyDirect `json:"proxy_direct" toml:"proxy_direct"` // 后端代理转发
}
