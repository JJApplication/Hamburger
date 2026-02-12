package resolver

// 规则解析结果
// 直接转换为可读的服务Service调用形式

const (
	Frontend = iota
	Backend
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
