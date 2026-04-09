package resolver

import (
	"Hamburger/gateway/balancer"
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"Hamburger/internal/constant"
	"Hamburger/internal/utils"
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
	Proxy      struct {
		Host string
		Port int
	}
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
				Proxy: struct {
					Host string
					Port int
				}{Host: backend.ProxyDirect.ProxyHost, Port: backend.ProxyDirect.ProxyPort},
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
			return RuleResult{
				ProxyError: errDomainsMapEmpty,
			}
		}
		r.rwLock.RLock()
		rules := r.apiRules[host]
		r.rwLock.RUnlock()

		serviceType, _ := runtime.DomainsRuntimeMap.ServiceMap.Get(serviceMap.Service)
		// 根据请求和域名判断转发到的真实服务
		switch serviceType.ServiceType {
		case constant.FrontendType:
			// 纯前端服务
			// 未匹配API转发时直接转到前端处理
			if result, ok := r.MatchAPIRule(req, rules); ok {
				return result
			}
			req.Header.Set(r.cfg.PxyFrontend.InternalFlag, serviceMap.Service)
			return RuleResult{
				ProxyToType: Frontend,
				ProxyTo:     serviceMap.Service,
				ProxyHost:   r.cfg.PxyFrontend.Host,
				ProxyPort:   r.cfg.PxyFrontend.Port,
				ProxyScheme: StaticSchema,
			}
		case constant.BackendType:
			// 纯后端服务
			ports, ok := runtime.ServicePortsMap.Get(serviceMap.Service)
			if !ok {
				return RuleResult{
					ProxyError: errDomainsPortEmpty,
				}
			}
			return RuleResult{
				ProxyToType: Backend,
				ProxyTo:     serviceMap.Service,
				ProxyHost:   StaticHost,
				ProxyPort:   balancer.PickOneRoundRobin(ports),
				ProxyScheme: StaticSchema,
			}
		case constant.CustomType:
			service, ok := runtime.DomainsRuntimeMap.ServiceMap.Get(serviceMap.Service)
			if !ok {
				return RuleResult{
					ProxyError: errDomainsMapEmpty,
				}
			}
			if result, ok := r.MatchCustomAPIRule(req, service.ProxyPass); ok {
				return result
			}

			req.Header.Set(r.cfg.PxyFrontend.InternalFlag, serviceMap.Service)
			return RuleResult{
				ProxyToType: Custom,
				ProxyTo:     serviceMap.Service,
				ProxyHost:   utils.DefaultString(service.Host, StaticHost),
				ProxyPort:   service.Port,
				ProxyScheme: StaticSchema,
			}
		default:
			return RuleResult{
				ProxyError: errUnknownPath,
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

		// 是否为单纯proxy端口转发
		if rule.Proxy.Host != "" && rule.Proxy.Port > 0 {
			return RuleResult{
				ProxyToType: Backend,
				ProxyTo:     "",
				ProxyPath:   targetPath,
				ProxyHost:   rule.Proxy.Host,
				ProxyPort:   rule.Proxy.Port,
				ProxyScheme: StaticSchema,
			}, true
		}

		// 转发到后端服务
		ports, ok := runtime.ServicePortsMap.Get(rule.Backend)
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

func (r *Ruler) MatchCustomAPIRule(req *http.Request, rules []config.ServiceProxy) (RuleResult, bool) {
	requestPath := req.URL.Path
	for _, rule := range rules {
		// 检查API路径和服务名是否配置
		if rule.API == "" || rule.Service == "" {
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

		// 是否为单纯proxy端口转发
		if rule.ProxyDirect.ProxyHost != "" && rule.ProxyDirect.ProxyPort > 0 {
			return RuleResult{
				ProxyToType: Custom,
				ProxyTo:     rule.Service,
				ProxyPath:   targetPath,
				ProxyHost:   rule.ProxyDirect.ProxyHost,
				ProxyPort:   rule.ProxyDirect.ProxyPort,
				ProxyScheme: StaticSchema,
			}, true
		}

		// TODO 静态文件代理

		// 转发到后端服务
		// 转发到后端自定义服务
		service, ok := runtime.DomainsRuntimeMap.ServiceMap.Get(rule.Service)
		if !ok {
			return RuleResult{
				ProxyError: errDomainsMapEmpty,
			}, true
		}
		if service.ServiceType == constant.CustomType {
			return RuleResult{
				ProxyToType: Custom,
				ProxyTo:     service.ServiceName,
				ProxyPath:   targetPath,
				ProxyHost:   service.Host,
				ProxyPort:   service.Port,
				ProxyScheme: StaticSchema,
			}, true
		} else {
			ports, ok := runtime.ServicePortsMap.Get(rule.Service)
			if !ok {
				return RuleResult{
					ProxyError: errDomainsPortEmpty,
				}, true
			}
			return RuleResult{
				ProxyToType: Custom,
				ProxyTo:     service.ServiceName,
				ProxyHost:   StaticHost,
				ProxyPath:   targetPath,
				ProxyPort:   balancer.PickOneRoundRobin(ports),
				ProxyScheme: StaticSchema,
			}, true
		}
	}

	return RuleResult{}, false
}

//go:inline
func IsBackend(req *http.Request) bool {
	if req.Header.Get("X-Hamburger-Backend") != "" {
		return true
	}

	return false
}
