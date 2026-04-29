/*
Package proxy_cache
代理转发缓存
记录代理规则解析后的真实Host转发缓存
*/
package proxy_cache

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/json"
	"Hamburger/internal/utils"
	"context"
	"fmt"
	"github.com/allegro/bigcache/v3"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type PxyCacheManager struct {
	cfg *config.Config
	bc  *bigcache.BigCache
}

type PxyCache struct {
	Key        string // 全局唯一的Key
	RequestURL string // 请求路径
	RealHost   string
	Host       string
	Port       int
	Schema     string
}

func (c PxyCache) ToJSON() []byte {
	data, err := json.Marshal(c)
	if err != nil {
		return nil
	}
	return data
}

func (c PxyCache) CacheToURL(req *http.Request) *url.URL {
	req.URL.Scheme = c.Schema
	req.URL.Host = fmt.Sprintf("%s:%d", c.Host, c.Port)
	req.Header.Set("Host", c.RealHost) // 设置真实HOST

	return req.URL
}

func ToCache(data []byte) PxyCache {
	var c PxyCache
	err := json.Unmarshal(data, &c)
	if err != nil {
		return PxyCache{}
	}
	return c
}

var (
	pxym *PxyCacheManager
	once sync.Once
)

func NewPxyCacheManager() *PxyCacheManager {
	cfg := loader.Get()
	if !cfg.Features.ProxyCache.Enabled {
		return &PxyCacheManager{cfg: cfg}
	}
	cache, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             64,
		LifeWindow:         24 * time.Hour, // 缓存的有效期一天
		CleanWindow:        time.Duration(utils.DefaultInt(cfg.Features.ProxyCache.CacheTTL, 300)) * time.Second,
		MaxEntriesInWindow: 256,
		MaxEntrySize:       utils.DefaultInt(cfg.Features.ProxyCache.CacheSize, 1024),
	})

	return &PxyCacheManager{
		cfg: cfg,
		bc:  cache,
	}
}

func M() *PxyCacheManager {
	once.Do(func() {
		pxym = NewPxyCacheManager()
	})
	return pxym
}

func uniqueCacheKey(req *http.Request) string {
	return fmt.Sprintf("%s+%s+%s+%s", req.Host, req.Proto, req.Method, req.URL.Path)
}

func (p *PxyCacheManager) IsEnabled() bool {
	return p.cfg.Features.ProxyCache.Enabled
}

// Set 存储rule转发解析规则 不再进行规则解析
func (p *PxyCacheManager) Set(req *http.Request, host string, port int, schema string) {
	if !p.IsEnabled() {
		return
	}
	key := uniqueCacheKey(req)
	p.bc.Set(key, PxyCache{
		Key:        key,
		RequestURL: req.RequestURI,
		RealHost:   req.Host,
		Host:       host,
		Port:       port,
		Schema:     schema,
	}.ToJSON())
}

func (p *PxyCacheManager) Get(req *http.Request) (PxyCache, bool) {
	if !p.IsEnabled() {
		return PxyCache{}, false
	}
	key := uniqueCacheKey(req)
	value, err := p.bc.Get(key)
	if err != nil {
		return PxyCache{}, false
	}
	return ToCache(value), true
}

func (p *PxyCacheManager) Del(req *http.Request) {
	if !p.IsEnabled() {
		return
	}
	key := uniqueCacheKey(req)
	p.bc.Delete(key)
}

func (p *PxyCacheManager) SetIfNotExists(req *http.Request, host string, port int, schema string) {
	if !p.IsEnabled() {
		return
	}
	key := uniqueCacheKey(req)
	_, err := p.bc.Get(key)
	if err != nil {
		p.Set(req, host, port, schema)
	}
}
