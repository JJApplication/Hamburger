package exp_config

type VpnServerConfig struct {
	Enabled   bool          `yaml:"enabled" json:"enabled"`
	Host      string        `yaml:"host" json:"host"`
	HttpPort  int           `yaml:"http_port" json:"http_port"`
	SocksPort int           `yaml:"socks_port" json:"socks_port"`
	Timeout   int           `yaml:"timeout" json:"timeout"`
	Auth      VpnAuthConfig `yaml:"auth" json:"auth"`
	TLS       VpnTLSConfig  `yaml:"tls" json:"tls"`
	Obfs      VpnObfsConfig `yaml:"obfs" json:"obfs"`
}

type VpnAuthConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

type VpnTLSConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	EnableHTTP  bool   `yaml:"enable_http" json:"enable_http"`
	EnableSocks bool   `yaml:"enable_socks" json:"enable_socks"`
	CertFile    string `yaml:"cert_file" json:"cert_file"`
	KeyFile     string `yaml:"key_file" json:"key_file"`
}

type VpnObfsConfig struct {
	Enabled            bool   `yaml:"enabled" json:"enabled"`
	Mode               string `yaml:"mode" json:"mode"`
	MinChunkSize       int    `yaml:"min_chunk_size" json:"min_chunk_size"`
	MaxChunkSize       int    `yaml:"max_chunk_size" json:"max_chunk_size"`
	MinDelayMs         int    `yaml:"min_delay_ms" json:"min_delay_ms"`
	MaxDelayMs         int    `yaml:"max_delay_ms" json:"max_delay_ms"`
	CamouflageEnabled  bool   `yaml:"camouflage_enabled" json:"camouflage_enabled"`
	CamouflageStatus   int    `yaml:"camouflage_status" json:"camouflage_status"`
	CamouflageBody     string `yaml:"camouflage_body" json:"camouflage_body"`
	CamouflageMimeType string `yaml:"camouflage_mime_type" json:"camouflage_mime_type"`
}
