package config

import (
	"Hamburger/internal/json"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// PxyBackendConfig 后端自定义服务器配置
type PxyBackendConfig struct {
	Enabled bool            `json:"enabled" toml:"enabled"` // 是否开启此特性
	Servers []BackendServer `json:"servers" toml:"servers"`
}

type BackendServerBase struct {
	ServiceName string `yaml:"service_name" json:"service_name"` // 后端服务名
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	Type        string `yaml:"type" json:"type"`
}

type BackendServer struct {
	BackendServerBase
	Http        BackendHTTPConfig        `yaml:"http" json:"http"`
	WebDav      BackendWebDavConfig      `yaml:"webdav" json:"webdav"`
	Transparent BackendTransparentConfig `yaml:"transparent" json:"transparent"`
	Tcp         BackendTCPConfig         `yaml:"tcp" json:"tcp"` // TCP代理配置
}

type BackendHTTPConfig struct {
	// 响应自定义
	Response     []Response `yaml:"response" json:"response"`
	EnableStatic bool       `yaml:"enable_static" json:"enable_static"`
	StaticDir    string     `yaml:"static_dir" json:"static_dir"`
	StaticPrefix string     `yaml:"static_prefix" json:"static_prefix"`
	User         string     `yaml:"user" json:"user"`
	Password     string     `yaml:"password" json:"password"`
}

type BackendWebDavConfig struct {
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	Root     string `yaml:"root" json:"root"`
}

type BackendTransparentConfig struct {
	Target        string `yaml:"target" json:"target"`
	Scheme        string `yaml:"scheme" json:"scheme"`
	RewritePrefix string `yaml:"rewrite_prefix" json:"rewrite_prefix"`
	RewriteTo     string `yaml:"rewrite_to" json:"rewrite_to"`
	User          string `yaml:"user" json:"user"`
	Password      string `yaml:"password" json:"password"`
}

type BackendTCPConfig struct {
	Target  string `yaml:"target" json:"target"`     // 转发目标地址
	Timeout int    `yaml:"timeout" json:"timeout"`   // 转发超时时间(秒)
	MaxBody int64  `yaml:"max_body" json:"max_body"` // 最大转发字节数
}

type Response struct {
	Path    string            `yaml:"path" json:"path"`
	Code    int               `yaml:"code" json:"code"`
	Msg     string            `yaml:"msg" json:"msg"`
	Headers map[string]string `yaml:"headers" json:"headers"`
}

func LoadBackendConfig(file string) (PxyBackendConfig, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return PxyBackendConfig{}, err
	}
	var cf PxyBackendConfig
	ext := filepath.Ext(file)
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &cf)
		return cf, err
	case ".toml":
		err = toml.Unmarshal(data, &cf)
		return cf, err
	default:
		err = json.Unmarshal(data, &cf)
		return PxyBackendConfig{}, err
	}
}
