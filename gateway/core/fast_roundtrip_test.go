package core

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestFastRoundTripStreamsBeforeBodyCompletes(t *testing.T) {
	payload := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 5000)
	const head = 1024
	released := make(chan struct{})
	headersOut := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		if _, err := w.Write(payload[:head]); err != nil {
			return
		}
		w.(http.Flusher).Flush()
		close(headersOut)
		<-released
		_, _ = w.Write(payload[head:])
	}))
	defer srv.Close()
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(released) }) }
	defer release()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/app.apk", nil)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewFastRoundTripper()
	var resp *http.Response
	done := make(chan struct{})
	go func() {
		resp, err = rt.RoundTrip(req)
		close(done)
	}()

	select {
	case <-headersOut:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not send response headers")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("RoundTrip blocked until the full body was buffered")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.ContentLength != int64(len(payload)) {
		t.Fatalf("ContentLength = %d, want %d", resp.ContentLength, len(payload))
	}
	if resp.Header.Get("Content-Length") != strconv.Itoa(len(payload)) {
		t.Fatalf("Content-Length header = %q", resp.Header.Get("Content-Length"))
	}

	gotHead := make([]byte, head)
	if _, err := io.ReadFull(resp.Body, gotHead); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotHead, payload[:head]) {
		t.Fatal("first bytes were not forwarded before the rest of the body")
	}
	release()

	rest, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(append(gotHead, rest...), payload) {
		t.Fatalf("body length = %d, want %d", head+len(rest), len(payload))
	}
}

func TestFastRoundTripCancelUnblocksRead(t *testing.T) {
	headersOut := make(chan struct{})
	hold := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(1<<20))
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(headersOut)
		<-hold
	}))
	defer srv.Close()
	defer close(hold)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/app.apk", nil)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewFastRoundTripper()
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	select {
	case <-headersOut:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not send response headers")
	}

	readDone := make(chan error, 1)
	go func() {
		buf := make([]byte, 32)
		_, readErr := resp.Body.Read(buf)
		readDone <- readErr
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-readDone:
		if err == nil {
			t.Fatal("expected read to stop after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("read stayed blocked after cancel")
	}
}

func TestFastRoundTripKeepAliveTracksConnection(t *testing.T) {
	payload := bytes.Repeat([]byte("k"), 64)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	rt := NewFastRoundTripper()
	doOnce := func(path string) *trackedConn {
		req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		fb, ok := resp.Body.(*fastBody)
		if !ok {
			t.Fatalf("body type = %T, want *fastBody", resp.Body)
		}
		if fb.conn == nil {
			t.Fatal("stream body missing tracked connection")
		}
		conn := fb.conn
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
		return conn
	}

	first := doOnce("/a.apk")
	second := doOnce("/b.apk")
	if first.localKey == "" || second.localKey == "" {
		t.Fatal("tracked connection missing local address key")
	}
	if first.localKey != second.localKey {
		t.Fatalf("expected keep-alive reuse, got %q then %q", first.localKey, second.localKey)
	}
	if lookupTrackedConn(first.LocalAddr(), first.RemoteAddr()) == nil {
		t.Fatal("pooled connection disappeared from liveConns after complete read")
	}
}

func TestFastRoundTripPreservesPartialContent(t *testing.T) {
	body := bytes.Repeat([]byte("z"), 100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Range"); got != "bytes=0-99" {
			t.Errorf("Range = %q, want bytes=0-99", got)
		}
		w.Header().Set("Content-Type", "application/vnd.android.package-archive")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", "bytes 0-99/1000")
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/app.apk", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=0-99")

	resp, err := NewFastRoundTripper().RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206", resp.StatusCode)
	}
	if resp.Header.Get("Content-Range") != "bytes 0-99/1000" {
		t.Fatalf("Content-Range = %q", resp.Header.Get("Content-Range"))
	}
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		t.Fatalf("Accept-Ranges = %q", resp.Header.Get("Accept-Ranges"))
	}
	if resp.ContentLength != 100 || resp.Header.Get("Content-Length") != "100" {
		t.Fatalf("length header=%q field=%d", resp.Header.Get("Content-Length"), resp.ContentLength)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("body length = %d, want %d", len(got), len(body))
	}
}
