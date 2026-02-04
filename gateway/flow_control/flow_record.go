/*
Create: 2022/9/7
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package flow
package flow

import (
	"net/http"
	"os"
	"sandwich/config"
	"sandwich/data"
	"sandwich/json"
	"sandwich/log"
	"strings"
	"sync"
	"time"
)

// 流量统计
// 基于时序数据库的流量统计
// 记录请求 请求接口 请求的client 请求时间 请求的响应码
// 新增：记录被限流的HOST和请求信息

var flowRecorder *FlowRecorder

func InitFlowRecorder() {
	flowRecorder = NewFlowRecorder()
}

func GetFlowRecorder() *FlowRecorder {
	return flowRecorder
}

// FlowRecorder 流量记录器
type FlowRecorder struct {
	config       *config.FlowRecordConfig
	recordBuffer []FlowRecord
	bufferMutex  sync.Mutex
	filePath     string
}

// FlowRecord 流量记录结构
type FlowRecord struct {
	Timestamp   time.Time `json:"timestamp"`    // 记录时间
	Host        string    `json:"host"`         // 请求的Host
	ClientIP    string    `json:"client_ip"`    // 客户端IP
	UserAgent   string    `json:"user_agent"`   // User-Agent
	URL         string    `json:"url"`          // 请求URL
	Method      string    `json:"method"`       // HTTP方法
	Status      string    `json:"status"`       // 状态: blocked, allowed
	RuleName    string    `json:"rule_name"`    // 触发的规则名称
	Reason      string    `json:"reason"`       // 限流原因
	Headers     string    `json:"headers"`      // 重要的请求头（JSON格式）
	Referer     string    `json:"referer"`      // 来源页面
	RequestSize int64     `json:"request_size"` // 请求大小
}

// NewFlowRecorder 创建流量记录器
func NewFlowRecorder() *FlowRecorder {
	cfg := config.Get().FlowControl.Recording
	recorder := &FlowRecorder{
		config:       &cfg,
		recordBuffer: make([]FlowRecord, 0, 100),
		filePath:     "flow_records.json",
	}

	// 启动定期刷新缓冲区
	if cfg.Enabled {
		go recorder.flushBuffer()
	}

	return recorder
}

// RecordBlocked 记录被限流的请求
func (fr *FlowRecorder) RecordBlocked(req *http.Request, result *FlowCheckResult) {
	if !fr.config.Enabled || !fr.config.RecordBlocked {
		return
	}

	record := fr.createRecord(req, "blocked", result.RuleName, result.Reason)
	fr.addRecord(record)

	log.GetLogger().Info().Str("Host", record.Host).Str("IP", record.ClientIP).
		Str("Rule", record.RuleName).Str("Reason", record.Reason).Msg("Flow blocked")
}

// RecordAllowed 记录通过的请求
func (fr *FlowRecorder) RecordAllowed(req *http.Request) {
	if !fr.config.Enabled || !fr.config.RecordAllowed {
		return
	}

	record := fr.createRecord(req, "allowed", "", "")
	fr.addRecord(record)
}

// createRecord 创建流量记录
func (fr *FlowRecorder) createRecord(req *http.Request, status, ruleName, reason string) FlowRecord {
	// 获取客户端IP
	clientIP := fr.getClientIP(req)

	// 获取重要的请求头
	importantHeaders := map[string]string{
		"User-Agent":      req.Header.Get("User-Agent"),
		"X-Forwarded-For": req.Header.Get("X-Forwarded-For"),
		"X-Real-IP":       req.Header.Get("X-Real-IP"),
		"Accept":          req.Header.Get("Accept"),
		"Accept-Language": req.Header.Get("Accept-Language"),
		"Content-Type":    req.Header.Get("Content-Type"),
	}

	headersJSON, _ := json.Marshal(importantHeaders)

	return FlowRecord{
		Timestamp:   time.Now(),
		Host:        req.Host,
		ClientIP:    clientIP,
		UserAgent:   req.Header.Get("User-Agent"),
		URL:         req.URL.String(),
		Method:      req.Method,
		Status:      status,
		RuleName:    ruleName,
		Reason:      reason,
		Headers:     string(headersJSON),
		Referer:     req.Header.Get("Referer"),
		RequestSize: req.ContentLength,
	}
}

// getClientIP 获取客户端IP
func (fr *FlowRecorder) getClientIP(req *http.Request) string {
	// 检查 X-Forwarded-For
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		// 取第一个IP
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// 检查 X-Real-IP
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 使用 RemoteAddr
	ip := req.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// addRecord 添加记录到缓冲区
func (fr *FlowRecorder) addRecord(record FlowRecord) {
	fr.bufferMutex.Lock()
	defer fr.bufferMutex.Unlock()

	fr.recordBuffer = append(fr.recordBuffer, record)

	// 如果缓冲区满了，立即刷新
	if len(fr.recordBuffer) >= 100 {
		go fr.doFlush()
	}
}

// flushBuffer 定期刷新缓冲区
func (fr *FlowRecorder) flushBuffer() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fr.doFlush()
	}
}

// doFlush 执行刷新操作
func (fr *FlowRecorder) doFlush() {
	fr.bufferMutex.Lock()
	if len(fr.recordBuffer) == 0 {
		fr.bufferMutex.Unlock()
		return
	}

	// 拷贝缓冲区数据
	records := make([]FlowRecord, len(fr.recordBuffer))
	copy(records, fr.recordBuffer)
	fr.recordBuffer = fr.recordBuffer[:0] // 清空缓冲区
	fr.bufferMutex.Unlock()

	// 根据配置选择存储方式
	switch fr.config.StorageType {
	case "influx":
		fr.storeToInflux(records)
	case "mongo":
		fr.storeToMongo(records)
	case "file":
		fr.storeToFile(records)
	default:
		fr.storeToFile(records) // 默认使用文件存储
	}
}

// storeToInflux 存储到InfluxDB
func (fr *FlowRecorder) storeToInflux(records []FlowRecord) {
	if !config.Get().Database.Influx.Enabled {
		log.Info("InfluxDB is not enabled, switching to file storage")
		fr.storeToFile(records)
		return
	}

	for _, record := range records {
		// 使用现有的data模块记录到InfluxDB
		if record.Status == "blocked" {
			// 模拟一个HTTP请求来调用data.AddInfluxData
			req := &http.Request{
				Host:       record.Host,
				RemoteAddr: record.ClientIP,
				Method:     record.Method,
				Header:     make(http.Header),
			}
			req.Header.Set("User-Agent", record.UserAgent)
			data.AddInfluxData(req, "flow_blocked")
		}
	}

	log.GetLogger().Info().Int("records", len(records)).Msg("Stored flow records to InfluxDB")
}

// storeToMongo 存储到MongoDB
func (fr *FlowRecorder) storeToMongo(records []FlowRecord) {
	// 这里需要实现MongoDB存储逻辑
	// 如果没有现成的MongoDB客户端，先使用文件存储
	log.Info("MongoDB storage not implemented yet, switching to file storage")
	fr.storeToFile(records)
}

// storeToFile 存储到文件
func (fr *FlowRecorder) storeToFile(records []FlowRecord) {
	file, err := os.OpenFile(fr.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.GetLogger().Error().Err(err).Msg("Failed to open flow record file")
		return
	}
	defer file.Close()

	for _, record := range records {
		data, err := json.Marshal(record)
		if err != nil {
			log.GetLogger().Error().Err(err).Msg("Failed to marshal flow record")
			continue
		}
		_, err = file.WriteString(string(data) + "\n")
		if err != nil {
			log.GetLogger().Error().Err(err).Msg("Failed to write flow record")
		}
	}

	log.GetLogger().Info().Str("file", fr.filePath).Int("records", len(records)).Msg("Stored flow records to file")
}

// GetBlockedHosts 获取被限流的Host统计
func (fr *FlowRecorder) GetBlockedHosts(since time.Time) map[string]int {
	// 这里可以实现从存储中查询被限流的Host统计
	// 目前返回空的map，后续可以扩展
	return make(map[string]int)
}

// GetFlowStats 获取流量统计
func (fr *FlowRecorder) GetFlowStats(since time.Time) (blocked, allowed int) {
	// 这里可以实现从存储中查询流量统计
	// 目前返回0，后续可以扩展
	return 0, 0
}
