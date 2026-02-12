package core

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// FastRoundTripper 使用 fasthttp 作为下游传输
// 负责将 *http.Request 映射为 fasthttp 请求并转换响应
type FastRoundTripper struct {
	// Client fasthttp客户端实例
	Client *fasthttp.Client
}

// NewFastRoundTripper 创建一个FastRoundTripper
// 根据现有http.Transport的超时与连接配置进行参照设置
func NewFastRoundTripper() *FastRoundTripper {
	return &FastRoundTripper{
		Client: &fasthttp.Client{
			MaxConnsPerHost:     100,
			MaxIdleConnDuration: 90 * time.Second,
			ReadTimeout:         30 * time.Second,
			WriteTimeout:        30 * time.Second,
		},
	}
}

// RoundTrip 实现 http.RoundTripper，使用 fasthttp 发起请求
func (f *FastRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var fr fasthttp.Request
	var resp fasthttp.Response

	fr.Header.SetMethod(req.Method)

	// 组装目标URI
	uri := fr.URI()
	if req.URL != nil {
		// path+query
		if req.URL.Opaque != "" {
			fr.SetRequestURI(req.URL.Opaque)
		} else {
			// 请求路径
			path := req.URL.Path
			if path == "" {
				path = "/"
			}
			if req.URL.RawQuery != "" {
				fr.SetRequestURI(path + "?" + req.URL.RawQuery)
			} else {
				fr.SetRequestURI(path)
			}
		}

		if req.URL.Scheme != "" {
			uri.SetScheme(req.URL.Scheme)
		}
		// 真实场景下req.HOST为请求域名 req.URL.Host为解析后的真实地址
		// 确保 Host header 设置正确
		if req.URL.Host != "" {
			uri.SetHost(req.URL.Host)       // URI的Host
			fr.SetHost(req.URL.Host)        // fr的Host
			fr.Header.SetHost(req.URL.Host) // 请求头的Host
			forwardHost := config.Get().ProxyHeader.ForwardHostHeader
			if forwardHost != "" {
				fr.Header.Set(forwardHost, req.Host) // 为方便后端服务获取原始Host 使用X-Forward-Host
			}
		}
	}

	// 复制请求头
	for k, vv := range req.Header {
		if strings.EqualFold("X-Forwarded-For", k) || strings.EqualFold("Host", k) {
			continue
		}
		for _, v := range vv {
			fr.Header.Add(k, v)
		}
	}

	logger.L().Debug().Str("Host", string(fr.Host())).Msg("fasthttp client")

	// 复制请求体（尽量流式）
	if req.Body != nil {
		if req.ContentLength > 0 {
			fr.SetBodyStream(req.Body, int(req.ContentLength))
		} else {
			// 长度未知时读入缓冲
			b, _ := io.ReadAll(req.Body)
			fr.SetBodyRaw(b)
		}
	}

	// 避免fasthttp无限重定向 增加专属请求头
	if req.URL != nil && req.URL.Scheme == "https" {
		fr.Header.Set("X-Forwarded-Proto", "https")
	}
	fr.DisableRedirectPathNormalizing = true
	// 发送请求
	if err := f.Client.Do(&fr, &resp); err != nil {
		return nil, err
	}

	// 转换响应
	dst := &http.Response{
		StatusCode:    resp.StatusCode(),
		Status:        http.StatusText(resp.StatusCode()),
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(resp.Body())),
		ContentLength: int64(len(resp.Body())),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
	}

	// 复制响应头
	resp.Header.VisitAll(func(k, v []byte) {
		key := string(k)
		val := string(v)
		// 多值合并为逗号分隔
		if prev, ok := dst.Header[key]; ok && len(prev) > 0 {
			dst.Header.Set(key, prev[0]+","+val)
		} else {
			dst.Header.Set(key, val)
		}
	})

	fr.Reset()

	return dst, nil
}
