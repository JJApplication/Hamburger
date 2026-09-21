package stat

import (
	runtimeDomains "Hamburger/gateway/runtime"
	"Hamburger/gateway/stat/model"
	"Hamburger/internal/structure"
	"Hamburger/internal/utils"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	ConnFrontServer = "front-server"
	ConnBackServer  = "back-server"
	ConnGateway     = "gateway"
)

type trackedConnPhase uint8

const (
	trackedConnNew trackedConnPhase = iota + 1
	trackedConnActive
	trackedConnIdle
)

type trackedConn struct {
	phase    trackedConnPhase
	inflight int
}

// connectionTracker keeps the current state of each accepted connection while
// retaining process-lifetime event totals. The mutex makes state transitions
// and snapshots consistent, which is important for HTTP/2 and HTTP/3 where
// callbacks for one connection can arrive concurrently.
type connectionTracker struct {
	mu       sync.Mutex
	entries  map[string]trackedConn
	new      int64
	active   int64
	idle     int64
	hijacked int64
	closed   int64
}

func newConnectionTracker() *connectionTracker {
	return &connectionTracker{entries: make(map[string]trackedConn)}
}

func (t *connectionTracker) onHTTPState(key string, state http.ConnState) {
	if t == nil || key == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	switch state {
	case http.StateNew:
		if _, exists := t.entries[key]; exists {
			return
		}
		t.entries[key] = trackedConn{phase: trackedConnNew}
		t.new++
	case http.StateActive:
		t.transitionLocked(key, trackedConnActive)
	case http.StateIdle:
		t.transitionLocked(key, trackedConnIdle)
	case http.StateHijacked:
		t.finishLocked(key, true)
	case http.StateClosed:
		t.finishLocked(key, false)
	}
}

func (t *connectionTracker) openIdle(key string) {
	if t == nil || key == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.entries[key]; exists {
		return
	}
	t.entries[key] = trackedConn{phase: trackedConnIdle}
	t.new++
	t.idle++
}

func (t *connectionTracker) requestStart(key string) {
	if t == nil || key == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, exists := t.entries[key]
	if !exists {
		return
	}
	entry.inflight++
	if entry.phase == trackedConnIdle {
		t.idle--
		t.active++
		entry.phase = trackedConnActive
	}
	t.entries[key] = entry
}

func (t *connectionTracker) requestEnd(key string) {
	if t == nil || key == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, exists := t.entries[key]
	if !exists || entry.inflight == 0 {
		return
	}
	entry.inflight--
	if entry.inflight == 0 && entry.phase == trackedConnActive {
		t.active--
		t.idle++
		entry.phase = trackedConnIdle
	}
	t.entries[key] = entry
}

func (t *connectionTracker) transitionLocked(key string, phase trackedConnPhase) {
	entry, exists := t.entries[key]
	if !exists {
		entry = trackedConn{phase: phase}
		t.entries[key] = entry
		if phase == trackedConnActive {
			t.active++
		} else if phase == trackedConnIdle {
			t.idle++
		}
		return
	}
	if entry.phase == phase {
		return
	}
	switch entry.phase {
	case trackedConnActive:
		t.active--
	case trackedConnIdle:
		t.idle--
	}
	switch phase {
	case trackedConnActive:
		t.active++
	case trackedConnIdle:
		t.idle++
	}
	entry.phase = phase
	t.entries[key] = entry
}

func (t *connectionTracker) finishLocked(key string, hijacked bool) {
	entry, exists := t.entries[key]
	if !exists {
		return
	}
	switch entry.phase {
	case trackedConnActive:
		t.active--
	case trackedConnIdle:
		t.idle--
	}
	delete(t.entries, key)
	if hijacked {
		t.hijacked++
	} else {
		t.closed++
	}
}

type connectionSnapshot struct {
	New      int64
	Active   int64
	Idle     int64
	Hijacked int64
	Closed   int64
}

func (t *connectionTracker) snapshot() connectionSnapshot {
	if t == nil {
		return connectionSnapshot{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return connectionSnapshot{
		New: t.new, Active: t.active, Idle: t.idle,
		Hijacked: t.hijacked, Closed: t.closed,
	}
}

func handleConnGw(conn net.Conn, state http.ConnState) {
	GetManager().handleConnGw(conn, state)
}

func (m *StatManager) handleConnGw(conn net.Conn, state http.ConnState) {
	key := trackerConnKey(conn)
	m.connStatGw.onHTTPState(key, state)
	switch state {
	case http.StateNew:
		m.connDomainMu.Lock()
		m.connDomains.Put(connKey(conn), map[string]struct{}{})
		m.connDomainMu.Unlock()
		return
	case http.StateActive:
		return
	case http.StateIdle:
		m.incrDomainConnStateByConn(conn, "idle", false)
		return
	case http.StateHijacked:
		m.incrDomainConnStateByConn(conn, "hijacked", true)
		return
	case http.StateClosed:
		m.incrDomainConnStateByConn(conn, "closed", true)
		return
	}
}

func handleConnFront(conn net.Conn, state http.ConnState) {
	GetManager().handleConnFront(conn, state)
}

func (m *StatManager) handleConnFront(conn net.Conn, state http.ConnState) {
	m.connStatFront.onHTTPState(trackerConnKey(conn), state)
}

func trackerConnKey(conn net.Conn) string {
	if conn == nil {
		return ""
	}
	v := reflect.ValueOf(conn)
	switch v.Kind() {
	case reflect.Pointer, reflect.UnsafePointer:
		return fmt.Sprintf("%T:%x", conn, v.Pointer())
	default:
		return fmt.Sprintf("%T:%s", conn, conn.RemoteAddr())
	}
}

// FrontConnOpened, FrontConnRequestStart, FrontConnRequestEnd and
// FrontConnClosed are used by the HTTP/3 server, which does not expose
// net/http.ConnState callbacks.
func FrontConnOpened(key string)       { GetManager().connStatFront.openIdle(key) }
func FrontConnRequestStart(key string) { GetManager().connStatFront.requestStart(key) }
func FrontConnRequestEnd(key string)   { GetManager().connStatFront.requestEnd(key) }
func FrontConnClosed(key string)       { GetManager().connStatFront.onHTTPState(key, http.StateClosed) }

func BindConnHost(remoteAddr, host string) {
	GetManager().bindConnHost(remoteAddr, host)
}

func (m *StatManager) bindConnHost(remoteAddr, host string) {
	remoteAddr = strings.TrimSpace(remoteAddr)
	host = normalizeConnHost(host)
	if remoteAddr == "" || host == "" {
		return
	}
	m.connDomainMu.Lock()
	defer m.connDomainMu.Unlock()
	domains, ok := m.connDomains.Get(remoteAddr)
	if !ok || domains == nil {
		domains = map[string]struct{}{}
	}
	if _, associated := domains[host]; !associated {
		domains[host] = struct{}{}
		m.connDomains.Put(remoteAddr, domains)
		m.incrDomainConnState(host, "active")
		m.domainConnState.Put(domainStateKey(remoteAddr, host), "active")
		return
	}
	stateKey := domainStateKey(remoteAddr, host)
	if previous, hasPrevious := m.domainConnState.Get(stateKey); hasPrevious && previous == "idle" {
		if states, found := m.domainConnStat.Get(host); found {
			if value, foundState := states.Get("idle"); foundState && value != nil {
				atomic.AddInt64(value, -1)
			}
		}
		m.incrDomainConnState(host, "active")
		m.domainConnState.Put(stateKey, "active")
	}
}

func (m *StatManager) incrDomainConnStateByConn(conn net.Conn, state string, clear bool) {
	key := connKey(conn)
	if key == "" {
		return
	}
	m.connDomainMu.Lock()
	defer m.connDomainMu.Unlock()
	domains, ok := m.connDomains.Get(key)
	if ok {
		for host := range domains {
			stateKey := domainStateKey(key, host)
			if previous, hasPrevious := m.domainConnState.Get(stateKey); hasPrevious && previous != state {
				if states, found := m.domainConnStat.Get(host); found {
					if value, foundState := states.Get(previous); foundState && value != nil && (previous == "active" || previous == "idle") {
						atomic.AddInt64(value, -1)
					}
				}
			}
			m.incrDomainConnState(host, state)
			if state == "idle" {
				m.domainConnState.Put(stateKey, "idle")
			} else {
				m.domainConnState.Delete(stateKey)
			}
		}
	}
	if clear {
		m.connDomains.Delete(key)
	}
}

func domainStateKey(connection, host string) string { return connection + "\x00" + host }

func (m *StatManager) incrDomainConnState(host, state string) {
	host = normalizeConnHost(host)
	if host == "" {
		return
	}
	hostConnStat := m.getOrInitDomainConnStat(host)
	ds, ok := hostConnStat.Get(state)
	if !ok {
		hostConnStat.Put(state, new(int64))
	} else {
		atomic.AddInt64(ds, 1)
	}
	if state == "active" || state == "idle" {
		active, _ := hostConnStat.Get("active")
		idle, _ := hostConnStat.Get("idle")
		current := int64(0)
		if active != nil {
			current += atomic.LoadInt64(active)
		}
		if idle != nil {
			current += atomic.LoadInt64(idle)
		}
		m.updateDomainPeak(host, current)
	}
}

func (m *StatManager) updateDomainPeak(host string, active int64) {
	peak, ok := m.domainConnPeak.Get(host)
	if !ok {
		peak = new(int64)
		m.domainConnPeak.Put(host, peak)
	}
	for {
		current := atomic.LoadInt64(peak)
		if active <= current || atomic.CompareAndSwapInt64(peak, current, active) {
			return
		}
	}
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
	if normalized, _, err := net.SplitHostPort(host); err == nil {
		host = normalized
	} else if strings.Count(host, ":") == 1 {
		if idx := strings.LastIndex(host, ":"); idx > 0 {
			host = host[:idx]
		}
	}
	host = strings.Trim(host, "[]")
	host = strings.ToLower(host)
	if item, ok := runtimeDomains.GetDomain2Service(host); ok && utils.IsDomainRegex(item.ServiceDomain) {
		return strings.TrimSpace(item.ServiceDomain)
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
	s := GetManager().connStatGw.snapshot()
	return model.GatewayConnModel{New: s.New, Active: s.Active, Idle: s.Idle, Hijacked: s.Hijacked, Closed: s.Closed}
}

func GetFrontConn() model.FrontConnModel {
	s := GetManager().connStatFront.snapshot()
	return model.FrontConnModel{New: s.New, Active: s.Active, Idle: s.Idle, Hijacked: s.Hijacked, Closed: s.Closed}
}

// GetDomainConnSnapshot returns a consistent, bounded snapshot for management
// dashboards. Keys originate from configured/runtime domains and are not
// created from arbitrary request Host values.
func GetDomainConnSnapshot() map[string]map[string]int64 {
	m := GetManager()
	result := map[string]map[string]int64{}
	if m == nil || m.domainConnStat == nil {
		return result
	}
	m.domainConnStat.Range(func(host string, states *structure.Map[*int64]) bool {
		item := map[string]int64{}
		for _, state := range []string{"active", "idle", "hijacked", "closed"} {
			if value, ok := states.Get(state); ok && value != nil {
				item[state] = atomic.LoadInt64(value)
			}
		}
		if peak, ok := m.domainConnPeak.Get(host); ok && peak != nil {
			item["peak"] = atomic.LoadInt64(peak)
		}
		result[host] = item
		return true
	})
	return result
}
