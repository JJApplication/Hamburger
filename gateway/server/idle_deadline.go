package server

import (
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// idleExtendInterval 限制顺延截止时间的频率，避免每个数据块都去改连接截止时间；
// 实际间隔不超过空闲超时的 1/4，保证顺延总能赶在截止之前。
const idleExtendInterval = time.Second

// wrapHandlerWithIdleDeadline 把 http.Server 的 ReadTimeout/WriteTimeout 从整次请求的总时长
// 改成空闲超时：请求体读到数据或响应写出数据时，读和写的截止时间一起顺延。
// 读写必须一起顺延：请求体读完后 net/http 会在后台读连接，读截止时间一到就会取消请求的 context，
// 正在进行的长下载也会被中止。
func wrapHandlerWithIdleDeadline(h http.Handler, idle time.Duration) http.Handler {
	if idle <= 0 {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isWebSocketUpgrade(r) {
			h.ServeHTTP(w, r)
			return
		}
		d := &idleDeadline{rc: http.NewResponseController(w), idle: idle}
		d.extend(true)
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = &idleBody{ReadCloser: r.Body, d: d}
		}
		h.ServeHTTP(&idleWriter{ResponseWriter: w, d: d}, r)
	})
}

// idleTimeoutOf 沿用已配置的读写超时作为空闲超时，取两者中较大的值。
func idleTimeoutOf(s *http.Server) time.Duration {
	return max(s.ReadTimeout, s.WriteTimeout)
}

type idleDeadline struct {
	rc       *http.ResponseController
	idle     time.Duration
	last     atomic.Int64
	deadline atomic.Int64
}

func (d *idleDeadline) extend(force bool) {
	now := time.Now().UnixNano()
	if !force {
		// 已经空闲超时的连接不能再被续命，否则静默的请求只要最后写一次就能绕过超时。
		if now >= d.deadline.Load() {
			return
		}
		if now-d.last.Load() < int64(min(idleExtendInterval, d.idle/4)) {
			return
		}
	}
	d.last.Store(now)
	deadline := now + int64(d.idle)
	d.deadline.Store(deadline)
	_ = d.rc.SetReadDeadline(time.Unix(0, deadline))
	_ = d.rc.SetWriteDeadline(time.Unix(0, deadline))
}

type idleBody struct {
	io.ReadCloser
	d *idleDeadline
}

func (b *idleBody) Read(p []byte) (int, error) {
	b.d.extend(false)
	return b.ReadCloser.Read(p)
}

type idleWriter struct {
	http.ResponseWriter
	d *idleDeadline
}

func (w *idleWriter) Write(p []byte) (int, error) {
	w.d.extend(false)
	return w.ResponseWriter.Write(p)
}

func (w *idleWriter) Flush() {
	w.d.extend(false)
	_ = w.d.rc.Flush()
}

func (w *idleWriter) FlushError() error {
	w.d.extend(false)
	return w.d.rc.Flush()
}

func (w *idleWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
