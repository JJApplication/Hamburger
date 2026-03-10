package imap

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"Hamburger/internal/config"

	goimapclient "github.com/emersion/go-imap/client"
)

var (
	// ErrProviderNotSupported 邮件服务类型不支持
	ErrProviderNotSupported = errors.New("mail provider not supported")
	// ErrSendNotSupported 邮件服务不支持发送
	ErrSendNotSupported = errors.New("mail provider does not support send")
)

// Service 邮件服务对接器
type Service struct {
	cfg config.NotifyMailConfig
}

// NewService 创建邮件服务对接器
func NewService(cfg config.NotifyMailConfig) *Service {
	return &Service{cfg: cfg}
}

// Send 发送邮件到指定收件人
func (s *Service) Send(ctx context.Context, to []string, subject, body, contentType string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(to) == 0 {
		return errors.New("recipient is empty")
	}
	if contentType == "" {
		contentType = "text/plain"
	}

	switch strings.ToLower(strings.TrimSpace(s.cfg.Provider)) {
	case "smtp":
		return s.sendBySMTP(to, subject, body, contentType)
	case "pop3":
		return ErrSendNotSupported
	default:
		return ErrProviderNotSupported
	}
}

// Ping 检查邮件系统连通性
func (s *Service) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(s.cfg.Provider)) {
	case "smtp":
		return s.pingSMTP()
	case "pop3":
		return s.pingPOP3()
	default:
		return ErrProviderNotSupported
	}
}

// PingIMAP 检查IMAP连通性
func (s *Service) PingIMAP(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	addr := net.JoinHostPort(s.cfg.IMAP.Host, fmt.Sprintf("%d", s.cfg.IMAP.Port))
	var (
		c   *goimapclient.Client
		err error
	)
	if s.cfg.IMAP.TLS {
		c, err = goimapclient.DialTLS(addr, nil)
	} else {
		c, err = goimapclient.Dial(addr)
	}
	if err != nil {
		return err
	}
	defer func() {
		_ = c.Logout()
	}()
	if s.cfg.IMAP.Username != "" {
		return c.Login(s.cfg.IMAP.Username, s.cfg.IMAP.Password)
	}
	return nil
}

// sendBySMTP 通过SMTP发送邮件
func (s *Service) sendBySMTP(to []string, subject, body, contentType string) error {
	addr := net.JoinHostPort(s.cfg.SMTP.Host, fmt.Sprintf("%d", s.cfg.SMTP.Port))
	from := s.cfg.From
	if from == "" {
		from = s.cfg.SMTP.Username
	}
	msg := buildMessage(from, to, subject, body, contentType)
	auth := smtp.PlainAuth("", s.cfg.SMTP.Username, s.cfg.SMTP.Password, s.cfg.SMTP.Host)

	if s.cfg.SMTP.TLS {
		tlsConn, err := tls.Dial("tcp", addr, &tls.Config{
			ServerName: s.cfg.SMTP.Host,
		})
		if err != nil {
			return err
		}
		defer func() {
			_ = tlsConn.Close()
		}()

		c, err := smtp.NewClient(tlsConn, s.cfg.SMTP.Host)
		if err != nil {
			return err
		}
		defer func() {
			_ = c.Quit()
		}()
		if s.cfg.SMTP.Username != "" {
			if err = c.Auth(auth); err != nil {
				return err
			}
		}
		if err = c.Mail(from); err != nil {
			return err
		}
		for _, recipient := range to {
			if err = c.Rcpt(recipient); err != nil {
				return err
			}
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err = w.Write([]byte(msg)); err != nil {
			return err
		}
		return w.Close()
	}

	if s.cfg.SMTP.Username == "" {
		auth = nil
	}
	return smtp.SendMail(addr, auth, from, to, []byte(msg))
}

// pingSMTP 检查SMTP连通性
func (s *Service) pingSMTP() error {
	addr := net.JoinHostPort(s.cfg.SMTP.Host, fmt.Sprintf("%d", s.cfg.SMTP.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

// pingPOP3 检查POP3连通性并执行登录
func (s *Service) pingPOP3() error {
	addr := net.JoinHostPort(s.cfg.POP3.Host, fmt.Sprintf("%d", s.cfg.POP3.Port))
	var (
		conn net.Conn
		err  error
	)
	if s.cfg.POP3.TLS {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", addr, &tls.Config{
			ServerName: s.cfg.POP3.Host,
		})
	} else {
		conn, err = net.DialTimeout("tcp", addr, 5*time.Second)
	}
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)
	if err = expectOK(reader); err != nil {
		return err
	}
	if s.cfg.POP3.Username != "" {
		if _, err = fmt.Fprintf(conn, "USER %s\r\n", s.cfg.POP3.Username); err != nil {
			return err
		}
		if err = expectOK(reader); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(conn, "PASS %s\r\n", s.cfg.POP3.Password); err != nil {
			return err
		}
		if err = expectOK(reader); err != nil {
			return err
		}
	}
	_, _ = fmt.Fprintf(conn, "QUIT\r\n")
	return nil
}

// expectOK 读取POP3响应并校验
func expectOK(reader *bufio.Reader) error {
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(line, "+OK") {
		return errors.New(strings.TrimSpace(line))
	}
	return nil
}

// buildMessage 构建邮件内容
func buildMessage(from string, to []string, subject, body, contentType string) string {
	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", strings.Join(to, ",")),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: %s; charset=UTF-8", contentType),
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
}
