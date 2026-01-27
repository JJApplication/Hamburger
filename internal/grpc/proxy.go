/*
Package grpc_proxy
gRPC代理模块，实现HTTP请求到gRPC调用的转换
*/
package grpc_proxy

import (
	"Hamburger/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// GrpcRequest HTTP到gRPC转换的请求结构
type GrpcRequest struct {
	Service string                 `json:"service"` // gRPC服务名，如 "user.UserService"
	Method  string                 `json:"method"`  // gRPC方法名，如 "GetUser"
	Data    map[string]interface{} `json:"data"`    // 请求参数
	Headers map[string]string      `json:"headers"` // 额外的gRPC metadata
	Timeout int                    `json:"timeout"` // 超时时间（秒），默认30秒
}

// GrpcResponse gRPC响应结构
type GrpcResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Code    int                    `json:"code"`
	Headers map[string]string      `json:"headers,omitempty"`
}

// GrpcProxy gRPC代理处理器
type GrpcProxy struct {
	config    *config.GrpcProxyConfig
	logger    *zerolog.Logger
	connPool  map[string]*grpc.ClientConn // 连接池
	connMutex sync.RWMutex
}

// NewGrpcProxy 创建新的gRPC代理实例
func NewGrpcProxy(cfg *config.GrpcProxyConfig, logger *zerolog.Logger) *GrpcProxy {
	return &GrpcProxy{
		config:   cfg,
		logger:   logger,
		connPool: make(map[string]*grpc.ClientConn),
	}
}

// IsGrpcRequest 判断是否为gRPC代理请求
func (p *GrpcProxy) IsGrpcRequest(r *http.Request) bool {
	if !p.config.Enabled {
		return false
	}

	// 检查gRPC标识头
	grpcFlag := r.Header.Get(p.config.GrpcHeader)
	return grpcFlag == "true" || grpcFlag == "1"
}

// ValidateGrpcAddr 验证gRPC地址是否在白名单中
func (p *GrpcProxy) ValidateGrpcAddr(addr string) bool {
	if len(p.config.Hosts) == 0 {
		return false // 如果没有配置白名单，则拒绝所有请求
	}

	for _, host := range p.config.Hosts {
		if addr == host || strings.HasPrefix(addr, host+":") {
			return true
		}
	}
	return false
}

// HandleGrpcRequest 处理gRPC代理请求
func (p *GrpcProxy) HandleGrpcRequest(w http.ResponseWriter, r *http.Request) {
	// 获取目标gRPC地址
	grpcAddr := r.Header.Get(p.config.GrpcAddr)
	if grpcAddr == "" {
		p.writeErrorResponse(w, "missing gRPC address header", http.StatusBadRequest)
		return
	}

	// 验证地址白名单
	if !p.ValidateGrpcAddr(grpcAddr) {
		p.logger.Warn().Str("address", grpcAddr).Msg("gRPC address not in whitelist")
		p.writeErrorResponse(w, "gRPC address not allowed", http.StatusForbidden)
		return
	}

	// 解析HTTP请求体
	grpcReq, err := p.parseHttpRequest(r)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to parse gRPC request")
		p.writeErrorResponse(w, fmt.Sprintf("invalid request format: %v\n", err), http.StatusBadRequest)
		return
	}

	// 执行gRPC调用
	resp, err := p.executeGrpcCall(grpcAddr, grpcReq)
	if err != nil {
		p.logger.Error().Err(err).Msg("gRPC call failed")
		p.writeErrorResponse(w, fmt.Sprintf("gRPC call failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回响应
	p.writeGrpcResponse(w, resp)
}

// parseHttpRequest 解析HTTP请求为gRPC请求结构
func (p *GrpcProxy) parseHttpRequest(r *http.Request) (*GrpcRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var grpcReq GrpcRequest
	if err := json.Unmarshal(body, &grpcReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	// 验证必要字段
	if grpcReq.Service == "" {
		return nil, fmt.Errorf("service field is required")
	}
	if grpcReq.Method == "" {
		return nil, fmt.Errorf("method field is required")
	}

	// 设置默认超时
	if grpcReq.Timeout <= 0 {
		grpcReq.Timeout = 30
	}

	return &grpcReq, nil
}

// executeGrpcCall 执行gRPC调用
func (p *GrpcProxy) executeGrpcCall(addr string, req *GrpcRequest) (*GrpcResponse, error) {
	// 获取连接
	conn, err := p.getConnection(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Second)
	defer cancel()

	// 添加metadata
	if len(req.Headers) > 0 {
		md := metadata.New(req.Headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	// 构造gRPC方法全名
	fullMethod := fmt.Sprintf("/%s/%s", req.Service, req.Method)

	// 将请求数据转换为protobuf格式
	reqData, err := json.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// 执行通用gRPC调用
	var respData json.RawMessage
	err = conn.Invoke(ctx, fullMethod, reqData, &respData)

	response := &GrpcResponse{
		Success: err == nil,
		Code:    http.StatusOK,
	}

	if err != nil {
		response.Error = err.Error()
		response.Code = http.StatusInternalServerError
		// 根据gRPC错误码设置HTTP状态码
		if strings.Contains(err.Error(), "NotFound") {
			response.Code = http.StatusNotFound
		} else if strings.Contains(err.Error(), "InvalidArgument") {
			response.Code = http.StatusBadRequest
		} else if strings.Contains(err.Error(), "Unauthenticated") {
			response.Code = http.StatusUnauthorized
		} else if strings.Contains(err.Error(), "PermissionDenied") {
			response.Code = http.StatusForbidden
		}
	} else {
		// 解析响应数据
		var data map[string]interface{}
		if err := json.Unmarshal(respData, &data); err == nil {
			response.Data = data
		}
	}

	// 获取响应headers
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		response.Headers = make(map[string]string)
		for k, v := range md {
			if len(v) > 0 {
				response.Headers[k] = v[0]
			}
		}
	}

	return response, nil
}

// getConnection 获取或创建gRPC连接
func (p *GrpcProxy) getConnection(addr string) (*grpc.ClientConn, error) {
	p.connMutex.RLock()
	if conn, exists := p.connPool[addr]; exists {
		p.connMutex.RUnlock()
		return conn, nil
	}
	p.connMutex.RUnlock()

	p.connMutex.Lock()
	defer p.connMutex.Unlock()

	// 双重检查
	if conn, exists := p.connPool[addr]; exists {
		return conn, nil
	}

	// 创建新连接
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server %s: %w", addr, err)
	}

	p.connPool[addr] = conn
	p.logger.Info().Str("address", addr).Msg("created new gRPC connection")

	return conn, nil
}

// writeErrorResponse 写入错误响应
func (p *GrpcProxy) writeErrorResponse(w http.ResponseWriter, message string, code int) {
	resp := &GrpcResponse{
		Success: false,
		Error:   message,
		Code:    code,
	}
	p.writeGrpcResponse(w, resp)
}

// writeGrpcResponse 写入gRPC响应
func (p *GrpcProxy) writeGrpcResponse(w http.ResponseWriter, resp *GrpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Code)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		p.logger.Error().Err(err).Msg("failed to encode gRPC response")
	}
}

// Close 关闭所有gRPC连接
func (p *GrpcProxy) Close() {
	p.connMutex.Lock()
	defer p.connMutex.Unlock()

	for addr, conn := range p.connPool {
		if err := conn.Close(); err != nil {
			p.logger.Error().Err(err).Str("address", addr).Msg("failed to close gRPC connection")
		}
	}
	p.connPool = make(map[string]*grpc.ClientConn)
}
