package modifier

import (
	"Hamburger/internal/config/loader"
	"Hamburger/internal/utils"
	"net/http"
)

type SecureHeaderModifier struct {
	enable bool
}

func NewSecureHeaderModifier() *SecureHeaderModifier {
	cfg := loader.Get()
	mod := new(SecureHeaderModifier)
	mod.enable = cfg.Middleware.SecureHeader
	return mod
}

func (s SecureHeaderModifier) Use(response *http.Response) bool {
	_ = s.ModifyResponse(response)
	return true
}

func (s SecureHeaderModifier) ModifyResponse(response *http.Response) error {
	if !s.enable {
		return nil
	}
	utils.AddSecureHeader(response)
	return nil
}

func (s SecureHeaderModifier) IsEnabled() bool {
	return s.enable
}

func (s SecureHeaderModifier) UpdateConfig() {
	//TODO implement me
	panic("implement me")
}

func (s SecureHeaderModifier) GetName() string {
	return "secure-header"
}
