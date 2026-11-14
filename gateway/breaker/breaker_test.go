package breaker

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"Hamburger/internal/config"
	"Hamburger/internal/config/core_config"
	"Hamburger/internal/config/loader"
)

func testBreaker(cfg core_config.BreakConfig) *Breaker {
	return newBreaker(cfg)
}

func TestZeroConfigDoesNotPanicAndOpens(t *testing.T) {
	previous := current.Load()
	previousCfg := loader.Get()
	t.Cleanup(func() {
		current.Store(previous)
		loader.Set(previousCfg)
	})

	loader.Set(&config.Config{})
	InitBreaker()

	if !Get("zero") {
		t.Fatal("first request should be allowed")
	}
	for i := 0; i < core_config.DefaultBreakMaxError; i++ {
		if !Set("zero") {
			t.Fatalf("failure %d was rejected before the circuit opened", i+1)
		}
	}
	if Get("zero") {
		t.Fatal("circuit remained closed after the default failure limit")
	}
	if Set("zero") {
		t.Fatal("open circuit accepted another failure")
	}
	snapshot := findSnapshot(t, Snapshot(), "zero")
	if snapshot.State != StateOpen || snapshot.Expiry.IsZero() {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	ResetAll()
	if Get("zero") == false {
		t.Fatal("reset did not close the circuit")
	}
}

func TestConsecutiveFailuresTripAndStaleGenerationIsIgnored(t *testing.T) {
	b := testBreaker(core_config.BreakConfig{MaxError: 2, Reset: 30, FailureRatio: 0})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b.setNow(func() time.Time { return now })

	first, err := b.Allow("api")
	if err != nil || first == 0 {
		t.Fatalf("allow = %d, %v", first, err)
	}
	b.Report("api", first, false)
	if !b.Get("api") {
		t.Fatal("single failure opened the circuit")
	}

	second, err := b.Allow("api")
	if err != nil {
		t.Fatal(err)
	}
	b.Report("api", second, false)
	if b.Get("api") {
		t.Fatal("two consecutive failures did not open the circuit")
	}
	b.Report("api", first, true)
	if b.Get("api") {
		t.Fatal("stale success closed an open circuit")
	}
}

func TestFailureRatioTripsWithoutConsecutiveLimit(t *testing.T) {
	b := testBreaker(core_config.BreakConfig{
		MaxError:     100,
		Reset:        60,
		FailureRatio: 0.5,
		MinRequests:  4,
	})
	failures := []bool{false, false, true, false}
	for _, success := range failures {
		generation, err := b.Allow("ratio")
		if err != nil {
			t.Fatal(err)
		}
		b.Report("ratio", generation, success)
	}
	if b.Get("ratio") {
		t.Fatal("failure ratio did not open the circuit")
	}
	snapshot := findSnapshot(t, b.Snapshot(), "ratio")
	if snapshot.State != StateOpen {
		t.Fatalf("state = %s", snapshot.State)
	}
}

func TestWindowResetClearsConsecutiveFailures(t *testing.T) {
	b := testBreaker(core_config.BreakConfig{MaxError: 3, Reset: 10, FailureRatio: 0})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b.setNow(func() time.Time { return now })

	for i := 0; i < 2; i++ {
		generation, err := b.Allow("window")
		if err != nil {
			t.Fatal(err)
		}
		b.Report("window", generation, false)
	}
	now = now.Add(10 * time.Second)
	generation, err := b.Allow("window")
	if err != nil {
		t.Fatal(err)
	}
	b.Report("window", generation, false)
	if !b.Get("window") {
		t.Fatal("window reset did not clear consecutive failures")
	}
	snapshot := findSnapshot(t, b.Snapshot(), "window")
	if snapshot.ConsecutiveFailures != 1 {
		t.Fatalf("consecutive failures = %d", snapshot.ConsecutiveFailures)
	}
}

func TestHalfOpenProbeRecoversOrReopens(t *testing.T) {
	b := testBreaker(core_config.BreakConfig{
		MaxError:            1,
		Reset:               5,
		FailureRatio:        0,
		HalfOpenMaxRequests: 1,
		SuccessThreshold:    2,
	})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b.setNow(func() time.Time { return now })

	generation, err := b.Allow("probe")
	if err != nil {
		t.Fatal(err)
	}
	b.Report("probe", generation, false)
	if _, err = b.Allow("probe"); err == nil {
		t.Fatal("open circuit allowed a request")
	}

	now = now.Add(5 * time.Second)
	probe, err := b.Allow("probe")
	if err != nil {
		t.Fatalf("half-open probe was rejected: %v", err)
	}
	if _, err = b.Allow("probe"); err == nil {
		t.Fatal("half-open accepted more probes than configured")
	}
	b.Report("probe", probe, true)
	if stateOf(t, b, "probe") != StateHalfOpen {
		t.Fatal("one success closed the circuit before the threshold")
	}

	probe, err = b.Allow("probe")
	if err != nil {
		t.Fatal(err)
	}
	b.Report("probe", probe, false)
	if stateOf(t, b, "probe") != StateOpen {
		t.Fatal("failed probe did not reopen the circuit")
	}

	now = now.Add(5 * time.Second)
	first, err := b.Allow("probe")
	if err != nil {
		t.Fatal(err)
	}
	b.Report("probe", first, true)
	second, err := b.Allow("probe")
	if err != nil {
		t.Fatal(err)
	}
	b.Report("probe", second, true)
	if stateOf(t, b, "probe") != StateClosed {
		t.Fatal("successful probes did not close the circuit")
	}
}

func TestStatusCodeAndClientCancel(t *testing.T) {
	b := testBreaker(core_config.BreakConfig{MaxError: 1, Reset: 30, TripStatusCodes: []int{418}})
	if b.IsFailureStatus(503) {
		t.Fatal("unconfigured 503 was treated as a failure")
	}
	if !b.IsFailureStatus(418) {
		t.Fatal("configured status was ignored")
	}

	generation, err := b.Allow("status")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b.Observe("status", generation, ctx, nil, context.Canceled)
	if stateOf(t, b, "status") != StateClosed {
		t.Fatal("client cancel opened the circuit")
	}

	generation, err = b.Allow("status")
	if err != nil {
		t.Fatal(err)
	}
	b.Observe("status", generation, context.Background(), &http.Response{StatusCode: http.StatusTeapot}, nil)
	if stateOf(t, b, "status") != StateOpen {
		t.Fatal("configured status code did not open the circuit")
	}
}

func TestDisabledBreakerNeverOpens(t *testing.T) {
	enabled := false
	b := testBreaker(core_config.BreakConfig{Enabled: &enabled, MaxError: 1, Reset: 1})
	for i := 0; i < 5; i++ {
		if !b.Set("off") {
			t.Fatal("disabled breaker rejected a request")
		}
	}
	if len(b.Snapshot()) != 0 {
		t.Fatal("disabled breaker stored circuit state")
	}
}

func TestBreakerConcurrentAccess(t *testing.T) {
	b := testBreaker(core_config.BreakConfig{
		MaxError:            30,
		Reset:               1,
		FailureRatio:        0.8,
		MinRequests:         50,
		HalfOpenMaxRequests: 4,
		SuccessThreshold:    2,
	})
	var wg sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			key := "shared"
			if worker%2 == 0 {
				key = "other"
			}
			for i := 0; i < 100; i++ {
				generation, err := b.Allow(key)
				if err == nil && generation != 0 {
					switch i % 4 {
					case 0:
						b.Ignore(key, generation)
					default:
						b.Report(key, generation, i%3 == 0)
					}
				}
				_ = b.Get(key)
				if i%25 == 0 {
					_ = b.Snapshot()
				}
			}
		}(worker)
	}
	wg.Wait()
	b.ResetTarget("shared")
	b.ResetAll()
}

func findSnapshot(t *testing.T, snapshots []TargetSnapshot, key string) TargetSnapshot {
	t.Helper()
	for _, snapshot := range snapshots {
		if snapshot.Key == key {
			return snapshot
		}
	}
	t.Fatalf("snapshot %s not found in %+v", key, snapshots)
	return TargetSnapshot{}
}

func stateOf(t *testing.T, b *Breaker, key string) State {
	t.Helper()
	return findSnapshot(t, b.Snapshot(), key).State
}
