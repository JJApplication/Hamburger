package notifier

// notifier 提供基于内存消息队列的通知推送能力。
//
// 消息推送方式示例：
//
//	package main
//
//	import (
//		"context"
//		"Hamburger/gateway/notifier"
//		"Hamburger/internal/queue"
//	)
//
//	func main() {
//		_ = notifier.Publish(context.Background(), queue.Message{
//			Level:       queue.LevelWarn,
//			Topic:       "gateway.notify",
//			Subject:     "网关告警",
//			Body:        "检测到后端响应延迟升高",
//			ContentType: queue.ContentTypeText,
//			To:          []string{"ops@example.com"},
//		})
//	}
//
// HTML 邮件示例：
//
//	_ = notifier.Publish(context.Background(), queue.Message{
//		Level:       queue.LevelCritical,
//		Topic:       "gateway.notify",
//		Subject:     "网关严重告警",
//		Body:        "<h1>服务不可用</h1><p>请立即处理</p>",
//		ContentType: queue.ContentTypeHTML,
//	})
