package stat

import (
	"runtime"
	"testing"
	"time"
)

func TestGCDelta(t *testing.T) {
	previous := runtime.MemStats{
		NumGC:        10,
		NumForcedGC:  2,
		PauseTotalNs: uint64(10 * time.Millisecond),
	}
	current := previous
	current.NumGC = 12
	current.NumForcedGC = 3
	current.PauseTotalNs = uint64(19 * time.Millisecond)
	current.PauseNs[(11+255)%uint32(len(current.PauseNs))] = uint64(2 * time.Millisecond)
	current.PauseNs[(12+255)%uint32(len(current.PauseNs))] = uint64(7 * time.Millisecond)

	cycles, forced, pauseTotal, pauseMax, buckets := gcDelta(previous, current)
	if cycles != 2 || forced != 1 || pauseTotal != uint64(9*time.Millisecond) || pauseMax != uint64(7*time.Millisecond) {
		t.Fatalf("gcDelta counters = cycles %d forced %d total %d max %d", cycles, forced, pauseTotal, pauseMax)
	}
	if buckets[latencyHistogramIndex(2_000)] != 1 || buckets[latencyHistogramIndex(7_000)] != 1 {
		t.Fatalf("gcDelta histogram = %+v", buckets)
	}

	reset := current
	reset.NumGC = 1
	reset.NumForcedGC = 0
	reset.PauseTotalNs = uint64(time.Millisecond)
	cycles, forced, pauseTotal, pauseMax, buckets = gcDelta(current, reset)
	if cycles != 0 || forced != 0 || pauseTotal != 0 || pauseMax != 0 || buckets != ([requestHistogramSize]int64{}) {
		t.Fatalf("gcDelta counter reset = cycles %d forced %d total %d max %d buckets %+v", cycles, forced, pauseTotal, pauseMax, buckets)
	}
}

func TestGCPressurePercent(t *testing.T) {
	pressure, ok := gcPressurePercent(1.5, 2.0, 10.0, 20.0)
	if !ok || pressure != 5 {
		t.Fatalf("gcPressurePercent() = %v, %t; want 5, true", pressure, ok)
	}
	if pressure, ok := gcPressurePercent(2, 1, 10, 20); ok || pressure != 0 {
		t.Fatalf("gcPressurePercent() with counter reset = %v, %t; want 0, false", pressure, ok)
	}
	if pressure, ok := gcPressurePercent(1, 2, 20, 20); ok || pressure != 0 {
		t.Fatalf("gcPressurePercent() with no CPU interval = %v, %t; want 0, false", pressure, ok)
	}
}
