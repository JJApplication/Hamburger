package grpc_web

import (
	"Hamburger/internal/config/core_config"
	"Hamburger/internal/logger"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

type GrpcWebProxy struct {
	config    *core_config.GrpcProxyConfig
	h2cClient *http.Client
	tlsClient *http.Client
}

type backendConnMode int

const (
	backendConnModeH2C backendConnMode = iota
	backendConnModeTLS
	backendConnModeUDS
)

func NewGrpcWebProxy(cfg *core_config.GrpcProxyConfig) *GrpcWebProxy {
	h2cTransport := &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			return dialer.Dial(network, addr)
		},
	}
	return &GrpcWebProxy{
		config: cfg,
		h2cClient: &http.Client{
			Transport: h2cTransport,
			Timeout:   60 * time.Second,
		},
		tlsClient: &http.Client{
			Transport: &http2.Transport{},
			Timeout:   60 * time.Second,
		},
	}
}

func (p *GrpcWebProxy) IsGrpcWebRequest(r *http.Request) bool {
	if !p.config.Enabled {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.HasPrefix(contentType, "application/grpc-web") {
		return true
	}
	return strings.TrimSpace(r.Header.Get("X-Grpc-Web")) == "1"
}

func (p *GrpcWebProxy) ValidateGrpcAddr(addr string) bool {
	if len(p.config.Hosts) == 0 {
		return false
	}
	for _, host := range p.config.Hosts {
		if addr == host || strings.HasPrefix(addr, host+":") {
			return true
		}
	}
	return false
}

func (p *GrpcWebProxy) HandleGrpcWebRequest(w http.ResponseWriter, r *http.Request) {
	grpcAddr := strings.TrimSpace(r.Header.Get(p.config.GrpcHeader))
	if grpcAddr == "" {
		p.writeErrorResponse(w, "missing gRPC address header", http.StatusBadRequest)
		return
	}
	if !p.ValidateGrpcAddr(grpcAddr) {
		logger.GetLogger().Warn().Str("address", grpcAddr).Msg("grpc-web address not in whitelist")
		p.writeErrorResponse(w, "gRPC address not allowed", http.StatusForbidden)
		return
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	isTextMode := strings.Contains(contentType, "grpc-web-text")
	resp, err := p.forward(grpcAddr, r, isTextMode)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("grpc-web proxy failed")
		p.writeErrorResponse(w, fmt.Sprintf("grpc-web proxy failed: %v", err), http.StatusBadGateway)
		return
	}
	for k, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(resp.Body)
}

type proxyResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func (p *GrpcWebProxy) forward(grpcAddr string, r *http.Request, textMode bool) (*proxyResponse, error) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if textMode {
		raw := make([]byte, base64.StdEncoding.DecodedLen(len(payload)))
		n, decodeErr := base64.StdEncoding.Decode(raw, bytes.TrimSpace(payload))
		if decodeErr != nil {
			return nil, fmt.Errorf("decode grpc-web-text body failed: %w", decodeErr)
		}
		payload = raw[:n]
	}

	targetURL, mode, udsPath, err := buildTargetURL(grpcAddr, r.URL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if mode == backendConnModeUDS {
		req.Host = "localhost"
	}
	copyGrpcHeaders(r.Header, req.Header)
	req.Header.Set("Content-Type", "application/grpc")
	req.Header.Set("TE", "trailers")
	req.Header.Set("Accept-Encoding", "identity")

	client := p.h2cClient
	if mode == backendConnModeTLS {
		client = p.tlsClient
	}
	if mode == backendConnModeUDS {
		client = newUDSClient(udsPath)
	}
	backendResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer backendResp.Body.Close()

	respBody, err := io.ReadAll(backendResp.Body)
	if err != nil {
		return nil, err
	}
	trailerFrame := encodeTrailerFrame(backendResp.Trailer)
	result := append(respBody, trailerFrame...)
	if textMode {
		encoded := base64.StdEncoding.EncodeToString(result)
		result = []byte(encoded)
	}
	headers := make(http.Header)
	headers.Set("Access-Control-Allow-Origin", "*")
	headers.Set("Access-Control-Expose-Headers", "grpc-status,grpc-message,grpc-status-details-bin")
	headers.Set("Vary", "Origin")
	if textMode {
		headers.Set("Content-Type", "application/grpc-web-text+proto")
	} else {
		headers.Set("Content-Type", "application/grpc-web+proto")
	}
	if gs := backendResp.Trailer.Get("grpc-status"); gs != "" {
		headers.Set("Grpc-Status", gs)
	}
	if gm := backendResp.Trailer.Get("grpc-message"); gm != "" {
		headers.Set("Grpc-Message", gm)
	}
	return &proxyResponse{
		StatusCode: http.StatusOK,
		Header:     headers,
		Body:       result,
	}, nil
}

func buildTargetURL(grpcAddr string, reqURL *url.URL) (string, backendConnMode, string, error) {
	raw := strings.TrimSpace(grpcAddr)
	if raw == "" {
		return "", backendConnModeH2C, "", fmt.Errorf("empty grpc address")
	}
	mode := backendConnModeH2C
	base := raw
	udsPath := ""
	if strings.HasPrefix(raw, "unix://") {
		udsPath = strings.TrimSpace(strings.TrimPrefix(raw, "unix://"))
		if udsPath == "" {
			return "", backendConnModeH2C, "", fmt.Errorf("empty uds path")
		}
		mode = backendConnModeUDS
		base = "localhost"
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", backendConnModeH2C, "", err
		}
		base = parsed.Host
		if parsed.Scheme == "https" {
			mode = backendConnModeTLS
		}
	}
	scheme := "http"
	if mode == backendConnModeTLS {
		scheme = "https"
	}
	path := "/"
	if reqURL != nil && reqURL.Path != "" {
		path = reqURL.Path
	}
	query := ""
	if reqURL != nil {
		query = reqURL.RawQuery
	}
	target := fmt.Sprintf("%s://%s%s", scheme, base, path)
	if query != "" {
		target += "?" + query
	}
	return target, mode, udsPath, nil
}

func newUDSClient(socketPath string) *http.Client {
	transport := &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			return dialer.Dial("unix", socketPath)
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}

func copyGrpcHeaders(src http.Header, dst http.Header) {
	for k, values := range src {
		lk := strings.ToLower(k)
		if lk == "content-type" || lk == "content-length" || lk == "host" || lk == "x-grpc-web" || lk == "origin" {
			continue
		}
		if strings.HasPrefix(lk, "grpc-") || strings.HasPrefix(lk, "x-") || lk == "authorization" {
			for _, v := range values {
				dst.Add(k, v)
			}
		}
	}
}

func encodeTrailerFrame(trailer http.Header) []byte {
	if len(trailer) == 0 {
		return nil
	}
	var b strings.Builder
	for k, values := range trailer {
		lk := strings.ToLower(strings.TrimSpace(k))
		for _, v := range values {
			b.WriteString(lk)
			b.WriteString(": ")
			b.WriteString(strings.TrimSpace(v))
			b.WriteString("\r\n")
		}
	}
	payload := []byte(b.String())
	frame := make([]byte, 5+len(payload))
	frame[0] = 0x80
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)
	return frame
}

func (p *GrpcWebProxy) writeErrorResponse(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(message))
}
