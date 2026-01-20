package frontend_proxy

import (
	"Hamburger/internal/config"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
	"time"
)

// CacheManager 缓存管理器
type CacheManager struct {
	config *config.PxyFrontConfig
	logger *zerolog.Logger
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(config *config.Config, logger *zerolog.Logger) *CacheManager {
	return &CacheManager{
		config: &config.PxyFrontend,
		logger: logger,
	}
}

// ShouldCache 检查文件是否应该被缓存
func (cm *CacheManager) ShouldCache(reqPath, filePath string) bool {
	if !cm.config.Cache.Enable {
		return false
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	ext := filepath.Ext(reqPath)
	for _, pattern := range cm.config.Cache.Matcher {
		if matched, _ := filepath.Match(pattern, "*"+ext); matched {
			return true
		}
	}
	return false
}

// GetCachedFile 获取缓存文件
func (cm *CacheManager) GetCachedFile(internalFlag, requestPath string) string {
	cacheDir := cm.config.Cache.Dir
	// 使用internal_flag和requestPath组合作为缓存key，避免资源冲突
	cacheKey := filepath.Join(internalFlag, requestPath)
	cachedFilePath := filepath.Join(cacheDir, cacheKey)

	// 检查缓存文件是否存在
	if stat, err := os.Stat(cachedFilePath); err == nil {
		// 检查是否过期（如果expire > 0）
		if cm.config.Cache.Expire > 0 {
			if time.Since(stat.ModTime()).Minutes() > float64(cm.config.Cache.Expire) {
				// 缓存已过期，删除文件
				os.Remove(cachedFilePath)
				return ""
			}
		}
		return cachedFilePath
	}
	return ""
}

// CacheFile 缓存文件
func (cm *CacheManager) CacheFile(internalFlag, requestPath, originalFilePath string) {
	cacheDir := cm.config.Cache.Dir
	// 使用internal_flag和requestPath组合作为缓存key，避免资源冲突
	cacheKey := filepath.Join(internalFlag, requestPath)
	cachedFilePath := filepath.Join(cacheDir, cacheKey)

	// 创建缓存目录
	os.MkdirAll(filepath.Dir(cachedFilePath), 0755)

	// 复制文件到缓存目录
	data, err := os.ReadFile(originalFilePath)
	if err != nil {
		cm.logger.Error().Err(err).Msg("Failed to read original file for caching")
		return
	}

	err = os.WriteFile(cachedFilePath, data, 0644)
	if err != nil {
		cm.logger.Error().Err(err).Msg("Failed to write cached file")
	}
}
