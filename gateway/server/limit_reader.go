package server

import (
	"fmt"
	"github.com/rs/zerolog"
	"io"
)

// limitedReader 用于限制请求体读取，超出限制时直接拒绝
type limitedReader struct {
	io.ReadCloser
	limit    int64
	read     int64
	logger   *zerolog.Logger
	host     string
	exceeded bool
}

func (lr *limitedReader) Read(p []byte) (n int, err error) {
	if lr.exceeded {
		return 0, fmt.Errorf("request entity too large")
	}

	n, err = lr.ReadCloser.Read(p)
	lr.read += int64(n)

	if lr.read > lr.limit {
		lr.exceeded = true
		if lr.logger != nil {
			lr.logger.Error().Str("Host", lr.host).Int64("Read", lr.read).Int64("Limit", lr.limit).Msg("请求体超出限制被拒绝")
		}
		return 0, fmt.Errorf("request entity too large")
	}

	return n, err
}
