package config

// inner latency server

type LatencyConfig struct {
	Enabled         bool     `yaml:"enabled" json:"enabled"` // 是否开启服务器
	Host            string   `yaml:"host" json:"host"`
	Port            int      `yaml:"port" json:"port"`
	DomainBlackList []string `yaml:"domain_black_list" json:"domain_black_list"` // 不允许测试的黑名单
	DomainWhiteList []string `yaml:"domain_white_list" json:"domain_white_list"`
}
