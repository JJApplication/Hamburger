package main

import (
	"Hamburger/internal/dsl_conf"
	"Hamburger/internal/json"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

const (
	formatJSON      = "json"
	formatYAML      = "yaml"
	formatTOML      = "toml"
	formatHamburger = "hamburger"
)

func main() {
	var inFile string
	var outFile string
	var from string
	var to string

	flag.StringVar(&inFile, "in", "", "输入文件路径")
	flag.StringVar(&outFile, "out", "", "输出文件路径，留空时输出到标准输出")
	flag.StringVar(&from, "from", "", "输入格式，可选: json|yaml|toml|hamburger")
	flag.StringVar(&to, "to", "", "输出格式，可选: json|yaml|toml|hamburger")
	flag.Parse()

	if strings.TrimSpace(inFile) == "" {
		writeErrorAndExit("参数 -in 不能为空")
	}

	output, err := convertFile(inFile, outFile, from, to)
	if err != nil {
		writeErrorAndExit(err.Error())
	}

	if strings.TrimSpace(outFile) == "" {
		_, _ = os.Stdout.Write(output)
	}
}

func convertFile(inFile, outFile, from, to string) ([]byte, error) {
	fromFormat, err := resolveFormat(from, inFile)
	if err != nil {
		return nil, err
	}

	toFormat, err := resolveOutputFormat(to, outFile)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(inFile)
	if err != nil {
		return nil, fmt.Errorf("读取输入文件失败: %w", err)
	}

	var decoded any
	if err := decodeByFormat(fromFormat, data, &decoded); err != nil {
		return nil, fmt.Errorf("解析输入文件失败: %w", err)
	}

	encoded, err := encodeByFormat(toFormat, decoded)
	if err != nil {
		return nil, fmt.Errorf("序列化输出内容失败: %w", err)
	}

	if strings.TrimSpace(outFile) != "" {
		if err := os.WriteFile(outFile, encoded, 0644); err != nil {
			return nil, fmt.Errorf("写入输出文件失败: %w", err)
		}
	}

	return encoded, nil
}

func resolveOutputFormat(to, outFile string) (string, error) {
	if parsed, err := parseFormat(to); err == nil {
		return parsed, nil
	}
	if strings.TrimSpace(outFile) != "" {
		return resolveFormat("", outFile)
	}
	return "", errors.New("无法确定输出格式，请指定 -to 或带有可识别后缀的 -out")
}

func resolveFormat(flagFormat, file string) (string, error) {
	if parsed, err := parseFormat(flagFormat); err == nil {
		return parsed, nil
	}
	ext := strings.ToLower(filepath.Ext(file))
	return parseFormat(ext)
}

func parseFormat(raw string) (string, error) {
	v := strings.TrimSpace(strings.ToLower(raw))
	switch v {
	case ".json", formatJSON:
		return formatJSON, nil
	case ".yaml", ".yml", formatYAML, "yml":
		return formatYAML, nil
	case ".toml", formatTOML:
		return formatTOML, nil
	case ".hamburger", formatHamburger:
		return formatHamburger, nil
	default:
		return "", fmt.Errorf("不支持的格式: %s", raw)
	}
}

func decodeByFormat(format string, data []byte, v any) error {
	switch format {
	case formatJSON:
		return json.Unmarshal(data, v)
	case formatYAML:
		return yaml.Unmarshal(data, v)
	case formatTOML:
		return toml.Unmarshal(data, v)
	case formatHamburger:
		return dsl_conf.Unmarshal(data, v)
	default:
		return fmt.Errorf("不支持的输入格式: %s", format)
	}
}

func encodeByFormat(format string, v any) ([]byte, error) {
	switch format {
	case formatJSON:
		return json.MarshalIndent(v, "", "  ")
	case formatYAML:
		return yaml.Marshal(v)
	case formatTOML:
		var buf bytes.Buffer
		encoder := toml.NewEncoder(&buf)
		if err := encoder.Encode(v); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case formatHamburger:
		return dsl_conf.MarshalIndent(v, "", "  ")
	default:
		return nil, fmt.Errorf("不支持的输出格式: %s", format)
	}
}

func writeErrorAndExit(msg string) {
	_, _ = os.Stderr.WriteString(msg + "\n")
	os.Exit(1)
}
