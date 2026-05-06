package traversal

import (
	"net"
	"time"

	kcp "github.com/xtaci/kcp-go/v5"
)

// listenTCP 创建 TCP 监听。
func listenTCP(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

// listenKCP 创建 KCP 监听，基于 UDP 端口。
func listenKCP(addr string) (net.Listener, error) {
	// 使用默认配置，后续如有需要再通过配置暴露 KCP 参数。
	return kcp.ListenWithOptions(addr, nil, 0, 0)
}

// dialTCP 使用带超时的 TCP 连接。
func dialTCP(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, timeout)
}

// dialKCP 使用 KCP 进行连接。
func dialKCP(addr string) (net.Conn, error) {
	// KCP 底层使用 UDP，不支持 DialTimeout，这里直接阻塞到建立完成或出错。
	return kcp.DialWithOptions(addr, nil, 0, 0)
}

// DialServer 根据协议拨号 traversal 服务端，目前支持 tcp / kcp。
func DialServer(protocol, addr string, timeout time.Duration) (net.Conn, error) {
	switch protocol {
	case "kcp":
		return dialKCP(addr)
	case "tcp", "":
		return dialTCP(addr, timeout)
	default:
		// 理论上在配置校验阶段已保证不会到这里。
		return dialTCP(addr, timeout)
	}
}

