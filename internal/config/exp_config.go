package config

// 实验特性

type ExpConfig struct {
	VpnServer    VpnServerConfig `yaml:"vpn_server" json:"vpn_server"`
	AnyTLSServer AnyTLSServer    `yaml:"any_tls_server" json:"any_tls_server"`
}

type AnyTLSServer struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	User        string `yaml:"user" json:"user"`
	Password    string `yaml:"password" json:"password"`
	CertFile    string `yaml:"cert_file" json:"cert_file"`
	KeyFile     string `yaml:"key_file" json:"key_file"`
	DialTimeout int    `yaml:"dial_timeout" json:"dial_timeout"` // 单位s
}
