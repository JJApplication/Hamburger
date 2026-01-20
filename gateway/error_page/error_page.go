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
	"net/http"
	"strings"
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

var CodeMap = map[int][]byte{
	Forbidden:   ForbiddenPage,
	Unavailable: UnavailablePage,
	Other:       []byte(serror.ERRORSendProxy),
}

//go:inline
func Cache(code int, w http.ResponseWriter, r *http.Request, resType int) {
	cf := config.Get()
	if cf.Security.StrictMode || !acceptHTML(r) {
		strictWrite(code, w)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	switch resType {
	case Forbidden:
		writeResponse(w, r, Forbidden)
		return
	case Unavailable:
		writeResponse(w, r, Unavailable)
		return
	default:
		writeResponse(w, r, Other)
		return
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
