package main

import (
	"Hamburger/exp/traversal"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const dialTimeout = 8 * time.Second
const relayIdleTimeout = 90 * time.Second

// clientConfig 内网穿透客户端配置。
type clientConfig struct {
	// ServerIP 服务端 IP 地址。
	ServerIP string `json:"server_ip"`
	// ServerPort 服务端控制端口。
	ServerPort int `json:"server_port"`
	// ServerProtocol 服务端协议，当前仅支持 tcp。
	ServerProtocol string `json:"server_protocol"`
	// AuthKey 认证密钥。
	AuthKey string `json:"auth_key"`
	// ProxyServers 服务端对外映射配置列表。
	ProxyServers []proxyServerConfig `json:"proxy_server"`
}

// proxyServerConfig 单个内网穿透映射配置。
type proxyServerConfig struct {
	// Name 映射服务名。
	Name string `json:"name"`
	// RemotePort 服务端对外端口。
	RemotePort int `json:"remote_port"`
	// LocalHost 本地被代理服务地址。
	LocalHost string `json:"local_host"`
	// LocalPort 本地被代理服务端口。
	LocalPort int `json:"local_port"`
}

func main() {
	configPath := "traversalc.json"
	flag.StringVar(&configPath, "c", "traversalc.json", "traversal client config file path")
	flag.StringVar(&configPath, "config", "traversalc.json", "traversal client config file path")
	flag.Parse()

	cfg, err := loadConfig(configPath)
	if err != nil {
		fatalf("加载配置失败: %v", err)
	}
	if err = cfg.validate(); err != nil {
		fatalf("配置非法: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client := &traversalClient{
		cfg: cfg,
	}

	// goroutine
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err = client.run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("客户端异常退出: %v，5s后重试", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
				continue
			}
			return
		}
	}()
	wg.Wait()
}

type traversalClient struct {
	cfg clientConfig
}

func (c *traversalClient) run(ctx context.Context) error {
	addr := net.JoinHostPort(c.cfg.ServerIP, strconv.Itoa(c.cfg.ServerPort))
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	bindings := make([]traversal.ProxyServerBinding, 0, len(c.cfg.ProxyServers))
	for _, item := range c.cfg.ProxyServers {
		bindings = append(bindings, traversal.ProxyServerBinding{
			Name:       item.Name,
			RemotePort: item.RemotePort,
		})
	}
	if err = traversal.WriteMessage(writer, traversal.Message{
		Type:         traversal.MessageTypeRegister,
		AuthKey:      c.cfg.AuthKey,
		ProxyServers: bindings,
	}); err != nil {
		return err
	}
	if err = writer.Flush(); err != nil {
		return err
	}

	reader := bufio.NewReader(conn)
	ack, err := traversal.ReadMessage(reader)
	if err != nil {
		return err
	}
	if ack.Type != traversal.MessageTypeRegisterAck || !ack.OK {
		if strings.TrimSpace(ack.Message) == "" {
			ack.Message = "register failed"
		}
		return errors.New(ack.Message)
	}

	log.Printf("已连接内网穿透服务端 %s", addr)
	for _, item := range c.cfg.ProxyServers {
		log.Printf(
			"[%s] 访问地址映射：%s:%d -> %s:%d",
			item.Name,
			c.cfg.ServerIP,
			item.RemotePort,
			item.LocalHost,
			item.LocalPort,
		)
	}

	done := make(chan error, 1)
	go c.readLoop(reader, done)

	select {
	case <-ctx.Done():
		return context.Canceled
	case err = <-done:
		return err
	}
}

func (c *traversalClient) readLoop(reader *bufio.Reader, done chan<- error) {
	for {
		msg, err := traversal.ReadMessage(reader)
		if err != nil {
			done <- err
			return
		}
		if msg.Type != traversal.MessageTypeOpen || strings.TrimSpace(msg.ConnID) == "" {
			continue
		}
		go func(openMsg traversal.Message) {
			if tunnelErr := c.handleOpen(openMsg); tunnelErr != nil {
				log.Printf(
					"处理隧道失败 conn_id=%s mapping=%s remote_port=%d err=%v",
					openMsg.ConnID,
					openMsg.MappingName,
					openMsg.RemotePort,
					tunnelErr,
				)
			}
		}(msg)
	}
}

func (c *traversalClient) handleOpen(openMsg traversal.Message) error {
	target, ok := c.findTarget(openMsg.MappingName, openMsg.RemotePort)
	if !ok {
		return fmt.Errorf("未找到映射配置: mapping=%s remote_port=%d", openMsg.MappingName, openMsg.RemotePort)
	}

	serverAddr := net.JoinHostPort(c.cfg.ServerIP, strconv.Itoa(c.cfg.ServerPort))
	dataConn, err := net.DialTimeout("tcp", serverAddr, dialTimeout)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(dataConn)
	if err = traversal.WriteMessage(writer, traversal.Message{
		Type:    traversal.MessageTypeJoin,
		AuthKey: c.cfg.AuthKey,
		ConnID:  openMsg.ConnID,
	}); err != nil {
		_ = dataConn.Close()
		return err
	}
	if err = writer.Flush(); err != nil {
		_ = dataConn.Close()
		return err
	}

	localAddr := net.JoinHostPort(target.LocalHost, strconv.Itoa(target.LocalPort))
	localConn, err := net.DialTimeout("tcp", localAddr, dialTimeout)
	if err != nil {
		_ = dataConn.Close()
		return err
	}

	relay(dataConn, localConn)
	return nil
}

func relay(a net.Conn, b net.Conn) {
	defer a.Close()
	defer b.Close()

	errCh := make(chan error, 2)
	go func() {
		_ = b.SetReadDeadline(time.Now().Add(relayIdleTimeout))
		_ = a.SetWriteDeadline(time.Now().Add(relayIdleTimeout))
		_, err := io.Copy(a, b)
		closeWrite(a)
		errCh <- err
	}()
	go func() {
		_ = a.SetReadDeadline(time.Now().Add(relayIdleTimeout))
		_ = b.SetWriteDeadline(time.Now().Add(relayIdleTimeout))
		_, err := io.Copy(b, a)
		closeWrite(b)
		errCh <- err
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			return
		}
	}
}

func closeWrite(conn net.Conn) {
	type closeWriter interface {
		CloseWrite() error
	}
	if cw, ok := conn.(closeWriter); ok {
		_ = cw.CloseWrite()
	}
}

func loadConfig(path string) (clientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return clientConfig{}, err
	}
	var cfg clientConfig
	if err = json.Unmarshal(data, &cfg); err != nil {
		return clientConfig{}, err
	}
	cfg.normalize()
	return cfg, nil
}

func (c clientConfig) validate() error {
	c.ServerIP = strings.TrimSpace(c.ServerIP)
	c.ServerProtocol = strings.ToLower(strings.TrimSpace(c.ServerProtocol))
	c.AuthKey = strings.TrimSpace(c.AuthKey)
	if len(c.ProxyServers) == 0 {
		return errors.New("proxy_server 至少需要配置一个映射")
	}

	nameSet := make(map[string]struct{}, len(c.ProxyServers))
	portSet := make(map[int]struct{}, len(c.ProxyServers))
	for _, item := range c.ProxyServers {
		if item.Name == "" {
			return errors.New("proxy_server.name 不能为空")
		}
		if item.RemotePort <= 0 {
			return errors.New("proxy_server.remote_port 必须大于 0")
		}
		if item.LocalHost == "" {
			return errors.New("proxy_server.local_host 不能为空")
		}
		if item.LocalPort <= 0 {
			return errors.New("proxy_server.local_port 必须大于 0")
		}
		if _, ok := nameSet[item.Name]; ok {
			return fmt.Errorf("proxy_server.name 重复: %s", item.Name)
		}
		nameSet[item.Name] = struct{}{}
		if _, ok := portSet[item.RemotePort]; ok {
			return fmt.Errorf("proxy_server.remote_port 重复: %d", item.RemotePort)
		}
		portSet[item.RemotePort] = struct{}{}
	}

	switch {
	case c.ServerIP == "":
		return errors.New("server_ip 不能为空")
	case c.ServerPort <= 0:
		return errors.New("server_port 必须大于 0")
	case c.ServerProtocol != "tcp":
		return errors.New("server_protocol 目前仅支持 tcp")
	case c.AuthKey == "":
		return errors.New("auth_key 不能为空")
	default:
		return nil
	}
}

func (c *clientConfig) normalize() {
	c.ServerIP = strings.TrimSpace(c.ServerIP)
	c.ServerProtocol = strings.ToLower(strings.TrimSpace(c.ServerProtocol))
	c.AuthKey = strings.TrimSpace(c.AuthKey)
	for i := range c.ProxyServers {
		c.ProxyServers[i].Name = strings.TrimSpace(c.ProxyServers[i].Name)
		c.ProxyServers[i].LocalHost = strings.TrimSpace(c.ProxyServers[i].LocalHost)
	}
}

func (c *traversalClient) findTarget(name string, remotePort int) (proxyServerConfig, bool) {
	for _, item := range c.cfg.ProxyServers {
		if item.Name == name {
			return item, true
		}
	}
	for _, item := range c.cfg.ProxyServers {
		if item.RemotePort == remotePort {
			return item, true
		}
	}
	return proxyServerConfig{}, false
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
