package resolver

import (
	"Hamburger/gateway/balancer"
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"errors"
	"github.com/rs/zerolog"
	"net/http"
	"strings"
	"sync"
)

// 配置的API转发规则解析

const (
	StaticHost   = "127.0.0.1"
	StaticSchema = "http"
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
			ProxyError: errors.New("host is empty"),
		}
	}
	if host == "localhost" || host == StaticHost {
		// 内部调用
	} else {
		serviceMap, ok := runtime.DomainsRuntimeMap.DomainsMap.Get(host)
		if !ok {
			return RuleResult{
				ProxyError: errors.New("domains map is empty"),
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
					ProxyError: errors.New("domains port is empty"),
				}
			}
			return RuleResult{
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
					ProxyTo:     serviceMap.Frontend,
					ProxyHost:   r.cfg.PxyFrontend.Host,
					ProxyPort:   r.cfg.PxyFrontend.Port,
					ProxyScheme: StaticSchema,
				}
			}
		}
	}
	return RuleResult{
		ProxyError: errors.New("unknown path"),
	}
}

func (r *Ruler) MatchAPIRule(req *http.Request, rules []Rule) (RuleResult, bool) {
	requestPath := req.URL.Path
	host := req.Host
	for _, rule := range rules {
		// 检查API路径和服务名是否配置
		if rule.API == "" || rule.Backend == "" {
			continue
		}

		// 检查请求路径是否匹配backend.api
		if strings.HasPrefix(requestPath, rule.API) {
			// 执行后端代理转发

			// 是否rewrite url
			targetPath := req.URL.Path
			if rule.UseRewrite {
				// 将API路径重写为指定的rewrite路径
				if strings.HasPrefix(targetPath, rule.API) {
					targetPath = strings.Replace(targetPath, rule.API, rule.Rewrite, 1)
				}
			}

			ports, ok := runtime.DomainPortsMap.Get(host)
			if !ok {
				return RuleResult{
					ProxyError: errors.New("domains port is empty"),
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

	return RuleResult{}, false
}

func IsBackend(req *http.Request) bool {
	if req.Header.Get("X-Hamburger-Backend") != "" {
		return true
	}

	return false
}
