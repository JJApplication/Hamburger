package config

import (
	"Hamburger/internal/config/backproxy_config"
	"Hamburger/internal/config/core_config"
	"Hamburger/internal/config/exp_config"
	"Hamburger/internal/config/frontproxy_config"
	"Hamburger/internal/config/svr_config"
	"time"
)

// Config 主配置结构体，包含所有服务配置信息
type Config struct {
	CoreProxy          core_config.ProxyConfig           `yaml:"proxy" json:"proxy"` // 核心代理配置
	ErrorConfig        core_config.ProxyErrorConfig      `yaml:"error_config" json:"error_config"`
	Servers            []core_config.ServerConfig        `yaml:"servers" json:"servers"`                     // 服务器配置列表
	Middleware         core_config.MiddlewareConfig      `yaml:"middleware" json:"middleware"`               // 中间件配置列表
	Features           core_config.FeatureConfig         `yaml:"features" json:"features"`                   // 功能特性配置
	GRPC               GRPCConfig                        `yaml:"grpc" json:"grpc"`                           // gRPC服务配置
	ApiServerConfig    svr_config.ApiServerConfig        `yaml:"api_server_config" json:"api_server_config"` // 内置API服务
	Database           core_config.DatabaseConfig        `yaml:"database" json:"database"`                   // 数据库配置
	Security           core_config.SecurityConfig        `yaml:"security" json:"security"`                   // 安全配置
	ProxyHeader        core_config.ProxyHeader           `yaml:"proxy_header" json:"proxy_header"`           // 内置的代理头配置
	Log                core_config.LogConfig             `yaml:"log" json:"log"`                             // 日志配置
	Module             []core_config.ModuleConfig        `yaml:"module" json:"module"`                       // 模块
	Stat               svr_config.StatConfig             `yaml:"stat" json:"stat"`                           // 状态统计配置
	Latency            svr_config.LatencyConfig          `yaml:"latency" json:"latency"`                     // 延迟统计配置
	CustomHeader       map[string]string                 `yaml:"custom_header" json:"custom_header"`         // 自定义Header
	Plugin             PluginConfig                      `yaml:"plugin" json:"plugin"`                       // 插件配置
	Lua                svr_config.LuaConfig              `yaml:"lua" json:"lua"`                             // Lua脚本配置
	Syncer             Syncer                            `yaml:"syncer" json:"syncer"`                       // 定时器时间
	Debug              bool                              `yaml:"debug" json:"debug"`                         // 调试模式
	DevMode            bool                              `yaml:"dev_mode" json:"dev_mode"`                   // 开发模式
	PProf              core_config.PProf                 `yaml:"pprof" json:"pprof"`
	MaxCores           int                               `yaml:"max_cores" json:"max_cores"`
	PxyBackend         backproxy_config.PxyBackendConfig `yaml:"pxy_backend" json:"pxy_backend"`
	PxyFrontend        frontproxy_config.PxyFrontConfig  `yaml:"pxy_frontend" json:"pxy_frontend"`
	StaticDirectServer svr_config.StaticDirectSvrConfig  `yaml:"static_direct_server" json:"static_direct_server"` // 静态直通
	ExpConfig          exp_config.ExpConfig              `yaml:"exp_config" json:"exp_config"`
	PreAuthConfig      svr_config.PreAuthConfig          `yaml:"pre_auth_config" json:"pre_auth_config"`
	NotifyConfig       svr_config.NotifyConfig           `yaml:"notify_config" json:"notify_config"` // 通知系统配置
}

// GRPCConfig gRPC服务配置结构体
type GRPCConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"` // 是否启用gRPC服务
	Address string `yaml:"address" json:"address"` // gRPC监听地址
}

// PluginConfig 插件配置
type PluginConfig struct {
	Enabled bool         `yaml:"enabled" json:"enabled"` // 是否启用
	Root    string       `yaml:"root" json:"root"`       // 插件根目录
	Plugins []PluginItem `yaml:"plugins" json:"plugins"` // 插件列表
}

// PluginItem 单个插件配置
type PluginItem struct {
	Name    string                 `yaml:"name" json:"name"`       // 插件名称
	Enabled bool                   `yaml:"enabled" json:"enabled"` // 是否启用
	Params  map[string]interface{} `yaml:"params" json:"params"`   // 扩展配置
}

// Syncer 定时器时间
type Syncer struct {
	JobSyncDomainsMap  SyncerTime `yaml:"job_sync_domains" json:"job_sync_domains"`           // 同步域名映射文件的任务
	JobSyncDomainPorts SyncerTime `yaml:"job_sync_domain_ports" json:"job_sync_domain_ports"` // 同步域名和服务端口映射
	JobSyncHealthProbe SyncerTime `yaml:"job_sync_health_probe" json:"job_sync_health_probe"` // 同步域名探针状态
}

type SyncerTime int64

// Get 获取默认值如果不存在
func (s *SyncerTime) Get(defaultValue time.Duration) time.Duration {
	if *s == 0 {
		return defaultValue * time.Second
	}
	return time.Duration(*s) * time.Second
}
