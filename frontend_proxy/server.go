package frontend_proxy

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/frontproxy_config"
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/net/http2"
	"html/template"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/rs/zerolog"

	"github.com/gin-gonic/gin"
)

// HeliosServer 服务器结构体
type HeliosServer struct {
	config        *frontproxy_config.PxyFrontConfig
	logger        *zerolog.Logger
	gin           *gin.Engine
	cacheManager  *CacheManager
	clientPool    *sync.Pool
	serverIndexMu sync.RWMutex
	serverIndex   map[string]frontproxy_config.FrontServerConfig
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

	server.refreshIndexes()
	server.setupGin()
	return server, nil
}

// refreshIndexes rebuilds the read-mostly indexes after configuration changes.
// Values are copied instead of storing pointers into the configuration slice so
// a later config replacement cannot leave a dangling slice-element pointer.
func (s *HeliosServer) refreshIndexes() {
	index := make(map[string]frontproxy_config.FrontServerConfig, len(s.config.Servers))
	for _, server := range s.config.Servers {
		if server.Name == "" {
			continue
		}
		// Keep the first-match behavior of the previous linear lookup if a
		// malformed configuration contains duplicate names.
		if _, exists := index[server.Name]; exists {
			continue
		}
		index[server.Name] = server
	}

	s.serverIndexMu.Lock()
	s.serverIndex = index
	s.serverIndexMu.Unlock()
	if s.cacheManager != nil {
		s.cacheManager.Reset()
	}
}

// RefreshConfig refreshes indexes and invalidates cached content after an
// in-place configuration reload.
func (s *HeliosServer) RefreshConfig() {
	s.refreshIndexes()
}

func (s *HeliosServer) lookupServer(name string) (frontproxy_config.FrontServerConfig, bool) {
	s.serverIndexMu.RLock()
	server, ok := s.serverIndex[name]
	s.serverIndexMu.RUnlock()
	return server, ok
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
	s.gin.Use(LoggingMiddleware(s))
	s.gin.Use(CustomHeadersMiddleware(s.config))
	// 后端代理中间件，优先级高于静态文件路由
	s.gin.Use(BackendProxyMiddleware(s))
	s.gin.Use(RoutingMiddleware(s))
}

func (s *HeliosServer) GetHandler() http.Handler {
	return s.gin
}

// HandleStaticFile 处理静态文件请求
func (s *HeliosServer) HandleStaticFile(c *gin.Context, serverConfig *frontproxy_config.FrontServerConfig) {
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

	// 先查内存缓存，再回落到磁盘缓存。内存命中在验证窗口内不触发
	// 文件系统调用；缓存层未命中时继续走下面的源文件和 try_file 逻辑。
	if s.cacheManager.ServeCached(c, internalFlag, requestPath, filePath) {
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
	s.logger.Error().Int("status", statusCode).Str("message", message).Msg("http error")

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
	// The control API reloads the shared config in place. Rebuild the indexes
	// before a restarted listener begins serving requests.
	s.refreshIndexes()
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	if s.config.ExpFastConnect.Http3.Enabled {
		return s.http3Server(addr)
	}
	if !s.config.ExpFastConnect.Enabled {
		s.logger.Info().Str("address", addr).Msg("starting helios server")
		return s.httpServer(addr)
	}

	return s.http2Server(addr)
}

// Shutdown 优雅关闭服务器
func (s *HeliosServer) Shutdown() {
	s.logger.Info().Msg("shutting down helios server...")

	s.logger.Info().Msg("server shutdown complete")
}

func (s *HeliosServer) Status() {
	s.logger.Info().Str("version", Version).Msg("starting helios server")
	s.logger.Info().Msgf("[Helios] server running on %s:%d", s.config.Host, s.config.Port)
	s.logger.Info().Msgf("[Helios] cache enabled: %v", s.config.Cache.Enable)
}

func (s *HeliosServer) httpServer(addr string) error {
	return s.gin.Run(addr)
}

func (s *HeliosServer) http2Server(addr string) error {
	proto := &http.Protocols{}
	proto.SetHTTP2(true)
	proto.SetHTTP1(true)
	proto.SetUnencryptedHTTP2(true)

	h2c := s.config.ExpFastConnect.Http2
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.gin,
		ReadTimeout:       time.Second * time.Duration(defaultInt64(h2c.ReadTimeout, 30)),
		WriteTimeout:      time.Second * time.Duration(defaultInt64(h2c.WriteTimeout, 30)),
		IdleTimeout:       time.Second * time.Duration(defaultInt64(h2c.IdleTimeout, 60)),
		ReadHeaderTimeout: time.Second * time.Duration(defaultInt64(h2c.ReadHeaderTimeout, 10)),
		MaxHeaderBytes:    int(defaultInt64(h2c.MaxHeaderBytes, 5<<20)),
		Protocols:         proto,
	}
	h2s := &http2.Server{}
	if h2c.MaxHandlers > 0 {
		h2s.MaxHandlers = h2c.MaxHandlers
	}
	if h2c.MaxConcurrentStreams > 0 {
		h2s.MaxConcurrentStreams = uint32(h2c.MaxConcurrentStreams)
	}
	if h2c.MaxUploadBufferPerConnection > 0 {
		h2s.MaxUploadBufferPerConnection = int32(h2c.MaxUploadBufferPerConnection)
	}
	if h2c.MaxUploadBufferPerStream > 0 {
		h2s.MaxUploadBufferPerStream = int32(h2c.MaxUploadBufferPerStream)
	}
	if err := http2.ConfigureServer(httpServer, h2s); err != nil {
		return err
	}

	listener, err := listenWithKeepAlive(addr, s.config.ExpFastConnect.Http2.KeepAlive)
	if err != nil {
		return err
	}

	s.logger.Info().Str("address", addr).Msg("starting helios http2 server")
	return httpServer.Serve(listener)
}

func (s *HeliosServer) http3Server(addr string) error {
	h3cfg := s.config.ExpFastConnect.Http3
	if h3cfg.CertFile == "" || h3cfg.KeyFile == "" {
		return fmt.Errorf("http3 cert or key not configured")
	}
	cert, err := tls.LoadX509KeyPair(h3cfg.CertFile, h3cfg.KeyFile)
	if err != nil {
		return err
	}
	tlsConfig := http3.ConfigureTLSConfig(&tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	})
	quicConfig := &quic.Config{
		EnableDatagrams: true,
	}
	if h3cfg.MaxConnections > 0 {
		quicConfig.MaxIncomingStreams = int64(h3cfg.MaxConnections)
	}
	if h3cfg.IdleTimeout > 0 {
		quicConfig.MaxIdleTimeout = time.Duration(h3cfg.IdleTimeout) * time.Second
	}
	if h3cfg.KeepAlive > 0 {
		quicConfig.KeepAlivePeriod = time.Duration(h3cfg.KeepAlive) * time.Second
	}
	server := http3.Server{
		Addr:       addr,
		Handler:    s.gin,
		TLSConfig:  tlsConfig,
		QUICConfig: quicConfig,
	}
	s.logger.Info().Str("address", addr).Msg("starting helios http3 server")
	return server.ListenAndServe()
}

func listenWithKeepAlive(addr string, keepAliveSeconds int64) (net.Listener, error) {
	listenerConfig := net.ListenConfig{}
	if keepAliveSeconds > 0 {
		listenerConfig.KeepAlive = time.Duration(keepAliveSeconds) * time.Second
	}
	return listenerConfig.Listen(context.Background(), "tcp", addr)
}

func defaultInt64(value int64, fallback int64) int64 {
	if value > 0 {
		return value
	}
	return fallback
}
