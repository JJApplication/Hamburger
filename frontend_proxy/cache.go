package frontend_proxy

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/frontproxy_config"
	"bytes"
	"container/list"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"golang.org/x/sync/singleflight"
)

const (
	defaultMemoryMaxMB            = 256
	defaultMemoryMaxFileMB        = 8
	defaultMemoryMaxEntries       = 4096
	defaultMemoryRevalidateSecond = 60
	diskCacheLockStripes          = 256
)

// CacheManager provides a memory-first cache backed by the existing on-disk
// cache. The source metadata in each entry lets us invalidate content after a
// deployment without reading the file body again.
type CacheManager struct {
	config *frontproxy_config.PxyFrontConfig
	logger *zerolog.Logger
	memory *memoryFileCache
	group  singleflight.Group
	// Striped locks coordinate disk readers with the two-file data/metadata
	// commit without retaining an unbounded lock for every request path.
	diskLocks [diskCacheLockStripes]sync.RWMutex
}

type memoryFileEntry struct {
	key         string
	sourcePath  string
	name        string
	data        []byte
	size        int64
	modTime     time.Time
	validatedAt time.Time
}

type memoryFileCache struct {
	mu         sync.Mutex
	items      map[string]*list.Element
	lru        *list.List
	maxBytes   int64
	maxEntries int
	usedBytes  int64
}

func newMemoryFileCache(maxBytes int64, maxEntries int) *memoryFileCache {
	return &memoryFileCache{
		items:      make(map[string]*list.Element),
		lru:        list.New(),
		maxBytes:   maxBytes,
		maxEntries: maxEntries,
	}
}

func (c *memoryFileCache) get(key string) (memoryFileEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	element, ok := c.items[key]
	if !ok {
		return memoryFileEntry{}, false
	}
	c.lru.MoveToFront(element)
	return element.Value.(memoryFileEntry), true
}

func (c *memoryFileCache) touch(key string, validatedAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.items[key]; ok {
		entry := element.Value.(memoryFileEntry)
		entry.validatedAt = validatedAt
		element.Value = entry
		c.lru.MoveToFront(element)
	}
}

func (c *memoryFileCache) put(entry memoryFileEntry) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.maxBytes <= 0 || c.maxEntries <= 0 || entry.size > c.maxBytes {
		return false
	}
	if element, ok := c.items[entry.key]; ok {
		old := element.Value.(memoryFileEntry)
		c.usedBytes -= old.size
		c.lru.Remove(element)
		delete(c.items, entry.key)
	}
	for c.usedBytes+entry.size > c.maxBytes || len(c.items) >= c.maxEntries {
		oldest := c.lru.Back()
		if oldest == nil {
			break
		}
		old := oldest.Value.(memoryFileEntry)
		c.usedBytes -= old.size
		delete(c.items, old.key)
		c.lru.Remove(oldest)
	}
	if c.usedBytes+entry.size > c.maxBytes || len(c.items) >= c.maxEntries {
		return false
	}
	element := c.lru.PushFront(entry)
	c.items[entry.key] = element
	c.usedBytes += entry.size
	return true
}

func (c *memoryFileCache) remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.items[key]; ok {
		entry := element.Value.(memoryFileEntry)
		c.usedBytes -= entry.size
		delete(c.items, key)
		c.lru.Remove(element)
	}
}

func (c *memoryFileCache) reset(maxBytes int64, maxEntries int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxBytes = maxBytes
	c.maxEntries = maxEntries
	c.items = make(map[string]*list.Element)
	c.lru.Init()
	c.usedBytes = 0
}

// NewCacheManager creates the frontend cache manager. New memory settings are
// defaulted here so older configuration files get the optimized behavior.
func NewCacheManager(cfg *config.Config, logger *zerolog.Logger) *CacheManager {
	frontCfg := &cfg.PxyFrontend
	maxMB := frontCfg.Cache.MemoryMaxMB
	if maxMB <= 0 {
		maxMB = defaultMemoryMaxMB
	}
	maxFileMB := frontCfg.Cache.MemoryMaxFileMB
	if maxFileMB <= 0 {
		maxFileMB = defaultMemoryMaxFileMB
	}
	maxEntries := frontCfg.Cache.MemoryMaxEntries
	if maxEntries <= 0 {
		maxEntries = defaultMemoryMaxEntries
	}
	return &CacheManager{
		config: frontCfg,
		logger: logger,
		memory: newMemoryFileCache(int64(maxMB)*1024*1024, maxEntries),
	}
}

func (cm *CacheManager) memoryEnabled() bool {
	cache := cm.config.Cache
	if !cache.Enable {
		return false
	}
	// A nil value means the option was omitted by an older configuration and
	// therefore receives the optimized default. An explicit false remains a
	// reliable opt-out.
	if cache.MemoryEnable == nil {
		return true
	}
	return *cache.MemoryEnable
}

func (cm *CacheManager) memoryMaxFileBytes() int64 {
	maxMB := cm.config.Cache.MemoryMaxFileMB
	if maxMB <= 0 {
		maxMB = defaultMemoryMaxFileMB
	}
	return int64(maxMB) * 1024 * 1024
}

func (cm *CacheManager) revalidateAfter() time.Duration {
	seconds := cm.config.Cache.MemoryRevalidateSeconds
	if seconds <= 0 {
		seconds = defaultMemoryRevalidateSecond
	}
	return time.Duration(seconds) * time.Second
}

// Reset drops process-local file content. It is called after a frontend
// configuration reload so an old root/index cannot be served from memory.
func (cm *CacheManager) Reset() {
	if cm.memory != nil {
		maxMB := cm.config.Cache.MemoryMaxMB
		if maxMB <= 0 {
			maxMB = defaultMemoryMaxMB
		}
		maxEntries := cm.config.Cache.MemoryMaxEntries
		if maxEntries <= 0 {
			maxEntries = defaultMemoryMaxEntries
		}
		cm.memory.reset(int64(maxMB)*1024*1024, maxEntries)
	}
}

// ShouldCache checks whether a source file is eligible for the configured
// frontend cache.
func (cm *CacheManager) ShouldCache(reqPath, filePath string) bool {
	if !cm.config.Cache.Enable {
		return false
	}
	if _, err := os.Stat(filePath); err != nil {
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

func (cm *CacheManager) cacheKey(internalFlag, requestPath string) string {
	return internalFlag + "\x00" + requestPath
}

func (cm *CacheManager) diskLock(key string) *sync.RWMutex {
	const (
		fnvOffset32 = uint32(2166136261)
		fnvPrime32  = uint32(16777619)
	)
	hash := fnvOffset32
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= fnvPrime32
	}
	return &cm.diskLocks[hash%diskCacheLockStripes]
}

func (cm *CacheManager) cachedFilePath(internalFlag, requestPath string) string {
	// Keep the existing human-readable layout while preventing a request path
	// from escaping the configured cache directory.
	cleanFlag := filepath.Clean(internalFlag)
	if cleanFlag == "." || cleanFlag == string(filepath.Separator) ||
		filepath.IsAbs(cleanFlag) || filepath.VolumeName(cleanFlag) != "" ||
		strings.Contains(cleanFlag, "..") {
		cleanFlag = "_default"
	}
	cleanPath := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(requestPath, "/")))
	if cleanPath == "." || cleanPath == string(filepath.Separator) ||
		filepath.IsAbs(cleanPath) || filepath.VolumeName(cleanPath) != "" ||
		strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) || cleanPath == ".." {
		cleanPath = "_root"
	}
	return filepath.Join(cm.config.Cache.Dir, cleanFlag, cleanPath)
}

func metadataPath(cachePath string) string {
	return cachePath + ".meta"
}

type cacheMetadata struct {
	SourcePath string `json:"source_path"`
	Size       int64  `json:"size"`
	ModTime    int64  `json:"mod_time_unix_nano"`
}

func sourceMetadata(path string) (cacheMetadata, os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		if err == nil {
			err = os.ErrInvalid
		}
		return cacheMetadata{}, nil, err
	}
	return cacheMetadata{
		SourcePath: path,
		Size:       info.Size(),
		ModTime:    info.ModTime().UnixNano(),
	}, info, nil
}

func (cm *CacheManager) diskExpired(info os.FileInfo) bool {
	return cm.config.Cache.Expire > 0 &&
		time.Since(info.ModTime()).Minutes() > float64(cm.config.Cache.Expire)
}

func (cm *CacheManager) readDiskMetadata(cachePath string) (cacheMetadata, bool) {
	data, err := os.ReadFile(metadataPath(cachePath))
	if err != nil {
		return cacheMetadata{}, false
	}
	var metadata cacheMetadata
	if json.Unmarshal(data, &metadata) != nil || metadata.SourcePath == "" {
		return cacheMetadata{}, false
	}
	return metadata, true
}

func (cm *CacheManager) diskEntry(internalFlag, requestPath, sourcePath string) (string, cacheMetadata, bool) {
	cachePath := cm.cachedFilePath(internalFlag, requestPath)
	cacheInfo, err := os.Stat(cachePath)
	if err != nil {
		return "", cacheMetadata{}, false
	}
	if !cacheInfo.Mode().IsRegular() || cm.diskExpired(cacheInfo) {
		return "", cacheMetadata{}, false
	}
	metadata, ok := cm.readDiskMetadata(cachePath)
	if !ok || metadata.SourcePath != sourcePath {
		return "", cacheMetadata{}, false
	}
	source, _, err := sourceMetadata(sourcePath)
	if err != nil || source.Size != metadata.Size || source.ModTime != metadata.ModTime {
		return "", cacheMetadata{}, false
	}
	return cachePath, metadata, true
}

func (cm *CacheManager) removeDiskFiles(internalFlag, requestPath string) {
	cachePath := cm.cachedFilePath(internalFlag, requestPath)
	_ = os.Remove(cachePath)
	_ = os.Remove(metadataPath(cachePath))
}

func (cm *CacheManager) invalidateDiskIfInvalid(key, internalFlag, requestPath, sourcePath string) {
	lock := cm.diskLock(key)
	lock.Lock()
	defer lock.Unlock()
	// A writer may have committed a fresh pair between the caller's failed
	// lookup and this exclusive section. Revalidate before deleting anything.
	if _, _, ok := cm.diskEntry(internalFlag, requestPath, sourcePath); ok {
		return
	}
	cm.removeDiskFiles(internalFlag, requestPath)
}

func serveMemory(c *gin.Context, entry memoryFileEntry) {
	http.ServeContent(c.Writer, c.Request, entry.name, entry.modTime, bytes.NewReader(entry.data))
}

func serveDisk(c *gin.Context, cachePath, name string, modTime time.Time, cacheHeader string) bool {
	file, err := os.Open(cachePath)
	if err != nil {
		return false
	}
	defer file.Close()
	c.Header(cacheHeader, "True")
	http.ServeContent(c.Writer, c.Request, name, modTime, file)
	return true
}

// ServeCached attempts the memory-first/disk-second lookup. It returns true
// only after writing a complete cached response; false lets the caller resolve
// and serve the source file.
func (cm *CacheManager) ServeCached(c *gin.Context, internalFlag, requestPath, sourcePath string) bool {
	if !cm.config.Cache.Enable {
		return false
	}
	key := cm.cacheKey(internalFlag, requestPath)
	if cm.memoryEnabled() {
		if entry, ok := cm.memory.get(key); ok {
			if entry.sourcePath == sourcePath {
				if time.Since(entry.validatedAt) <= cm.revalidateAfter() {
					c.Header(cm.config.CacheHeader, "True")
					serveMemory(c, entry)
					return true
				}
				metadata, _, err := sourceMetadata(sourcePath)
				if err == nil && metadata.Size == entry.size && metadata.ModTime == entry.modTime.UnixNano() {
					cm.memory.touch(key, time.Now())
					c.Header(cm.config.CacheHeader, "True")
					serveMemory(c, entry)
					return true
				}
			}
			cm.memory.remove(key)
			cm.invalidateDiskIfInvalid(key, internalFlag, requestPath, sourcePath)
		}
	}

	diskLock := cm.diskLock(key)
	diskLock.RLock()
	cachePath, metadata, ok := cm.diskEntry(internalFlag, requestPath, sourcePath)
	if !ok {
		diskLock.RUnlock()
		cm.invalidateDiskIfInvalid(key, internalFlag, requestPath, sourcePath)
		return false
	}
	cacheInfo, err := os.Stat(cachePath)
	if err != nil {
		diskLock.RUnlock()
		cm.invalidateDiskIfInvalid(key, internalFlag, requestPath, sourcePath)
		return false
	}
	if cm.memoryEnabled() && cacheInfo.Size() <= cm.memoryMaxFileBytes() {
		data, err := os.ReadFile(cachePath)
		if err == nil {
			diskLock.RUnlock()
			entry := memoryFileEntry{
				key:         key,
				sourcePath:  sourcePath,
				name:        filepath.Base(sourcePath),
				data:        data,
				size:        int64(len(data)),
				modTime:     time.Unix(0, metadata.ModTime),
				validatedAt: time.Now(),
			}
			cm.memory.put(entry)
			c.Header(cm.config.CacheHeader, "True")
			serveMemory(c, entry)
			return true
		}
	}
	// Large files remain streamed from disk instead of being copied into the
	// process heap. ServeContent keeps Range, HEAD, and conditional semantics
	// aligned with the memory path while using the source file's mtime.
	served := serveDisk(c, cachePath, filepath.Base(sourcePath), time.Unix(0, metadata.ModTime), cm.config.CacheHeader)
	diskLock.RUnlock()
	return served
}

// CacheFile reads and stores a source file in the disk cache and, when allowed,
// in the memory LRU. Concurrent misses for the same key are coalesced.
func (cm *CacheManager) CacheFile(internalFlag, requestPath, originalFilePath string) {
	if !cm.config.Cache.Enable || !cm.ShouldCache(requestPath, originalFilePath) {
		return
	}
	key := cm.cacheKey(internalFlag, requestPath)
	_, _, _ = cm.group.Do(key, func() (interface{}, error) {
		metadata, _, err := sourceMetadata(originalFilePath)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(originalFilePath)
		if err != nil {
			cm.logger.Error().Err(err).Msg("failed to read original file for caching")
			return nil, err
		}
		cachePath := cm.cachedFilePath(internalFlag, requestPath)
		diskLock := cm.diskLock(key)
		diskLock.Lock()
		writeErr := writeAtomicCache(cachePath, metadata, data)
		diskLock.Unlock()
		if writeErr != nil {
			cm.logger.Error().Err(writeErr).Msg("failed to write cached file")
		}
		if cm.memoryEnabled() && int64(len(data)) <= cm.memoryMaxFileBytes() {
			cm.memory.put(memoryFileEntry{
				key:         key,
				sourcePath:  originalFilePath,
				name:        filepath.Base(originalFilePath),
				data:        data,
				size:        int64(len(data)),
				modTime:     time.Unix(0, metadata.ModTime),
				validatedAt: time.Now(),
			})
		}
		return nil, nil
	})
}

func writeAtomicCache(cachePath string, metadata cacheMetadata, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}
	dataTemp, err := os.CreateTemp(filepath.Dir(cachePath), ".cache-data-*")
	if err != nil {
		return err
	}
	dataTempName := dataTemp.Name()
	defer os.Remove(dataTempName)
	if _, err := dataTemp.Write(data); err != nil {
		_ = dataTemp.Close()
		return err
	}
	if err := dataTemp.Close(); err != nil {
		return err
	}
	if err := renameReplace(dataTempName, cachePath); err != nil {
		return err
	}

	metadataData, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	metaPath := metadataPath(cachePath)
	metadataTemp, err := os.CreateTemp(filepath.Dir(metaPath), ".cache-meta-*")
	if err != nil {
		return err
	}
	metadataTempName := metadataTemp.Name()
	defer os.Remove(metadataTempName)
	if _, err := metadataTemp.Write(metadataData); err != nil {
		_ = metadataTemp.Close()
		return err
	}
	if err := metadataTemp.Close(); err != nil {
		return err
	}
	if err := renameReplace(metadataTempName, metaPath); err != nil {
		return err
	}
	return nil
}

func renameReplace(source, target string) error {
	// Use one replacement attempt and never remove the target first. The
	// process-local disk lock prevents readers from observing the data/metadata
	// commit while os.Rename avoids the previous explicit missing-file window.
	return os.Rename(source, target)
}

// GetCachedFile is retained for callers that only need the legacy disk path.
func (cm *CacheManager) GetCachedFile(internalFlag, requestPath string) string {
	if !cm.config.Cache.Enable {
		return ""
	}
	key := cm.cacheKey(internalFlag, requestPath)
	diskLock := cm.diskLock(key)
	diskLock.Lock()
	defer diskLock.Unlock()
	cachePath := cm.cachedFilePath(internalFlag, requestPath)
	info, err := os.Stat(cachePath)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	if cm.diskExpired(info) {
		cm.removeDiskFiles(internalFlag, requestPath)
		return ""
	}
	return cachePath
}
