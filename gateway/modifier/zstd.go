package modifier

import (
	"Hamburger/internal/config/loader"
	"Hamburger/internal/logger"
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

type ZstdModifier struct {
	enabled     bool
	level       int
	types       []string
	threshold   int
	encoderPool sync.Pool
	bufferPool  sync.Pool
}

func NewZstdModifier() *ZstdModifier {
	cfg := loader.Get()
	zm := &ZstdModifier{
		enabled:   cfg.Middleware.Zstd.Enabled,
		level:     cfg.Middleware.Zstd.Level,
		types:     cfg.Middleware.Zstd.Types,
		threshold: cfg.Middleware.Zstd.Threshold,
	}

	zm.encoderPool = sync.Pool{
		New: func() interface{} {
			opts := make([]zstd.EOption, 0, 1)
			if zm.level != 0 {
				opts = append(opts, zstd.WithEncoderLevel(zstd.EncoderLevel(zm.level)))
			}
			enc, err := zstd.NewWriter(nil, opts...)
			if err != nil {
				enc, _ = zstd.NewWriter(nil)
			}
			return enc
		},
	}

	zm.bufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}

	return zm
}

func (z *ZstdModifier) Use(response *http.Response) bool {
	_ = z.ModifyResponse(response)
	return true
}

func (z *ZstdModifier) ModifyResponse(response *http.Response) error {
	if !z.enabled {
		return nil
	}

	best := bestCompressionEncoding(response.Request, loader.Get().Middleware.Gzip.Enabled, z.enabled)
	if best != "zstd" {
		return nil
	}

	if !z.shouldCompress(response) {
		return nil
	}

	if response.Header.Get("Content-Encoding") != "" {
		logger.GetLogger().Debug().Msg("response already compressed, skipping zstd processing")
		return nil
	}

	cl := response.Header.Get("Content-Length")
	if cl != "" {
		size, err := strconv.Atoi(cl)
		if err == nil && size <= z.threshold {
			return nil
		}
	}

	var buf bytes.Buffer
	tee := io.TeeReader(response.Body, &buf)
	originalBody, err := io.ReadAll(tee)
	if err != nil {
		logger.GetLogger().Debug().Err(err).Msg("failed to read response body")
		return err
	}

	if len(originalBody) <= z.threshold {
		response.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		return nil
	}

	compressedBody, err := z.compressData(originalBody)
	if err != nil {
		logger.GetLogger().Debug().Err(err).Msg("zstd compression failed")
		response.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		return nil
	}

	if len(compressedBody) >= len(originalBody) {
		logger.GetLogger().Debug().Msg("compressed size not reduced, using original response")
		response.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		return nil
	}

	response.Header.Set("Content-Encoding", "zstd")
	response.Header.Set("Content-Length", strconv.Itoa(len(compressedBody)))
	response.Header.Del("Content-Range")
	response.Body = io.NopCloser(bytes.NewReader(compressedBody))

	if loader.Get().Debug {
		logger.GetLogger().Debug().
			Int("original_size", len(originalBody)).
			Int("compressed_size", len(compressedBody)).
			Float64("compression_ratio", float64(len(originalBody)-len(compressedBody))/float64(len(originalBody))*100).Msg("zstd compression successful")
	}
	return nil
}

func (z *ZstdModifier) shouldCompress(response *http.Response) bool {
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		return false
	}

	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(strings.ToLower(mainType))

	for _, allowedType := range z.types {
		if strings.EqualFold(mainType, strings.TrimSpace(allowedType)) {
			return true
		}
	}

	logger.GetLogger().Debug().Str("content_type", contentType).Msg("content type not in compressible list")
	return false
}

func (z *ZstdModifier) compressData(data []byte) ([]byte, error) {
	buf := z.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		z.bufferPool.Put(buf)
	}()

	enc := z.encoderPool.Get().(*zstd.Encoder)
	defer func() {
		enc.Reset(nil)
		z.encoderPool.Put(enc)
	}()

	enc.Reset(buf)
	_, err := enc.Write(data)
	if err != nil {
		return nil, err
	}
	err = enc.Close()
	if err != nil {
		return nil, err
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

func (z *ZstdModifier) IsEnabled() bool {
	return z.enabled
}

func (z *ZstdModifier) GetLevel() int {
	return z.level
}

func (z *ZstdModifier) GetTypes() []string {
	return z.types
}

func (z *ZstdModifier) UpdateConfig() {
	cfg := loader.Get()
	oldLevel := z.level
	z.enabled = cfg.Middleware.Zstd.Enabled
	z.level = cfg.Middleware.Zstd.Level
	z.types = cfg.Middleware.Zstd.Types
	z.threshold = cfg.Middleware.Zstd.Threshold

	if oldLevel != z.level {
		z.encoderPool = sync.Pool{
			New: func() interface{} {
				opts := make([]zstd.EOption, 0, 1)
				if z.level != 0 {
					opts = append(opts, zstd.WithEncoderLevel(zstd.EncoderLevel(z.level)))
				}
				enc, err := zstd.NewWriter(nil, opts...)
				if err != nil {
					enc, _ = zstd.NewWriter(nil)
				}
				return enc
			},
		}
		logger.GetLogger().Debug().Int("level", z.level).Msg("zstd compression level updated, encoder pool re-initialized")
	}

	logger.GetLogger().Debug().Bool("enable", z.enabled).Int("level", z.level).Any("types", z.types).Msg("zstd configuration updated")
}

func (z *ZstdModifier) GetName() string {
	return "zstd"
}

func acceptEncodingQ(acceptEncoding string, encoding string) (float64, bool) {
	encoding = strings.ToLower(strings.TrimSpace(encoding))
	if acceptEncoding == "" || encoding == "" {
		return 0, false
	}

	for _, part := range strings.Split(acceptEncoding, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segments := strings.Split(part, ";")
		token := strings.ToLower(strings.TrimSpace(segments[0]))
		if token != encoding {
			continue
		}
		q := 1.0
		for _, seg := range segments[1:] {
			seg = strings.TrimSpace(seg)
			if !strings.HasPrefix(seg, "q=") {
				continue
			}
			if v, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(seg, "q=")), 64); err == nil {
				q = v
			}
		}
		return q, true
	}

	return 0, false
}

func bestCompressionEncoding(request *http.Request, enableGzip bool, enableZstd bool) string {
	if request == nil {
		return ""
	}
	ae := request.Header.Get("Accept-Encoding")
	if ae == "" {
		return ""
	}

	qStar, okStar := acceptEncodingQ(ae, "*")
	qGzip, okGzip := acceptEncodingQ(ae, "gzip")
	qZstd, okZstd := acceptEncodingQ(ae, "zstd")

	if !okGzip && okStar {
		qGzip, okGzip = qStar, true
	}
	if !okZstd && okStar {
		qZstd, okZstd = qStar, true
	}

	if !enableGzip {
		qGzip, okGzip = 0, true
	}
	if !enableZstd {
		qZstd, okZstd = 0, true
	}

	if !okGzip && !okZstd {
		return ""
	}

	if qZstd <= 0 && qGzip <= 0 {
		return ""
	}

	if qZstd > qGzip {
		return "zstd"
	}
	if qGzip > qZstd {
		return "gzip"
	}

	if qZstd > 0 {
		return "zstd"
	}
	if qGzip > 0 {
		return "gzip"
	}
	return ""
}
