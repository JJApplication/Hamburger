/*
Create: 2022/7/31
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

package error_page

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"Hamburger/internal/serror"
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"strings"
	"sync"
)

// 静态文件的缓存
// 缓存headers支持gzip压缩

// gzip压缩后的静态页面缓存
var (
	ForbiddenPageGzip   []byte
	UnavailablePageGzip []byte
)

const (
	Forbidden = iota
	Unavailable
	Other
)

const (
	ErrorModeText         = "text"
	ErrorModeJSON         = "json"
	ErrorModeHTML         = "html"          // 自定义页面
	ErrorModeInternalHTML = "internal_html" // 内置页面
)

var CodeMap = map[int][]byte{
	Forbidden:   ForbiddenPage,
	Unavailable: UnavailablePage,
	Other:       []byte(serror.ERRORSendProxy),
}

type ErrorPageManager struct {
	cfg                *config.Config
	logger             *zerolog.Logger
	lo                 sync.RWMutex
	Mode               string
	ErrorPage          map[int]string
	ErrorPageCache     map[int][]byte // 自定义页面缓存
	ErrorPageCacheGzip map[int][]byte // 自定义页面 带gzip的缓存
	EnablePageCache    bool           // 是否开启gzip缓存 默认为byte流缓存
}

var EPM *ErrorPageManager

func InitErrorPageManager(cfg *config.Config, logger *zerolog.Logger) {
	EPM = &ErrorPageManager{
		cfg:                cfg,
		logger:             logger,
		Mode:               cfg.ErrorConfig.ErrorMode,
		ErrorPage:          cfg.ErrorConfig.ErrorPage,
		ErrorPageCache:     make(map[int][]byte),
		ErrorPageCacheGzip: make(map[int][]byte),
		EnablePageCache:    cfg.ErrorConfig.EnablePageCache,
	}
	EPM.Init()
}

func (em *ErrorPageManager) Init() {
	switch em.Mode {
	case ErrorModeHTML:
		for code, file := range em.ErrorPage {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			em.lo.Lock()
			em.ErrorPageCache[code] = data
			em.lo.Unlock()
			if em.EnablePageCache {
				em.lo.Lock()
				gzipData, err := compressData(data)
				if err == nil {
					em.ErrorPageCacheGzip[code] = gzipData
				}
				em.lo.Unlock()
			}
		}
	case ErrorModeInternalHTML:
		InitErrorPageCache()
	}
}

func (em *ErrorPageManager) Response(code int, w http.ResponseWriter, r *http.Request) {
	switch em.Mode {
	case ErrorModeText:
		em.textWriter(code, w, r)
		return
	case ErrorModeJSON:
		em.jsonWriter(code, w, r)
		return
	case ErrorModeHTML:
		em.htmlWriter(code, w, r)
		return
	case ErrorModeInternalHTML:
		em.internalHtmlWriter(code, w, r)
		return
	default:
		em.textWriter(code, w, r)
	}
}

//go:inline
func (em *ErrorPageManager) textWriter(code int, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(code)
	w.Write([]byte(serror.ERRORSendProxy))
}

//go:inline
func (em *ErrorPageManager) jsonWriter(code int, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(serror.ErrorProxyErrorJSON))
}

//go:inline
func (em *ErrorPageManager) htmlWriter(code int, w http.ResponseWriter, r *http.Request) {
	if em.EnablePageCache {
		if data, ok := em.ErrorPageCacheGzip[code]; ok {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(data)
			return
		} else {
			// 没有缓存时压缩后计入缓存
			if data, ok = em.ErrorPageCache[code]; ok {
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				minify(w, data)
				em.lo.Lock()
				defer em.lo.Unlock()
				if gzipData, err := compressData(data); err == nil {
					em.ErrorPageCacheGzip[code] = gzipData
				}
			} else {
				// 未映射的页面统一使用默认502对应的页面 如果不存在再降级到text
				if data, ok = em.ErrorPageCache[http.StatusBadGateway]; ok {
					w.WriteHeader(http.StatusOK)
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					minify(w, data)
					em.lo.Lock()
					defer em.lo.Unlock()
					if gzipData, err := compressData(data); err == nil {
						em.ErrorPageCacheGzip[code] = gzipData
					}
					return
				}
				em.textWriter(code, w, r)
				return
			}
		}
		em.textWriter(code, w, r)
	} else {
		if data, ok := em.ErrorPageCache[code]; ok {
			w.WriteHeader(code)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
		// 未映射的页面统一使用默认502对应的页面 如果不存在再降级到text
		if data, ok := em.ErrorPageCache[http.StatusBadGateway]; ok {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
		em.textWriter(code, w, r)
	}
}

//go:inline
func (em *ErrorPageManager) internalHtmlWriter(code int, w http.ResponseWriter, r *http.Request) {
	// 严格模式直接按响应码返回
	if em.cfg.Security.StrictMode {
		strictWrite(code, w)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if code == http.StatusInternalServerError {
		writeResponse(w, r, Unavailable)
	} else {
		writeResponse(w, r, Forbidden)
	}
}

func acceptHTML(r *http.Request) bool {
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		return true
	}
	return false
}

func strictWrite(code int, w http.ResponseWriter) {
	w.WriteHeader(code)
	w.Write([]byte(serror.ERRORSendProxy))
}

func writeResponse(w http.ResponseWriter, request *http.Request, t int) {
	if useGzip(request) {
		// 检查是否有预压缩的缓存
		if gzipData := getGzipCache(t); gzipData != nil {
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(gzipData)
		} else {
			minify(w, CodeMap[t])
		}
	} else {
		w.Write(CodeMap[t])
	}
}

// InitErrorPageCache 初始化gzip缓存
func InitErrorPageCache() {
	var err error
	ForbiddenPageGzip, err = compressData(ForbiddenPage)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("compress ForbiddenPage error")
	}
	UnavailablePageGzip, err = compressData(UnavailablePage)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("compress UnavailablePage error")
	}
	logger.GetLogger().Info().Msg("gzip cache initialized")
}
