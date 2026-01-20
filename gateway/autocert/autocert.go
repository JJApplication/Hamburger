package autocert

// 自动TLS证书申请

import (
	"golang.org/x/crypto/acme/autocert"
)

// NewCertManager 无法使用dns使用80端口验证 需要在验证时关闭80端口
func NewCertManager(domains []string, email string) *autocert.Manager {
	cm := &autocert.Manager{
		Prompt:                 autocert.AcceptTOS,
		Cache:                  autocert.DirCache("./autocert"),
		HostPolicy:             autocert.HostWhitelist(domains...),
		RenewBefore:            0,
		Client:                 nil,
		Email:                  email,
		ForceRSA:               false,
		ExtraExtensions:        nil,
		ExternalAccountBinding: nil,
	}

	return cm
}
