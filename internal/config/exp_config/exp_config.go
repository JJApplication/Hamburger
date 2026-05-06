package exp_config

// 实验特性

type ExpConfig struct {
	VpnServer    VpnServerConfig       `yaml:"vpn_server" json:"vpn_server"`
	AnyTLSServer AnyTLSServer          `yaml:"any_tls_server" json:"any_tls_server"`
	DNSServer    DNSServerConfig       `yaml:"dns_server" json:"dns_server"`
	WebDAV       WebDAVConfig          `yaml:"webdav" json:"webdav"`
	Traversal    TraversalServerConfig `yaml:"traversal" json:"traversal"`
	TrojanServer string                `yaml:"trojan_server" json:"trojan_server"`
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

// TraversalServerConfig 内网穿透服务端配置。
type TraversalServerConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	// ListenHost 控制通道监听地址。
	ListenHost string `yaml:"listen_host" json:"listen_host"`
	// ListenPort 控制通道监听端口。
	ListenPort int `yaml:"listen_port" json:"listen_port"`
	// Protocol 控制通道监听协议。
	// 支持：
	//   - "tcp"     : 仅启用 TCP 控制监听（默认）
	//   - "kcp"     : 仅启用 KCP 控制监听
	//   - "tcp+kcp" : 同时启用 TCP 和 KCP 控制监听
	Protocol string `yaml:"protocol" json:"protocol"`
	// KCP KCP 控制通道参数（仅当 protocol 启用 kcp 时生效）。
	KCP TraversalKCPControlConfig `yaml:"kcp" json:"kcp"`
	// AuthKey 客户端认证密钥。
	AuthKey string `yaml:"auth_key" json:"auth_key"`
}

// TraversalKCPControlConfig KCP 控制通道参数。
type TraversalKCPControlConfig struct {
	// HeartbeatInterval 心跳间隔（秒）。<=0 表示关闭心跳。
	HeartbeatInterval int `yaml:"heartbeat_interval" json:"heartbeat_interval"`
	// HeartbeatTimeout 心跳超时（秒）。当超过该时间未收到对端任何控制消息（含 pong），将断开控制连接。
	HeartbeatTimeout int `yaml:"heartbeat_timeout" json:"heartbeat_timeout"`
}
