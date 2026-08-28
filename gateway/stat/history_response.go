package stat

import (
	"math"
	"sort"
	"strings"
	"time"
)

type StatMeta struct {
	Range         string          `json:"range"`
	StartTime     string          `json:"start_time"`
	EndTime       string          `json:"end_time"`
	BucketSeconds int64           `json:"bucket_seconds"`
	GeneratedAt   string          `json:"generated_at"`
	RetentionDays int             `json:"retention_days"`
	Capabilities  map[string]bool `json:"capabilities"`
}

type StatusSummary struct {
	Status1xx int64 `json:"1xx"`
	Status2xx int64 `json:"2xx"`
	Status3xx int64 `json:"3xx"`
	Status4xx int64 `json:"4xx"`
	Status5xx int64 `json:"5xx"`
}

type LatencySummary struct {
	AvgMS float64 `json:"avg_ms"`
	P95MS float64 `json:"p95_ms"`
	MaxMS float64 `json:"max_ms"`
}

type TrafficSummary struct {
	RequestBytes  int64 `json:"request_bytes"`
	ResponseBytes int64 `json:"response_bytes"`
	TotalBytes    int64 `json:"total_bytes"`
}

type StatSummary struct {
	TotalRequests    int64          `json:"total_requests"`
	FrontendRequests int64          `json:"frontend_requests"`
	BackendRequests  int64          `json:"backend_requests"`
	UnknownRequests  int64          `json:"unknown_requests"`
	ErrorRequests    int64          `json:"error_requests"`
	RPS              float64        `json:"rps"`
	ErrorRate        float64        `json:"error_rate"`
	Status           StatusSummary  `json:"status"`
	Latency          LatencySummary `json:"latency"`
	FrontendTraffic  TrafficSummary `json:"frontend_traffic"`
	BackendTraffic   TrafficSummary `json:"backend_traffic"`
	TotalTraffic     TrafficSummary `json:"total_traffic"`
}

type RequestSeriesPoint struct {
	Timestamp    string  `json:"timestamp"`
	Frontend     int64   `json:"frontend"`
	Backend      int64   `json:"backend"`
	Unknown      int64   `json:"unknown"`
	Total        int64   `json:"total"`
	Errors       int64   `json:"errors"`
	Status1xx    int64   `json:"status_1xx"`
	Status2xx    int64   `json:"status_2xx"`
	Status3xx    int64   `json:"status_3xx"`
	Status4xx    int64   `json:"status_4xx"`
	Status5xx    int64   `json:"status_5xx"`
	RPS          float64 `json:"rps"`
	AvgLatencyMS float64 `json:"avg_latency_ms"`
	P95LatencyMS float64 `json:"p95_latency_ms"`
	MaxLatencyMS float64 `json:"max_latency_ms"`
}

type TrafficSeriesPoint struct {
	Timestamp             string `json:"timestamp"`
	FrontendRequestBytes  int64  `json:"frontend_request_bytes"`
	FrontendResponseBytes int64  `json:"frontend_response_bytes"`
	BackendRequestBytes   int64  `json:"backend_request_bytes"`
	BackendResponseBytes  int64  `json:"backend_response_bytes"`
	RequestBytes          int64  `json:"request_bytes"`
	ResponseBytes         int64  `json:"response_bytes"`
}

type SystemSeriesPoint struct {
	Timestamp         string   `json:"timestamp"`
	CPUPercent        *float64 `json:"cpu_percent"`
	CPUPeakPercent    *float64 `json:"cpu_peak_percent"`
	MemoryPercent     *float64 `json:"memory_percent"`
	MemoryPeakPercent *float64 `json:"memory_peak_percent"`
	NetworkRXBytes    *int64   `json:"network_rx_bytes"`
	NetworkTXBytes    *int64   `json:"network_tx_bytes"`
	DiskReadBytes     *int64   `json:"disk_read_bytes"`
	DiskWriteBytes    *int64   `json:"disk_write_bytes"`
}

type ProcessSeriesPoint struct {
	Timestamp         string   `json:"timestamp"`
	CPUPercent        *float64 `json:"cpu_percent"`
	CPUPeakPercent    *float64 `json:"cpu_peak_percent"`
	MemoryBytes       *int64   `json:"memory_bytes"`
	MemoryPeakBytes   *int64   `json:"memory_peak_bytes"`
	MemoryPercent     *float64 `json:"memory_percent"`
	MemoryPeakPercent *float64 `json:"memory_peak_percent"`
	DiskReadBytes     *int64   `json:"disk_read_bytes"`
	DiskWriteBytes    *int64   `json:"disk_write_bytes"`
}

type StatSeries struct {
	Requests []RequestSeriesPoint `json:"requests"`
	Traffic  []TrafficSeriesPoint `json:"traffic"`
	System   []SystemSeriesPoint  `json:"system"`
	Process  []ProcessSeriesPoint `json:"process"`
}

type DomainSummary struct {
	Domain        string `json:"domain"`
	Requests      int64  `json:"requests"`
	Errors        int64  `json:"errors"`
	RequestBytes  int64  `json:"request_bytes"`
	ResponseBytes int64  `json:"response_bytes"`
}

type DomainSeriesPoint struct {
	Timestamp     string `json:"timestamp"`
	Requests      int64  `json:"requests"`
	Errors        int64  `json:"errors"`
	RequestBytes  int64  `json:"request_bytes"`
	ResponseBytes int64  `json:"response_bytes"`
}

type StatResponse struct {
	// Legacy cumulative fields remain stable for existing dashboard clients.
	Total  int64 `json:"total"`
	API    int64 `json:"api"`
	Static int64 `json:"static"`
	Fail   int64 `json:"fail"`
	Today  int64 `json:"today"`

	Meta         StatMeta               `json:"meta"`
	Summary      StatSummary            `json:"summary"`
	Series       StatSeries             `json:"series"`
	Connections  map[string]interface{} `json:"connections"`
	Domains      []DomainSummary        `json:"domains"`
	DomainSeries []DomainSeriesPoint    `json:"domain_series,omitempty"`
}

type requestAggregate struct {
	requestBucket
	Frontend              int64
	Backend               int64
	Unknown               int64
	FrontendRequestBytes  int64
	FrontendResponseBytes int64
	BackendRequestBytes   int64
	BackendResponseBytes  int64
}

func (a *requestAggregate) add(route RouteType, value requestBucket) {
	mergeRequestBucket(&a.requestBucket, value)
	switch route {
	case RouteFrontend:
		a.Frontend += value.Requests
		a.FrontendRequestBytes += value.RequestBytes
		a.FrontendResponseBytes += value.ResponseBytes
	case RouteBackend:
		a.Backend += value.Requests
		a.BackendRequestBytes += value.RequestBytes
		a.BackendResponseBytes += value.ResponseBytes
	default:
		a.Unknown += value.Requests
	}
}

type domainAggregate struct {
	domainBucket
}

func newStatResponse(spec RangeSpec, now time.Time, m *StatManager) StatResponse {
	retention := HistoryRetentionDays
	if m != nil && m.history != nil {
		retention = m.history.RetentionDays()
	}
	return StatResponse{
		Meta: StatMeta{
			Range: spec.Name, StartTime: now.Add(-spec.Duration).UTC().Format(time.RFC3339), EndTime: now.UTC().Format(time.RFC3339),
			BucketSeconds: int64(spec.BucketWidth / time.Second), GeneratedAt: now.UTC().Format(time.RFC3339Nano),
			RetentionDays: retention, Capabilities: m.HistoryCapabilities(),
		},
		Series: StatSeries{
			Requests: make([]RequestSeriesPoint, 0), Traffic: make([]TrafficSeriesPoint, 0),
			System: make([]SystemSeriesPoint, 0), Process: make([]ProcessSeriesPoint, 0),
		},
		Connections: map[string]interface{}{
			"gateway": GetGatewayConn(),
			"front":   GetFrontConn(),
		},
		Domains: make([]DomainSummary, 0),
	}
}

// QueryStat builds the public response and includes the current in-memory
// bucket even when it has not reached the five-second SQLite flush yet.
func (m *StatManager) QueryStat(rangeValue, domain string) (StatResponse, error) {
	spec, err := ParseRange(rangeValue)
	if err != nil {
		return StatResponse{}, err
	}
	now := time.Now().UTC()
	response := newStatResponse(spec, now, m)
	if m == nil {
		if strings.TrimSpace(domain) != "" {
			response.DomainSeries = make([]DomainSeriesPoint, 0)
		}
		return response, nil
	}
	response.Total = m.Get(Total)
	response.API = m.Get(API)
	response.Static = m.Get(Static)
	response.Fail = m.Get(Fail)
	response.Today = m.Get(Today)
	if m == nil || m.history == nil {
		if strings.TrimSpace(domain) != "" {
			response.DomainSeries = make([]DomainSeriesPoint, 0)
		}
		return response, nil
	}

	start := now.Add(-spec.Duration)
	rows, err := m.history.QueryRows(start, now, "")
	if err != nil {
		return response, err
	}
	requestedDomain := NormalizeDomain(domain)
	requestBuckets := make(map[int64]*requestAggregate)
	trafficBuckets := make(map[int64]*requestAggregate)
	var total requestAggregate
	for _, row := range rows.Requests {
		bucket := row.bucket()
		route := normalizeRoute(row.Route)
		total.add(route, bucket)
		outputBucket := outputBucketStart(row.BucketStart, spec.BucketWidth)
		requestAgg := requestBuckets[outputBucket]
		if requestAgg == nil {
			requestAgg = &requestAggregate{}
			requestBuckets[outputBucket] = requestAgg
		}
		requestAgg.add(route, bucket)
		trafficAgg := trafficBuckets[outputBucket]
		if trafficAgg == nil {
			trafficAgg = &requestAggregate{}
			trafficBuckets[outputBucket] = trafficAgg
		}
		trafficAgg.add(route, bucket)
	}
	if len(rows.Requests) > 0 {
		response.Series.Requests = buildRequestSeries(requestBuckets, spec, start, now)
		response.Series.Traffic = buildTrafficSeries(trafficBuckets, spec, start, now)
	}
	response.Summary = buildSummary(total, spec.Duration)

	domainBuckets := make(map[int64]*domainAggregate)
	domainTotals := make(map[string]*domainAggregate)
	var selectedDomainBuckets = make(map[int64]*domainAggregate)
	for _, row := range rows.Domains {
		value := domainBucket{Requests: row.Requests, Errors: row.Errors, RequestBytes: row.RequestBytes, ResponseBytes: row.ResponseBytes}
		outputBucket := outputBucketStart(row.BucketStart, spec.BucketWidth)
		bucket := domainBuckets[outputBucket]
		if bucket == nil {
			bucket = &domainAggregate{}
			domainBuckets[outputBucket] = bucket
		}
		bucket.domainBucket.Requests += value.Requests
		bucket.domainBucket.Errors += value.Errors
		bucket.domainBucket.RequestBytes += value.RequestBytes
		bucket.domainBucket.ResponseBytes += value.ResponseBytes
		totalDomain := domainTotals[row.Domain]
		if totalDomain == nil {
			totalDomain = &domainAggregate{}
			domainTotals[row.Domain] = totalDomain
		}
		totalDomain.domainBucket.Requests += value.Requests
		totalDomain.domainBucket.Errors += value.Errors
		totalDomain.domainBucket.RequestBytes += value.RequestBytes
		totalDomain.domainBucket.ResponseBytes += value.ResponseBytes
		if requestedDomain != "" && row.Domain == requestedDomain {
			selected := selectedDomainBuckets[outputBucket]
			if selected == nil {
				selected = &domainAggregate{}
				selectedDomainBuckets[outputBucket] = selected
			}
			selected.domainBucket.Requests += value.Requests
			selected.domainBucket.Errors += value.Errors
			selected.domainBucket.RequestBytes += value.RequestBytes
			selected.domainBucket.ResponseBytes += value.ResponseBytes
		}
	}
	response.Domains = sortedDomainSummaries(domainTotals)
	if strings.TrimSpace(domain) != "" {
		response.DomainSeries = buildDomainSeries(selectedDomainBuckets, spec, start, now)
	}

	resourceBuckets := make(map[int64]*resourceBucket)
	for _, row := range rows.Resources {
		outputBucket := outputBucketStart(row.BucketStart, spec.BucketWidth)
		value := row.bucket()
		if current := resourceBuckets[outputBucket]; current != nil {
			mergeResourceBucket(current, value)
		} else {
			copyValue := value
			resourceBuckets[outputBucket] = &copyValue
		}
	}
	if len(rows.Resources) > 0 {
		response.Series.System, response.Series.Process = buildResourceSeries(resourceBuckets, spec, start, now)
	}
	return response, nil
}

func normalizeRoute(value string) RouteType {
	switch RouteType(value) {
	case RouteFrontend:
		return RouteFrontend
	case RouteBackend:
		return RouteBackend
	default:
		return RouteUnknown
	}
}

func buildSummary(total requestAggregate, duration time.Duration) StatSummary {
	result := StatSummary{
		TotalRequests: total.Requests, FrontendRequests: total.Frontend, BackendRequests: total.Backend,
		UnknownRequests: total.Unknown, ErrorRequests: total.Errors,
		Status:          StatusSummary{Status1xx: total.Status1xx, Status2xx: total.Status2xx, Status3xx: total.Status3xx, Status4xx: total.Status4xx, Status5xx: total.Status5xx},
		Latency:         latencySummary(total.requestBucket),
		FrontendTraffic: TrafficSummary{RequestBytes: total.FrontendRequestBytes, ResponseBytes: total.FrontendResponseBytes, TotalBytes: total.FrontendRequestBytes + total.FrontendResponseBytes},
		BackendTraffic:  TrafficSummary{RequestBytes: total.BackendRequestBytes, ResponseBytes: total.BackendResponseBytes, TotalBytes: total.BackendRequestBytes + total.BackendResponseBytes},
	}
	if duration > 0 {
		result.RPS = float64(total.Requests) / duration.Seconds()
	}
	if total.Requests > 0 {
		result.ErrorRate = float64(total.Errors) / float64(total.Requests)
	}
	// Route-specific traffic is reconstructed from the route counts in a
	// second pass by callers when needed; the flat total remains exact here.
	result.TotalTraffic = TrafficSummary{RequestBytes: total.RequestBytes, ResponseBytes: total.ResponseBytes, TotalBytes: total.RequestBytes + total.ResponseBytes}
	return result
}

func latencySummary(value requestBucket) LatencySummary {
	result := LatencySummary{}
	if value.Requests > 0 {
		result.AvgMS = float64(value.LatencySumUS) / float64(value.Requests) / 1000
	}
	result.P95MS = approximateP95(value.LatencyBuckets)
	result.MaxMS = float64(value.LatencyMaxUS) / 1000
	return result
}

func sortedOutputBuckets[T any](values map[int64]*T, spec RangeSpec, start, now time.Time) []int64 {
	if len(values) == 0 {
		return []int64{}
	}
	width := int64(spec.BucketWidth / time.Second)
	from := outputBucketStart(bucketStart(start), spec.BucketWidth)
	to := outputBucketStart(bucketStart(now), spec.BucketWidth)
	keys := make([]int64, 0, ((to-from)/width)+1)
	for key := from; key <= to; key += width {
		keys = append(keys, key)
	}
	return keys
}

func buildRequestSeries(values map[int64]*requestAggregate, spec RangeSpec, start, now time.Time) []RequestSeriesPoint {
	keys := sortedOutputBuckets(values, spec, start, now)
	result := make([]RequestSeriesPoint, 0, len(keys))
	widthSeconds := spec.BucketWidth.Seconds()
	for _, key := range keys {
		value := values[key]
		if value == nil {
			value = &requestAggregate{}
		}
		result = append(result, RequestSeriesPoint{
			Timestamp: time.Unix(key, 0).UTC().Format(time.RFC3339), Frontend: value.Frontend, Backend: value.Backend, Unknown: value.Unknown,
			Total: value.Requests, Errors: value.Errors, Status1xx: value.Status1xx, Status2xx: value.Status2xx, Status3xx: value.Status3xx,
			Status4xx: value.Status4xx, Status5xx: value.Status5xx, RPS: float64(value.Requests) / widthSeconds,
			AvgLatencyMS: latencySummary(value.requestBucket).AvgMS, P95LatencyMS: latencySummary(value.requestBucket).P95MS, MaxLatencyMS: latencySummary(value.requestBucket).MaxMS,
		})
	}
	return result
}

func buildTrafficSeries(values map[int64]*requestAggregate, spec RangeSpec, start, now time.Time) []TrafficSeriesPoint {
	keys := sortedOutputBuckets(values, spec, start, now)
	result := make([]TrafficSeriesPoint, 0, len(keys))
	for _, key := range keys {
		value := values[key]
		if value == nil {
			value = &requestAggregate{}
		}
		result = append(result, TrafficSeriesPoint{
			Timestamp:            time.Unix(key, 0).UTC().Format(time.RFC3339),
			FrontendRequestBytes: value.FrontendRequestBytes, FrontendResponseBytes: value.FrontendResponseBytes,
			BackendRequestBytes: value.BackendRequestBytes, BackendResponseBytes: value.BackendResponseBytes,
			RequestBytes: value.RequestBytes, ResponseBytes: value.ResponseBytes,
		})
	}
	return result
}

func sortedDomainSummaries(values map[string]*domainAggregate) []DomainSummary {
	result := make([]DomainSummary, 0, len(values))
	for domain, value := range values {
		result = append(result, DomainSummary{Domain: domain, Requests: value.Requests, Errors: value.Errors, RequestBytes: value.RequestBytes, ResponseBytes: value.ResponseBytes})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Requests == result[j].Requests {
			return result[i].Domain < result[j].Domain
		}
		return result[i].Requests > result[j].Requests
	})
	return result
}

func buildDomainSeries(values map[int64]*domainAggregate, spec RangeSpec, start, now time.Time) []DomainSeriesPoint {
	keys := sortedOutputBuckets(values, spec, start, now)
	result := make([]DomainSeriesPoint, 0, len(keys))
	for _, key := range keys {
		value := values[key]
		if value == nil {
			value = &domainAggregate{}
		}
		result = append(result, DomainSeriesPoint{Timestamp: time.Unix(key, 0).UTC().Format(time.RFC3339), Requests: value.Requests, Errors: value.Errors, RequestBytes: value.RequestBytes, ResponseBytes: value.ResponseBytes})
	}
	return result
}

func buildResourceSeries(values map[int64]*resourceBucket, spec RangeSpec, start, now time.Time) ([]SystemSeriesPoint, []ProcessSeriesPoint) {
	keys := sortedOutputBuckets(values, spec, start, now)
	system := make([]SystemSeriesPoint, 0, len(keys))
	process := make([]ProcessSeriesPoint, 0, len(keys))
	for _, key := range keys {
		value := values[key]
		if value == nil {
			value = &resourceBucket{}
		}
		system = append(system, SystemSeriesPoint{
			Timestamp:  time.Unix(key, 0).UTC().Format(time.RFC3339),
			CPUPercent: averageFloat(value.SystemCPUSum, value.SystemCPUCount), CPUPeakPercent: optionalFloat(value.SystemCPUPeak, value.SystemCPUCount),
			MemoryPercent: averageFloat(value.SystemMemorySum, value.SystemMemoryCount), MemoryPeakPercent: optionalFloat(value.SystemMemoryPeak, value.SystemMemoryCount),
			NetworkRXBytes: optionalInt64(value.SystemNetworkRX, value.SystemNetworkRXAvail), NetworkTXBytes: optionalInt64(value.SystemNetworkTX, value.SystemNetworkTXAvail),
			DiskReadBytes: optionalInt64(value.SystemDiskRead, value.SystemDiskReadAvail), DiskWriteBytes: optionalInt64(value.SystemDiskWrite, value.SystemDiskWriteAvail),
		})
		process = append(process, ProcessSeriesPoint{
			Timestamp:  time.Unix(key, 0).UTC().Format(time.RFC3339),
			CPUPercent: averageFloat(value.ProcessCPUSum, value.ProcessCPUCount), CPUPeakPercent: optionalFloat(value.ProcessCPUPeak, value.ProcessCPUCount),
			MemoryBytes: averageInt64(value.ProcessMemorySum, value.ProcessMemoryCount), MemoryPeakBytes: optionalInt64(int64(value.ProcessMemoryPeak), value.ProcessMemoryCount > 0),
			MemoryPercent: averageFloat(value.ProcessMemoryPercentSum, value.ProcessMemoryPercentCount), MemoryPeakPercent: optionalFloat(value.ProcessMemoryPercentPeak, value.ProcessMemoryPercentCount),
			DiskReadBytes: optionalInt64(value.ProcessDiskRead, value.ProcessDiskReadAvail), DiskWriteBytes: optionalInt64(value.ProcessDiskWrite, value.ProcessDiskWriteAvail),
		})
	}
	return system, process
}

func averageFloat(sum float64, count int64) *float64 {
	if count <= 0 {
		return nil
	}
	value := sum / float64(count)
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	return &value
}

func optionalFloat(value float64, available int64) *float64 {
	if available <= 0 {
		return nil
	}
	return &value
}

func averageInt64(sum float64, count int64) *int64 {
	if count <= 0 {
		return nil
	}
	value := int64(sum / float64(count))
	return &value
}

func optionalInt64(value int64, available bool) *int64 {
	if !available {
		return nil
	}
	return &value
}
