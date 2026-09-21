package stat

import (
	"Hamburger/internal/config"
	"Hamburger/internal/structure"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

type snapshotConn struct{ address net.Addr }

func (c snapshotConn) Read([]byte) (int, error)           { return 0, net.ErrClosed }
func (c snapshotConn) Write([]byte) (int, error)          { return 0, net.ErrClosed }
func (c snapshotConn) Close() error                       { return nil }
func (c snapshotConn) LocalAddr() net.Addr                { return c.address }
func (c snapshotConn) RemoteAddr() net.Addr               { return c.address }
func (c snapshotConn) SetDeadline(_ time.Time) error      { return nil }
func (c snapshotConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c snapshotConn) SetWriteDeadline(_ time.Time) error { return nil }

func TestDomainConnectionTracksSharedConnectionAssociations(t *testing.T) {
	m := NewStatManager(&config.Config{})
	conn := snapshotConn{address: staticAddr("10.0.0.1:1000")}
	m.bindConnHost(conn.RemoteAddr().String(), "A.example")
	m.bindConnHost(conn.RemoteAddr().String(), "B.example")
	snapshot := m.domainSnapshotForTest()
	if snapshot["a.example"]["active"] != 1 || snapshot["b.example"]["active"] != 1 {
		t.Fatalf("shared connection was not associated with both domains: %#v", snapshot)
	}
	m.incrDomainConnStateByConn(conn, "closed", true)
	snapshot = m.domainSnapshotForTest()
	if snapshot["a.example"]["closed"] != 1 || snapshot["b.example"]["closed"] != 1 {
		t.Fatalf("shared connection close was not released per domain: %#v", snapshot)
	}
}

type staticAddr string

func (a staticAddr) Network() string { return "tcp" }
func (a staticAddr) String() string  { return string(a) }

func (m *StatManager) domainSnapshotForTest() map[string]map[string]int64 {
	return GetDomainConnSnapshotForManager(m)
}

func GetDomainConnSnapshotForManager(m *StatManager) map[string]map[string]int64 {
	result := map[string]map[string]int64{}
	m.domainConnStat.Range(func(host string, states *structure.Map[*int64]) bool {
		item := map[string]int64{}
		for _, state := range []string{"active", "idle", "closed"} {
			if value, ok := states.Get(state); ok && value != nil {
				item[state] = atomic.LoadInt64(value)
			}
		}
		result[host] = item
		return true
	})
	return result
}
