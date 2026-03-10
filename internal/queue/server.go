package queue

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/cskr/pubsub"
)

const (
	// LevelInfo 普通通知
	LevelInfo = "INFO"
	// LevelWarn 告警通知
	LevelWarn = "WARN"
	// LevelError 错误通知
	LevelError = "ERROR"
	// LevelCritical 严重通知
	LevelCritical = "CRITICAL"
)

const (
	// ContentTypeText 文本邮件内容类型
	ContentTypeText = "text/plain"
	// ContentTypeHTML HTML邮件内容类型
	ContentTypeHTML = "text/html"
)

// Message 通知消息
type Message struct {
	ID          string            // 消息ID
	Level       string            // 消息级别
	Topic       string            // 消息主题
	Subject     string            // 邮件主题
	Body        string            // 消息正文
	ContentType string            // 消息内容类型
	To          []string          // 覆盖收件人
	Metadata    map[string]string // 扩展字段
	CreatedAt   time.Time         // 消息创建时间
}

// Subscription 消息订阅句柄
type Subscription struct {
	C     <-chan Message
	close func()
	once  sync.Once
}

// Close 关闭订阅
func (s *Subscription) Close() {
	if s == nil || s.close == nil {
		return
	}
	s.once.Do(s.close)
}

// Server 内存消息服务器
type Server struct {
	pubsub       *pubsub.PubSub
	buffer       int
	defaultTopic string
}

var (
	defaultServer *Server
	defaultOnce   sync.Once
)

// NewServer 创建内存消息服务器
func NewServer(defaultTopic string, buffer int) *Server {
	if buffer <= 0 {
		buffer = 128
	}
	if strings.TrimSpace(defaultTopic) == "" {
		defaultTopic = "gateway.notify"
	}
	return &Server{
		pubsub:       pubsub.New(buffer),
		buffer:       buffer,
		defaultTopic: defaultTopic,
	}
}

// InitDefault 初始化全局消息服务器
func InitDefault(defaultTopic string, buffer int) *Server {
	defaultOnce.Do(func() {
		defaultServer = NewServer(defaultTopic, buffer)
	})
	return defaultServer
}

// Default 获取全局消息服务器
func Default() *Server {
	return defaultServer
}

// Publish 发布消息
func Publish(ctx context.Context, msg Message) error {
	if defaultServer == nil {
		return errors.New("queue server not initialized")
	}
	return defaultServer.Publish(ctx, msg)
}

// Subscribe 订阅主题
func Subscribe(topics ...string) (*Subscription, error) {
	if defaultServer == nil {
		return nil, errors.New("queue server not initialized")
	}
	return defaultServer.Subscribe(topics...)
}

// Publish 发布消息到主题
func (s *Server) Publish(ctx context.Context, msg Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	if strings.TrimSpace(msg.Level) == "" {
		msg.Level = LevelInfo
	}
	if strings.TrimSpace(msg.ContentType) == "" {
		msg.ContentType = ContentTypeText
	}
	topic := msg.Topic
	if strings.TrimSpace(topic) == "" {
		topic = s.defaultTopic
	}
	s.pubsub.Pub(msg, topic)
	return nil
}

// Subscribe 订阅主题并返回类型安全通道
func (s *Server) Subscribe(topics ...string) (*Subscription, error) {
	cleanTopics := make([]string, 0, len(topics))
	for _, topic := range topics {
		if strings.TrimSpace(topic) != "" {
			cleanTopics = append(cleanTopics, topic)
		}
	}
	if len(cleanTopics) == 0 {
		cleanTopics = []string{s.defaultTopic}
	}

	rawCh := s.pubsub.Sub(cleanTopics...)
	ch := make(chan Message, s.buffer)
	stop := make(chan struct{})

	go func() {
		defer close(ch)
		for {
			select {
			case <-stop:
				return
			case raw, ok := <-rawCh:
				if !ok {
					return
				}
				msg, ok := raw.(Message)
				if !ok {
					continue
				}
				ch <- msg
			}
		}
	}()

	return &Subscription{
		C: ch,
		close: func() {
			close(stop)
			s.pubsub.Unsub(rawCh, cleanTopics...)
		},
	}, nil
}

// DefaultTopic 返回默认主题
func (s *Server) DefaultTopic() string {
	return s.defaultTopic
}
