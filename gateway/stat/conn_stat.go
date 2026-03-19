package stat

import (
	"Hamburger/gateway/stat/model"
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
		ds, ok := m.connStatGw.Get("active")
		if !ok {
			m.connStatGw.Put("active", new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
		return
	case http.StateIdle:
		ds, ok := m.connStatGw.Get("idle")
		if !ok {
			m.connStatGw.Put("idle", new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
		return
	case http.StateHijacked:
		ds, ok := m.connStatGw.Get("hijacked")
		if !ok {
			m.connStatGw.Put("hijacked", new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
		return
	case http.StateClosed:
		ds, ok := m.connStatGw.Get("closed")
		if !ok {
			m.connStatGw.Put("closed", new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
		return
	}
}

func handleConnFront(conn net.Conn, state http.ConnState) {
	GetManager().handleConnFront(conn, state)
}

func (m *StatManager) handleConnFront(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		ds, ok := m.connStatFront.Get("new")
		if !ok {
			m.connStatFront.Put("new", new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
		return
	case http.StateActive:
		ds, ok := m.connStatFront.Get("active")
		if !ok {
			m.connStatFront.Put("active", new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
		return
	case http.StateIdle:
		ds, ok := m.connStatFront.Get("idle")
		if !ok {
			m.connStatFront.Put("idle", new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
		return
	case http.StateHijacked:
		ds, ok := m.connStatFront.Get("hijacked")
		if !ok {
			m.connStatFront.Put("hijacked", new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
		return
	case http.StateClosed:
		ds, ok := m.connStatFront.Get("closed")
		if !ok {
			m.connStatFront.Put("closed", new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
		return
	}
}

// HandleConn 记录服务器内部连接数
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

func GetGatewayConn() model.GatewayConnModel {
	tmp := model.GatewayConnModel{}
	if t, ok := GetManager().connStatGw.Get("new"); ok {
		tmp.New = *t
	}
	if t, ok := GetManager().connStatFront.Get("active"); ok {
		tmp.Active = *t
	}
	if t, ok := GetManager().connStatFront.Get("idle"); ok {
		tmp.Idle = *t
	}
	if t, ok := GetManager().connStatFront.Get("hijacked"); ok {
		tmp.Hijacked = *t
	}
	if t, ok := GetManager().connStatFront.Get("closed"); ok {
		tmp.Closed = *t
	}

	return tmp
}

func GetFrontConn() model.FrontConnModel {
	tmp := model.FrontConnModel{}
	if t, ok := GetManager().connStatFront.Get("new"); ok {
		tmp.New = *t
	}
	if t, ok := GetManager().connStatFront.Get("active"); ok {
		tmp.Active = *t
	}
	if t, ok := GetManager().connStatFront.Get("idle"); ok {
		tmp.Idle = *t
	}
	if t, ok := GetManager().connStatFront.Get("hijacked"); ok {
		tmp.Hijacked = *t
	}
	if t, ok := GetManager().connStatFront.Get("closed"); ok {
		tmp.Closed = *t
	}

	return tmp
}
