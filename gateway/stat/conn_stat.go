package stat

import (
	"Hamburger/gateway/stat/model"
	"Hamburger/internal/structure"
	"net"
	"net/http"
	"strings"
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
		m.incrConnState(m.connStatGw, "new")
		m.connHostMap.Put(connKey(conn), "")
		return
	case http.StateActive:
		m.incrConnState(m.connStatGw, "active")
		return
	case http.StateIdle:
		m.incrConnState(m.connStatGw, "idle")
		m.incrDomainConnStateByConn(conn, "idle", false)
		return
	case http.StateHijacked:
		m.incrConnState(m.connStatGw, "hijacked")
		m.incrDomainConnStateByConn(conn, "hijacked", true)
		return
	case http.StateClosed:
		m.incrConnState(m.connStatGw, "closed")
		m.incrDomainConnStateByConn(conn, "closed", true)
		return
	}
}

func handleConnFront(conn net.Conn, state http.ConnState) {
	GetManager().handleConnFront(conn, state)
}

func (m *StatManager) handleConnFront(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		m.incrConnState(m.connStatFront, "new")
		return
	case http.StateActive:
		m.incrConnState(m.connStatFront, "active")
		return
	case http.StateIdle:
		m.incrConnState(m.connStatFront, "idle")
		return
	case http.StateHijacked:
		m.incrConnState(m.connStatFront, "hijacked")
		return
	case http.StateClosed:
		m.incrConnState(m.connStatFront, "closed")
		return
	}
}

func BindConnHost(remoteAddr, host string) {
	GetManager().bindConnHost(remoteAddr, host)
}

func (m *StatManager) bindConnHost(remoteAddr, host string) {
	remoteAddr = strings.TrimSpace(remoteAddr)
	host = normalizeConnHost(host)
	if remoteAddr == "" || host == "" {
		return
	}
	prevHost, ok := m.connHostMap.Get(remoteAddr)
	m.connHostMap.Put(remoteAddr, host)
	if !ok || prevHost == "" {
		m.incrDomainConnState(host, "active")
	}
}

func (m *StatManager) incrConnState(connStat *structure.Map[*int64], state string) {
	ds, ok := connStat.Get(state)
	if !ok {
		connStat.Put(state, new(int64))
		return
	}
	atomic.AddInt64(ds, 1)
}

func (m *StatManager) incrDomainConnStateByConn(conn net.Conn, state string, clear bool) {
	key := connKey(conn)
	if key == "" {
		return
	}
	host, ok := m.connHostMap.Get(key)
	if ok && host != "" {
		m.incrDomainConnState(host, state)
	}
	if clear {
		m.connHostMap.Delete(key)
	}
}

func (m *StatManager) incrDomainConnState(host, state string) {
	host = normalizeConnHost(host)
	if host == "" {
		return
	}
	hostConnStat := m.getOrInitDomainConnStat(host)
	ds, ok := hostConnStat.Get(state)
	if !ok {
		hostConnStat.Put(state, new(int64))
		return
	}
	atomic.AddInt64(ds, 1)
}

func (m *StatManager) getOrInitDomainConnStat(host string) *structure.Map[*int64] {
	domainConnStat, ok := m.domainConnStat.Get(host)
	if ok {
		return domainConnStat
	}
	newStat := structure.NewMap[*int64]()
	newStat.Put("active", new(int64))
	newStat.Put("idle", new(int64))
	newStat.Put("hijacked", new(int64))
	newStat.Put("closed", new(int64))
	m.domainConnStat.Put(host, newStat)
	return newStat
}

func connKey(conn net.Conn) string {
	if conn == nil || conn.RemoteAddr() == nil {
		return ""
	}
	return conn.RemoteAddr().String()
}

func normalizeConnHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
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
