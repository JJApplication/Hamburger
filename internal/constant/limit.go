package constant

var (
	// LIMIT 限制100/10s
	LIMIT = 10
	// RESET 经过RESET * Duration次无请求后，从map中删除定时器
	RESET = 10
)

var (
	BreakerLimit = 10 // 限制内部错误的次数
	BreakerReset = 60 // 默认的重置时间当服务down时 等待60s后重试
)
