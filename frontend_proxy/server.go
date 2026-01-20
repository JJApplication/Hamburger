package frontend_proxy

import (
	"Hamburger/internal/config"
	"fmt"
	"github.com/rs/zerolog"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HeliosServer 服务器结构体
type HeliosServer struct {
	config       *config.PxyFrontConfig
	logger       *zerolog.Logger
	gin          *gin.Engine
	cacheManager *CacheManager
	clientPool   *sync.Pool
}

// NewFrontServer 创建新的服务器实例
func NewFrontServer(cfg *config.Config, logger *zerolog.Logger) (*HeliosServer, error) {
	cacheManager := NewCacheManager(cfg, logger)

	// 创建HTTP客户端池
	clientPool := &sync.Pool{
		New: func() interface{} {
			return &http.Client{
				Timeout: 30 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 10,
					IdleConnTimeout:     90 * time.Second,
					DisableCompression:  false,
					DisableKeepAlives:   false,
				},
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					// 禁用自动重定向以提高性能
					return http.ErrUseLastResponse
				},
			}
		},
	}

	server := &HeliosServer{
		config:       &cfg.PxyFrontend,
		logger:       logger,
		cacheManager: cacheManager,
		clientPool:   clientPool,
	}

	server.setupGin()
	return server, nil
}

// GetHTTPClient 从池中获取HTTP客户端
func (s *HeliosServer) GetHTTPClient() *http.Client {
	return s.clientPool.Get().(*http.Client)
}

// PutHTTPClient 将HTTP客户端归还到池中
func (s *HeliosServer) PutHTTPClient(client *http.Client) {
	s.clientPool.Put(client)
}

// setupGin 设置Gin引擎
func (s *HeliosServer) setupGin() {
	gin.SetMode(gin.ReleaseMode)
	s.gin = gin.New()

	// 添加中间件
	s.gin.Use(LoggingMiddleware(s.logger, s.config))
	s.gin.Use(CustomHeadersMiddleware(s.config))
	// 后端代理中间件，优先级高于静态文件路由
	s.gin.Use(BackendProxyMiddleware(s))
	s.gin.Use(RoutingMiddleware(s))
}

// HandleStaticFile 处理静态文件请求
func (s *HeliosServer) HandleStaticFile(c *gin.Context, serverConfig *config.FrontServerConfig) {
	requestPath := c.Request.URL.Path
	if requestPath == "/" {
		requestPath = "/" + serverConfig.Index
	}

	// 处理alias路径代理
	var filePath string
	aliasMatched := false
	for aliasPath, aliasRoot := range serverConfig.Alias {
		if strings.HasPrefix(requestPath, aliasPath) {
			// 移除alias前缀，获取相对路径
			relativePath := strings.TrimPrefix(requestPath, aliasPath)
			filePath = filepath.Join(aliasRoot, relativePath)
			aliasMatched = true
			break
		}
	}

	// 如果没有匹配到alias，使用默认的root路径
	if !aliasMatched {
		filePath = filepath.Join(serverConfig.Root, requestPath)
	}

	// 获取internal_flag
	internalFlag := c.GetHeader(s.config.InternalFlag)

	// 先检查缓存是否存在
	cachedFile := s.cacheManager.GetCachedFile(internalFlag, requestPath)
	if cachedFile != "" {
		// 缓存命中，添加响应头标识
		c.Header(s.config.CacheHeader, "True")
		c.File(cachedFile)
		return
	}

	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	// 检查是否匹配try_files
	isTryFiles := false
	if os.IsNotExist(err) {
		// 尝试tryFile机制
		if serverConfig.TryFile != "" {
			tryFilePath := filepath.Join(serverConfig.Root, serverConfig.TryFile)
			if _, err := os.Stat(tryFilePath); err == nil {
				filePath = tryFilePath
				isTryFiles = true
			} else {
				s.HandleError(c, 404, "File not found")
				return
			}
		} else {
			s.HandleError(c, 404, "File not found")
			return
		}
	} else if err == nil && fileInfo.IsDir() {
		// 处理目录访问
		if serverConfig.Type == "FileServer" {
			// FileServer模式：显示目录列表
			s.HandleDirectoryListing(c, filePath)
			return
		} else {
			// WebServer模式：目录访问视作前端路由
			c.File(filePath)
			return
		}
	}

	// 缓存文件（如果启用）
	if !isTryFiles && s.cacheManager.ShouldCache(requestPath, filePath) {
		s.cacheManager.CacheFile(internalFlag, requestPath, filePath)
	}

	c.File(filePath)
}

// HandleDirectoryListing 处理目录列表显示
func (s *HeliosServer) HandleDirectoryListing(c *gin.Context, dirPath string) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		s.HandleError(c, 500, "Failed to read directory")
		return
	}

	// 构建文件信息列表
	type FileInfo struct {
		Name    string
		IsDir   bool
		Size    int64
		ModTime string
	}

	var fileInfos []FileInfo
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, FileInfo{
			Name:    file.Name(),
			IsDir:   file.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// HTML模板
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Directory Listing - {{.Path}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .dir { color: #0066cc; }
        .file { color: #333; }
    </style>
</head>
<body>
    <h1>Directory Listing: {{.Path}}</h1>
    <table>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Size</th>
            <th>Modified</th>
        </tr>
        {{range .Files}}
        <tr>
            <td><a href="{{.Name}}{{if .IsDir}}/{{end}}" class="{{if .IsDir}}dir{{else}}file{{end}}">{{.Name}}</a></td>
            <td>{{if .IsDir}}Directory{{else}}File{{end}}</td>
            <td>{{if not .IsDir}}{{.Size}} bytes{{else}}-{{end}}</td>
            <td>{{.ModTime}}</td>
        </tr>
        {{end}}
    </table>
</body>
</html>`

	tmpl, err := template.New("directory").Parse(htmlTemplate)
	if err != nil {
		s.HandleError(c, 500, "Template parsing error")
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	err = tmpl.Execute(c.Writer, struct {
		Path  string
		Files []FileInfo
	}{
		Path:  c.Request.URL.Path,
		Files: fileInfos,
	})

	if err != nil {
		s.HandleError(c, 500, "Template execution error")
	}
}

// HandleError 处理错误响应
func (s *HeliosServer) HandleError(c *gin.Context, statusCode int, message string) {
	s.logger.Error().Int("status", statusCode).Str("message", message).Msg("HTTP Error")

	// 检查是否有自定义错误页面
	var errorPage string
	switch statusCode {
	case 404:
		errorPage = s.config.Error.NotFound
	case 500:
		errorPage = s.config.Error.InternalServerError
	}

	// 如果配置了自定义错误页面且文件存在，使用自定义页面
	if errorPage != "" {
		errorFilePath := filepath.Clean(errorPage)
		if _, err := os.Stat(errorFilePath); err == nil {
			c.File(errorFilePath)
			return
		}
	}

	// 使用embed的默认错误页面
	switch statusCode {
	case 404:
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Data(200, "text/html; charset=utf-8", PageNotFound)
		return
	case 500:
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Data(200, "text/html; charset=utf-8", PageInternalError)
		return
	}

	// 返回默认JSON错误响应
	c.JSON(statusCode, gin.H{"error": message})
}

// Start 启动服务器
func (s *HeliosServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info().Str("address", addr).Msg("Starting Helios server")
	return s.gin.Run(addr)
}

// Shutdown 优雅关闭服务器
func (s *HeliosServer) Shutdown() {
	s.logger.Info().Msg("Shutting down Helios server...")

	s.logger.Info().Msg("Server shutdown complete")
}

func (s *HeliosServer) Status() {
	s.logger.Info().Str("version", Version).Msg("Starting Helios Server")
	s.logger.Info().Msgf("[Helios] Server running on %s:%d", s.config.Host, s.config.Port)
	s.logger.Info().Msgf("[Helios] Cache enabled: %v", s.config.Cache.Enable)
	s.logger.Info().Msgf("[Helios] Log level: %v", s.config.Log.LogLevel)
}
