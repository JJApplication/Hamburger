package service

import (
	"strings"
	"testing"
)

func TestEditableRootKeepsUnknownJSONFields(t *testing.T) {
	root, err := editableRoot([]byte(`{"proxy":{"proxy_mode":"fasthttp","future_field":{"enabled":true}}}`), "settings.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := setConfigPath(root, "proxy.proxy_mode", "http"); err != nil {
		t.Fatal(err)
	}
	encoded, err := encodeConfigFile(root, "settings.json")
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if !strings.Contains(text, `"future_field"`) || !strings.Contains(text, `"proxy_mode": "http"`) {
		t.Fatalf("unknown JSON field was lost: %s", text)
	}
}

func TestSetConfigPathAllowsTrojanCompatibilityFlag(t *testing.T) {
	root := map[string]interface{}{"exp_config": map[string]interface{}{"trojan_server": "trojan.json"}}
	if err := setConfigPath(root, "exp_config.trojan_enabled", false); err != nil {
		t.Fatal(err)
	}
	if root["exp_config"].(map[string]interface{})["trojan_enabled"] != false {
		t.Fatalf("compatibility flag was not inserted: %#v", root)
	}
}

func TestPatchDSLSourcePreservesUnchangedSource(t *testing.T) {
	source := []byte("# keep this comment\n{\n  proxy: {\n    # keep nested comment\n    proxy_mode: \"fasthttp\"\n    max_conns_per_host: 100\n  }\n}\n")
	patched, ok, err := patchDSLSource(source, map[string]interface{}{"proxy.proxy_mode": "http"})
	if err != nil || !ok {
		t.Fatalf("patchDSLSource() ok=%v err=%v", ok, err)
	}
	text := string(patched)
	if !strings.Contains(text, "# keep nested comment") || !strings.Contains(text, "proxy_mode: \"http\"") || !strings.Contains(text, "max_conns_per_host: 100") {
		t.Fatalf("source was not preserved: %s", text)
	}
}

func TestPatchDSLSourceRejectsUnknownField(t *testing.T) {
	patched, ok, err := patchDSLSource([]byte("{\n  proxy: {\n    proxy_mode: \"fasthttp\"\n  }\n}\n"), map[string]interface{}{"proxy.missing": true})
	if err != nil || ok || patched != nil {
		t.Fatalf("expected fallback for unknown field, got ok=%v err=%v", ok, err)
	}
}

func TestPatchDSLSourceAddsTrojanCompatibilityFlag(t *testing.T) {
	source := []byte("{\n  exp_config: {\n    trojan_server: \"config/trojan.json\"\n  }\n}\n")
	patched, ok, err := patchDSLSource(source, map[string]interface{}{"exp_config.trojan_enabled": false})
	if err != nil || !ok {
		t.Fatalf("patchDSLSource() ok=%v err=%v", ok, err)
	}
	text := string(patched)
	if !strings.Contains(text, "trojan_enabled: false") || !strings.Contains(text, "trojan_server: \"config/trojan.json\"") {
		t.Fatalf("compatibility flag or path missing: %s", text)
	}
}
