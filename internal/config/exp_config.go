package config

// 实验特性

type ExpConfig struct {
	VpnServer    VpnServerConfig `yaml:"vpn_server" json:"vpn_server"`
	AnyTLSServer AnyTLSServer    `yaml:"any_tls_server" json:"any_tls_server"`
	DNSServer    DNSServerConfig `yaml:"dns_server" json:"dns_server"`
	WebDAV       WebDAVConfig    `yaml:"webdav" json:"webdav"`
	TrojanServer string          `yaml:"trojan_server" json:"trojan_server"`
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

type DNSServerConfig struct {
	Enabled  bool               `yaml:"enabled" json:"enabled"`
	Host     string             `yaml:"host" json:"host"`
	Port     int                `yaml:"port" json:"port"`
	Upstream string             `yaml:"upstream" json:"upstream"`
	Timeout  int                `yaml:"timeout" json:"timeout"`
	DOH      DNSServerDOHConfig `yaml:"doh" json:"doh"`
}

type DNSServerDOHConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Path     string `yaml:"path" json:"path"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
}
