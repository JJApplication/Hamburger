package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.env")
	fileB := filepath.Join(dir, "b.env")
	contentA := `
# base env
APP_NAME="hamburger dev"
APP_PORT=8080
TEXT_WITH_HASH=abc#v1
SINGLE='quoted value'
`
	contentB := `
APP_PORT=9090
APP_HOST=127.0.0.1
`
	if err := os.WriteFile(fileA, []byte(contentA), 0644); err != nil {
		t.Fatalf("write a.env failed: %v", err)
	}
	if err := os.WriteFile(fileB, []byte(contentB), 0644); err != nil {
		t.Fatalf("write b.env failed: %v", err)
	}

	if err := LoadDir(dir); err != nil {
		t.Fatalf("load dir failed: %v", err)
	}

	if v := os.Getenv("APP_NAME"); v != "hamburger dev" {
		t.Fatalf("unexpected APP_NAME: %q", v)
	}
	if v := os.Getenv("APP_PORT"); v != "9090" {
		t.Fatalf("unexpected APP_PORT: %q", v)
	}
	if v := os.Getenv("APP_HOST"); v != "127.0.0.1" {
		t.Fatalf("unexpected APP_HOST: %q", v)
	}
	if v := os.Getenv("TEXT_WITH_HASH"); v != "abc#v1" {
		t.Fatalf("unexpected TEXT_WITH_HASH: %q", v)
	}
	if v := os.Getenv("SINGLE"); v != "quoted value" {
		t.Fatalf("unexpected SINGLE: %q", v)
	}
}

func TestLoadDirKeepSystemEnv(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "hamburger.env")
	contentA := "APP_PORT=8080\n"
	if err := os.WriteFile(fileA, []byte(contentA), 0644); err != nil {
		t.Fatalf("write env file failed: %v", err)
	}
	t.Setenv("APP_PORT", "6000")

	if err := LoadDir(dir); err != nil {
		t.Fatalf("load dir failed: %v", err)
	}
	if v := os.Getenv("APP_PORT"); v != "6000" {
		t.Fatalf("system env should not be overridden: %q", v)
	}
}

func TestLoadFromConfigFile(t *testing.T) {
	base := t.TempDir()
	configDir := filepath.Join(base, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "hamburger.env"), []byte("APP_TOKEN=abc\n"), 0644); err != nil {
		t.Fatalf("write env failed: %v", err)
	}
	configFile := filepath.Join(configDir, "config.hamburger")
	if err := os.WriteFile(configFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("write config file failed: %v", err)
	}

	if err := LoadFromConfigFile(configFile); err != nil {
		t.Fatalf("load from config failed: %v", err)
	}
	if v := os.Getenv("APP_TOKEN"); v != "abc" {
		t.Fatalf("unexpected APP_TOKEN: %q", v)
	}
}
