package utils

import (
	"Hamburger/internal/dsl_conf"
	"Hamburger/internal/json"
	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const (
	DataExtJSON      = ".json"
	DataExtTOML      = ".toml"
	DataExtYAML      = ".yaml"
	DataExtHamburger = ".hamburger"
)

func DataUnmarshal(ext string, data []byte, v interface{}) error {
	switch ext {
	case DataExtJSON:
		return json.Unmarshal(data, v)
	case DataExtTOML:
		return toml.Unmarshal(data, v)
	case DataExtHamburger:
		return dsl_conf.Unmarshal(data, v)
	case DataExtYAML:
		return yaml.Unmarshal(data, v)
	default:
		return json.Unmarshal(data, v)
	}
}

func FileUnmarshal(file string, v interface{}) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	ext := filepath.Ext(file)
	switch ext {
	case DataExtJSON:
		err = json.Unmarshal(data, v)
		return err
	case DataExtTOML:
		err = toml.Unmarshal(data, v)
		return err
	case DataExtYAML:
		return yaml.Unmarshal(data, v)
	case DataExtHamburger:
		err = dsl_conf.Unmarshal(data, v)
		return err
	default:
		err = json.Unmarshal(data, v)
		return err
	}
}
