package trojan

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"Hamburger/exp/trojan/common"
	"Hamburger/exp/trojan/core/statistic"
	"Hamburger/exp/trojan/core/statistic/memory"
	"Hamburger/exp/trojan/core/tunnel"
	"Hamburger/exp/trojan/core/tunnel/mux"
	"Hamburger/exp/trojan/log"
)

const (
	MaxPacketSize = 1024 * 8
)

const (
	Connect   tunnel.Command = 1
	Associate tunnel.Command = 3
	Mux       tunnel.Command = 0x7f
)

type OutboundConn struct {
	// WARNING: do not change the order of these fields.
	// 64-bit fields that use `sync/atomic` package functions
	// must be 64-bit aligned on 32-bit systems.
	// Reference: https://github.com/golang/go/issues/599
	// Solution: https://github.com/golang/go/issues/11891#issuecomment-433623786
	sent uint64
	recv uint64

	metadata          *tunnel.Metadata
	user              statistic.User
	headerWrittenOnce sync.Once
	net.Conn
}

func (c *OutboundConn) Metadata() *tunnel.Metadata {
	return c.metadata
}

func (c *OutboundConn) WriteHeader(payload []byte) (bool, error) {
	var err error
	written := false
	c.headerWrittenOnce.Do(func() {
		hash := c.user.Hash()
		var headerBuf [512]byte
		header := headerBuf[:0]
		header = append(header, hash...)
		header = append(header, '\r', '\n')
		header = append(header, byte(c.metadata.Command))
		header, err = appendAddressBytes(header, c.metadata.Address)
		if err != nil {
			return
		}
		c.metadata.Address.NetworkType = "tcp"
		header = append(header, '\r', '\n')
		if payload != nil {
			buffers := net.Buffers{header, payload}
			_, err = buffers.WriteTo(c.Conn)
		} else {
			_, err = c.Conn.Write(header)
		}
		if err == nil {
			written = true
		}
	})
	return written, err
}

func (c *OutboundConn) Write(p []byte) (int, error) {
	written, err := c.WriteHeader(p)
	if err != nil {
		return 0, common.NewError("trojan failed to flush header with payload").Base(err)
	}
	if written {
		return len(p), nil
	}
	n, err := c.Conn.Write(p)
	c.user.AddTraffic(n, 0)
	atomic.AddUint64(&c.sent, uint64(n))
	return n, err
}

func (c *OutboundConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.user.AddTraffic(0, n)
	atomic.AddUint64(&c.recv, uint64(n))
	return n, err
}

func (c *OutboundConn) Close() error {
	log.Info("connection to", c.metadata, "closed", "sent:", common.HumanFriendlyTraffic(atomic.LoadUint64(&c.sent)), "recv:", common.HumanFriendlyTraffic(atomic.LoadUint64(&c.recv)))
	return c.Conn.Close()
}

type Client struct {
	underlay tunnel.Client
	user     statistic.User
	ctx      context.Context
	cancel   context.CancelFunc
}

func (c *Client) Close() error {
	c.cancel()
	return c.underlay.Close()
}

func (c *Client) DialConn(addr *tunnel.Address, overlay tunnel.Tunnel) (tunnel.Conn, error) {
	conn, err := c.underlay.DialConn(addr, &Tunnel{})
	if err != nil {
		return nil, err
	}
	newConn := &OutboundConn{
		Conn: conn,
		user: c.user,
		metadata: &tunnel.Metadata{
			Command: Connect,
			Address: addr,
		},
	}
	if _, ok := overlay.(*mux.Tunnel); ok {
		newConn.metadata.Command = Mux
	}

	go func(newConn *OutboundConn) {
		// if the trojan header is still buffered after 100 ms, the client may expect data from the server
		// so we flush the trojan header
		time.Sleep(time.Millisecond * 100)
		newConn.WriteHeader(nil)
	}(newConn)
	return newConn, nil
}

func (c *Client) DialPacket(tunnel.Tunnel) (tunnel.PacketConn, error) {
	fakeAddr := &tunnel.Address{
		DomainName:  "UDP_CONN",
		AddressType: tunnel.DomainName,
	}
	conn, err := c.underlay.DialConn(fakeAddr, &Tunnel{})
	if err != nil {
		return nil, err
	}
	return &PacketConn{
		Conn: &OutboundConn{
			Conn: conn,
			user: c.user,
			metadata: &tunnel.Metadata{
				Command: Associate,
				Address: fakeAddr,
			},
		},
	}, nil
}

func NewClient(ctx context.Context, client tunnel.Client) (*Client, error) {
	ctx, cancel := context.WithCancel(ctx)
	auth, err := statistic.NewAuthenticator(ctx, memory.Name)
	if err != nil {
		cancel()
		return nil, err
	}

	var user statistic.User
	for _, u := range auth.ListUsers() {
		user = u
		break
	}
	if user == nil {
		cancel()
		return nil, common.NewError("no valid user found")
	}

	log.Debug("trojan client created")
	return &Client{
		underlay: client,
		ctx:      ctx,
		user:     user,
		cancel:   cancel,
	}, nil
}
