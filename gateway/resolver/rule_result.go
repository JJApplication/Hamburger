package resolver

import (
	"net"
	"strconv"
	"strings"
)

// 规则解析结果
// 直接转换为可读的服务Service调用形式

const (
	Frontend = iota
	Backend
	Custom
)

type RuleResult struct {
	ProxyToType int
	ProxyTo     string // service 前端或后端或前后端合并项目
	ProxyPath   string // API路径
	ProxyHost   string
	ProxyPort   int
	ProxyScheme string // http 或者 grpc
	ProxyError  error  // 错误
}

// BreakerKey 是熔断器使用的后端目标。
// 有服务名时按服务隔离；纯端口转发按 host:port 隔离。
func (r RuleResult) BreakerKey() string {
	if name := strings.TrimSpace(r.ProxyTo); name != "" {
		return name
	}
	if r.ProxyHost != "" && r.ProxyPort > 0 {
		return net.JoinHostPort(r.ProxyHost, strconv.Itoa(r.ProxyPort))
	}
	return ""
}
