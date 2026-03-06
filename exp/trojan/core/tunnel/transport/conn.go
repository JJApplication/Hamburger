package transport

import (
	"net"

	"Hamburger/exp/trojan/core/tunnel"
)

type Conn struct {
	net.Conn
}

func (c *Conn) Metadata() *tunnel.Metadata {
	return nil
}
