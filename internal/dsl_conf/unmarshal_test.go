package dsl_conf

import (
	"Hamburger/internal/constant"
	"runtime"
	"strings"
	"testing"
	"time"
)

type sampleConfig struct {
	Name     string            `json:"name"`
	Port     int               `json:"port"`
	Size     int               `json:"size"`
	Enabled  bool              `json:"enabled"`
	NowDate  string            `json:"now_date"`
	NowTime  string            `json:"now_time"`
	Headers  map[string]string `json:"headers"`
	Numbers  []int             `json:"numbers"`
	Computed int               `json:"computed"`
}

func TestUnmarshalDSL(t *testing.T) {
	t.Setenv("DSL_PORT", "8088")
	t.Setenv("DSL_ENABLED", "true")
	data := `
# 顶部注释
{
  name: @AppName,
  port: $DSL_PORT,
  size: 1<<10,
  enabled: $DSL_ENABLED,
  now_date: @DATE,
  now_time: @DATETIME,
  headers: {
    Server: @AppName,
    Copyright: @Copyright
  },
  numbers: [1, 2, 3+4,],
  computed: (5+5)*2
}
`
	var cfg sampleConfig
	if err := Unmarshal([]byte(data), &cfg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if cfg.Name != constant.AppName {
		t.Fatalf("unexpected name: %s", cfg.Name)
	}
	if cfg.Port != 8088 {
		t.Fatalf("unexpected port: %d", cfg.Port)
	}
	if cfg.Size != 1024 {
		t.Fatalf("unexpected size: %d", cfg.Size)
	}
	if !cfg.Enabled {
		t.Fatalf("unexpected enabled: %v", cfg.Enabled)
	}
	if _, err := time.Parse("2006-01-02", cfg.NowDate); err != nil {
		t.Fatalf("invalid date: %s", cfg.NowDate)
	}
	if _, err := time.Parse("2006-01-02 15:04:05", cfg.NowTime); err != nil {
		t.Fatalf("invalid datetime: %s", cfg.NowTime)
	}
	if cfg.Headers["Server"] != constant.AppName {
		t.Fatalf("unexpected header Server: %s", cfg.Headers["Server"])
	}
	if cfg.Headers["Copyright"] != constant.Copyright {
		t.Fatalf("unexpected header Copyright: %s", cfg.Headers["Copyright"])
	}
	if len(cfg.Numbers) != 3 || cfg.Numbers[2] != 7 {
		t.Fatalf("unexpected numbers: %v", cfg.Numbers)
	}
	if cfg.Computed != 20 {
		t.Fatalf("unexpected computed: %d", cfg.Computed)
	}
}

func TestRuntimeSymbols(t *testing.T) {
	type runtimeCfg struct {
		Arch      string `json:"arch"`
		OS        string `json:"os"`
		Version   string `json:"version"`
		NumCore   int    `json:"num_core"`
		Kernel    string `json:"kernel"`
		NowDate   string `json:"now_date"`
		NowTime   string `json:"now_time"`
		AppName   string `json:"app_name"`
		Copyright string `json:"copyright"`
	}
	data := `
{
  arch: @ARCH,
  os: @GOOS,
  version: @GOVERSION,
  num_core: @NUMCORE,
  kernel: @KERNEL,
  now_date: @DATE,
  now_time: @DATETIME,
  app_name: @AppName,
  copyright: @Copyright
}
`
	var cfg runtimeCfg
	if err := Unmarshal([]byte(data), &cfg); err != nil {
		t.Fatalf("unmarshal runtime symbols failed: %v", err)
	}
	if cfg.Arch != runtime.GOARCH {
		t.Fatalf("unexpected arch: %s", cfg.Arch)
	}
	if cfg.OS != runtime.GOOS {
		t.Fatalf("unexpected os: %s", cfg.OS)
	}
	if !strings.HasPrefix(cfg.Version, "go") {
		t.Fatalf("unexpected go version: %s", cfg.Version)
	}
	if cfg.NumCore < 1 {
		t.Fatalf("unexpected num_core: %d", cfg.NumCore)
	}
	if cfg.Kernel == "" {
		t.Fatalf("unexpected kernel: %q", cfg.Kernel)
	}
	if cfg.AppName != constant.AppName {
		t.Fatalf("unexpected app_name: %s", cfg.AppName)
	}
	if cfg.Copyright != constant.Copyright {
		t.Fatalf("unexpected copyright: %s", cfg.Copyright)
	}
}

func TestWithSymbols(t *testing.T) {
	type symbolCfg struct {
		Message string `json:"message"`
	}
	data := `{ message: @CustomMessage }`
	var cfg symbolCfg
	err := Unmarshal([]byte(data), &cfg, WithSymbols(map[string]any{
		"CustomMessage": "hello",
	}))
	if err != nil {
		t.Fatalf("unmarshal with symbols failed: %v", err)
	}
	if cfg.Message != "hello" {
		t.Fatalf("unexpected message: %s", cfg.Message)
	}
}

func TestMissingEnv(t *testing.T) {
	type envCfg struct {
		Value int `json:"value"`
	}
	data := `{ value: $NOT_EXIST_ENV }`
	var cfg envCfg
	err := Unmarshal([]byte(data), &cfg)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "NOT_EXIST_ENV") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNullCollections(t *testing.T) {
	type nullCfg struct {
		Module []sampleConfig    `json:"module"`
		Dict   map[string]string `json:"dict"`
	}
	data := `
{
  module: null,
  dict: null
}
`
	var cfg nullCfg
	if err := Unmarshal([]byte(data), &cfg); err != nil {
		t.Fatalf("unmarshal null collections failed: %v", err)
	}
	if cfg.Module != nil {
		t.Fatalf("expected nil module, got: %#v", cfg.Module)
	}
	if cfg.Dict != nil {
		t.Fatalf("expected nil dict, got: %#v", cfg.Dict)
	}
}
