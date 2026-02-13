package tls

import (
	"Hamburger/internal/config"
	"crypto/tls"
	"fmt"
)

// 处理多域名SNI

func (m *TLSManager) GetCertificateFunc() func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if info == nil {
			return nil, nil
		}
		sniName := info.ServerName
		if sniName == "" {
			return nil, nil
		}
		domainCert, domainKey := m.GetCert(sniName)
		if domainCert == "" || domainKey == "" {
			return nil, nil
		}
		// 加载证书和私钥
		cert, err := tls.LoadX509KeyPair(domainCert, domainKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate: %v", err)
		}

		return &cert, nil
	}
}

func (m *TLSManager) GetCert(sni string) (certFile, keyFile string) {
	m.certMu.RLock()
	defer m.certMu.RUnlock()
	for _, cert := range m.certMap {
		for _, domain := range cert.Domains {
			if sni == domain {
				return cert.CertFile, cert.KeyFile
			}
		}
	}

	return "", ""
}

func (m *TLSManager) InitCertMap() {
	m.certMu.Lock()
	defer m.certMu.Unlock()

	m.certMap = make(map[string]config.CertConfig)
	for _, server := range m.config.Servers {
		if server.TLS != nil && server.TLS.CertMap != nil {
			for groupName, cert := range server.TLS.CertMap {
				m.certMap[groupName] = cert
			}
		}
	}
}
