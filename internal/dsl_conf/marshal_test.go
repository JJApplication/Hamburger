package dsl_conf

import (
	"strings"
	"testing"
)

type marshalNested struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

type marshalConfig struct {
	Name    string                 `json:"name"`
	Count   int                    `json:"count"`
	Labels  []string               `json:"labels"`
	Headers map[string]string      `json:"headers"`
	Nested  marshalNested          `json:"nested"`
	AnyMap  map[string]any         `json:"any_map"`
	Numbers []int                  `json:"numbers"`
	NullRef *marshalNested         `json:"null_ref"`
	Meta    map[string]interface{} `json:"meta"`
}

func TestMarshalAndUnmarshalRoundTrip(t *testing.T) {
	src := marshalConfig{
		Name:   "hamburger",
		Count:  3,
		Labels: []string{"a", "b"},
		Headers: map[string]string{
			"Proxy-Server": "Hamburger",
			"trace.id":     "x-1",
		},
		Nested: marshalNested{
			Enabled: true,
			Port:    8080,
		},
		AnyMap: map[string]any{
			"k1":  "v1",
			"k2":  2.5,
			"403": "forbidden",
		},
		Numbers: []int{1, 2, 3},
		NullRef: nil,
		Meta: map[string]interface{}{
			"x": true,
			"y": nil,
		},
	}

	data, err := MarshalIndent(src, "", "  ")
	if err != nil {
		t.Fatalf("marshal indent failed: %v", err)
	}

	text := string(data)
	if !strings.Contains(text, "\"403\"") {
		t.Fatalf("expected quoted map key, got: %s", text)
	}
	if !strings.Contains(text, "nested: {") {
		t.Fatalf("expected nested object, got: %s", text)
	}

	var out marshalConfig
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal round-trip failed: %v", err)
	}

	if out.Name != src.Name || out.Count != src.Count || out.Nested.Port != src.Nested.Port {
		t.Fatalf("unexpected round-trip result: %#v", out)
	}
	if len(out.Labels) != 2 || out.Labels[0] != "a" || out.Labels[1] != "b" {
		t.Fatalf("unexpected labels: %#v", out.Labels)
	}
	if out.Headers["Proxy-Server"] != "Hamburger" {
		t.Fatalf("unexpected headers: %#v", out.Headers)
	}
	if out.NullRef != nil {
		t.Fatalf("expected nil null_ref, got: %#v", out.NullRef)
	}
}

func TestMarshalCompact(t *testing.T) {
	data, err := Marshal(map[string]any{
		"name": "h",
		"port": 80,
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "\n") {
		t.Fatalf("compact marshal should not contain line break: %q", text)
	}
	if !strings.Contains(text, "{") || !strings.Contains(text, "}") {
		t.Fatalf("invalid marshal result: %q", text)
	}
}
