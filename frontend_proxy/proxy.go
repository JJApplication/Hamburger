package frontend_proxy

import (
	"Hamburger/internal/config"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// BackendProxyMiddleware 后端代理中间件
func BackendProxyMiddleware(server *HeliosServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取内部标志头
		internalFlag := c.GetHeader(server.config.InternalFlag)
		server.logger.Info().Str("internal_flag", internalFlag).Msg("Received request")
		if internalFlag == "" {
			c.Next()
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
			c.Next()
			return
		}

		// 检查是否配置了backends
		if len(serverConfig.Backends) == 0 {
			c.Next()
			return
		}

		// 遍历所有backend配置，查找匹配的API路径
		requestPath := c.Request.URL.Path
		for _, backend := range serverConfig.Backends {
			// 检查API路径和服务名是否配置
			if backend.API == "" || backend.Service == "" {
				continue
			}

			// 检查请求路径是否匹配backend.api
			if strings.HasPrefix(requestPath, backend.API) {
				// 执行后端代理转发
				proxyToBackend(server, c, &backend)
				// 代理转发完成后直接返回，不继续执行后续中间件
				c.Abort()
				return
			}
		}

		// 没有匹配的backend配置，继续执行后续中间件
		c.Next()
	}
}

// proxyToBackend 代理请求到后端服务
func proxyToBackend(server *HeliosServer, c *gin.Context, backend *config.BackendConfig) {
	server.logger.Debug().Str("backend_service", backend.Service).Msg("Proxy request to backend")
	// 构建目标URL，处理URL重写
	targetPath := c.Request.URL.Path
	if backend.UseRewrite {
		// 将API路径重写为指定的rewrite路径
		if strings.HasPrefix(targetPath, backend.API) {
			targetPath = strings.Replace(targetPath, backend.API, backend.Rewrite, 1)
		}
	}

	balancer := server.config.Balancer
	if !strings.HasPrefix(balancer, "http://") {
		balancer = fmt.Sprintf("http://%s", balancer)
	}
	targetURL, err := url.Parse(balancer)
	if err != nil {
		server.logger.Error().Err(err).Msg("Failed to parse balancer URL")
		server.HandleError(c, 500, "Internal server error")
		return
	}

	// 设置重写后的路径
	targetURL.Path = targetPath
	targetURL.RawQuery = c.Request.URL.RawQuery

	// 读取请求体
	var bodyReader io.Reader
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			server.logger.Error().Err(err).Msg("Failed to read request body")
			server.HandleError(c, 500, "Internal server error")
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, c.Request.Method, targetURL.String(), bodyReader)
	if err != nil {
		server.logger.Error().Err(err).Msg("Failed to create request")
		server.HandleError(c, 500, "Internal server error")
		return
	}

	// 复制原始请求头
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// 添加内部标识头
	req.Header.Set(server.config.InternalLocalFlag, "yes")
	req.Header.Set(server.config.InternalBackendFlag, backend.Service)
	// 重置HOST
	if req.Header.Get("X-Forwarded-Host") != "" {
		req.Host = req.Header.Get("X-Forwarded-Host")
	} else if req.Header.Get("X-Proxy-Internal-Host") != "" {
		req.Host = req.Header.Get("X-Proxy-Internal-Host")
	}

	server.logger.Debug().Str("target_url", targetURL.String()).Str("backend_service", backend.Service).Msg("Proxy request")

	// 从池中获取HTTP客户端
	client := server.GetHTTPClient()
	defer server.PutHTTPClient(client)

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		server.logger.Error().Err(err).Msg("Failed to proxy request")
		server.HandleError(c, 500, "Internal server error")
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// 设置状态码并优化流式传输响应体
	c.Status(resp.StatusCode)
	// 使用缓冲区优化传输性能
	buf := make([]byte, 32*1024) // 32KB缓冲区
	_, err = io.CopyBuffer(c.Writer, resp.Body, buf)
	if err != nil {
		server.logger.Error().Err(err).Msg("Failed to copy response body")
	}

	// 记录代理日志
	server.logger.Info().
		Str("original_path", c.Request.URL.Path).
		Str("target_url", targetURL.String()).
		Str("backend_service", backend.Service).Int("status_code", resp.StatusCode).
		Msg("Backend proxy request")
}
