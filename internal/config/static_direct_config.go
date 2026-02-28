package config

type StaticDirectSvrConfig struct {
	Enabled      bool     `json:"enabled" yaml:"enabled"`
	Host         string   `json:"host" yaml:"host"`
	Port         int      `json:"port" yaml:"port"`
	Blacklist    []string `json:"blacklist" yaml:"blacklist"` // 黑名单文件
	BlacklistExt []string `json:"blacklist_ext" yaml:"blacklist_ext"`
}
