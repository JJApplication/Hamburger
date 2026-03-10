package notifier

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"Hamburger/internal/config"
	"Hamburger/internal/imap"
	"Hamburger/internal/logger"
	"Hamburger/internal/queue"
)

// Service 通知服务
type Service struct {
	cfg         config.NotifyConfig
	queueServer *queue.Server
	mailService *imap.Service
	subscription *queue.Subscription
	log         *loggerWrapper
}

type loggerWrapper struct{}

func (l *loggerWrapper) Info(msg string, kv ...string) {
	event := logger.GetLogger().Info()
	for i := 0; i+1 < len(kv); i += 2 {
		event = event.Str(kv[i], kv[i+1])
	}
	event.Msg(msg)
}

func (l *loggerWrapper) Error(err error, msg string, kv ...string) {
	event := logger.GetLogger().Error().Err(err)
	for i := 0; i+1 < len(kv); i += 2 {
		event = event.Str(kv[i], kv[i+1])
	}
	event.Msg(msg)
}

var (
	globalService *Service
	serviceLock   sync.RWMutex
)

// Register 注册并启动通知服务
func Register(cfg config.NotifyConfig) (*Service, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	q := queue.Default()
	if q == nil {
		q = queue.InitDefault(cfg.Queue.Topic, cfg.Queue.Buffer)
	}
	mail := imap.NewService(cfg.Mail)

	service := &Service{
		cfg:         cfg,
		queueServer: q,
		mailService: mail,
		log:         &loggerWrapper{},
	}

	if err := service.start(); err != nil {
		return nil, err
	}

	serviceLock.Lock()
	globalService = service
	serviceLock.Unlock()
	return service, nil
}

// Get 获取全局通知服务
func Get() *Service {
	serviceLock.RLock()
	defer serviceLock.RUnlock()
	return globalService
}

// Publish 发布消息到通知系统
func Publish(ctx context.Context, msg queue.Message) error {
	s := Get()
	if s == nil {
		return queue.Publish(ctx, msg)
	}
	return s.queueServer.Publish(ctx, msg)
}

// start 启动订阅并消费消息
func (s *Service) start() error {
	sub, err := s.queueServer.Subscribe(s.cfg.Queue.Topic)
	if err != nil {
		return err
	}
	s.subscription = sub

	if err = s.mailService.Ping(context.Background()); err != nil {
		s.log.Error(err, "notifier mail ping failed")
	}

	go s.consume()
	s.log.Info("notifier started", "topic", s.cfg.Queue.Topic)
	return nil
}

// consume 消费通知消息并发送邮件
func (s *Service) consume() {
	for msg := range s.subscription.C {
		recipients := s.resolveRecipients(msg)
		if len(recipients) == 0 {
			s.log.Info("skip notify because recipients is empty", "level", msg.Level)
			continue
		}
		subject := buildSubject(msg)
		contentType := msg.ContentType
		if strings.TrimSpace(contentType) == "" {
			contentType = queue.ContentTypeText
		}
		sendErr := s.mailService.Send(context.Background(), recipients, subject, msg.Body, contentType)
		if sendErr != nil {
			s.log.Error(sendErr, "send notify email failed", "subject", subject)
			continue
		}
		s.log.Info("notify email sent", "subject", subject, "to", strings.Join(recipients, ","))
	}
}

// resolveRecipients 解析收件人
func (s *Service) resolveRecipients(msg queue.Message) []string {
	if len(msg.To) > 0 {
		return msg.To
	}
	return s.cfg.DefaultRecipients
}

// buildSubject 组装邮件主题
func buildSubject(msg queue.Message) string {
	level := strings.ToUpper(strings.TrimSpace(msg.Level))
	if level == "" {
		level = queue.LevelInfo
	}
	subject := strings.TrimSpace(msg.Subject)
	if subject == "" {
		subject = strings.TrimSpace(msg.Topic)
	}
	if subject == "" {
		subject = "Gateway Notification"
	}
	return fmt.Sprintf("[%s] %s %s", level, subject, time.Now().Format(time.DateTime))
}
