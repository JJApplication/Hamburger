package core

import (
	"Hamburger/gateway/grpc_proxy"
	"Hamburger/gateway/grpc_web"
	"Hamburger/internal/config"
	"Hamburger/internal/constant"
	"Hamburger/internal/utils"
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

type myTransport struct {
	conf         *config.Config
	Transport    http.RoundTripper
	h2cTransport http.RoundTripper
	h3Transport  http.RoundTripper
}

func (t *myTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL == nil {
		return t.Transport.RoundTrip(req)
	}
	// 检查是否为gRPC代理请求
	if req.URL.Scheme == constant.SchemeGrpc {
		return t.handleGrpcProxy(req)
	}
	if req.URL.Scheme == constant.SchemeGrpcWeb {
		return t.handleGrpcWebProxy(req)
	}

	transport := t.Transport
	if h3Target, ok := t.shouldUseH3(req); ok && t.h3Transport != nil {
		if req.URL != nil {
			req.URL.Scheme = "https"
			if h3Target != "" {
				req.URL.Host = h3Target
			}
		}
		transport = t.h3Transport
	} else if t.shouldUseH2C(req) && t.h2cTransport != nil {
		transport = t.h2cTransport
	}

	if t.conf.Debug {
		start := time.Now()
		resp, err := transport.RoundTrip(req)

		if t.conf.Debug {
			utils.PerformCalc("round-trip", start)
		}

		return resp, err
	}

	return transport.RoundTrip(req)
}

func (t *myTransport) shouldUseH3(req *http.Request) (string, bool) {
	if t.conf == nil {
		return "", false
	}
	if !t.conf.CoreProxy.EnableHTTP3 {
		return "", false
	}
	if !t.conf.PxyFrontend.ExpFastConnect.Http3.Enabled {
		return "", false
	}
	if req.URL == nil || req.URL.Host == "" {
		return "", false
	}
	defaultTarget := fmt.Sprintf("%s:%d", t.conf.PxyFrontend.Host, t.conf.PxyFrontend.Port)
	h3Target := t.frontHttp3Target()
	if h3Target == "" {
		return "", false
	}
	if req.URL.Host != defaultTarget && req.URL.Host != h3Target {
		return "", false
	}
	return h3Target, true
}

func (t *myTransport) shouldUseH2C(req *http.Request) bool {
	if t.conf == nil {
		return false
	}
	if !t.conf.PxyFrontend.ExpFastConnect.Enabled {
		return false
	}
	if req.URL == nil || req.URL.Scheme != "http" || req.URL.Host == "" {
		return false
	}
	target := fmt.Sprintf("%s:%d", t.conf.PxyFrontend.Host, t.conf.PxyFrontend.Port)
	return req.URL.Host == target
}

func (t *myTransport) frontHttp3Target() string {
	host := t.conf.PxyFrontend.ExpFastConnect.Http3.Host
	if host == "" {
		host = t.conf.PxyFrontend.Host
	}
	port := t.conf.PxyFrontend.ExpFastConnect.Http3.Port
	if port == 0 {
		port = t.conf.PxyFrontend.Port
	}
	if host == "" || port == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func newH2CTransport(cfg *config.Config) http.RoundTripper {
	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
	}
	if cfg != nil && cfg.PxyFrontend.ExpFastConnect.Http2.KeepAlive > 0 {
		dialer.KeepAlive = time.Duration(cfg.PxyFrontend.ExpFastConnect.Http2.KeepAlive) * time.Second
	}
	return &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
	}
}

func newH3Transport(cfg *config.Config) http.RoundTripper {
	h3cfg := cfg.PxyFrontend.ExpFastConnect.Http3
	host := h3cfg.Host
	if host == "" {
		host = cfg.PxyFrontend.Host
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: h3cfg.InsecureSkipVerify,
	}
	if host != "" && net.ParseIP(host) == nil {
		tlsConfig.ServerName = host
	}
	quicConfig := &quic.Config{}
	if h3cfg.MaxConnections > 0 {
		quicConfig.MaxIncomingStreams = int64(h3cfg.MaxConnections)
	}
	if h3cfg.IdleTimeout > 0 {
		quicConfig.MaxIdleTimeout = time.Duration(h3cfg.IdleTimeout) * time.Second
	}
	if h3cfg.KeepAlive > 0 {
		quicConfig.KeepAlivePeriod = time.Duration(h3cfg.KeepAlive) * time.Second
	}
	return &http3.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig:      quicConfig,
	}
}

// handleGrpcProxy 处理gRPC代理请求
func (t *myTransport) handleGrpcProxy(req *http.Request) (*http.Response, error) {
	proxy := grpc_proxy.GetGrpcProxy()
	if proxy == nil {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Status:     "503 Service Unavailable",
			Header:     make(http.Header),
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	// 创建一个ResponseWriter来捕获gRPC代理的响应
	recorder := &responseRecorder{
		header: make(http.Header),
		body:   &bytes.Buffer{},
	}

	// 处理gRPC请求
	proxy.HandleGrpcRequest(recorder, req)

	// 构造HTTP响应
	resp := &http.Response{
		StatusCode:    recorder.statusCode,
		Status:        http.StatusText(recorder.statusCode),
		Header:        recorder.header,
		Body:          &bodyReader{bytes.NewReader(recorder.body.Bytes())},
		ContentLength: int64(recorder.body.Len()),
		Request:       req,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
	}

	return resp, nil
}

func (t *myTransport) handleGrpcWebProxy(req *http.Request) (*http.Response, error) {
	proxy := grpc_web.GetGrpcWebProxy()
	if proxy == nil {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Status:     "503 Service Unavailable",
			Header:     make(http.Header),
			Body:       http.NoBody,
			Request:    req,
		}, nil
	}

	recorder := &responseRecorder{
		header: make(http.Header),
		body:   &bytes.Buffer{},
	}

	proxy.HandleGrpcWebRequest(recorder, req)

	resp := &http.Response{
		StatusCode:    recorder.statusCode,
		Status:        http.StatusText(recorder.statusCode),
		Header:        recorder.header,
		Body:          &bodyReader{bytes.NewReader(recorder.body.Bytes())},
		ContentLength: int64(recorder.body.Len()),
		Request:       req,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
	}

	return resp, nil
}

// responseRecorder 用于捕获gRPC代理的响应
type responseRecorder struct {
	header     http.Header
	body       *bytes.Buffer
	statusCode int
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	return r.body.Write(data)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

// bodyReader 实现io.ReadCloser接口
type bodyReader struct {
	*bytes.Reader
}

func (b *bodyReader) Close() error {
	return nil
}
