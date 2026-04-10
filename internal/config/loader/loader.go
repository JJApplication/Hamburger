package loader

import (
	"Hamburger/internal/config"
	"Hamburger/internal/dsl_conf"
	"Hamburger/internal/env"
	"Hamburger/internal/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
)

// 配置加载器 加载不同配置后合并
// 优先级顺序逐层增加

// config模型为最终的全局配置模型
// app_config为配置文件的格式

var globalConfig *config.Config
var globalConfigLock sync.RWMutex

func LoadConfig(file string) (*config.AppConfig, error) {
	if err := env.LoadFromConfigFile(file); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var config config.AppConfig
	ext := filepath.Ext(file)
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &config)
		return &config, err
	case ".toml":
		err = toml.Unmarshal(data, &config)
		return &config, err
	case ".hamburger":
		err = dsl_conf.Unmarshal(data, &config)
		return &config, err
	default:
		err = json.Unmarshal(data, &config)
		return &config, err
	}
}

func Set(cfg *config.Config) {
	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()
	globalConfig = cfg
}

// Get 获取全局唯一的配置
func Get() *config.Config {
	return globalConfig
}
