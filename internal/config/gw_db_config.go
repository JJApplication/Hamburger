package config

// MongoConfig MongoDB配置结构体
type MongoConfig struct {
	URL      string `yaml:"url" json:"url"`           // MongoDB连接URL
	Database string `yaml:"database" json:"database"` // 数据库名称
	Timeout  int    `yaml:"timeout" json:"timeout"`   // 连接超时时间
}

// InfluxConfig InfluxDB配置结构体
type InfluxConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`   // 是否启用InfluxDB
	URL      string `yaml:"url" json:"url"`           // InfluxDB连接URL
	Token    string `yaml:"token" json:"token"`       // 访问令牌
	Org      string `yaml:"org" json:"org"`           // 组织名称
	Bucket   string `yaml:"bucket" json:"bucket"`     // 存储桶名称
	Password string `yaml:"password" json:"password"` // 密码
}
