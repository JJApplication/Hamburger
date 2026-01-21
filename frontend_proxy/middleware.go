package frontend_proxy

import (
	"Hamburger/internal/config"
	"github.com/rs/zerolog"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggingMiddleware 日志中间件
func LoggingMiddleware(logger *zerolog.Logger, config *config.PxyFrontConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		// 获取请求体大小
		bodySize := c.Request.ContentLength
		if bodySize < 0 {
			bodySize = 0
		}

		// 获取内部标志
		internalFlag := c.GetHeader(config.InternalFlag)

		// 检查是否需要记录详细访问日志
		var shouldLogAccess bool
		for _, srv := range config.Servers {
			if srv.Name == internalFlag && srv.Access {
				shouldLogAccess = true
				break
			}
		}

		if shouldLogAccess {
			// 详细访问日志
			logger.Info().
				Str("internal_flag", internalFlag).
				Any("headers", c.Request.Header).
				Str("method", c.Request.Method).
				Int64("body_size", bodySize).
				Str("path", c.Request.URL.Path).
				Dur("response_time", latency).
				Int("status_code", c.Writer.Status()).
				Str("ip", c.ClientIP()).Msg("access log")
		} else {
			// 简单请求日志
			logger.Info().
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Int("status", c.Writer.Status()).
				Dur("latency", latency).
				Str("ip", c.ClientIP()).
				Msg("http request")
		}
	}
}

// CustomHeadersMiddleware 自定义响应头中间件
func CustomHeadersMiddleware(config *config.PxyFrontConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, header := range config.CustomHeaders {
			c.Header(header.Name, header.Value)
		}
		c.Next()
	}
}

// RoutingMiddleware 路由中间件
func RoutingMiddleware(server *HeliosServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取内部标志头
		internalFlag := c.GetHeader(server.config.InternalFlag)
		if internalFlag == "" {
			server.HandleError(c, 404, "Internal flag header not found")
			return
		}

		// 查找对应的服务器配置
		var serverConfig *config.FrontServerConfig

		for _, srv := range server.config.Servers {
			if srv.Name == internalFlag {
				serverConfig = &srv
				break
			}
		}

		if serverConfig == nil {
			server.HandleError(c, 404, "Server not found for internal flag: "+internalFlag)
			return
		}

		// 处理静态文件请求
		server.HandleStaticFile(c, serverConfig)
	}
}
