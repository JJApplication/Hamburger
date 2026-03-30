package config

type WebDAVConfig struct {
	Enabled      bool               `yaml:"enabled" json:"enabled"`
	Host         string             `yaml:"host" json:"host"`
	Port         int                `yaml:"port" json:"port"`
	PathPrefix   string             `yaml:"path_prefix" json:"path_prefix"`
	ReadOnly     bool               `yaml:"read_only" json:"read_only"`
	ReadTimeout  int                `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout int                `yaml:"write_timeout" json:"write_timeout"`
	Users        []WebDAVUserConfig `yaml:"users" json:"users"`
}

type WebDAVUserConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	RootDir  string `yaml:"root_dir" json:"root_dir"`
	ReadOnly bool   `yaml:"read_only" json:"read_only"`
}
