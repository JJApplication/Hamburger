/*
Project: Sandwich header.go
Created: 2021/12/12 by Landers
*/

package utils

import (
	"Hamburger/internal/config"
	"net/http"
)

// 自定义的响应头部

func AddHeader(response *http.Response, headers map[string]string) {
	for key, value := range headers {
		if response.Header.Get(key) == "" {
			response.Header.Add(key, value)
		}
	}
}

func AddTrace(response *http.Response, traceHeader string) {
	// 设置请求的Trace-Id
	traceIdHeader := config.Get().ProxyHeader.TraceId
	if traceIdHeader == "" {
		traceIdHeader = traceHeader
	}
	traceId := response.Request.Header.Get(traceIdHeader)

	// 仅当traceID不存在时才生成并设置新的TraceID
	if traceId == "" {
		// 生成唯一的TraceID
		newTraceId := generateTraceId()
		response.Header.Set(traceIdHeader, newTraceId)
	} else {
		// 如果请求中已有TraceID，传递到响应中
		response.Header.Set(traceIdHeader, traceId)
	}
}

// AddSecureHeader 为响应添加安全头部，防止XSS和CSRF攻击
func AddSecureHeader(response *http.Response) {
	cfg := config.Get()
	response.Header.Set("X-Content-Type-Options", "nosniff")
	// 防止XSS攻击
	if cfg.Security.XssProtection {
		response.Header.Set("X-XSS-Protection", "1; mode=block")
	}
	if cfg.Security.IFrameProtection {
		response.Header.Set("X-Frame-Options", "DENY")
	}

	// HTTPS相关安全头部
	if cfg.Security.HSTS {
		var hstsHeader = "max-age=31536000;"
		if cfg.Security.HSTSSubdomain {
			hstsHeader += "includeSubDomains;"
		}
		if cfg.Security.HSTSPreload {
			hstsHeader += "preload"
		}
		response.Header.Set("Strict-Transport-Security", hstsHeader)
	}

	if cfg.Security.SameSite {
		// 防止CSRF攻击 - SameSite Cookie 策略
		response.Header.Set("Set-Cookie", "SameSite=Strict; Path=/; Secure; HttpOnly")
	}

	// 引用策略 - 控制Referer头信息泄露
	response.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
}
