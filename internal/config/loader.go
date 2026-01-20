package config

import (
	"Hamburger/internal/json"
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
	"sync"
)

// 配置加载器 加载不同配置后合并
// 优先级顺序逐层增加

// config模型为最终的全局配置模型
// app_config为配置文件的格式

var globalConfig *Config
var globalConfigLock sync.RWMutex

func LoadConfig(file string) (*AppConfig, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var config AppConfig
	ext := filepath.Ext(file)
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &config)
		return &config, err
	case ".toml":
		err = toml.Unmarshal(data, &config)
		return &config, err
	default:
		err = json.Unmarshal(data, &config)
		return &config, err
	}
}

func Set(cfg *Config) {
	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()
	globalConfig = cfg
}

// Get 获取全局唯一的配置
func Get() *Config {
	return globalConfig
}
