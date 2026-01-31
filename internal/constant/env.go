/*
   Create: 2025/8/5
   Project: AppName
   Github: https://github.com/landers1037
   Copyright Renj
*/

package constant

import (
	"os"
	"strconv"
)

type SandwichEnv struct {
	Raw string
}

func LoaderEnv(env string) *SandwichEnv {
	return &SandwichEnv{Raw: os.Getenv(env)}
}

func (s *SandwichEnv) String(def string) string {
	if s.Raw == "" {
		return def
	}
	return s.Raw
}

func (s *SandwichEnv) Int(def int) int {
	if s.Raw == "" {
		return def
	}
	i, err := strconv.Atoi(s.Raw)
	if err != nil {
		return def
	}
	return i
}

func (s *SandwichEnv) Float(def float64) float64 {
	if s.Raw == "" {
		return def
	}
	f, err := strconv.ParseFloat(s.Raw, 64)
	if err != nil {
		return def
	}
	return f
}

func (s *SandwichEnv) Bool(def bool) bool {
	if s.Raw == "" {
		return def
	}
	b, err := strconv.ParseBool(s.Raw)
	if err != nil {
		return def
	}
	return b
}
