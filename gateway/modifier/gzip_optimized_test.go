package modifier

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sandwich/config"
	"testing"
)

// 基准测试：优化版本 vs 原版本
func BenchmarkOptimizedGzip_SmallData(b *testing.B) {
	benchmarkOptimizedGzip(b, 1024) // 1KB
}

func BenchmarkOptimizedGzip_MediumData(b *testing.B) {
	benchmarkOptimizedGzip(b, 10*1024) // 10KB
}

func BenchmarkOptimizedGzip_LargeData(b *testing.B) {
	benchmarkOptimizedGzip(b, 100*1024) // 100KB
}

func BenchmarkOptimizedGzip_VeryLargeData(b *testing.B) {
	benchmarkOptimizedGzip(b, 1024*1024) // 1MB
}

func benchmarkOptimizedGzip(b *testing.B, dataSize int) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html", "application/json"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewOptimizedGzipModifier()
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

// 基准测试：缓存效果
func BenchmarkOptimizedGzip_WithCache(b *testing.B) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewOptimizedGzipModifier()
	modifier.cacheEnabled = true
	testData := generateTestData(50 * 1024)
	
	// 预热缓存
	resp := createTestResponse(testData, "text/html")
	modifier.ModifyResponse(resp)
	resp.Body.Close()
	
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

// 基准测试：异步压缩
func BenchmarkOptimizedGzip_AsyncCompression(b *testing.B) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewOptimizedGzipModifier()
	modifier.asyncThreshold = 50 * 1024 // 50KB异步阈值
	testData := generateTestData(100 * 1024) // 100KB数据
	
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

// 基准测试：并发性能
func BenchmarkOptimizedGzip_Concurrent(b *testing.B) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewOptimizedGzipModifier()
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

// 对比测试：原版本 vs 优化版本
func BenchmarkComparison_Original_50KB(b *testing.B) {
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

func BenchmarkComparison_Optimized_50KB(b *testing.B) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewOptimizedGzipModifier()
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

// 功能测试
func TestOptimizedGzipModifier_BasicFunctionality(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html", "application/json"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewOptimizedGzipModifier()
	
	// 测试数据
	testData := generateTestData(5 * 1024) // 5KB
	
	// 创建测试请求和响应
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	
	resp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(testData)),
		Request:    req,
	}
	resp.Header.Set("Content-Type", "text/html")
	
	// 执行压缩
	err := modifier.ModifyResponse(resp)
	if err != nil {
		t.Fatalf("压缩失败: %v", err)
	}
	
	// 验证响应头
	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Error("Content-Encoding头未设置为gzip")
	}
	
	if resp.Header.Get("Vary") != "Accept-Encoding" {
		t.Error("Vary头未正确设置")
	}
	
	// 读取压缩后的数据
	compressedData, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取压缩数据失败: %v", err)
	}
	
	// 验证数据被压缩
	if len(compressedData) >= len(testData) {
		t.Error("数据未被有效压缩")
	}
	
	t.Logf("原始大小: %d bytes, 压缩后大小: %d bytes, 压缩比: %.2f%%", 
		len(testData), len(compressedData), 
		float64(len(compressedData))/float64(len(testData))*100)
}

func TestOptimizedGzipModifier_CacheHit(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewOptimizedGzipModifier()
	modifier.cacheEnabled = true
	
	testData := generateTestData(5 * 1024)
	
	// 第一次压缩
	resp1 := createTestResponse(testData, "text/html")
	err := modifier.ModifyResponse(resp1)
	if err != nil {
		t.Fatalf("第一次压缩失败: %v", err)
	}
	resp1.Body.Close()
	
	// 第二次压缩（应该命中缓存）
	resp2 := createTestResponse(testData, "text/html")
	err = modifier.ModifyResponse(resp2)
	if err != nil {
		t.Fatalf("第二次压缩失败: %v", err)
	}
	resp2.Body.Close()
	
	// 检查统计信息
	stats := modifier.GetStats()
	if stats["cache_hits"].(int64) == 0 {
		t.Error("缓存未命中")
	}
	
	t.Logf("统计信息: %+v", stats)
}

func TestOptimizedGzipModifier_Stats(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.Features.Gzip.Enabled = true
	cfg.Features.Gzip.Level = 6
	cfg.Features.Gzip.Types = []string{"text/html"}
	cfg.Features.Gzip.Threshold = 1024
	
	loader := config.NewConfigLoader("")
	loader.RegisterGlobalConfig(cfg)
	
	modifier := NewOptimizedGzipModifier()
	modifier.cacheEnabled = false // 禁用缓存以测试压缩次数
	
	// 执行多次压缩
	for i := 0; i < 5; i++ {
		testData := generateTestData(5*1024 + i) // 每次生成不同的数据
		resp := createTestResponse(testData, "text/html")
		modifier.ModifyResponse(resp)
		resp.Body.Close()
	}
	
	// 检查统计信息
	stats := modifier.GetStats()
	
	if stats["total_requests"].(int64) != 5 {
		t.Errorf("总请求数不正确: 期望5, 实际%d", stats["total_requests"])
	}
	
	if stats["compressed_count"].(int64) != 5 {
		t.Errorf("压缩次数不正确: 期望5, 实际%d", stats["compressed_count"])
	}
	
	t.Logf("统计信息: %+v", stats)
}