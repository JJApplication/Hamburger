package stat

import (
	"context"
	"math"
	"os"
	"runtime"
	runtimeMetrics "runtime/metrics"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// ResourceReading contains one point-in-time sample. Nil means the platform
// did not expose that metric; a non-nil zero is a valid first/reset sample.
type ResourceReading struct {
	Timestamp time.Time

	SystemCPUPercent    *float64
	SystemMemoryPercent *float64

	ProcessCPUPercent    *float64
	ProcessMemoryBytes   *int64
	ProcessMemoryPercent *float64

	SystemNetworkRXBytes  *int64
	SystemNetworkTXBytes  *int64
	SystemDiskReadBytes   *int64
	SystemDiskWriteBytes  *int64
	ProcessDiskReadBytes  *int64
	ProcessDiskWriteBytes *int64

	GCCycles          *int64
	GCForcedCycles    *int64
	GCPauseTotalNS    *int64
	GCPauseMaxNS      *int64
	GCPauseBuckets    [requestHistogramSize]int64
	GCPressurePercent *float64
}

type resourceCapabilities struct {
	SystemCPU     bool
	SystemMemory  bool
	SystemNetwork bool
	SystemDiskIO  bool
	ProcessCPU    bool
	ProcessMemory bool
	ProcessDiskIO bool
	RuntimeGC     bool
}

func (c resourceCapabilities) mapValue() map[string]bool {
	return map[string]bool{
		"system_cpu":      c.SystemCPU,
		"system_memory":   c.SystemMemory,
		"system_network":  c.SystemNetwork,
		"system_disk_io":  c.SystemDiskIO,
		"process_cpu":     c.ProcessCPU,
		"process_memory":  c.ProcessMemory,
		"process_disk_io": c.ProcessDiskIO,
		"runtime_gc":      c.RuntimeGC,
		"program_traffic": true,
	}
}

type resourceSampler struct {
	process *process.Process

	mu            sync.Mutex
	lastWall      time.Time
	lastSystem    *cpu.TimesStat
	lastProcess   *cpu.TimesStat
	lastNetwork   *net.IOCountersStat
	lastDisk      *disk.IOCountersStat
	networkReady  bool
	diskReady     bool
	lastProcRead  uint64
	lastProcWrite uint64
	procIOReady   bool
	lastGC        runtime.MemStats
	gcReady       bool
	lastGCCPU     float64
	lastTotalCPU  float64
	gcCPUReady    bool
	caps          resourceCapabilities
}

func newResourceSampler() *resourceSampler {
	p, _ := process.NewProcess(int32(os.Getpid()))
	return &resourceSampler{process: p, caps: resourceCapabilities{RuntimeGC: true}}
}

func (s *resourceSampler) capabilities() resourceCapabilities {
	if s == nil {
		return resourceCapabilities{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.caps
}

func (s *resourceSampler) sample(ctx context.Context, now time.Time) ResourceReading {
	if s == nil {
		return ResourceReading{Timestamp: now}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	reading := ResourceReading{Timestamp: now.UTC()}

	var gcStats runtime.MemStats
	runtime.ReadMemStats(&gcStats)
	reading.GCCycles = int64Ptr(0)
	reading.GCForcedCycles = int64Ptr(0)
	reading.GCPauseTotalNS = int64Ptr(0)
	reading.GCPauseMaxNS = int64Ptr(0)
	gcCPU, totalCPU, metricsOK := readRuntimeGCMetrics()
	if metricsOK {
		if s.gcCPUReady {
			if pressure, ok := gcPressurePercent(s.lastGCCPU, gcCPU, s.lastTotalCPU, totalCPU); ok {
				reading.GCPressurePercent = float64Ptr(pressure)
			}
		}
		s.lastGCCPU = gcCPU
		s.lastTotalCPU = totalCPU
		s.gcCPUReady = true
	} else {
		// Keep a compatibility fallback for runtimes that do not expose the
		// runtime/metrics CPU classes.
		reading.GCPressurePercent = float64Ptr(clampPercent(gcStats.GCCPUFraction * 100))
	}
	s.caps.RuntimeGC = true
	if s.gcReady {
		cycles, forced, pauseTotal, pauseMax, pauseBuckets := gcDelta(s.lastGC, gcStats)
		reading.GCCycles = int64Ptr(cycles)
		reading.GCForcedCycles = int64Ptr(forced)
		reading.GCPauseTotalNS = int64Ptr(safeUint64ToInt64(pauseTotal))
		reading.GCPauseMaxNS = int64Ptr(safeUint64ToInt64(pauseMax))
		reading.GCPauseBuckets = pauseBuckets
	}
	s.lastGC = gcStats
	s.gcReady = true

	if times, err := cpu.TimesWithContext(ctx, false); err == nil && len(times) > 0 {
		current := times[0]
		value := 0.0
		if s.lastSystem != nil {
			value = cpuBusyPercent(*s.lastSystem, current)
		}
		reading.SystemCPUPercent = float64Ptr(value)
		s.caps.SystemCPU = true
		s.lastSystem = &current
	}

	if memory, err := mem.VirtualMemoryWithContext(ctx); err == nil && memory != nil {
		value := memory.UsedPercent
		reading.SystemMemoryPercent = float64Ptr(value)
		s.caps.SystemMemory = true
	}

	if counters, err := net.IOCountersWithContext(ctx, false); err == nil && len(counters) > 0 {
		current := counters[0]
		rx := counterDeltaValue(s.lastNetworkValue(func(v *net.IOCountersStat) uint64 { return v.BytesRecv }), current.BytesRecv, s.networkReady)
		tx := counterDeltaValue(s.lastNetworkValue(func(v *net.IOCountersStat) uint64 { return v.BytesSent }), current.BytesSent, s.networkReady)
		reading.SystemNetworkRXBytes = int64Ptr(safeUint64ToInt64(rx))
		reading.SystemNetworkTXBytes = int64Ptr(safeUint64ToInt64(tx))
		s.caps.SystemNetwork = true
		s.lastNetwork = &current
		s.networkReady = true
	}

	if counters, err := disk.IOCountersWithContext(ctx); err == nil && len(counters) > 0 {
		current := sumDiskCounters(counters)
		read := counterDeltaValue(s.lastDiskValue(func(v *disk.IOCountersStat) uint64 { return v.ReadBytes }), current.ReadBytes, s.diskReady)
		write := counterDeltaValue(s.lastDiskValue(func(v *disk.IOCountersStat) uint64 { return v.WriteBytes }), current.WriteBytes, s.diskReady)
		reading.SystemDiskReadBytes = int64Ptr(safeUint64ToInt64(read))
		reading.SystemDiskWriteBytes = int64Ptr(safeUint64ToInt64(write))
		s.caps.SystemDiskIO = true
		s.lastDisk = &current
		s.diskReady = true
	}

	if s.process != nil {
		if times, err := s.process.TimesWithContext(ctx); err == nil && times != nil {
			value := 0.0
			if s.lastProcess != nil {
				value = processCPUPercent(*s.lastProcess, *times, s.lastWall, now)
			}
			reading.ProcessCPUPercent = float64Ptr(value)
			s.caps.ProcessCPU = true
			copyTimes := *times
			s.lastProcess = &copyTimes
		}
		if memory, err := s.process.MemoryInfoWithContext(ctx); err == nil && memory != nil {
			rss := safeUint64ToInt64(memory.RSS)
			reading.ProcessMemoryBytes = int64Ptr(rss)
			if systemMemory, memErr := mem.VirtualMemoryWithContext(ctx); memErr == nil && systemMemory != nil && systemMemory.Total > 0 {
				reading.ProcessMemoryPercent = float64Ptr(float64(memory.RSS) / float64(systemMemory.Total) * 100)
			}
			s.caps.ProcessMemory = true
		}
		if ioStat, err := s.process.IOCountersWithContext(ctx); err == nil && ioStat != nil {
			read := counterDeltaValue(s.lastProcRead, ioStat.ReadBytes, s.procIOReady)
			write := counterDeltaValue(s.lastProcWrite, ioStat.WriteBytes, s.procIOReady)
			reading.ProcessDiskReadBytes = int64Ptr(safeUint64ToInt64(read))
			reading.ProcessDiskWriteBytes = int64Ptr(safeUint64ToInt64(write))
			s.caps.ProcessDiskIO = true
			s.lastProcRead = ioStat.ReadBytes
			s.lastProcWrite = ioStat.WriteBytes
			s.procIOReady = true
		}
	}
	s.lastWall = now
	return reading
}

func gcDelta(previous, current runtime.MemStats) (int64, int64, uint64, uint64, [requestHistogramSize]int64) {
	var buckets [requestHistogramSize]int64
	if current.NumGC < previous.NumGC || current.NumForcedGC < previous.NumForcedGC || current.PauseTotalNs < previous.PauseTotalNs {
		return 0, 0, 0, 0, buckets
	}

	cycles := current.NumGC - previous.NumGC
	forced := current.NumForcedGC - previous.NumForcedGC
	pauseTotal := current.PauseTotalNs - previous.PauseTotalNs
	var pauseMax uint64
	start := previous.NumGC + 1
	if cycles > uint32(len(current.PauseNs)) {
		start = current.NumGC - uint32(len(current.PauseNs)) + 1
	}
	for cycle := start; cycle <= current.NumGC; cycle++ {
		pause := current.PauseNs[(cycle+255)%uint32(len(current.PauseNs))]
		if pause > pauseMax {
			pauseMax = pause
		}
		microseconds := pause / uint64(time.Microsecond)
		if pause > 0 && microseconds == 0 {
			microseconds = 1
		}
		if microseconds > uint64(^uint64(0)>>1) {
			microseconds = uint64(^uint64(0) >> 1)
		}
		buckets[latencyHistogramIndex(int64(microseconds))]++
	}
	return int64(cycles), int64(forced), pauseTotal, pauseMax, buckets
}

func readRuntimeGCMetrics() (float64, float64, bool) {
	samples := []runtimeMetrics.Sample{
		{Name: "/cpu/classes/gc/total:cpu-seconds"},
		{Name: "/cpu/classes/total:cpu-seconds"},
	}
	runtimeMetrics.Read(samples)
	if samples[0].Value.Kind() != runtimeMetrics.KindFloat64 || samples[1].Value.Kind() != runtimeMetrics.KindFloat64 {
		return 0, 0, false
	}
	return samples[0].Value.Float64(), samples[1].Value.Float64(), true
}

func gcPressurePercent(previousGC, currentGC, previousTotal, currentTotal float64) (float64, bool) {
	gcDelta := currentGC - previousGC
	totalDelta := currentTotal - previousTotal
	if math.IsNaN(gcDelta) || math.IsInf(gcDelta, 0) || math.IsNaN(totalDelta) || math.IsInf(totalDelta, 0) || gcDelta < 0 || totalDelta <= 0 {
		return 0, false
	}
	return clampPercent(gcDelta / totalDelta * 100), true
}

func (s *resourceSampler) lastNetworkValue(read func(*net.IOCountersStat) uint64) uint64 {
	if s.lastNetwork == nil {
		return 0
	}
	return read(s.lastNetwork)
}

func (s *resourceSampler) lastDiskValue(read func(*disk.IOCountersStat) uint64) uint64 {
	if s.lastDisk == nil {
		return 0
	}
	return read(s.lastDisk)
}

func cpuBusyPercent(previous, current cpu.TimesStat) float64 {
	previousTotal := previous.Total()
	currentTotal := current.Total()
	if currentTotal <= previousTotal {
		return 0
	}
	previousBusy := previousTotal - previous.Idle - previous.Iowait
	currentBusy := currentTotal - current.Idle - current.Iowait
	if currentBusy <= previousBusy {
		return 0
	}
	value := (currentBusy - previousBusy) / (currentTotal - previousTotal) * 100
	return clampPercent(value)
}

func processCPUPercent(previous, current cpu.TimesStat, previousAt, currentAt time.Time) float64 {
	if previousAt.IsZero() || !currentAt.After(previousAt) {
		return 0
	}
	previousTotal := previous.User + previous.System
	currentTotal := current.User + current.System
	if currentTotal <= previousTotal {
		return 0
	}
	value := (currentTotal - previousTotal) / currentAt.Sub(previousAt).Seconds() * 100
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	return value
}

func clampPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func counterDelta(previous, current uint64) uint64 {
	if previous == 0 || current < previous {
		return 0
	}
	return current - previous
}

func counterDeltaValue(previous, current uint64, ready bool) uint64 {
	if !ready || current < previous {
		return 0
	}
	return current - previous
}

func safeUint64ToInt64(value uint64) int64 {
	if value > uint64(^uint64(0)>>1) {
		return int64(^uint64(0) >> 1)
	}
	return int64(value)
}

func sumDiskCounters(counters map[string]disk.IOCountersStat) disk.IOCountersStat {
	var total disk.IOCountersStat
	for _, item := range counters {
		total.ReadBytes += item.ReadBytes
		total.WriteBytes += item.WriteBytes
	}
	return total
}

func float64Ptr(value float64) *float64 { return &value }
func int64Ptr(value int64) *int64       { return &value }
