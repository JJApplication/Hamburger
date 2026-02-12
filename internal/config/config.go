package config

import "time"

// Config 主配置结构体，包含所有服务配置信息
type Config struct {
	CoreProxy        ProxyConfig            `yaml:"proxy" json:"proxy"`                 // 核心代理配置
	Servers          []ServerConfig         `yaml:"servers" json:"servers"`             // 服务器配置列表
	Middleware       MiddlewareConfig       `yaml:"middleware" json:"middleware"`       // 中间件配置列表
	Features         FeatureConfig          `yaml:"features" json:"features"`           // 功能特性配置
	Database         DatabaseConfig         `yaml:"database" json:"database"`           // 数据库配置
	Security         SecurityConfig         `yaml:"security" json:"security"`           // 安全配置
	ProxyHeader      ProxyHeader            `yaml:"proxy_header" json:"proxy_header"`   // 内置的代理头配置
	Log              LogConfig              `yaml:"log" json:"log"`                     // 日志配置
	Module           []ModuleConfig         `yaml:"module" json:"module"`               // 模块
	Stat             StatConfig             `yaml:"stat" json:"stat"`                   // 状态统计配置
	CustomHeader     map[string]string      `yaml:"custom_header" json:"custom_header"` // 自定义Header
	Syncer           Syncer                 `yaml:"syncer" json:"syncer"`               // 定时器时间
	Debug            bool                   `yaml:"debug" json:"debug"`                 // 调试模式
	PProf            PProf                  `yaml:"pprof" json:"pprof"`
	MaxCores         int                    `yaml:"max_cores" json:"max_cores"`
	PxyBackend       PxyBackendConfig       `yaml:"pxy_backend" json:"pxy_backend"`
	PxyFrontend      PxyFrontConfig         `yaml:"pxy_frontend" json:"pxy_frontend"`
	PxyCustomService PxyCustomServiceConfig `yaml:"pxy_custom_service" json:"pxy_custom_service"` // 定义的转发服务
}

// Syncer 定时器时间
type Syncer struct {
	JobSyncDomainsMap  SyncerTime `yaml:"job_sync_domains" json:"job_sync_domains"`           // 同步域名映射文件的任务
	JobSyncDomainPorts SyncerTime `yaml:"job_sync_domain_ports" json:"job_sync_domain_ports"` // 同步域名和服务端口映射
}

type SyncerTime int64

// Get 获取默认值如果不存在
func (s *SyncerTime) Get(defaultValue time.Duration) time.Duration {
	if *s == 0 {
		return defaultValue
	}
	return time.Duration(*s) * time.Second
}
