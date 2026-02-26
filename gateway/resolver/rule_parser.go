package resolver

import (
	"Hamburger/gateway/balancer"
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

// 配置的API转发规则解析

const (
	StaticHost   = "127.0.0.1"
	StaticSchema = "http"
)

var (
	errHostEmpty        = errors.New("host is empty")
	errDomainsMapEmpty  = errors.New("domains map is empty")
	errDomainsPortEmpty = errors.New("domains port is empty")
	errUnknownPath      = errors.New("unknown path")
	errUnknownHost      = errors.New("unknown host")
)

type Ruler struct {
	cfg      *config.Config
	logger   *zerolog.Logger
	apiRules map[string][]Rule // 记录域名和对应的API服务转发映射
	rwLock   sync.RWMutex
}

type Rule struct {
	API        string
	Rewrite    string
	UseRewrite bool
	Backend    string
}

func NewRuler(cfg *config.Config, logger *zerolog.Logger) *Ruler {
	apiServers := cfg.PxyFrontend.Servers
	rules := make(map[string][]Rule)
	// 转换为域名的规则映射
	for _, server := range apiServers {
		domain, ok := runtime.DomainsRuntimeMap.DomainFrontMap.Get(server.Name)
		if !ok {
			continue
		}
		rules[domain] = make([]Rule, 0)
		for _, backend := range server.Backends {
			rules[domain] = append(rules[domain], Rule{
				API:        backend.API,
				Rewrite:    backend.Rewrite,
				UseRewrite: backend.UseRewrite,
				Backend:    backend.Service,
			})
		}
	}

	return &Ruler{
		cfg:      cfg,
		logger:   logger,
		apiRules: rules,
	}
}

func (r *Ruler) Parse(req *http.Request) RuleResult {
	// 首先通过域名判断是什么服务组
	host := req.Host
	// 代理服务都是http协议服务器
	if host == "" {
		return RuleResult{
			ProxyError: errHostEmpty,
		}
	}
	if host == "localhost" || host == StaticHost {
		// 内部调用
		r.logger.Debug().Str("URL", req.URL.RawPath).Msg("host is localhost")
	} else {
		serviceMap, ok := runtime.DomainsRuntimeMap.DomainsMap.Get(host)
		if !ok {
			// 未匹配运行时域名映射时尝试自定义服务映射
			if r.IsCustomServiceEnabled() {
				return r.ResolveCustomService(host)
			}
			return RuleResult{
				ProxyError: errDomainsMapEmpty,
			}
		}
		r.rwLock.RLock()
		rules := r.apiRules[host]
		r.rwLock.RUnlock()

		// 根据请求和域名判断转发到的真实服务
		if serviceMap.Frontend != "" && serviceMap.Backend == "" {
			// 纯前端服务
			req.Header.Set(r.cfg.PxyFrontend.InternalFlag, serviceMap.Frontend)
			return RuleResult{
				ProxyToType: Frontend,
				ProxyTo:     serviceMap.Frontend,
				ProxyHost:   r.cfg.PxyFrontend.Host,
				ProxyPort:   r.cfg.PxyFrontend.Port,
				ProxyScheme: StaticSchema,
			}
		}
		if serviceMap.Frontend == "" && serviceMap.Backend != "" {
			// 纯后端服务
			ports, ok := runtime.DomainPortsMap.Get(host)
			if !ok {
				return RuleResult{
					ProxyError: errDomainsPortEmpty,
				}
			}
			return RuleResult{
				ProxyToType: Backend,
				ProxyTo:     serviceMap.Backend,
				ProxyHost:   StaticHost,
				ProxyPort:   balancer.PickOneRoundRobin(ports),
				ProxyScheme: StaticSchema,
			}
		}

		result, ok := r.MatchAPIRule(req, rules)
		if ok {
			return result
		} else {
			if serviceMap.Backend != "" && serviceMap.Frontend != "" {
				// 前后端分离服务
				req.Header.Set(r.cfg.PxyFrontend.InternalFlag, serviceMap.Frontend)
				return RuleResult{
					ProxyToType: Frontend,
					ProxyTo:     serviceMap.Frontend,
					ProxyHost:   r.cfg.PxyFrontend.Host,
					ProxyPort:   r.cfg.PxyFrontend.Port,
					ProxyScheme: StaticSchema,
				}
			}
		}
	}
	return RuleResult{
		ProxyError: errUnknownPath,
	}
}

//go:inline
func (r *Ruler) MatchAPIRule(req *http.Request, rules []Rule) (RuleResult, bool) {
	requestPath := req.URL.Path
	host := req.Host
	for _, rule := range rules {
		// 检查API路径和服务名是否配置
		if rule.API == "" || rule.Backend == "" {
			continue
		}

		// 检查请求路径是否匹配backend.api
		if !strings.HasPrefix(requestPath, rule.API) {
			continue
		}
		// 执行后端代理转发

		// 是否rewrite url
		targetPath := requestPath
		if rule.UseRewrite {
			targetPath = rule.Rewrite + requestPath[len(rule.API):]
		}

		ports, ok := runtime.DomainPortsMap.Get(host)
		if !ok {
			return RuleResult{
				ProxyError: errDomainsPortEmpty,
			}, true
		}
		return RuleResult{
			ProxyTo:     rule.Backend,
			ProxyHost:   StaticHost,
			ProxyPath:   targetPath,
			ProxyPort:   balancer.PickOneRoundRobin(ports),
			ProxyScheme: StaticSchema,
		}, true
	}

	return RuleResult{}, false
}

//go:inline
func (r *Ruler) IsCustomServiceEnabled() bool {
	return r.cfg.PxyCustomService.Enable
}

// ResolveCustomService 处理自定义后端服务转发
//
//go:inline
func (r *Ruler) ResolveCustomService(host string) RuleResult {
	for _, serviceConfig := range r.cfg.PxyCustomService.CustomService {
		if host == serviceConfig.Domain {
			addrs := make([]struct {
				Host string
				Port int
			}, len(serviceConfig.Upstream))
			for i, upstream := range serviceConfig.Upstream {
				addrs[i] = struct {
					Host string
					Port int
				}{Host: upstream.Host, Port: upstream.Port}
			}
			realHost, port := balancer.PickOneAddrRoundRobin(addrs)
			return RuleResult{
				ProxyToType: Backend,
				ProxyTo:     "",
				ProxyHost:   realHost,
				ProxyPort:   port,
				ProxyScheme: StaticSchema,
			}
		}
	}
	return RuleResult{
		ProxyError: errUnknownHost,
	}
}

//go:inline
func IsBackend(req *http.Request) bool {
	if req.Header.Get("X-Hamburger-Backend") != "" {
		return true
	}

	return false
}
