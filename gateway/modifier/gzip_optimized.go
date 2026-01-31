package modifier

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// OptimizedGzipModifier 优化版本的Gzip修改器
type OptimizedGzipModifier struct {
	enabled    bool
	level      int
	types      []string
	threshold  int
	writerPool sync.Pool // gzip.Writer 对象池
	bufferPool sync.Pool // bytes.Buffer 对象池

	// 新增优化配置
	asyncThreshold   int           // 异步压缩阈值
	maxWorkers       int           // 最大工作协程数
	workerPool       chan struct{} // 工作协程池
	compressionCache sync.Map      // 压缩结果缓存
	cacheEnabled     bool          // 是否启用缓存
	cacheTTL         time.Duration // 缓存过期时间

	// 性能统计
	stats struct {
		sync.RWMutex
		totalRequests      int64
		compressedCount    int64
		cacheHits          int64
		avgCompressionTime time.Duration
	}
}

// CacheEntry 缓存条目
type CacheEntry struct {
	data      []byte
	headers   map[string]string
	timestamp time.Time
}

// NewOptimizedGzipModifier 创建优化版本的Gzip修改器
func NewOptimizedGzipModifier() *OptimizedGzipModifier {
	cfg := config.Get()

	modifier := &OptimizedGzipModifier{
		enabled:        cfg.Middleware.Gzip.Enabled,
		level:          cfg.Middleware.Gzip.Level,
		types:          cfg.Middleware.Gzip.Types,
		threshold:      cfg.Middleware.Gzip.Threshold,
		asyncThreshold: 100 * 1024, // 100KB以上异步压缩
		maxWorkers:     runtime.NumCPU(),
		cacheEnabled:   true,
		cacheTTL:       5 * time.Minute,
	}

	// 初始化工作协程池
	modifier.workerPool = make(chan struct{}, modifier.maxWorkers)
	for i := 0; i < modifier.maxWorkers; i++ {
		modifier.workerPool <- struct{}{}
	}

	// 初始化gzip.Writer对象池
	modifier.writerPool = sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, modifier.level)
			return w
		},
	}

	// 初始化bytes.Buffer对象池
	modifier.bufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}

	// 启动缓存清理协程
	if modifier.cacheEnabled {
		go modifier.cleanupCache()
	}

	logger.GetLogger().Info().
		Bool("enable", modifier.enabled).
		Int("level", modifier.level).
		Int("threshold", modifier.threshold).
		Int("asyncThreshold", modifier.asyncThreshold).Msg("optimized gzip modifier initialized")

	return modifier
}

// Use 应用修改器
func (g *OptimizedGzipModifier) Use(response *http.Response) {
	if g.enabled {
		g.ModifyResponse(response)
	}
}

// ModifyResponse 修改响应（优化版本）
func (g *OptimizedGzipModifier) ModifyResponse(response *http.Response) error {
	start := time.Now()
	defer func() {
		g.updateStats(time.Since(start))
	}()

	// 基本检查
	if !g.enabled || response == nil || response.Request == nil {
		return nil
	}

	// 检查客户端是否支持gzip
	if !g.clientSupportsGzip(response.Request) {
		return nil
	}

	// 检查是否应该压缩
	if !g.shouldCompress(response) {
		return nil
	}

	// 读取原始响应体
	originalData, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	response.Body.Close()

	// 检查数据大小是否达到压缩阈值
	if len(originalData) < g.threshold {
		response.Body = io.NopCloser(bytes.NewReader(originalData))
		return nil
	}

	// 尝试从缓存获取
	if g.cacheEnabled {
		if cached := g.getFromCache(originalData); cached != nil {
			g.applyCachedResponse(response, cached)
			g.stats.Lock()
			g.stats.cacheHits++
			g.stats.Unlock()
			return nil
		}
	}

	// 根据数据大小选择压缩策略
	if len(originalData) >= g.asyncThreshold {
		return g.compressAsync(response, originalData)
	} else {
		return g.compressSync(response, originalData)
	}
}

// compressSync 同步压缩
func (g *OptimizedGzipModifier) compressSync(response *http.Response, data []byte) error {
	compressedData, headers, err := g.compressData(data)
	if err != nil {
		// 压缩失败，返回原始数据
		response.Body = io.NopCloser(bytes.NewReader(data))
		return nil
	}

	// 检查压缩效果
	if len(compressedData) >= len(data) {
		// 压缩效果不佳，返回原始数据
		response.Body = io.NopCloser(bytes.NewReader(data))
		return nil
	}

	// 应用压缩结果
	g.applyCompression(response, compressedData, headers)

	// 缓存结果
	if g.cacheEnabled {
		g.cacheResult(data, compressedData, headers)
	}

	return nil
}

// compressAsync 异步压缩
func (g *OptimizedGzipModifier) compressAsync(response *http.Response, data []byte) error {
	// 获取工作协程
	select {
	case <-g.workerPool:
		defer func() {
			g.workerPool <- struct{}{}
		}()
	case <-time.After(100 * time.Millisecond):
		// 超时则使用同步压缩
		return g.compressSync(response, data)
	}

	// 在协程中执行压缩
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resultChan := make(chan struct {
		data    []byte
		headers map[string]string
		err     error
	}, 1)

	go func() {
		compressedData, headers, err := g.compressData(data)
		select {
		case resultChan <- struct {
			data    []byte
			headers map[string]string
			err     error
		}{compressedData, headers, err}:
		case <-ctx.Done():
		}
	}()

	select {
	case result := <-resultChan:
		if result.err != nil || len(result.data) >= len(data) {
			// 压缩失败或效果不佳，返回原始数据
			response.Body = io.NopCloser(bytes.NewReader(data))
			return nil
		}

		g.applyCompression(response, result.data, result.headers)

		// 缓存结果
		if g.cacheEnabled {
			g.cacheResult(data, result.data, result.headers)
		}

		return nil
	case <-ctx.Done():
		// 超时，返回原始数据
		response.Body = io.NopCloser(bytes.NewReader(data))
		return nil
	}
}

// compressData 压缩数据（优化版本）
func (g *OptimizedGzipModifier) compressData(data []byte) ([]byte, map[string]string, error) {
	// 从对象池获取buffer
	buf := g.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		g.bufferPool.Put(buf)
	}()

	// 从对象池获取writer
	writer := g.writerPool.Get().(*gzip.Writer)
	defer g.writerPool.Put(writer)

	// 重置writer
	writer.Reset(buf)

	// 写入数据
	if _, err := writer.Write(data); err != nil {
		return nil, nil, err
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		return nil, nil, err
	}

	// 准备响应头
	headers := map[string]string{
		"Content-Encoding": "gzip",
		"Content-Length":   strconv.Itoa(buf.Len()),
		"Vary":             "Accept-Encoding",
	}

	// 复制压缩后的数据
	compressedData := make([]byte, buf.Len())
	copy(compressedData, buf.Bytes())

	return compressedData, headers, nil
}

// applyCompression 应用压缩结果
func (g *OptimizedGzipModifier) applyCompression(response *http.Response, data []byte, headers map[string]string) {
	// 设置响应体
	response.Body = io.NopCloser(bytes.NewReader(data))

	// 设置响应头
	for key, value := range headers {
		response.Header.Set(key, value)
	}

	// 移除原始Content-Length（如果存在）
	response.Header.Del("Content-Length")
	response.Header.Set("Content-Length", strconv.Itoa(len(data)))

	g.stats.Lock()
	g.stats.compressedCount++
	g.stats.Unlock()
}

// 缓存相关方法
func (g *OptimizedGzipModifier) getFromCache(data []byte) *CacheEntry {
	if !g.cacheEnabled {
		return nil
	}

	key := g.generateCacheKey(data)
	if value, ok := g.compressionCache.Load(key); ok {
		entry := value.(*CacheEntry)
		if time.Since(entry.timestamp) < g.cacheTTL {
			return entry
		}
		// 过期删除
		g.compressionCache.Delete(key)
	}
	return nil
}

func (g *OptimizedGzipModifier) cacheResult(original, compressed []byte, headers map[string]string) {
	if !g.cacheEnabled {
		return
	}

	key := g.generateCacheKey(original)
	entry := &CacheEntry{
		data:      compressed,
		headers:   headers,
		timestamp: time.Now(),
	}
	g.compressionCache.Store(key, entry)
}

func (g *OptimizedGzipModifier) generateCacheKey(data []byte) string {
	// 简单的哈希键生成（实际应用中可以使用更好的哈希算法）
	return strconv.Itoa(len(data)) + "_" + strconv.Itoa(int(data[0])) + "_" + strconv.Itoa(int(data[len(data)-1]))
}

func (g *OptimizedGzipModifier) applyCachedResponse(response *http.Response, entry *CacheEntry) {
	response.Body = io.NopCloser(bytes.NewReader(entry.data))
	for key, value := range entry.headers {
		response.Header.Set(key, value)
	}
}

func (g *OptimizedGzipModifier) cleanupCache() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		g.compressionCache.Range(func(key, value interface{}) bool {
			entry := value.(*CacheEntry)
			if time.Since(entry.timestamp) > g.cacheTTL {
				g.compressionCache.Delete(key)
			}
			return true
		})
	}
}

// 性能统计方法
func (g *OptimizedGzipModifier) updateStats(duration time.Duration) {
	g.stats.Lock()
	defer g.stats.Unlock()

	g.stats.totalRequests++
	// 计算平均压缩时间
	if g.stats.totalRequests == 1 {
		g.stats.avgCompressionTime = duration
	} else {
		g.stats.avgCompressionTime = (g.stats.avgCompressionTime*time.Duration(g.stats.totalRequests-1) + duration) / time.Duration(g.stats.totalRequests)
	}
}

// GetStats 获取性能统计
func (g *OptimizedGzipModifier) GetStats() map[string]interface{} {
	g.stats.RLock()
	defer g.stats.RUnlock()

	return map[string]interface{}{
		"total_requests":       g.stats.totalRequests,
		"compressed_count":     g.stats.compressedCount,
		"cache_hits":           g.stats.cacheHits,
		"avg_compression_time": g.stats.avgCompressionTime.String(),
		"compression_ratio":    float64(g.stats.compressedCount) / float64(g.stats.totalRequests),
		"cache_hit_ratio":      float64(g.stats.cacheHits) / float64(g.stats.totalRequests),
	}
}

// 保持原有的辅助方法
func (g *OptimizedGzipModifier) clientSupportsGzip(request *http.Request) bool {
	acceptEncoding := request.Header.Get("Accept-Encoding")
	return strings.Contains(strings.ToLower(acceptEncoding), "gzip")
}

func (g *OptimizedGzipModifier) shouldCompress(response *http.Response) bool {
	// 检查是否已经压缩
	if response.Header.Get("Content-Encoding") != "" {
		return false
	}

	// 检查Content-Type
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		return false
	}

	// 检查是否在支持的类型列表中
	for _, supportedType := range g.types {
		if strings.Contains(strings.ToLower(contentType), strings.ToLower(supportedType)) {
			return true
		}
	}

	return false
}

// 配置更新方法
func (g *OptimizedGzipModifier) UpdateConfig() {
	cfg := config.Get()
	oldLevel := g.level
	g.enabled = cfg.Middleware.Gzip.Enabled
	g.level = cfg.Middleware.Gzip.Level
	g.types = cfg.Middleware.Gzip.Types
	g.threshold = cfg.Middleware.Gzip.Threshold

	// 如果压缩级别发生变化，需要重新初始化 writer 对象池
	if oldLevel != g.level {
		g.writerPool = sync.Pool{
			New: func() interface{} {
				w, _ := gzip.NewWriterLevel(nil, g.level)
				return w
			},
		}
		logger.GetLogger().Debug().Int("level", g.level).Msg("gzip compression level updated, writer pool re-initialized")
	}

	logger.GetLogger().Debug().Bool("enable", g.enabled).Int("level", g.level).Any("types", g.types).Msg("gzip configuration updated")
}

func (g *OptimizedGzipModifier) GetName() string {
	return "optimized-gzip"
}
