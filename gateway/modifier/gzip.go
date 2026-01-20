/*
Create: 2025/1/15
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

package modifier

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// GzipModifier gzip压缩响应处理中间件
type GzipModifier struct {
	enabled    bool
	level      int
	types      []string
	threshold  int
	writerPool sync.Pool // gzip.Writer 对象池
	bufferPool sync.Pool // bytes.Buffer 对象池
}

// NewGzipModifier 创建新的gzip中间件实例
func NewGzipModifier() *GzipModifier {
	cfg := config.Get()
	gm := &GzipModifier{
		enabled:   cfg.Features.Gzip.Enabled,
		level:     cfg.Features.Gzip.Level,
		types:     cfg.Features.Gzip.Types,
		threshold: cfg.Features.Gzip.Threshold,
	}

	// 初始化 gzip.Writer 对象池
	gm.writerPool = sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, gm.level)
			return w
		},
	}

	// 初始化 bytes.Buffer 对象池
	gm.bufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}

	return gm
}

func (g *GzipModifier) Use(response *http.Response) {
	_ = g.ModifyResponse(response)
}

// ModifyResponse 处理响应的gzip压缩
func (g *GzipModifier) ModifyResponse(response *http.Response) error {
	// 检查是否启用gzip
	if !g.enabled {
		return nil
	}

	// 检查客户端是否支持gzip
	if !g.clientSupportsGzip(response.Request) {
		return nil
	}

	// 检查响应内容类型是否需要压缩
	if !g.shouldCompress(response) {
		return nil
	}

	// 检查响应是否已经被压缩
	if response.Header.Get("Content-Encoding") != "" {
		logger.GetLogger().Debug().Msg("响应已被压缩，跳过gzip处理")
		return nil
	}

	ct := response.Header.Get("Content-Length")
	if ct != "" {
		size, err := strconv.Atoi(ct)
		if err == nil && size <= g.threshold {
			return nil
		}
	}
	// 读取原始响应体
	var buf bytes.Buffer
	tee := io.TeeReader(response.Body, &buf)
	originalBody, err := io.ReadAll(tee)
	if err != nil {
		logger.GetLogger().Debug().Err(err).Msg("读取响应体失败")
		return err
	}

	// 检查响应体大小，太小的响应不需要压缩
	if len(originalBody) <= g.threshold { // 小于1KB不压缩
		response.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		return nil
	}

	// 压缩响应体
	compressedBody, err := g.compressData(originalBody)
	if err != nil {
		logger.GetLogger().Debug().Err(err).Msg("gzip压缩失败")
		// 压缩失败时返回原始响应
		response.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		return nil
	}

	// 检查压缩效果，如果压缩后更大则不使用压缩
	if len(compressedBody) >= len(originalBody) {
		logger.GetLogger().Debug().Msg("压缩后大小未减少，使用原始响应")
		response.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		return nil
	}

	// 设置压缩相关的响应头
	response.Header.Set("Content-Encoding", "gzip")
	response.Header.Set("Content-Length", strconv.Itoa(len(compressedBody)))
	response.Header.Del("Content-Range") // 移除Range相关头部，因为内容已改变

	// 设置新的响应体
	response.Body = io.NopCloser(bytes.NewReader(compressedBody))

	if config.Get().Debug {
		logger.GetLogger().Debug().
			Int("原始大小", len(originalBody)).
			Int("压缩大小", len(compressedBody)).
			Float64("压缩率", float64(len(originalBody)-len(compressedBody))/float64(len(originalBody))*100).Msg("gzip压缩成功")
	}
	return nil
}

// clientSupportsGzip 检查客户端是否支持gzip
func (g *GzipModifier) clientSupportsGzip(request *http.Request) bool {
	if request == nil {
		return false
	}

	acceptEncoding := request.Header.Get("Accept-Encoding")
	return strings.Contains(strings.ToLower(acceptEncoding), "gzip")
}

// shouldCompress 检查响应是否应该被压缩
func (g *GzipModifier) shouldCompress(response *http.Response) bool {
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		return false
	}

	// 提取主要的MIME类型（去除参数部分）
	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(strings.ToLower(mainType))

	// 检查是否在可压缩类型列表中
	for _, allowedType := range g.types {
		if strings.EqualFold(mainType, strings.TrimSpace(allowedType)) {
			return true
		}
	}

	logger.GetLogger().Debug().Str("响应类型", contentType).Msg("响应类型不在可压缩列表中")
	return false
}

// compressData 压缩数据（使用对象池优化）
func (g *GzipModifier) compressData(data []byte) ([]byte, error) {
	// 从对象池获取 buffer
	buf := g.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()           // 重置buffer内容
		g.bufferPool.Put(buf) // 归还到对象池
	}()

	// 从对象池获取 gzip writer
	gzw := g.writerPool.Get().(*gzip.Writer)
	defer func() {
		gzw.Reset(nil)        // 重置writer
		g.writerPool.Put(gzw) // 归还到对象池
	}()

	// 重置 writer 到新的 buffer
	gzw.Reset(buf)

	// 写入数据
	_, err := gzw.Write(data)
	if err != nil {
		return nil, err
	}

	// 关闭 writer 以完成压缩
	err = gzw.Close()
	if err != nil {
		return nil, err
	}

	// 复制数据到新的字节切片返回
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// IsEnabled 返回gzip是否启用
func (g *GzipModifier) IsEnabled() bool {
	return g.enabled
}

// GetLevel 返回压缩级别
func (g *GzipModifier) GetLevel() int {
	return g.level
}

// GetTypes 返回可压缩的MIME类型列表
func (g *GzipModifier) GetTypes() []string {
	return g.types
}

// UpdateConfig 更新配置（支持热更新）
func (g *GzipModifier) UpdateConfig() {
	cfg := config.Get()
	oldLevel := g.level
	g.enabled = cfg.Features.Gzip.Enabled
	g.level = cfg.Features.Gzip.Level
	g.types = cfg.Features.Gzip.Types

	// 如果压缩级别发生变化，需要重新初始化 writer 对象池
	if oldLevel != g.level {
		g.writerPool = sync.Pool{
			New: func() interface{} {
				w, _ := gzip.NewWriterLevel(nil, g.level)
				return w
			},
		}
		logger.GetLogger().Debug().Int("level", g.level).Msg("gzip压缩级别已更新，writer对象池已重新初始化")
	}
	logger.GetLogger().Debug().Bool("enable", g.enabled).Int("level", g.level).Any("types", g.types).Msg("gzip配置已更新")
}

// GetName 获取修改器名称
func (g *GzipModifier) GetName() string {
	return "gzip"
}
