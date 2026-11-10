package core

import (
	"Hamburger/internal/config/loader"
	"Hamburger/internal/logger"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"
)

// defaultUpstreamIdleTimeout 是与后端之间连续无数据的最长时间。
// 用空闲超时而不是整次请求的总时长，大文件上传/下载只要持续有数据就不会被切断。
const defaultUpstreamIdleTimeout = 30 * time.Second

// streamBodyThreshold 让带 Content-Length 的响应走流式读取。
// fasthttp 只有在正文超过 MaxResponseBodySize 时才会停止整包缓冲。
const streamBodyThreshold = 1

// FastRoundTripper 使用 fasthttp 作为下游传输。
// 响应头到达后立即把正文流转给反向代理，而不是先读完整段 body。
type FastRoundTripper struct {
	Client      *fasthttp.Client
	idleTimeout time.Duration
}

// NewFastRoundTripper 创建一个 FastRoundTripper。
func NewFastRoundTripper() *FastRoundTripper {
	return newFastRoundTripper(defaultUpstreamIdleTimeout)
}

func newFastRoundTripper(idle time.Duration) *FastRoundTripper {
	rt := &FastRoundTripper{idleTimeout: idle}
	rt.Client = &fasthttp.Client{
		MaxConnsPerHost:     100,
		MaxIdleConnDuration: 90 * time.Second,
		MaxResponseBodySize: streamBodyThreshold,
		StreamResponseBody:  true,
		Dial:                rt.dial,
	}
	return rt
}

// RoundTrip 实现 http.RoundTripper。
func (f *FastRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}

	var fr fasthttp.Request
	defer fr.Reset()
	resp := fasthttp.AcquireResponse()

	fr.Header.SetMethod(req.Method)
	f.writeRequestURI(&fr, req)
	f.copyRequestHeaders(&fr, req)

	logger.L().Debug().Str("Host", string(fr.Host())).Msg("fasthttp client")

	if req.Body != nil && req.Body != http.NoBody {
		if req.ContentLength > 0 {
			fr.SetBodyStream(req.Body, int(req.ContentLength))
		} else {
			b, err := io.ReadAll(req.Body)
			if err != nil {
				fasthttp.ReleaseResponse(resp)
				return nil, err
			}
			fr.SetBodyRaw(b)
		}
	}

	if req.URL != nil && req.URL.Scheme == "https" {
		fr.Header.Set("X-Forwarded-Proto", "https")
	}
	fr.DisableRedirectPathNormalizing = true

	owner := &dialOwner{ctx: req.Context()}
	gid := currentGoroutineID()
	dialOwners.Store(gid, owner)
	defer dialOwners.Delete(gid)

	stopWatch := make(chan struct{})
	go func() {
		select {
		case <-req.Context().Done():
			owner.closeAll()
		case <-stopWatch:
		}
	}()
	defer close(stopWatch)

	if err := f.Client.Do(&fr, resp); err != nil {
		fasthttp.ReleaseResponse(resp)
		if ctxErr := req.Context().Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}

	dst := &http.Response{
		StatusCode: resp.StatusCode(),
		Status:     http.StatusText(resp.StatusCode()),
		Header:     make(http.Header),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Request:    req,
	}
	copyResponseHeaders(dst, resp)

	stream := resp.BodyStream()
	if stream == nil {
		fasthttp.ReleaseResponse(resp)
		dst.Body = http.NoBody
		return dst, nil
	}

	// keep-alive 复用时不会再次 Dial，按连接地址从全局表找回连接，
	// 取消下载时才能直接关掉套接字。
	conn := lookupTrackedConn(resp.LocalAddr(), resp.RemoteAddr())
	if conn == nil {
		conn = owner.latest()
	}
	dst.Body = newFastBody(req.Context(), resp, stream, conn)
	return dst, nil
}

func (f *FastRoundTripper) writeRequestURI(fr *fasthttp.Request, req *http.Request) {
	uri := fr.URI()
	if req.URL == nil {
		return
	}
	if req.URL.Opaque != "" {
		fr.SetRequestURI(req.URL.Opaque)
	} else {
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
	if req.URL.Host != "" {
		uri.SetHost(req.URL.Host)
		fr.SetHost(req.URL.Host)
		fr.Header.SetHost(req.URL.Host)
		if cfg := loader.Get(); cfg != nil {
			forwardHost := cfg.ProxyHeader.ForwardHostHeader
			if forwardHost != "" && fr.Header.Peek(forwardHost) == nil {
				fr.Header.Set(forwardHost, req.Host)
			}
		}
	}
}

func (f *FastRoundTripper) copyRequestHeaders(fr *fasthttp.Request, req *http.Request) {
	for k, vv := range req.Header {
		if strings.EqualFold(k, "Host") || strings.EqualFold(k, "X-Forwarded-For") {
			continue
		}
		for _, v := range vv {
			fr.Header.Add(k, v)
		}
	}
}

func copyResponseHeaders(dst *http.Response, resp *fasthttp.Response) {
	resp.Header.VisitAll(func(k, v []byte) {
		key := string(k)
		val := string(v)
		if prev, ok := dst.Header[key]; ok && len(prev) > 0 {
			dst.Header.Set(key, prev[0]+","+val)
		} else {
			dst.Header.Set(key, val)
		}
	})

	// 流式读取已经解开 chunked，长度以 fasthttp 解析结果为准，避免头和正文不一致。
	cl := resp.Header.ContentLength()
	dst.Header.Del("Transfer-Encoding")
	if cl >= 0 {
		dst.ContentLength = int64(cl)
		dst.Header.Set("Content-Length", strconv.Itoa(cl))
		return
	}
	dst.ContentLength = -1
	dst.Header.Del("Content-Length")
}

func (f *FastRoundTripper) dial(addr string) (net.Conn, error) {
	conn, err := fasthttp.Dial(addr)
	if err != nil {
		return nil, err
	}
	tracked := newTrackedConn(conn, f.idleTimeout)
	if v, ok := dialOwners.Load(currentGoroutineID()); ok {
		owner := v.(*dialOwner)
		if err := owner.ctx.Err(); err != nil {
			_ = tracked.Close()
			return nil, err
		}
		owner.add(tracked)
	}
	return tracked, nil
}

var (
	dialOwners sync.Map // goroutine id -> *dialOwner
	liveConns  sync.Map // connKey -> *trackedConn
)

type dialOwner struct {
	ctx   context.Context
	mu    sync.Mutex
	conns []*trackedConn
}

func (o *dialOwner) add(conn *trackedConn) {
	o.mu.Lock()
	o.conns = append(o.conns, conn)
	o.mu.Unlock()
}

func (o *dialOwner) closeAll() {
	o.mu.Lock()
	conns := append([]*trackedConn(nil), o.conns...)
	o.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Close()
	}
}

func (o *dialOwner) latest() *trackedConn {
	o.mu.Lock()
	defer o.mu.Unlock()
	if n := len(o.conns); n > 0 {
		return o.conns[n-1]
	}
	return nil
}

type trackedConn struct {
	net.Conn
	localKey string
	idle     time.Duration
	closed   atomic.Bool
}

func newTrackedConn(conn net.Conn, idle time.Duration) *trackedConn {
	tc := &trackedConn{Conn: conn, idle: idle}
	if key := connKey(conn.LocalAddr(), conn.RemoteAddr()); key != "" {
		tc.localKey = key
		liveConns.Store(key, tc)
	}
	return tc
}

// connKey 同时带上远端地址：同一个本地端口可以连向不同上游。
func connKey(local, remote net.Addr) string {
	if local == nil || remote == nil {
		return ""
	}
	return local.String() + "|" + remote.String()
}

func lookupTrackedConn(local, remote net.Addr) *trackedConn {
	key := connKey(local, remote)
	if key == "" {
		return nil
	}
	if v, ok := liveConns.Load(key); ok {
		return v.(*trackedConn)
	}
	return nil
}

func (c *trackedConn) Read(p []byte) (int, error) {
	if c.idle > 0 {
		_ = c.Conn.SetReadDeadline(time.Now().Add(c.idle))
	}
	return c.Conn.Read(p)
}

func (c *trackedConn) Write(p []byte) (int, error) {
	if c.idle > 0 {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(c.idle))
	}
	return c.Conn.Write(p)
}

// fasthttp 设置的是整次请求的绝对截止时间，会切断慢速的大文件上传/下载。
// 截止时间统一在 Read/Write 里按空闲超时维护。
func (c *trackedConn) SetDeadline(time.Time) error      { return nil }
func (c *trackedConn) SetReadDeadline(time.Time) error  { return nil }
func (c *trackedConn) SetWriteDeadline(time.Time) error { return nil }

func (c *trackedConn) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return net.ErrClosed
	}
	if c.localKey != "" {
		liveConns.Delete(c.localKey)
	}
	return c.Conn.Close()
}

func currentGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	const prefix = "goroutine "
	stack := buf[:n]
	if !bytes.HasPrefix(stack, []byte(prefix)) {
		return 0
	}
	stack = stack[len(prefix):]
	end := bytes.IndexByte(stack, ' ')
	if end < 0 {
		return 0
	}
	id, _ := strconv.ParseUint(string(stack[:end]), 10, 64)
	return id
}

type fastBody struct {
	ctx    context.Context
	resp   *fasthttp.Response
	r      io.Reader
	conn   *trackedConn
	read   atomic.Int64
	sawEOF atomic.Bool
	stop   chan struct{}
	once   sync.Once
}

func newFastBody(ctx context.Context, resp *fasthttp.Response, stream io.Reader, conn *trackedConn) *fastBody {
	body := &fastBody{
		ctx:  ctx,
		resp: resp,
		r:    stream,
		conn: conn,
		stop: make(chan struct{}),
	}
	go func() {
		select {
		case <-ctx.Done():
			if body.conn != nil {
				_ = body.conn.Close()
			}
		case <-body.stop:
		}
	}()
	return body
}

func (b *fastBody) Read(p []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := b.r.Read(p)
	if n > 0 {
		b.read.Add(int64(n))
	}
	if err == io.EOF {
		b.sawEOF.Store(true)
		return n, io.EOF
	}
	if ctxErr := b.ctx.Err(); ctxErr != nil && err == nil {
		return n, ctxErr
	}
	return n, err
}

func (b *fastBody) Close() error {
	b.once.Do(func() {
		close(b.stop)
		// 只有真正读到 EOF 才允许连接回池。按 Content-Length 提前停读时，
		// 上游可能还有残留字节，回池会污染下一个请求。
		complete := b.sawEOF.Load()
		if b.resp != nil && (!complete || b.ctx.Err() != nil) {
			b.resp.SetConnectionClose()
			if b.conn != nil {
				_ = b.conn.Close()
			}
		}
		if b.resp != nil {
			_ = b.resp.CloseBodyStream()
			fasthttp.ReleaseResponse(b.resp)
			b.resp = nil
		}
	})
	return nil
}
