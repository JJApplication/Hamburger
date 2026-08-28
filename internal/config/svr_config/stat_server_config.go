package svr_config

// inner stat server

type StatConfig struct {
	DBFile       string         `yaml:"db_file" json:"db_file"`         // SQLite数据库文件路径
	UseDB        bool           `yaml:"use_db" json:"use_db"`           // 传统stat是否使用数据库记录
	Compatible   bool           `yaml:"compatible" json:"compatible"`   // 兼容加载file到DB中
	Enabled      bool           `yaml:"enabled" json:"enabled"`         // 是否开启服务器 不开启服务器也会统计
	EnableStat   bool           `yaml:"enable_stat" json:"enable_stat"` // 开启统计
	SyncDuration int            `yaml:"sync_duration" json:"sync_duration"`
	SaveDuration int            `yaml:"save_duration" json:"save_duration"`
	SaveFile     string         `json:"save_file"`
	GeoFile      string         `json:"geo_file"`
	DomainFile   string         `json:"domain_file"`
	GeoDB        string         `json:"geo_db"`                   // geo数据库
	Sequence     SequenceConfig `yaml:"sequence" json:"sequence"` // 时序统计配置
}

// SequenceConfig 时序统计配置结构体
// 用于配置是否启用时序统计、数据库文件路径以及统计时间间隔
type SequenceConfig struct {
	Enabled         bool `yaml:"enabled" json:"enabled"`                   // 是否启用分钟级历史统计
	Interval        int  `yaml:"interval" json:"interval"`                 // 存储粒度，当前固定为60秒
	RetentionDays   int  `yaml:"retention_days" json:"retention_days"`     // 历史数据保留天数
	FlushInterval   int  `yaml:"flush_interval" json:"flush_interval"`     // 增量刷盘间隔（秒）
	CleanupInterval int  `yaml:"cleanup_interval" json:"cleanup_interval"` // 过期数据清理间隔（秒）
}
