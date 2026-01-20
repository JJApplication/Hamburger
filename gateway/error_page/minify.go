package error_page

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"bytes"
	"compress/gzip"
	"net/http"
	"strings"
)

// 压缩html文件
func minify(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Encoding", "gzip")
	gzw := gzip.NewWriter(w)
	defer func() {
		if e := gzw.Close(); e != nil {
			logger.GetLogger().Error().Err(e).Msg("gzip write error")
		}
	}()
	_, _ = gzw.Write(b)
}

func useGzip(request *http.Request) bool {
	cf := config.Get()
	if !cf.Features.Gzip.Enabled {
		return false
	}
	accept := request.Header.Get("Accept-Encoding")
	return strings.Contains(accept, "gzip")
}

// 压缩数据到字节数组
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write(data)
	if err != nil {
		return nil, err
	}
	err = gzw.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// 获取对应的gzip缓存数据
func getGzipCache(t int) []byte {
	// 通过比较字节数组来确定是哪个页面
	if t == Forbidden {
		return ForbiddenPageGzip
	}
	if t == Unavailable {
		return UnavailablePageGzip
	}
	return nil
}
