package service

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/dsl_conf"
	"Hamburger/internal/json"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// ManagementConfig is the safe editable view used by the dashboard. Secrets
// are replaced with a marker and are never returned to the browser.
type ManagementConfig struct {
	SourceFile string                 `json:"source_file"`
	Version    string                 `json:"version"`
	Values     map[string]interface{} `json:"values"`
	Pending    bool                   `json:"pending"`
}

type managementOperation struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

var managementStore = struct {
	sync.Mutex
	applyMu sync.Mutex
	pending map[string]bool
	ops     map[string]managementOperation
}{pending: map[string]bool{}, ops: map[string]managementOperation{}}

func (s *APIService) managementSource() string {
	if s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.SourceFile) != "" {
		return s.cfg.SourceFile
	}
	return "config/config.hamburger"
}

func configVersion(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// editableRoot decodes the original document into a generic tree so fields
// that are newer than this binary (or intentionally unknown to AppConfig) are
// carried through a JSON/TOML save.  Validation still happens against the
// typed AppConfig model below before the replacement is committed.
func editableRoot(data []byte, path string) (map[string]interface{}, error) {
	root := map[string]interface{}{}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".hamburger":
		if err := dsl_conf.Unmarshal(data, &root); err != nil {
			return nil, err
		}
	case ".toml":
		if _, err := toml.Decode(string(data), &root); err != nil {
			return nil, err
		}
	default:
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, err
		}
	}
	if root == nil {
		root = map[string]interface{}{}
	}
	return root, nil
}

func redactConfig(value interface{}) {
	switch item := value.(type) {
	case map[string]interface{}:
		for key, child := range item {
			lower := strings.ToLower(key)
			secret := strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "private_key") || lower == "key"
			if secret {
				if text, ok := child.(string); ok && strings.TrimSpace(text) != "" {
					item[key] = "********"
				}
				continue
			}
			redactConfig(child)
		}
	case []interface{}:
		for _, child := range item {
			redactConfig(child)
		}
	}
}

func (s *APIService) GetManagementConfig() (ManagementConfig, error) {
	path := s.managementSource()
	data, err := os.ReadFile(path)
	if err != nil {
		return ManagementConfig{}, err
	}
	if _, err := loader.LoadConfig(path); err != nil {
		return ManagementConfig{}, err
	}
	values, err := editableRoot(data, path)
	if err != nil {
		return ManagementConfig{}, err
	}
	redactConfig(values)
	managementStore.Lock()
	pending := managementStore.pending[path]
	managementStore.Unlock()
	return ManagementConfig{SourceFile: path, Version: configVersion(data), Values: values, Pending: pending}, nil
}

func setConfigPath(root map[string]interface{}, path string, value interface{}) error {
	parts := strings.Split(strings.Trim(path, "."), ".")
	if len(parts) == 0 || parts[0] == "" {
		return errors.New("configuration field is empty")
	}
	current := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]interface{})
		if !ok {
			return fmt.Errorf("configuration field %s does not exist", path)
		}
		current = next
	}
	last := parts[len(parts)-1]
	if _, ok := current[last]; !ok {
		// Trojan's explicit enabled flag was added after the original
		// configuration format.  It is intentionally optional so old files can
		// opt in without forcing users to rewrite the rest of the object.
		if strings.Join(parts, ".") == "exp_config.trojan_enabled" {
			current[last] = value
			return nil
		}
		return fmt.Errorf("configuration field %s does not exist", path)
	}
	// A masked secret means “leave the existing value unchanged”.
	if text, ok := value.(string); ok && text == "********" {
		return nil
	}
	current[last] = value
	return nil
}

func encodeConfigFile(root map[string]interface{}, path string) ([]byte, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".hamburger":
		return dsl_conf.MarshalIndent(root, "", "  ")
	case ".toml":
		var builder strings.Builder
		if err := toml.NewEncoder(&builder).Encode(root); err != nil {
			return nil, err
		}
		return []byte(builder.String()), nil
	default:
		return json.MarshalIndent(root, "", "  ")
	}
}

func dslLiteral(value interface{}) string {
	switch item := value.(type) {
	case string:
		return strconv.Quote(item)
	case bool:
		if item {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(item, 'f', -1, 64)
	case int, int8, int16, int32, int64:
		return fmt.Sprint(item)
	default:
		encoded, _ := dsl_conf.Marshal(item)
		return string(encoded)
	}
}

// patchDSLSource changes scalar fields in place. This keeps comments,
// expressions and environment references outside the edited field intact.
func patchDSLSource(data []byte, updates map[string]interface{}) ([]byte, bool, error) {
	lines := strings.SplitAfter(string(data), "\n")
	stack := []string{}
	matched := map[string]bool{}
	fieldPattern := regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_-]*)\s*:\s*([^#\r\n]*?)(\s*(?:#.*)?\r?\n?)$`)
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		match := fieldPattern.FindStringSubmatch(line)
		if len(match) > 0 {
			pathParts := make([]string, 0, len(stack)+1)
			for _, item := range stack {
				if item != "" {
					pathParts = append(pathParts, item)
				}
			}
			pathParts = append(pathParts, match[2])
			path := strings.Join(pathParts, ".")
			if value, ok := updates[path]; ok {
				lines[index] = match[1] + match[2] + ": " + dslLiteral(value) + match[4]
				matched[path] = true
			}
		}
		key := ""
		if len(match) > 0 {
			key = match[2]
		}
		open := strings.Count(trimmed, "{")
		close := strings.Count(trimmed, "}")
		if open > 0 {
			for count := 0; count < open; count++ {
				if count == 0 && key != "" {
					stack = append(stack, key)
				} else {
					stack = append(stack, "")
				}
			}
		}
		for count := 0; count < close && len(stack) > 0; count++ {
			stack = stack[:len(stack)-1]
		}
	}
	for key := range updates {
		if !matched[key] && key == "exp_config.trojan_enabled" {
			patched, ok := insertDSLField([]byte(strings.Join(lines, "")), "exp_config", "trojan_enabled", updates[key])
			if ok {
				return patched, true, nil
			}
		}
		if !matched[key] {
			return nil, false, nil
		}
	}
	return []byte(strings.Join(lines, "")), true, nil
}

// insertDSLField adds the compatibility-only Trojan flag inside an existing
// object while leaving every other source line byte-for-byte unchanged.
func insertDSLField(data []byte, parent, field string, value interface{}) ([]byte, bool) {
	lines := strings.SplitAfter(string(data), "\n")
	parentPattern := regexp.MustCompile(`^(\s*)` + regexp.QuoteMeta(parent) + `\s*:\s*\{`)
	for start, line := range lines {
		match := parentPattern.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		depth := strings.Count(line, "{") - strings.Count(line, "}")
		for index := start + 1; index < len(lines); index++ {
			nextDepth := depth + strings.Count(lines[index], "{") - strings.Count(lines[index], "}")
			if nextDepth <= 0 {
				ending := "\n"
				if strings.Contains(line, "\r\n") {
					ending = "\r\n"
				}
				indent := match[1] + "  "
				lines = append(lines[:index], append([]string{indent + field + ": " + dslLiteral(value) + ending}, lines[index:]...)...)
				return []byte(strings.Join(lines, "")), true
			}
			depth = nextDepth
		}
		return nil, false
	}
	return nil, false
}

func validateConfigBytes(data []byte, path string) error {
	var target config.AppConfig
	switch strings.ToLower(filepath.Ext(path)) {
	case ".hamburger":
		return dsl_conf.Unmarshal(data, &target)
	case ".toml":
		_, err := toml.Decode(string(data), &target)
		return err
	default:
		return json.Unmarshal(data, &target)
	}
}

// SaveManagementConfig validates all updates before atomically replacing the
// source file and marks it pending until the user applies the new snapshot.
func (s *APIService) SaveManagementConfig(version string, updates map[string]interface{}) (ManagementConfig, error) {
	path := s.managementSource()
	data, err := os.ReadFile(path)
	if err != nil {
		return ManagementConfig{}, err
	}
	if strings.TrimSpace(version) != "" && configVersion(data) != strings.TrimSpace(version) {
		return ManagementConfig{}, errors.New("configuration changed outside dashboard")
	}
	if _, err := loader.LoadConfig(path); err != nil {
		return ManagementConfig{}, err
	}
	root, err := editableRoot(data, path)
	if err != nil {
		return ManagementConfig{}, err
	}
	for key, value := range updates {
		if err := setConfigPath(root, key, value); err != nil {
			return ManagementConfig{}, err
		}
	}
	encoded, patched, patchErr := []byte(nil), false, error(nil)
	if strings.EqualFold(filepath.Ext(path), ".hamburger") {
		encoded, patched, patchErr = patchDSLSource(data, updates)
		if patchErr != nil {
			return ManagementConfig{}, patchErr
		}
	}
	if !patched {
		encoded, err = encodeConfigFile(root, path)
	}
	if err != nil {
		return ManagementConfig{}, err
	}
	if err := validateConfigBytes(encoded, path); err != nil {
		return ManagementConfig{}, err
	}
	tmp := fmt.Sprintf("%s.dashboard.%d.tmp", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, encoded, 0644); err != nil {
		return ManagementConfig{}, err
	}
	backup := fmt.Sprintf("%s.dashboard.%d.bak", path, time.Now().UnixNano())
	if err := os.Rename(path, backup); err != nil {
		_ = os.Remove(tmp)
		return ManagementConfig{}, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Rename(backup, path)
		_ = os.Remove(tmp)
		return ManagementConfig{}, err
	}
	managementStore.Lock()
	managementStore.pending[path] = true
	managementStore.Unlock()
	return s.GetManagementConfig()
}

func (s *APIService) MarkManagementApplied() {
	path := s.managementSource()
	managementStore.Lock()
	delete(managementStore.pending, path)
	managementStore.Unlock()
}

func NewManagementOperation() string {
	id := fmt.Sprintf("op-%d", time.Now().UnixNano())
	managementStore.Lock()
	managementStore.ops[id] = managementOperation{Status: "queued"}
	managementStore.Unlock()
	return id
}

// ApplyManagementConfig serializes service restarts and exposes their result
// through the operation store used by both management protocol facades.
func (s *APIService) ApplyManagementConfig(services []string) string {
	selected := make([]string, 0, len(services))
	for _, name := range services {
		if value := strings.TrimSpace(name); value != "" {
			selected = append(selected, value)
		}
	}
	if len(selected) == 0 {
		selected = []string{"gateway"}
	}
	id := NewManagementOperation()
	go func() {
		ManagementApplyLock()
		defer ManagementApplyUnlock()
		SetManagementOperation(id, "running", nil)
		if s == nil {
			SetManagementOperation(id, "failed", errors.New("api service unavailable"))
			return
		}
		for _, name := range selected {
			if err := s.RestartServer(name); err != nil {
				SetManagementOperation(id, "failed", err)
				return
			}
		}
		s.MarkManagementApplied()
		SetManagementOperation(id, "complete", nil)
	}()
	return id
}

// ManagementApplyLock serializes configuration application across dashboard
// requests so two reloads cannot interleave service state transitions.
func ManagementApplyLock() { managementStore.applyMu.Lock() }

func ManagementApplyUnlock() { managementStore.applyMu.Unlock() }

func SetManagementOperation(id, status string, operationError error) {
	op := managementOperation{Status: status}
	if operationError != nil {
		op.Error = operationError.Error()
	}
	managementStore.Lock()
	managementStore.ops[id] = op
	managementStore.Unlock()
}

func GetManagementOperation(id string) (map[string]string, bool) {
	managementStore.Lock()
	op, ok := managementStore.ops[id]
	managementStore.Unlock()
	if !ok {
		return nil, false
	}
	result := map[string]string{"status": op.Status}
	if op.Error != "" {
		result["error"] = op.Error
	}
	return result, true
}
