package tls

import (
	autocert2 "Hamburger/gateway/autocert"
	"Hamburger/internal/config"
	"Hamburger/internal/structure"
	"crypto/tls"
	"fmt"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/singleflight"
	"net"
	"sync"
)

type TLSManager struct {
	config *config.Config
	logger *zerolog.Logger
	// AutoTLS
	AcmeMgr *autocert.Manager  // autocert 管理器
	acmeMU  sync.Mutex         // 刷新证书过程的互斥锁，避免并发冲突
	sf      singleflight.Group // 用于合并并发的证书请求

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
		return nil, nil, fmt.Errorf("HTTPS 服务器缺少 TLS 配置")
	}

	if tlsConfig.AutoTLS {
		// 使用 autocert 自动管理证书，返回用于标准 http.Server 的 *tls.Config
		// 构建域名白名单（必须提供域名，否则不可启动AutoTLS）
		domains := GetTlsDomains(m.config)
		if len(domains) == 0 {
			return nil, nil, fmt.Errorf("自动TLS已启用但未配置任何域名，无法申请证书")
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

				m.logger.Info().Str("域名", hello.ServerName).Msg("AutoTLS: 即将获取/刷新证书, 准备处理80端口")
				if err := m.beforeAutoCert(); err != nil {
					m.logger.Error().Err(err).Msg("AutoTLS: beforeHandleAutoCert 失败")
				}

				cert, err := origGetCert(hello)

				if err2 := m.afterAutoCert(); err2 != nil {
					m.logger.Error().Err(err2).Msg("AutoTLS: afterHandleAutoCert 失败")
				}

				if err != nil {
					m.logger.Error().Err(err).Msg("AutoTLS: 获取证书失败")
					return nil, err
				} else {
					m.logger.Info().Str("域名", hello.ServerName).Msg("AutoTLS: 证书获取/刷新完成")
					return cert, nil
				}
			})

			if err != nil {
				return nil, err
			}
			return val.(*tls.Certificate), nil
		}

		// 强化TLS安全参数
		base.MinVersion = tls.VersionTLS12
		base.PreferServerCipherSuites = true

		// 应用 TLS 配置
		lis := tls.NewListener(listener, base)

		return base, lis, nil
	}

	// 加载证书和私钥
	cert, err := tls.LoadX509KeyPair(tlsConfig.CertFile, tlsConfig.KeyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("加载 TLS 证书失败: %v", err)
	}

	// 配置 TLS
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// 设置最低 TLS 版本
		MinVersion: tls.VersionTLS12,
		// 优先使用服务器的密码套件顺序
		PreferServerCipherSuites: true,
	}

	// 应用 TLS 配置
	lis := tls.NewListener(listener, tlsCfg)

	return tlsCfg, lis, nil
}

func (m *TLSManager) GetTlsConfig(tlsConfig *config.TLSConfig) *tls.Config {
	// 加载证书和私钥
	cert, err := tls.LoadX509KeyPair(tlsConfig.CertFile, tlsConfig.KeyFile)
	if err != nil {
		return nil
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// 设置最低 TLS 版本
		MinVersion: tls.VersionTLS12,
		// 优先使用服务器的密码套件顺序
		PreferServerCipherSuites: true,
	}

	return tlsCfg
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
