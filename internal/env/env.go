package env

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

func LoadFromConfigFile(configFile string) error {
	trimmed := strings.TrimSpace(configFile)
	if trimmed == "" {
		trimmed = filepath.Join("config", "config.hamburger")
	}
	dir := filepath.Dir(filepath.Clean(trimmed))
	if !filepath.IsAbs(dir) {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		dir = absDir
	}
	return LoadDir(dir)
}

func LoadDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", dir)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.env"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	protected := snapshotEnvKeys()
	for _, file := range files {
		if err := loadFile(file, protected); err != nil {
			return err
		}
	}
	return nil
}

func loadFile(file string, protected map[string]struct{}) error {
	pairs, err := godotenv.Read(file)
	if err != nil {
		return err
	}
	for key, value := range pairs {
		if _, exists := protected[key]; exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

func snapshotEnvKeys() map[string]struct{} {
	out := map[string]struct{}{}
	for _, item := range os.Environ() {
		if item == "" {
			continue
		}
		idx := strings.Index(item, "=")
		if idx <= 0 {
			continue
		}
		out[item[:idx]] = struct{}{}
	}
	return out
}
