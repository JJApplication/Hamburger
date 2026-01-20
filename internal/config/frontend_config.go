package config

import (
	"Hamburger/internal/json"
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
)

// FrontCacheConfig 缓存配置
type FrontCacheConfig struct {
	Enable  bool     `json:"enable" toml:"enable"`
	Dir     string   `json:"dir" toml:"dir"`
	Expire  int      `json:"expire" toml:"expire"`
	Matcher []string `json:"matcher" toml:"matcher"`
}

// BackendConfig 后端配置
type BackendConfig struct {
	API        string `json:"api" toml:"api"`
	Service    string `json:"service" toml:"service"`
	UseRewrite bool   `json:"use_rewrite" toml:"use_rewrite"`
	Rewrite    string `json:"rewrite" toml:"rewrite"`
}

// FrontServerConfig 服务器配置
type FrontServerConfig struct {
	Type     string            `json:"type" toml:"type"`
	Name     string            `json:"name" toml:"name"`
	Root     string            `json:"root" toml:"root"`
	Index    string            `json:"index" toml:"index"`
	TryFile  string            `json:"try_file" toml:"try_file"`
	Access   bool              `json:"access" toml:"access"`
	Compress bool              `json:"compress" toml:"compress"`
	Alias    map[string]string `json:"alias" toml:"alias"`
	Backends []BackendConfig   `json:"backends" toml:"backends"`
}

// ErrorConfig 错误页面配置
type ErrorConfig struct {
	NotFound            string `json:"not_found" toml:"not_found"`
	InternalServerError string `json:"internal_server_error" toml:"internal_server_error"`
}

// CustomHeaderConfig 自定义头配置
type CustomHeaderConfig struct {
	Name  string `json:"name" toml:"name"`
	Value string `json:"value" toml:"value"`
}

// PxyFrontConfig 前端服务器配置
type PxyFrontConfig struct {
	Log                 LogConfig            `json:"log" toml:"log"`
	Host                string               `json:"host" toml:"host"`
	Port                int                  `json:"port" toml:"port"`
	Balancer            string               `json:"balancer" toml:"balancer"`
	Cache               FrontCacheConfig     `json:"cache" toml:"cache"`
	InternalFlag        string               `json:"internal_flag" toml:"internal_flag"`
	InternalLocalFlag   string               `json:"internal_local_flag" toml:"internal_local_flag"`
	InternalBackendFlag string               `json:"internal_backend_flag" toml:"internal_backend_flag"`
	CacheHeader         string               `json:"cache_header" toml:"cache_header"`
	Servers             []FrontServerConfig  `json:"servers" toml:"servers"`
	Error               ErrorConfig          `json:"error" toml:"error"`
	CustomHeaders       []CustomHeaderConfig `json:"custom_headers" toml:"custom_headers"`
}

func LoadFrontConfig(file string) (PxyFrontConfig, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return PxyFrontConfig{}, err
	}
	var config AppConfig
	ext := filepath.Ext(file)
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &config)
		return PxyFrontConfig{}, err
	case ".toml":
		err = toml.Unmarshal(data, &config)
		return PxyFrontConfig{}, err
	default:
		err = json.Unmarshal(data, &config)
		return PxyFrontConfig{}, err
	}
}
