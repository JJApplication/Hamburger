package core

import (
	"Hamburger/gateway/grpc_proxy"
	"Hamburger/internal/config"
	"Hamburger/internal/constant"
	"Hamburger/internal/utils"
	"bytes"
	"net/http"
	"time"
)

type myTransport struct {
	conf      *config.Config
	Transport http.RoundTripper
}

func (t *myTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL == nil {
		return t.Transport.RoundTrip(req)
	}
	// 检查是否为gRPC代理请求
	if req.URL.Scheme == constant.SchemeGrpc {
		return t.handleGrpcProxy(req)
	}

	if t.conf.Debug {
		start := time.Now()
		resp, err := t.Transport.RoundTrip(req)

		if t.conf.Debug {
			utils.PerformCalc("round-trip", start)
		}

		return resp, err
	}

	return t.Transport.RoundTrip(req)
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
