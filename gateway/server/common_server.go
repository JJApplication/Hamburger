package server

import (
	"Hamburger/gateway/tls"
	"Hamburger/internal/config"
	"Hamburger/internal/utils"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// CommonHttpServer 通用http服务器
func CommonHttpServer(cfg *config.Config, serverConfig config.ServerConfig, logger *zerolog.Logger, h http.Handler, tlsManager *tls.TLSManager) (*ServerInstance, error) {
	// 创建服务器实例
	instance := &ServerInstance{
		Name:   serverConfig.Name,
		Config: serverConfig,
		TLS:    serverConfig.Protocol == "https" || serverConfig.Protocol == "http3",
		Logger: logger,
		lock:   sync.RWMutex{},
	}

	// 创建监听地址
	addr := fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port)

	// 设置支持协议
	proto := &http.Protocols{}
	if serverConfig.UseHttp2 {
		proto.SetHTTP2(true)
		proto.SetHTTP1(true)
		proto.SetUnencryptedHTTP2(true)
	} else {
		proto.SetHTTP2(false)
		proto.SetUnencryptedHTTP2(false)
		proto.SetHTTP1(true)
	}

	// 创建 HTTP 服务器
	instance.Server = &http.Server{
		Addr:    addr,
		Handler: h,
		// 设置合理的超时时间
		ReadTimeout:       time.Second * time.Duration(utils.DefaultInt64(serverConfig.ReadTimeout, 30)),
		WriteTimeout:      time.Second * time.Duration(utils.DefaultInt64(serverConfig.WriteTimeout, 30)),
		IdleTimeout:       time.Second * time.Duration(utils.DefaultInt64(serverConfig.IdleTimeout, 60)),
		ReadHeaderTimeout: time.Second * time.Duration(utils.DefaultInt64(serverConfig.ReadHeaderTimeout, 10)),
		// 设置最大请求体大小
		MaxHeaderBytes: utils.DefaultInt(int(serverConfig.MaxHeaderBytes), 5<<20), // 1MB header limit
		Protocols:      proto,
	}

	// 配置最大请求体大小和自动重定向
	if serverConfig.MaxRequestBody > 0 || serverConfig.Protocol == "http" {
		// 通过中间件控制请求体大小和处理重定向
		originalHandler := instance.Server.Handler
		instance.Server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 内部请求不需要判断重定向

			// 检查是否需要自动重定向HTTP到HTTPS
			if r.Header.Get(cfg.ProxyHeader.BackendHeader) == "" && serverConfig.Protocol == "http" {
				// 添加调试日志
				logger.Debug().Str("Host", r.Host).Str("URI", r.RequestURI).Str("Protocol", serverConfig.Protocol).Msg("http request")

				// 检查请求的域名是否配置了自动重定向
				host := r.Host
				// 移除端口号
				if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
					host = host[:colonIndex]
				}

				logger.Debug().Str("domain", host).Str("original_host", r.Host).Msg("processing domain")

				// 查找匹配的域名配置
				for i, domainConfig := range serverConfig.DomainConfig {
					logger.Debug().Int("domain_group", i).Bool("auto_redirect", domainConfig.AutoRedirect).Any("domains", domainConfig.Domains).Msg("checking domain configuration")

					if domainConfig.AutoRedirect {
						// 检查当前域名是否在配置的域名列表中
						for j, configuredDomain := range domainConfig.Domains {
							logger.Debug().Int("domain_group", j).Str("Host", host).Str("Domain", configuredDomain).Msg("checking domain match")

							if host == configuredDomain || (strings.HasPrefix(configuredDomain, "*.") && strings.HasSuffix(host, configuredDomain[1:])) {
								// 构建HTTPS重定向URL
								httpsURL := fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
								logger.Debug().Str("source_url", r.URL.String()).Str("target_url", httpsURL).Str("config_domain", configuredDomain).Msg("domain match success, performing redirect")
								// 执行301永久重定向
								w.Header().Set("Location", httpsURL)

								// 根据配置设置HSTS头部
								if domainConfig.HSTSMaxAge > 0 {
									hstsValue := fmt.Sprintf("max-age=%d", domainConfig.HSTSMaxAge)
									if domainConfig.HSTSSubdomains {
										hstsValue += "; includeSubDomains"
									}
									if domainConfig.HSTSPreload {
										hstsValue += "; preload"
									}
									w.Header().Set("Strict-Transport-Security", hstsValue)
									logger.Debug().Str("hsts_header", hstsValue).Msg("setting hsts header")
								} else {
									logger.Debug().Msg("hsts header not set (HSTSMaxAge=0)")
								}

								w.WriteHeader(http.StatusMovedPermanently)
								return
							}
						}
					}
				}
			}

			// 检查Content-Length头，如果超出限制直接返回413
			if serverConfig.MaxRequestBody > 0 && r.ContentLength > serverConfig.MaxRequestBody {
				w.Header().Set("Connection", "close")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				w.Write([]byte("Request entity too large"))
				logger.Printf("request entity too large, rejected: Host=%s, ContentLength=%d, Limit=%d",
					r.Host, r.ContentLength, serverConfig.MaxRequestBody)
				return
			}

			// 如果没有Content-Length头或为-1，使用limitedReader进行保护性限制
			if serverConfig.MaxRequestBody > 0 && (r.ContentLength == -1 || r.ContentLength == 0) {
				r.Body = &limitedReader{
					ReadCloser: r.Body,
					limit:      serverConfig.MaxRequestBody,
					logger:     logger,
					host:       r.Host,
				}
			}

			originalHandler.ServeHTTP(w, r)
		})
		logger.Printf("server %s max request body size set: %d bytes (%.2f MB)",
			serverConfig.Name, serverConfig.MaxRequestBody, float64(serverConfig.MaxRequestBody)/(1024*1024))
	}

	// 创建监听器
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %v", err)
	}
	instance.Listener = listener

	// 如果是 HTTPS，配置 TLS
	if instance.TLS {
		if tlsManager != nil {
			instance.lock.Lock()
			tlsConfig, lis, err := tlsManager.ConfigureTLS(instance.Config.TLS, instance.Listener)
			if err != nil {
				listener.Close()
				return nil, fmt.Errorf("failed to configure TLS: %v", err)
			}
			instance.Listener = lis
			instance.Server.TLSConfig = tlsConfig
			instance.lock.Unlock()
		}
	}
	logger.Info().Msgf("server %s starting listener on %s (protocol: %s)",
		serverConfig.Name, addr, serverConfig.Protocol)

	return instance, nil
}
