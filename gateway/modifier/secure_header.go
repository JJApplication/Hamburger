package modifier

import (
	"Hamburger/internal/config"
	"Hamburger/internal/utils"
	"net/http"
)

type SecureHeaderModifier struct {
	enable bool
}

func NewSecureHeaderModifier() *SecureHeaderModifier {
	cfg := config.Get()
	mod := new(SecureHeaderModifier)
	mod.enable = cfg.Features.SecureHeader
	return mod
}

func (s SecureHeaderModifier) Use(response *http.Response) {
	_ = s.ModifyResponse(response)
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
