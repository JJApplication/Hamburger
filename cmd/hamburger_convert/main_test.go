package main

import (
	"Hamburger/internal/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConvertFileJSONToHamburger(t *testing.T) {
	tmpDir := t.TempDir()
	inFile := filepath.Join(tmpDir, "input.json")
	outFile := filepath.Join(tmpDir, "output.hamburger")

	content := `{"name":"demo","port":8080,"enabled":true,"items":[1,2,3]}`
	if err := os.WriteFile(inFile, []byte(content), 0644); err != nil {
		t.Fatalf("write input file failed: %v", err)
	}

	if _, err := convertFile(inFile, outFile, "", ""); err != nil {
		t.Fatalf("convert file failed: %v", err)
	}

	outData, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output failed: %v", err)
	}
	var cfg struct {
		Name    string `json:"name"`
		Port    int    `json:"port"`
		Enabled bool   `json:"enabled"`
		Items   []int  `json:"items"`
	}
	if err := decodeByFormat(formatHamburger, outData, &cfg); err != nil {
		t.Fatalf("decode hamburger failed: %v", err)
	}
	if cfg.Name != "demo" || cfg.Port != 8080 || !cfg.Enabled || len(cfg.Items) != 3 {
		t.Fatalf("unexpected output cfg: %#v", cfg)
	}
}

func TestConvertFileHamburgerToYAML(t *testing.T) {
	tmpDir := t.TempDir()
	inFile := filepath.Join(tmpDir, "input.hamburger")
	outFile := filepath.Join(tmpDir, "output.yaml")

	content := `
{
  name: "demo",
  value: 123,
  nested: {
    enabled: true
  }
}
`
	if err := os.WriteFile(inFile, []byte(content), 0644); err != nil {
		t.Fatalf("write input failed: %v", err)
	}

	if _, err := convertFile(inFile, outFile, "", ""); err != nil {
		t.Fatalf("convert file failed: %v", err)
	}

	outData, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output failed: %v", err)
	}
	var data map[string]any
	if err := decodeByFormat(formatYAML, outData, &data); err != nil {
		t.Fatalf("decode yaml failed: %v", err)
	}
	if data["name"] != "demo" {
		t.Fatalf("unexpected name: %#v", data["name"])
	}
}

func TestResolveOutputFormat(t *testing.T) {
	if _, err := resolveOutputFormat("", ""); err == nil {
		t.Fatalf("expected resolve output format error")
	}
	if v, err := resolveOutputFormat("yaml", ""); err != nil || v != formatYAML {
		t.Fatalf("unexpected output format: %s, err=%v", v, err)
	}
}

func TestEncodeJSON(t *testing.T) {
	out, err := encodeByFormat(formatJSON, map[string]any{
		"name": "demo",
		"port": 8080,
	})
	if err != nil {
		t.Fatalf("encode json failed: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(out, &data); err != nil {
		t.Fatalf("decode json failed: %v", err)
	}
	if data["name"] != "demo" {
		t.Fatalf("unexpected data: %#v", data)
	}
}
