package core

import (
	"net/http"
	"time"
)

func OriginRoundTrip() *http.Transport {
	return &http.Transport{
		// 连接池配置
		MaxIdleConns:        100,              // 最大空闲连接数
		MaxIdleConnsPerHost: 20,               // 每个主机最大空闲连接数
		MaxConnsPerHost:     50,               // 每个主机最大连接数
		IdleConnTimeout:     90 * time.Second, // 空闲连接超时
		// 超时配置
		ResponseHeaderTimeout: 30 * time.Second, // 响应头超时
		ExpectContinueTimeout: 1 * time.Second,  // 100-continue超时
		// 启用TCP keep-alive
		DisableKeepAlives: false,
		// 启用HTTP/2支持
		ForceAttemptHTTP2: true,
		// 禁用压缩以减少CPU开销（如果不需要）
		DisableCompression: false,
	}
}
