package core

import (
	"Hamburger/gateway/prehandler"
	"Hamburger/gateway/proxy_cache"
	"Hamburger/static_direct"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"Hamburger/gateway/breaker"
	"Hamburger/gateway/error_page"
	"Hamburger/gateway/grpc_proxy"
	"Hamburger/gateway/modifier"
	"Hamburger/gateway/resolver"
	"Hamburger/gateway/stat"
	"Hamburger/internal/config"
	"Hamburger/internal/constant"
	"Hamburger/internal/serror"
	"Hamburger/internal/utils"

	"github.com/rs/zerolog"
)

// 拦截所有请求并根据配置进行转发

const (
	FlushInterval = 100
	BufferSize    = 32 * 1024
)

var (
	// 全局共享的Transport实例，避免重复创建
	sharedTransport *myTransport
	transportOnce   sync.Once
)

// getOptimizedTransport 获取优化的HTTP传输层配置
//
//go:inline
func getOptimizedTransport(transport string) *myTransport {
	transportOnce.Do(func() {
		cfg := config.Get()
		switch transport {
		case constant.ProxyMode_HTTP:
			sharedTransport = &myTransport{
				Transport: OriginRoundTrip(),
				conf:      cfg,
			}
		case constant.ProxyMode_FastHTTP:
			sharedTransport = &myTransport{
				Transport: NewFastRoundTripper(),
				conf:      cfg,
			}
		default:
			sharedTransport = &myTransport{
				Transport: OriginRoundTrip(),
				conf:      cfg,
			}
		}
		if cfg != nil && cfg.PxyFrontend.ExpFastConnect.Enabled {
			sharedTransport.h2cTransport = newH2CTransport(cfg)
		}
		if cfg != nil && cfg.CoreProxy.EnableHTTP3 && cfg.PxyFrontend.ExpFastConnect.Http3.Enabled {
			sharedTransport.h3Transport = newH3Transport(cfg)
		}
	})
	return sharedTransport
}

type syncBufferPool struct {
	pool    sync.Pool
	bufSize int
}

func (s *syncBufferPool) Get() []byte {
	return s.pool.Get().([]byte)
}

func (s *syncBufferPool) Put(buf []byte) {
	if cap(buf) == s.bufSize {
		s.pool.Put(buf)
	}
}

//go:inline
func getBufferPool(bufSize int) httputil.BufferPool {
	return &syncBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, bufSize)
			},
		},
		bufSize: bufSize,
	}
}

// ProxyDirector 代理请求处理逻辑
func ProxyDirector(cfg *config.Config, logger *zerolog.Logger) func(request *http.Request) {
	return func(request *http.Request) {
		logger.Debug().
			Any("URL", request.URL).
			Any("Header", request.Header).
			Str("Host", request.Host).
			Str("Trace-ID", request.Header.Get(cfg.ProxyHeader.TraceId)).
			Msg("parse request")
		// 检查是否为gRPC代理请求
		if grpc_proxy.IsEnabled() {
			proxy := grpc_proxy.GetGrpcProxy()
			if proxy != nil && proxy.IsGrpcRequest(request) {
				logger.Debug().Msg("detected gRPC proxy request")
				// 设置特殊的scheme来标识gRPC请求，后续在Transport中处理
				request.URL = &url.URL{Scheme: constant.SchemeGrpc}
				return
			}
		}

		if static_direct.IsEnabled() {
			sd := static_direct.GetSvr()
			if uri, ok := sd.IsStaticDirect(request); ok {
				request.URL = uri
				return
			}
		}

		// 请求次数
		stat.Add(stat.Total)
		// 统计域名
		stat.AddDomainStat(request.Host)
		// 统计远程地址
		stat.AddGeo(request.RemoteAddr)

		pm := prehandler.GetManager()
		for _, handler := range pm.GetPreHandlers() {
			if err := handler.Handle(request); err != nil {
				logger.Debug().Err(err).Str("Name", handler.Name()).Err(err).Msg("pre handler failed")
				request.URL = &url.URL{Scheme: constant.SchemeSandwich}
				return
			}
		}
		// 处理解析缓存 只缓存后端前端由Helios控制
		if cfg.Features.ProxyCache.Enabled {
			cache, ok := proxy_cache.M().Get(request)
			if ok {
				request.URL = cache.CacheToURL(request)
			} else {
				// 缓存未命中
				finalURL, result := resolver.OneResolver(cfg, logger).Parse(request)
				request.URL = finalURL
				if result.ProxyToType != resolver.Frontend {
					proxy_cache.M().Set(request, result.ProxyHost, result.ProxyPort, result.ProxyScheme)
				}
			}
		} else {
			request.URL, _ = resolver.OneResolver(cfg, logger).Parse(request)
		}
		logger.Debug().Any("URL", request.URL).Msg("parse request")
	}
}

// ProxyModifyResponse 响应修改逻辑
func ProxyModifyResponse(cfg *config.Config, logger *zerolog.Logger) func(response *http.Response) error {
	return func(response *http.Response) error {
		mods := modifier.GetManager().GetModifiers()
		logger.Debug().Int("Count", len(mods)).Any("mods", mods).Msg("modifier manager")
		if cfg.Debug {
			start, end, sub := utils.PerformTime(func() {
				for _, mod := range mods {
					mod.Use(response)
				}
			})
			logger.Debug().Time("start", start).Time("end", end).Dur("sub", sub).Msg("perform time for response modifier")
			return nil
		}

		for _, mod := range mods {
			mod.Use(response)
		}
		return nil
	}
}

// ProxyErrorHandler 错误处理逻辑
func ProxyErrorHandler(logger *zerolog.Logger) func(writer http.ResponseWriter, request *http.Request, err error) {
	return func(writer http.ResponseWriter, request *http.Request, err error) {
		logger.Debug().
			Str("host", request.Host).
			Any("url", request.URL).
			Str("proto", request.Proto).
			Str("method", request.Method).
			Err(err).Msg("proxy error")
		stat.Add(stat.Fail)
		// 熔断判断
		switch request.Header.Get(serror.SandwichInternalFlag) {
		case serror.SandwichBucketLimit:
			logger.Error().Str("host", request.Host).Msg("reach breaker limit")
			writer.WriteHeader(http.StatusGatewayTimeout)
			return
		case serror.SandwichReqLimit:
			logger.Error().Str("host", request.Host).Msg("reach flow control limit")
			error_page.Cache(http.StatusTooManyRequests, writer, request, error_page.Forbidden)
			return
		case serror.SandwichDomainNotAllow:
			logger.Error().Str("host", request.Host).Msg("http: no host in request url")
			error_page.Cache(http.StatusForbidden, writer, request, error_page.Forbidden)
			return
		case serror.SandwichBackendError:
			breaker.Set(request.Host)
			logger.Error().Str("host", request.Host).Msg("backend: service is down")
			error_page.Cache(http.StatusBadGateway, writer, request, error_page.Unavailable)
			return
		}
		logger.Error().Err(err).Str("host", request.Host).Msg("proxy connect error")
		error_page.Cache(http.StatusBadGateway, writer, request, error_page.Unavailable)
	}
}
