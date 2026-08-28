package stat

import (
	"errors"
	"math"
	"strings"
	"time"
)

const (
	HistoryBucketSeconds = int64(60)
	HistoryRetentionDays = 30
	requestHistogramSize = 11
)

var (
	ErrInvalidRange = errors.New("unsupported stat range")
	AllowedRanges   = []string{"1h", "5h", "24h", "7d", "30d"}
)

// RangeSpec describes the accepted API windows and their output resampling.
type RangeSpec struct {
	Name        string
	Duration    time.Duration
	BucketWidth time.Duration
}

func ParseRange(value string) (RangeSpec, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "1h":
		return RangeSpec{Name: "1h", Duration: time.Hour, BucketWidth: time.Minute}, nil
	case "5h":
		return RangeSpec{Name: value, Duration: 5 * time.Hour, BucketWidth: 5 * time.Minute}, nil
	case "24h":
		return RangeSpec{Name: value, Duration: 24 * time.Hour, BucketWidth: 15 * time.Minute}, nil
	case "7d":
		return RangeSpec{Name: value, Duration: 7 * 24 * time.Hour, BucketWidth: time.Hour}, nil
	case "30d":
		return RangeSpec{Name: value, Duration: 30 * 24 * time.Hour, BucketWidth: 2 * time.Hour}, nil
	default:
		return RangeSpec{}, ErrInvalidRange
	}
}

func bucketStart(t time.Time) int64 {
	return t.UTC().Unix() / HistoryBucketSeconds * HistoryBucketSeconds
}

func outputBucketStart(value int64, width time.Duration) int64 {
	widthSeconds := int64(width / time.Second)
	if widthSeconds <= 0 {
		widthSeconds = HistoryBucketSeconds
	}
	return value / widthSeconds * widthSeconds
}

// Histogram upper bounds in microseconds. The last bucket is an overflow
// bucket; quantiles are intentionally approximate as advertised by the API.
var latencyHistogramBounds = [...]int64{
	1_000,
	5_000,
	10_000,
	25_000,
	50_000,
	100_000,
	250_000,
	500_000,
	1_000_000,
	5_000_000,
}

func latencyHistogramIndex(microseconds int64) int {
	if microseconds < 0 {
		microseconds = 0
	}
	for index, bound := range latencyHistogramBounds {
		if microseconds <= bound {
			return index
		}
	}
	return requestHistogramSize - 1
}

func approximateP95(hist [requestHistogramSize]int64) float64 {
	var total int64
	for _, count := range hist {
		total += count
	}
	if total <= 0 {
		return 0
	}
	target := int64(math.Ceil(float64(total) * 0.95))
	if target < 1 {
		target = 1
	}
	var seen int64
	for index, count := range hist {
		seen += count
		if seen >= target {
			if index >= len(latencyHistogramBounds) {
				return float64(latencyHistogramBounds[len(latencyHistogramBounds)-1]) / 1000
			}
			return float64(latencyHistogramBounds[index]) / 1000
		}
	}
	return float64(latencyHistogramBounds[len(latencyHistogramBounds)-1]) / 1000
}

type requestBucketKey struct {
	BucketStart int64
	Route       RouteType
}

type domainBucketKey struct {
	BucketStart int64
	Domain      string
}

type requestBucket struct {
	Requests       int64
	Errors         int64
	Status1xx      int64
	Status2xx      int64
	Status3xx      int64
	Status4xx      int64
	Status5xx      int64
	RequestBytes   int64
	ResponseBytes  int64
	LatencySumUS   int64
	LatencyMaxUS   int64
	LatencyBuckets [requestHistogramSize]int64
}

type domainBucket struct {
	Requests      int64
	Errors        int64
	RequestBytes  int64
	ResponseBytes int64
}

type resourceBucket struct {
	SampleCount int64

	SystemCPUSum   float64
	SystemCPUCount int64
	SystemCPUPeak  float64

	SystemMemorySum   float64
	SystemMemoryCount int64
	SystemMemoryPeak  float64

	ProcessCPUSum   float64
	ProcessCPUCount int64
	ProcessCPUPeak  float64

	ProcessMemorySum   float64
	ProcessMemoryCount int64
	ProcessMemoryPeak  float64

	ProcessMemoryPercentSum   float64
	ProcessMemoryPercentCount int64
	ProcessMemoryPercentPeak  float64

	SystemNetworkRX       int64
	SystemNetworkTX       int64
	SystemNetworkRXAvail  bool
	SystemNetworkTXAvail  bool
	SystemDiskRead        int64
	SystemDiskWrite       int64
	SystemDiskReadAvail   bool
	SystemDiskWriteAvail  bool
	ProcessDiskRead       int64
	ProcessDiskWrite      int64
	ProcessDiskReadAvail  bool
	ProcessDiskWriteAvail bool
}

type historyBatch struct {
	Requests  map[requestBucketKey]requestBucket
	Domains   map[domainBucketKey]domainBucket
	Resources map[int64]resourceBucket
}

func newHistoryBatch() *historyBatch {
	return &historyBatch{
		Requests:  make(map[requestBucketKey]requestBucket),
		Domains:   make(map[domainBucketKey]domainBucket),
		Resources: make(map[int64]resourceBucket),
	}
}

func (b *historyBatch) empty() bool {
	return b == nil || (len(b.Requests) == 0 && len(b.Domains) == 0 && len(b.Resources) == 0)
}

func cloneRequestBucket(src requestBucket) requestBucket {
	dst := src
	return dst
}

func cloneResourceBucket(src resourceBucket) resourceBucket {
	return src
}

func (b *historyBatch) clone() *historyBatch {
	copyBatch := newHistoryBatch()
	if b == nil {
		return copyBatch
	}
	for key, value := range b.Requests {
		copyBatch.Requests[key] = cloneRequestBucket(value)
	}
	for key, value := range b.Domains {
		copyBatch.Domains[key] = value
	}
	for key, value := range b.Resources {
		copyBatch.Resources[key] = cloneResourceBucket(value)
	}
	return copyBatch
}

func mergeHistoryBatch(dst, src *historyBatch) {
	if dst == nil || src == nil {
		return
	}
	for key, value := range src.Requests {
		current := dst.Requests[key]
		mergeRequestBucket(&current, value)
		dst.Requests[key] = current
	}
	for key, value := range src.Domains {
		current := dst.Domains[key]
		current.Requests += value.Requests
		current.Errors += value.Errors
		current.RequestBytes += value.RequestBytes
		current.ResponseBytes += value.ResponseBytes
		dst.Domains[key] = current
	}
	for key, value := range src.Resources {
		current := dst.Resources[key]
		mergeResourceBucket(&current, value)
		dst.Resources[key] = current
	}
}

func (b *historyBatch) subtract(other *historyBatch) {
	if b == nil || other == nil {
		return
	}
	for key, value := range other.Requests {
		current, ok := b.Requests[key]
		if !ok {
			continue
		}
		current.Requests -= value.Requests
		current.Errors -= value.Errors
		current.Status1xx -= value.Status1xx
		current.Status2xx -= value.Status2xx
		current.Status3xx -= value.Status3xx
		current.Status4xx -= value.Status4xx
		current.Status5xx -= value.Status5xx
		current.RequestBytes -= value.RequestBytes
		current.ResponseBytes -= value.ResponseBytes
		current.LatencySumUS -= value.LatencySumUS
		current.LatencyMaxUS = 0
		for index := range current.LatencyBuckets {
			current.LatencyBuckets[index] -= value.LatencyBuckets[index]
		}
		if current.Requests <= 0 {
			delete(b.Requests, key)
			continue
		}
		b.Requests[key] = current
	}
	for key, value := range other.Domains {
		current, ok := b.Domains[key]
		if !ok {
			continue
		}
		current.Requests -= value.Requests
		current.Errors -= value.Errors
		current.RequestBytes -= value.RequestBytes
		current.ResponseBytes -= value.ResponseBytes
		if current.Requests <= 0 {
			delete(b.Domains, key)
			continue
		}
		b.Domains[key] = current
	}
	for key, value := range other.Resources {
		current, ok := b.Resources[key]
		if !ok {
			continue
		}
		current.SampleCount -= value.SampleCount
		current.SystemCPUSum -= value.SystemCPUSum
		current.SystemCPUCount -= value.SystemCPUCount
		current.SystemMemorySum -= value.SystemMemorySum
		current.SystemMemoryCount -= value.SystemMemoryCount
		current.ProcessCPUSum -= value.ProcessCPUSum
		current.ProcessCPUCount -= value.ProcessCPUCount
		current.ProcessMemorySum -= value.ProcessMemorySum
		current.ProcessMemoryCount -= value.ProcessMemoryCount
		current.ProcessMemoryPercentSum -= value.ProcessMemoryPercentSum
		current.ProcessMemoryPercentCount -= value.ProcessMemoryPercentCount
		current.SystemNetworkRX -= value.SystemNetworkRX
		current.SystemNetworkTX -= value.SystemNetworkTX
		current.SystemDiskRead -= value.SystemDiskRead
		current.SystemDiskWrite -= value.SystemDiskWrite
		current.ProcessDiskRead -= value.ProcessDiskRead
		current.ProcessDiskWrite -= value.ProcessDiskWrite
		if current.SampleCount <= 0 {
			delete(b.Resources, key)
			continue
		}
		b.Resources[key] = current
	}
}
