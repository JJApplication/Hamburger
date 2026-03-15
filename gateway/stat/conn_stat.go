package stat

import (
	"net"
	"net/http"
	"sync/atomic"
)

var (
	ConnFrontServer = "front-server"
	ConnBackServer  = "back-server"
	ConnGateway     = "gateway"
)

func handleConnGw(conn net.Conn, state http.ConnState) {
	GetManager().handleConnGw(conn, state)
}

func (m *StatManager) handleConnGw(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		ds, ok := m.connStatGw.Get("new")
		if !ok {
			m.connStatGw.Put("new", new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
		return
	case http.StateActive:
	case http.StateIdle:
	case http.StateHijacked:
	case http.StateClosed:

	}
}

func handleConnFront(conn net.Conn, state http.ConnState) {
	GetManager().handleConnFront(conn, state)
}

func (m *StatManager) handleConnFront(conn net.Conn, state http.ConnState) {

}

func HandleConn(connType string) func(conn net.Conn, state http.ConnState) {
	return func(conn net.Conn, state http.ConnState) {
		switch connType {
		case ConnGateway:
			handleConnGw(conn, state)
		case ConnFrontServer:
			handleConnFront(conn, state)
		}
	}
}
