package svr_config

// LuaConfig 定义Lua脚本运行时配置。
type LuaConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"` // 是否启用Lua脚本扩展

	ScriptsRoot string `yaml:"scripts_root" json:"scripts_root"` // Lua脚本目录
}
