package resolver

import (
	"Hamburger/internal/config"
	"Hamburger/internal/serror"
	"fmt"
	"github.com/rs/zerolog"
	"net/http"
	"net/url"
	"sync"
)

// 解析器 解析为前端或后端代理转发

var once = sync.Once{}
var resolver Resolver

type Resolver struct {
	cfg    *config.Config
	logger *zerolog.Logger
	ruler  *Ruler
}

func OneResolver(cfg *config.Config, logger *zerolog.Logger) *Resolver {
	once.Do(func() {
		resolver = Resolver{
			cfg:    cfg,
			logger: logger,
			ruler:  NewRuler(cfg, logger),
		}
	})

	return &resolver
}

func (r *Resolver) Parse(request *http.Request) *url.URL {
	host := request.Host
	result := r.ruler.Parse(request)

	r.logger.Debug().Any("Result", result).Err(result.ProxyError).Msg("rule-parse request")
	if r.ResolveError(result, request) {
		return nil
	}

	request.URL.Scheme = result.ProxyScheme
	request.URL.Host = fmt.Sprintf("%s:%d", result.ProxyHost, result.ProxyPort)
	request.Header.Set("Host", host)                               // 设置真实HOST
	request.Header.Set(r.cfg.ProxyHeader.FrontendHostHeader, host) // 设置真实HOST

	return request.URL
}

func (r *Resolver) ResolveError(result RuleResult, req *http.Request) (hasError bool) {
	// 包含标准错误
	if result.ProxyError != nil {
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichBackendError)
		return true
	}
	// 转发的端口 host为空
	if result.ProxyHost == "" || result.ProxyPort == 0 {
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichBackendError)
		return true
	}

	return false
}
