package config

import (
	"Hamburger/internal/json"
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
)

// PxyBackendConfig 后端自定义服务器配置
type PxyBackendConfig struct {
	Enabled bool            `json:"enabled" toml:"enabled"` // 是否开启此特性
	Servers []BackendServer `json:"servers" toml:"servers"`
}

type BackendServer struct {
	ServiceName string `yaml:"service_name" json:"service_name"` // 后端服务名
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	// 响应自定义
	Response []Response `yaml:"response" json:"response"`
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
