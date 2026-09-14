package stat

import (
	"net/http"
	"strconv"
	"sync"
	"testing"
)

func TestConnectionTrackerTracksCurrentStateAndTotals(t *testing.T) {
	tracker := newConnectionTracker()

	tracker.onHTTPState("conn-1", http.StateNew)
	tracker.onHTTPState("conn-1", http.StateActive)
	tracker.onHTTPState("conn-1", http.StateIdle)
	snapshot := tracker.snapshot()
	if snapshot.New != 1 || snapshot.Active != 0 || snapshot.Idle != 1 || snapshot.Closed != 0 {
		t.Fatalf("unexpected idle snapshot: %+v", snapshot)
	}

	tracker.onHTTPState("conn-1", http.StateActive)
	tracker.onHTTPState("conn-1", http.StateActive)
	snapshot = tracker.snapshot()
	if snapshot.Active != 1 || snapshot.Idle != 0 {
		t.Fatalf("repeated active transition changed gauges: %+v", snapshot)
	}

	tracker.onHTTPState("conn-1", http.StateClosed)
	tracker.onHTTPState("conn-1", http.StateClosed)
	snapshot = tracker.snapshot()
	if snapshot.Active != 0 || snapshot.Idle != 0 || snapshot.Closed != 1 {
		t.Fatalf("unexpected closed snapshot: %+v", snapshot)
	}
}

func TestConnectionTrackerTracksHijackOnce(t *testing.T) {
	tracker := newConnectionTracker()
	tracker.onHTTPState("conn-1", http.StateNew)
	tracker.onHTTPState("conn-1", http.StateIdle)
	tracker.onHTTPState("conn-1", http.StateHijacked)
	tracker.onHTTPState("conn-1", http.StateHijacked)

	snapshot := tracker.snapshot()
	if snapshot.New != 1 || snapshot.Idle != 0 || snapshot.Hijacked != 1 || snapshot.Closed != 0 {
		t.Fatalf("unexpected hijack snapshot: %+v", snapshot)
	}
}

func TestConnectionTrackerTracksHTTP3RequestConcurrency(t *testing.T) {
	tracker := newConnectionTracker()
	tracker.openIdle("quic-1")
	tracker.requestStart("quic-1")
	tracker.requestStart("quic-1")

	snapshot := tracker.snapshot()
	if snapshot.New != 1 || snapshot.Active != 1 || snapshot.Idle != 0 {
		t.Fatalf("unexpected active HTTP/3 snapshot: %+v", snapshot)
	}

	tracker.requestEnd("quic-1")
	snapshot = tracker.snapshot()
	if snapshot.Active != 1 || snapshot.Idle != 0 {
		t.Fatalf("connection became idle while a request remained: %+v", snapshot)
	}

	tracker.requestEnd("quic-1")
	snapshot = tracker.snapshot()
	if snapshot.Active != 0 || snapshot.Idle != 1 {
		t.Fatalf("connection did not return to idle: %+v", snapshot)
	}

	tracker.onHTTPState("quic-1", http.StateClosed)
	tracker.onHTTPState("quic-1", http.StateClosed)
	snapshot = tracker.snapshot()
	if snapshot.Active != 0 || snapshot.Idle != 0 || snapshot.Closed != 1 {
		t.Fatalf("unexpected closed HTTP/3 snapshot: %+v", snapshot)
	}
}

func TestConnectionTrackerConcurrentConnections(t *testing.T) {
	tracker := newConnectionTracker()
	const count = 256
	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		key := "conn-" + strconv.Itoa(i)
		go func() {
			defer wg.Done()
			tracker.onHTTPState(key, http.StateNew)
			tracker.onHTTPState(key, http.StateActive)
			tracker.onHTTPState(key, http.StateIdle)
			tracker.onHTTPState(key, http.StateClosed)
		}()
	}
	wg.Wait()

	snapshot := tracker.snapshot()
	if snapshot.New != count || snapshot.Active != 0 || snapshot.Idle != 0 || snapshot.Closed != count {
		t.Fatalf("unexpected concurrent snapshot: %+v", snapshot)
	}
}
