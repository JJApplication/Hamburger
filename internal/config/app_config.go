package config

// AppConfig 配置文件格式模型
type AppConfig struct {
	PxyBackendFile  string `yaml:"pxy_backend_file" json:"pxy_backend_file"` // 配置文件路径
	PxyFrontendFile string `yaml:"pxy_frontend_file" json:"pxy_frontend_file"`
	DomainMap       string `yaml:"domain_map" json:"domain_map"` // 域名映射文件

	CoreProxy    ProxyConfig        `yaml:"proxy" json:"proxy"`                 // 核心代理配置
	Servers      []ServerConfig     `yaml:"servers" json:"servers"`             // 服务器配置列表
	Middleware   []MiddlewareConfig `yaml:"middleware" json:"middleware"`       // 中间件配置列表
	Features     FeatureConfig      `yaml:"features" json:"features"`           // 功能特性配置
	Database     DatabaseConfig     `yaml:"database" json:"database"`           // 数据库配置
	Security     SecurityConfig     `yaml:"security" json:"security"`           // 安全配置
	ProxyHeader  ProxyHeader        `yaml:"proxy_header" json:"proxy_header"`   // 内置的代理头配置
	Log          LogConfig          `yaml:"log" json:"log"`                     // 日志配置
	Module       []ModuleConfig     `yaml:"module" json:"module"`               // 模块
	Stat         StatConfig         `yaml:"stat" json:"stat"`                   // 状态统计配置
	CustomHeader map[string]string  `yaml:"custom_header" json:"custom_header"` // 自定义Header
	Syncer       Syncer             `yaml:"syncer" json:"syncer"`               // 定时器时间
	Debug        bool               `yaml:"debug" json:"debug"`                 // 调试模式
	PProf        PProf              `yaml:"pprof" json:"pprof"`
	MaxCores     int                `yaml:"max_cores" json:"max_cores"`

	// 第二优先级
	PxyBackend  PxyBackendConfig `yaml:"pxy_backend" json:"pxy_backend"`
	PxyFrontend PxyFrontConfig   `yaml:"pxy_frontend" json:"pxy_frontend"`
}
