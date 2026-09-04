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
var configReloadLock sync.Mutex

// WithReloadLock serializes configuration mutations performed by the
// management API and server control callbacks. Connect keeps its own atomic
// snapshot, while the rest of the application continues to use the shared
// in-place configuration pointer.
func WithReloadLock(fn func() error) error {
	configReloadLock.Lock()
	defer configReloadLock.Unlock()
	return fn()
}

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

// Get returns the legacy shared configuration pointer. The pointer lock does
// not protect later field reads; concurrent request handlers must use Snapshot
// or a dedicated accessor such as IsDevMode instead.
func Get() *config.Config {
	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()
	return globalConfig
}

// Snapshot copies the current configuration while reload writers are excluded.
// Nested maps and slices are read-only: reload replaces them instead of mutating
// their contents. Do not use the returned pointer as an in-place reload target.
func Snapshot() *config.Config {
	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()
	return copyConfig(globalConfig)
}

// SnapshotOf reads a retained configuration pointer under the same lock used
// by ReplaceInPlace, for components constructed with a specific configuration.
func SnapshotOf(cfg *config.Config) *config.Config {
	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()
	return copyConfig(cfg)
}

func copyConfig(cfg *config.Config) *config.Config {
	if cfg == nil {
		return nil
	}
	snapshot := *cfg
	return &snapshot
}

// ReplaceInPlace preserves the pointer retained by existing components while
// synchronizing the entire struct assignment with snapshot readers. A nil
// target selects the current global configuration (or initializes it).
func ReplaceInPlace(target, next *config.Config) {
	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()
	if target == nil {
		target = globalConfig
	}
	if target == nil {
		target = new(config.Config)
	}
	*target = *next
	globalConfig = target
}

// IsDevMode avoids exposing a mutable configuration pointer to authentication.
func IsDevMode() bool {
	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()
	return globalConfig != nil && globalConfig.DevMode
}
