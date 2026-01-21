package modifier

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"Hamburger/internal/config"
	"strings"
	"testing"
	"time"
)

// 测试数据生成器
func generateTestData(size int) []byte {
	// 生成可压缩的重复数据
	pattern := "Hello World! This is a test string for gzip compression. "
	data := make([]byte, 0, size)
	for len(data) < size {
		remaining := size - len(data)
		if remaining < len(pattern) {
			data = append(data, pattern[:remaining]...)
		} else {
			data = append(data, pattern...)
		}
	}
	return data
}

// 生成JSON格式的测试数据
func generateJSONData(size int) []byte {
	baseJSON := `{"id":1,"name":"test","description":"This is a test description for JSON compression","data":[1,2,3,4,5],"timestamp":"2024-01-01T00:00:00Z"}`
	data := make([]byte, 0, size)
	for len(data) < size {
		remaining := size - len(data)
		if remaining < len(baseJSON) {
			data = append(data, baseJSON[:remaining]...)
		} else {
			data = append(data, baseJSON...)
		}
	}
	return data
}

// 创建测试响应
func createTestResponse(data []byte, contentType string) *http.Response {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	
	resp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(data)),
		Request:    req,
	}
	resp.Header.Set("Content-Type", contentType)
	resp.Header.Set("Content-Length", string(rune(len(data))))
	
	return resp
}

// 基准测试：不同大小的数据压缩
func BenchmarkGzipModifier_SmallData(b *testing.B) {
	benchmarkGzipModifier(b, 1024) // 1KB
}

func BenchmarkGzipModifier_MediumData(b *testing.B) {
	benchmarkGzipModifier(b, 10*1024) // 10KB
}

func BenchmarkGzipModifier_LargeData(b *testing.B) {
	benchmarkGzipModifier(b, 100*1024) // 100KB
}

func BenchmarkGzipModifier_VeryLargeData(b *testing.B) {
	benchmarkGzipModifier(b, 1024*1024) // 1MB
}

// 通用基准测试函数
func benchmarkGzipModifier(b *testing.B, dataSize int) {
	// 初始化配置
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html", "application/json"}
	cfg.Features.Gzip.Threshold = 1024
	
	// 使用ConfigLoader设置配置
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewGzipModifier()
	testData := generateTestData(dataSize)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		resp := createTestResponse(testData, "text/html")
		err := modifier.ModifyResponse(resp)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// 基准测试：不同压缩级别
func BenchmarkGzipModifier_Level1(b *testing.B) {
	benchmarkGzipLevel(b, 1)
}

func BenchmarkGzipModifier_Level6(b *testing.B) {
	benchmarkGzipLevel(b, 6)
}

func BenchmarkGzipModifier_Level9(b *testing.B) {
	benchmarkGzipLevel(b, 9)
}

func benchmarkGzipLevel(b *testing.B, level int) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = level
	cfg.Features.Gzip.Types = []string{"text/html", "application/json"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewGzipModifier()
	testData := generateTestData(50 * 1024) // 50KB
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		resp := createTestResponse(testData, "text/html")
		err := modifier.ModifyResponse(resp)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// 基准测试：不同内容类型
func BenchmarkGzipModifier_HTML(b *testing.B) {
	benchmarkGzipContentType(b, "text/html", generateTestData(50*1024))
}

func BenchmarkGzipModifier_JSON(b *testing.B) {
	benchmarkGzipContentType(b, "application/json", generateJSONData(50*1024))
}

func BenchmarkGzipModifier_CSS(b *testing.B) {
	cssData := []byte(strings.Repeat("body { margin: 0; padding: 0; } ", 1000))
	benchmarkGzipContentType(b, "text/css", cssData)
}

func benchmarkGzipContentType(b *testing.B, contentType string, data []byte) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html", "application/json", "text/css"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewGzipModifier()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		resp := createTestResponse(data, contentType)
		err := modifier.ModifyResponse(resp)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// 基准测试：对象池效果
func BenchmarkGzipModifier_WithPool(b *testing.B) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewGzipModifier()
	testData := generateTestData(50 * 1024)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		resp := createTestResponse(testData, "text/html")
		err := modifier.ModifyResponse(resp)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// 基准测试：不使用对象池（对比）
func BenchmarkGzipModifier_WithoutPool(b *testing.B) {
	testData := generateTestData(50 * 1024)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		resp := createTestResponse(testData, "text/html")
		
		// 直接压缩，不使用对象池
		var buf bytes.Buffer
		writer, _ := gzip.NewWriterLevel(&buf, 6)
		writer.Write(testData)
		writer.Close()
		
		resp.Body.Close()
	}
}

// 基准测试：阈值检查的影响
func BenchmarkGzipModifier_BelowThreshold(b *testing.B) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html"}
	cfg.Features.Gzip.Threshold = 10240 // 10KB阈值
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewGzipModifier()
	testData := generateTestData(5 * 1024) // 5KB数据，低于阈值
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		resp := createTestResponse(testData, "text/html")
		err := modifier.ModifyResponse(resp)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// 基准测试：并发压缩
func BenchmarkGzipModifier_Concurrent(b *testing.B) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewGzipModifier()
	testData := generateTestData(50 * 1024)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp := createTestResponse(testData, "text/html")
			err := modifier.ModifyResponse(resp)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// 性能分析：压缩比和时间的关系
func BenchmarkGzipModifier_CompressionRatio(b *testing.B) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewGzipModifier()
	testData := generateTestData(100 * 1024)
	
	b.ResetTimer()
	
	start := time.Now()
	for i := 0; i < b.N; i++ {
		resp := createTestResponse(testData, "text/html")
		err := modifier.ModifyResponse(resp)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
	duration := time.Since(start)
	
	b.ReportMetric(float64(len(testData)), "original_bytes")
	b.ReportMetric(duration.Seconds()/float64(b.N), "seconds_per_op")
}