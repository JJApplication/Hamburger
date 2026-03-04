package server

import (
	"github.com/lesismal/nbio/lmux"
	"net"
	"sync"
)

type nbioMuxListener struct {
	base *lmux.ChanListener
	mux  *lmux.ListenerMux
}

func (l *nbioMuxListener) Accept() (net.Conn, error) {
	conn, err := l.base.Accept()
	if err != nil {
		return nil, err
	}
	return &nbioConn{
		Conn:    conn,
		onClose: l.base.Decrease,
	}, nil
}

func (l *nbioMuxListener) Close() error {
	if l.mux != nil {
		l.mux.Stop()
		return nil
	}
	return nil
}

func (l *nbioMuxListener) Addr() net.Addr {
	return l.base.Addr()
}

type nbioConn struct {
	net.Conn
	onClose func()
	once    sync.Once
}

func (c *nbioConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() {
		if c.onClose != nil {
			c.onClose()
		}
	})
	return err
}
