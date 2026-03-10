package tls

import (
	autocert2 "Hamburger/gateway/autocert"
	"Hamburger/internal/config"
	"Hamburger/internal/structure"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"sort"
	"sync"
	"time"

	lltls "github.com/lesismal/llib/std/crypto/tls"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/singleflight"
)

var (
	TLSVersionMap = map[string]uint16{
		"TLS10": tls.VersionTLS10,
		"TLS11": tls.VersionTLS11,
		"TLS12": tls.VersionTLS12,
		"TLS13": tls.VersionTLS13,
	}
)

type TLSManager struct {
	config *config.Config
	logger *zerolog.Logger
	// AutoTLS
	AcmeMgr *autocert.Manager  // autocert 管理器
	acmeMU  sync.Mutex         // 刷新证书过程的互斥锁，避免并发冲突
	sf      singleflight.Group // 用于合并并发的证书请求
	// cert map
	certMu  sync.RWMutex
	certMap map[string]config.CertConfig
	// SelfTLS
	selfCertMu sync.Mutex
	selfCert   *tls.Certificate

	beforeAutoCert func() error
	afterAutoCert  func() error
}

func NewTLSManager(config *config.Config, logger *zerolog.Logger) *TLSManager {
	return &TLSManager{
		config: config,
		logger: logger,
	}
}

func (m *TLSManager) RegisterBeforeAutoCert(f func() error) {
	m.beforeAutoCert = f
}

func (m *TLSManager) RegisterAfterAutoCert(f func() error) {
	m.afterAutoCert = f
}

// ConfigureTLS 配置 TLS
func (m *TLSManager) ConfigureTLS(tlsConfig *config.TLSConfig, listener net.Listener) (*tls.Config, net.Listener, error) {
	if tlsConfig == nil {
		return nil, nil, fmt.Errorf("HTTPS server missing TLS configuration")
	}

	if tlsConfig.AutoTLS {
		// 使用 autocert 自动管理证书，返回用于标准 http.Server 的 *tls.Config
		// 构建域名白名单（必须提供域名，否则不可启动AutoTLS）
		domains := GetTlsDomains(m.config)
		if len(domains) == 0 {
			return nil, nil, fmt.Errorf("autotls enabled but no domains configured, cannot request certificate")
		}

		// 初始化或复用 autocert 管理器
		if m.AcmeMgr == nil {
			m.AcmeMgr = autocert2.NewCertManager(domains, m.config.Features.AutoCert.Email)
		}

		// 基础TLS配置来自autocert
		base := m.AcmeMgr.TLSConfig()
		// 包装 GetCertificate，在每次自动刷新前释放80端口并启用挑战处理，刷新后恢复
		origGetCert := base.GetCertificate
		base.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			// 使用singleflight合并并发的证书请求
			// 避免同一域名并发握手时多次触发80端口停止/启动
			val, err, _ := m.sf.Do("cert:"+hello.ServerName, func() (interface{}, error) {
				m.acmeMU.Lock()
				defer m.acmeMU.Unlock()

				m.logger.Info().Str("domain", hello.ServerName).Msg("autotls: about to obtain/renew certificate, preparing to handle port 80")
				if err := m.beforeAutoCert(); err != nil {
					m.logger.Error().Err(err).Msg("autotls: beforehandleautocert failed")
				}

				cert, err := origGetCert(hello)

				if err2 := m.afterAutoCert(); err2 != nil {
					m.logger.Error().Err(err2).Msg("autotls: afterhandleautocert failed")
				}

				if err != nil {
					m.logger.Error().Err(err).Msg("autotls: failed to obtain certificate")
					return nil, err
				} else {
					m.logger.Info().Str("domain", hello.ServerName).Msg("autotls: certificate obtained/renewed successfully")
					return cert, nil
				}
			})

			if err != nil {
				return nil, err
			}
			return val.(*tls.Certificate), nil
		}

		// 强化TLS安全参数
		base.MinVersion = GetTLSVersion(tlsConfig.MinVersion)
		base.PreferServerCipherSuites = true

		// 开启h2
		base.NextProtos = []string{"h2", "http/1.1"}

		// 应用 TLS 配置
		lis := tls.NewListener(listener, base)

		return base, lis, nil
	}

	if m.shouldUseSelfTLS(tlsConfig) {
		cert, err := m.getOrCreateSelfSignedCert()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{*cert},
			GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return cert, nil
			},
			MinVersion:               GetTLSVersion(tlsConfig.MinVersion),
			PreferServerCipherSuites: true,
			NextProtos:               []string{"h2", "http/1.1"},
		}
		lis := tls.NewListener(listener, tlsCfg)
		return tlsCfg, lis, nil
	}

	// 配置 TLS
	tlsCfg := &tls.Config{
		GetCertificate: m.GetCertificateFunc(),
		// 设置最低 TLS 版本
		MinVersion: GetTLSVersion(tlsConfig.MinVersion),
		// 优先使用服务器的密码套件顺序
		PreferServerCipherSuites: true,
		NextProtos:               []string{"h2", "http/1.1"},
	}

	// 应用 TLS 配置
	lis := tls.NewListener(listener, tlsCfg)

	return tlsCfg, lis, nil
}

func (m *TLSManager) GetTlsConfig(tlsConfig *config.TLSConfig) (*tls.Config, error) {
	if tlsConfig == nil {
		return nil, fmt.Errorf("HTTP3 server missing TLS configuration")
	}
	if m.shouldUseSelfTLS(tlsConfig) {
		cert, err := m.getOrCreateSelfSignedCert()
		if err != nil {
			return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{*cert},
			GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return cert, nil
			},
			MinVersion:               GetTLSVersion(tlsConfig.MinVersion),
			PreferServerCipherSuites: true,
			NextProtos:               []string{"h3", "h2", "http/1.1"},
		}, nil
	}
	tlsCfg := &tls.Config{
		GetCertificate: m.GetCertificateFunc(),
		// 设置最低 TLS 版本
		MinVersion: GetTLSVersion(tlsConfig.MinVersion),
		// 优先使用服务器的密码套件顺序
		PreferServerCipherSuites: true,
		NextProtos:               []string{"h3", "h2", "http/1.1"},
	}

	return tlsCfg, nil
}

func (m *TLSManager) GetNbioTLSConfig(tlsConfig *config.TLSConfig) (*lltls.Config, error) {
	if tlsConfig == nil {
		return nil, fmt.Errorf("HTTPS server missing TLS configuration")
	}

	if tlsConfig.AutoTLS {
		domains := GetTlsDomains(m.config)
		if len(domains) == 0 {
			return nil, fmt.Errorf("autotls enabled but no domains configured, cannot request certificate")
		}

		if m.AcmeMgr == nil {
			m.AcmeMgr = autocert2.NewCertManager(domains, m.config.Features.AutoCert.Email)
		}

		origGetCert := m.AcmeMgr.GetCertificate
		getCert := func(hello *lltls.ClientHelloInfo) (*lltls.Certificate, error) {
			if hello == nil {
				return nil, nil
			}
			val, err, _ := m.sf.Do("cert:"+hello.ServerName, func() (interface{}, error) {
				m.acmeMU.Lock()
				defer m.acmeMU.Unlock()

				if m.beforeAutoCert != nil {
					if err := m.beforeAutoCert(); err != nil {
						m.logger.Error().Err(err).Msg("autotls: beforehandleautocert failed")
					}
				}

				stdHello := &tls.ClientHelloInfo{
					ServerName: hello.ServerName,
				}
				stdCert, err := origGetCert(stdHello)

				if m.afterAutoCert != nil {
					if err2 := m.afterAutoCert(); err2 != nil {
						m.logger.Error().Err(err2).Msg("autotls: afterhandleautocert failed")
					}
				}

				if err != nil {
					m.logger.Error().Err(err).Msg("autotls: failed to obtain certificate")
					return nil, err
				}

				return toNbioCertificate(stdCert), nil
			})
			if err != nil {
				return nil, err
			}
			return val.(*lltls.Certificate), nil
		}

		tlsCfg := &lltls.Config{
			GetCertificate:           getCert,
			MinVersion:               lltls.VersionTLS12,
			NextProtos:               []string{"h2", "http/1.1"},
			PreferServerCipherSuites: true,
		}
		tlsCfg.PreferServerCipherSuites = true
		return tlsCfg, nil
	}
	if m.shouldUseSelfTLS(tlsConfig) {
		cert, err := m.getOrCreateSelfSignedCert()
		if err != nil {
			return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}
		nbioCert := toNbioCertificate(cert)
		tlsCfg := &lltls.Config{
			GetCertificate: func(_ *lltls.ClientHelloInfo) (*lltls.Certificate, error) {
				return nbioCert, nil
			},
			MinVersion:               lltls.VersionTLS12,
			NextProtos:               []string{"h2", "http/1.1"},
			PreferServerCipherSuites: true,
		}
		return tlsCfg, nil
	}

	tlsCfg := &lltls.Config{
		GetCertificate:           m.GetNBIOCertificateFunc(),
		MinVersion:               lltls.VersionTLS12,
		NextProtos:               []string{"h2", "http/1.1"},
		PreferServerCipherSuites: true,
	}
	return tlsCfg, nil
}

func toNbioCertificate(cert *tls.Certificate) *lltls.Certificate {
	if cert == nil {
		return nil
	}
	var sigAlgs []lltls.SignatureScheme
	if len(cert.SupportedSignatureAlgorithms) > 0 {
		sigAlgs = make([]lltls.SignatureScheme, len(cert.SupportedSignatureAlgorithms))
		for i, alg := range cert.SupportedSignatureAlgorithms {
			sigAlgs[i] = lltls.SignatureScheme(alg)
		}
	}
	return &lltls.Certificate{
		Certificate:                  cert.Certificate,
		PrivateKey:                   cert.PrivateKey,
		SupportedSignatureAlgorithms: sigAlgs,
		OCSPStaple:                   cert.OCSPStaple,
		SignedCertificateTimestamps:  cert.SignedCertificateTimestamps,
		Leaf:                         cert.Leaf,
	}
}

// GetTlsDomains 获取sever和autoCert配置中的ssl域名 取交集
func GetTlsDomains(config *config.Config) []string {
	domains := structure.NewSet[string]()
	for _, serverConfig := range config.Servers {
		if serverConfig.TLS != nil || serverConfig.Protocol == "https" {
			for _, domainConfig := range serverConfig.DomainConfig {
				for _, domain := range domainConfig.Domains {
					domains.Add(domain)
				}
			}
		}
	}

	if len(config.Features.AutoCert.Domains) > 0 {
		return config.Features.AutoCert.Domains
	}
	return domains.List()
}

func GetTLSVersion(minVersion string) uint16 {
	if minVersion == "" {
		return tls.VersionTLS12
	}
	if v, ok := TLSVersionMap[minVersion]; ok {
		return v
	}
	return tls.VersionTLS12
}

func (m *TLSManager) shouldUseSelfTLS(tlsConfig *config.TLSConfig) bool {
	if tlsConfig == nil {
		return false
	}
	return !tlsConfig.AutoTLS && tlsConfig.SelfTls && len(tlsConfig.CertMap) == 0
}

func (m *TLSManager) getOrCreateSelfSignedCert() (*tls.Certificate, error) {
	m.selfCertMu.Lock()
	defer m.selfCertMu.Unlock()
	if m.selfCert != nil {
		return m.selfCert, nil
	}

	dnsNames, ipAddresses := m.getSelfSignedSANs()
	cert, err := generateSelfSignedCertificate(dnsNames, ipAddresses)
	if err != nil {
		return nil, err
	}
	m.selfCert = cert
	return m.selfCert, nil
}

func (m *TLSManager) getSelfSignedSANs() ([]string, []net.IP) {
	dnsSet := map[string]struct{}{
		"localhost": {},
	}
	ipSet := map[string]net.IP{
		"127.0.0.1": net.ParseIP("127.0.0.1"),
		"::1":       net.ParseIP("::1"),
	}
	for _, server := range m.config.Servers {
		if server.Host != "" && server.Host != "0.0.0.0" && server.Host != "::" {
			if ip := net.ParseIP(server.Host); ip != nil {
				ipSet[ip.String()] = ip
			} else {
				dnsSet[server.Host] = struct{}{}
			}
		}
		for _, domainCfg := range server.DomainConfig {
			for _, domain := range domainCfg.Domains {
				if domain != "" {
					dnsSet[domain] = struct{}{}
				}
			}
		}
	}

	dnsNames := make([]string, 0, len(dnsSet))
	for dnsName := range dnsSet {
		dnsNames = append(dnsNames, dnsName)
	}
	sort.Strings(dnsNames)

	ipAddresses := make([]net.IP, 0, len(ipSet))
	for _, ip := range ipSet {
		if ip != nil {
			ipAddresses = append(ipAddresses, ip)
		}
	}
	return dnsNames, ipAddresses
}

func generateSelfSignedCertificate(dnsNames []string, ipAddresses []net.IP) (*tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, err
	}
	commonName := "Hamburger"
	if len(dnsNames) > 0 {
		commonName = dnsNames[0]
	}
	if len(ipAddresses) > 0 {
		commonName = ipAddresses[0].String()
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"JJApps Hamburger"},
		},
		NotBefore:             time.Now().Add(-5 * time.Minute),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	cert.Leaf, _ = x509.ParseCertificate(derBytes)
	return &cert, nil
}
