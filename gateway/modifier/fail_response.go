package modifier

import (
	"Hamburger/internal/config/loader"
	"net/http"
)

type FailResponseModifier struct {
	enabled bool
	codes   map[int]struct{}
}

func NewFailResponseModifier() *FailResponseModifier {
	cfg := loader.Get()
	codes := cfg.Middleware.FailResponse.Code
	if len(codes) == 0 {
		codes = []int{http.StatusInternalServerError}
	}

	codeMap := make(map[int]struct{}, len(codes))
	for _, code := range codes {
		codeMap[code] = struct{}{}
	}

	return &FailResponseModifier{
		enabled: cfg.Middleware.FailResponse.Enabled,
		codes:   codeMap,
	}
}

func (f *FailResponseModifier) Use(response *http.Response) bool {
	if !f.enabled || response == nil {
		return true
	}
	if response.StatusCode < http.StatusInternalServerError || response.StatusCode > 599 {
		return true
	}
	_, blocked := f.codes[response.StatusCode]
	return !blocked
}

func (f *FailResponseModifier) ModifyResponse(response *http.Response) error {
	return nil
}

func (f *FailResponseModifier) IsEnabled() bool {
	return f.enabled
}

func (f *FailResponseModifier) UpdateConfig() {
	cfg := loader.Get()
	codes := cfg.Middleware.FailResponse.Code
	if len(codes) == 0 {
		codes = []int{http.StatusInternalServerError}
	}

	codeMap := make(map[int]struct{}, len(codes))
	for _, code := range codes {
		codeMap[code] = struct{}{}
	}

	f.enabled = cfg.Middleware.FailResponse.Enabled
	f.codes = codeMap
}

func (f *FailResponseModifier) GetName() string {
	return "fail-response"
}
