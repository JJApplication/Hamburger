package utils

import (
	"crypto/rand"
	"fmt"
	"time"
)

// generateTraceId 生成唯一的TraceID
// 格式: timestamp-random (例如: 20250111150430-a1b2c3d4e5f6)
func generateTraceId() string {
	// 获取当前时间戳 (yyyyMMddHHmmss格式)
	timestamp := time.Now().Format("20060102150405")

	// 生成6字节随机数
	randomBytes := make([]byte, 6)
	rand.Read(randomBytes)

	// 转换为十六进制字符串
	randomHex := fmt.Sprintf("%x", randomBytes)

	// 组合成最终的TraceID
	return fmt.Sprintf("%s-%s", timestamp, randomHex)
}
