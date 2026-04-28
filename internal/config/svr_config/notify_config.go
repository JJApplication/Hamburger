package svr_config

// NotifyConfig 通知系统配置
type NotifyConfig struct {
	Enabled           bool              `yaml:"enabled" json:"enabled"`                       // 是否启用通知系统
	Queue             NotifyQueueConfig `yaml:"queue" json:"queue"`                           // 消息系统配置
	Mail              NotifyMailConfig  `yaml:"mail" json:"mail"`                             // 邮件系统配置
	DefaultRecipients []string          `yaml:"default_recipients" json:"default_recipients"` // 默认收件人
}

// NotifyQueueConfig 消息系统配置
type NotifyQueueConfig struct {
	Topic  string `yaml:"topic" json:"topic"`   // 默认主题
	Buffer int    `yaml:"buffer" json:"buffer"` // 队列缓冲区大小
}

// NotifyMailConfig 邮件系统总配置
type NotifyMailConfig struct {
	Provider string           `yaml:"provider" json:"provider"` // 邮件服务类型：smtp/pop3
	From     string           `yaml:"from" json:"from"`         // 发件人地址
	SMTP     NotifySMTPConfig `yaml:"smtp" json:"smtp"`         // SMTP配置
	POP3     NotifyPOP3Config `yaml:"pop3" json:"pop3"`         // POP3配置
	IMAP     NotifyIMAPConfig `yaml:"imap" json:"imap"`         // IMAP配置
}

// NotifySMTPConfig SMTP对接配置
type NotifySMTPConfig struct {
	Host     string `yaml:"host" json:"host"`         // SMTP主机
	Port     int    `yaml:"port" json:"port"`         // SMTP端口
	Username string `yaml:"username" json:"username"` // SMTP账号
	Password string `yaml:"password" json:"password"` // SMTP密码
	TLS      bool   `yaml:"tls" json:"tls"`           // 是否启用TLS
}

// NotifyPOP3Config POP3对接配置
type NotifyPOP3Config struct {
	Host     string `yaml:"host" json:"host"`         // POP3主机
	Port     int    `yaml:"port" json:"port"`         // POP3端口
	Username string `yaml:"username" json:"username"` // POP3账号
	Password string `yaml:"password" json:"password"` // POP3密码
	TLS      bool   `yaml:"tls" json:"tls"`           // 是否启用TLS
}

// NotifyIMAPConfig IMAP对接配置
type NotifyIMAPConfig struct {
	Host     string `yaml:"host" json:"host"`         // IMAP主机
	Port     int    `yaml:"port" json:"port"`         // IMAP端口
	Username string `yaml:"username" json:"username"` // IMAP账号
	Password string `yaml:"password" json:"password"` // IMAP密码
	TLS      bool   `yaml:"tls" json:"tls"`           // 是否启用TLS
}
