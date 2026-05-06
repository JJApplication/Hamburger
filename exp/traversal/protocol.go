package traversal

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const (
	// MessageTypeRegister 客户端注册控制连接消息。
	MessageTypeRegister = "register"
	// MessageTypeRegisterAck 服务端注册响应消息。
	MessageTypeRegisterAck = "register_ack"
	// MessageTypeOpen 服务端通知客户端创建数据通道消息。
	MessageTypeOpen = "open"
	// MessageTypeJoin 客户端绑定数据通道消息。
	MessageTypeJoin = "join"
	// MessageTypePing 控制通道心跳探测。
	MessageTypePing = "ping"
	// MessageTypePong 控制通道心跳响应。
	MessageTypePong = "pong"
	// MessageTypeError 错误响应消息。
	MessageTypeError = "error"
)

// Message 穿透控制面消息。
type Message struct {
	// Type 消息类型。
	Type string `json:"type"`
	// AuthKey 认证密钥。
	AuthKey string `json:"auth_key,omitempty"`
	// ConnID 连接标识。
	ConnID string `json:"conn_id,omitempty"`
	// MappingName 映射服务名。
	MappingName string `json:"mapping_name,omitempty"`
	// RemotePort 映射对应的服务端对外端口。
	RemotePort int `json:"remote_port,omitempty"`
	// ProxyServers 客户端申请的服务端对外代理端口列表。
	ProxyServers []ProxyServerBinding `json:"proxy_server,omitempty"`
	// OK 表示操作是否成功。
	OK bool `json:"ok,omitempty"`
	// Message 文本说明。
	Message string `json:"message,omitempty"`
}

// ProxyServerBinding 客户端申请的服务端端口映射。
type ProxyServerBinding struct {
	// Name 映射服务名。
	Name string `json:"name,omitempty"`
	// RemotePort 服务端对外端口。
	RemotePort int `json:"remote_port,omitempty"`
}

// ReadMessage 读取一条换行分隔的 JSON 消息。
func ReadMessage(r *bufio.Reader) (Message, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return Message{}, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return Message{}, errors.New("empty message")
	}
	var msg Message
	if err = json.Unmarshal([]byte(line), &msg); err != nil {
		return Message{}, err
	}
	return msg, nil
}

// WriteMessage 写入一条换行分隔的 JSON 消息。
func WriteMessage(w io.Writer, msg Message) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err = w.Write(raw); err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}
