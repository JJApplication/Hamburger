package stat

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"Hamburger/gateway/stat/db"
	"Hamburger/internal/config/svr_config"
	"Hamburger/internal/logger"

	"gorm.io/gorm"
)

const (
	requestBucketTable  = "stat_request_buckets"
	domainBucketTable   = "stat_domain_buckets"
	resourceSampleTable = "stat_resource_samples"
)

const historySchema = `
CREATE TABLE IF NOT EXISTS stat_request_buckets (
    bucket_start INTEGER NOT NULL,
    route TEXT NOT NULL,
    requests INTEGER NOT NULL DEFAULT 0,
    errors INTEGER NOT NULL DEFAULT 0,
    status_1xx INTEGER NOT NULL DEFAULT 0,
    status_2xx INTEGER NOT NULL DEFAULT 0,
    status_3xx INTEGER NOT NULL DEFAULT 0,
    status_4xx INTEGER NOT NULL DEFAULT 0,
    status_5xx INTEGER NOT NULL DEFAULT 0,
    request_bytes INTEGER NOT NULL DEFAULT 0,
    response_bytes INTEGER NOT NULL DEFAULT 0,
    latency_sum_us INTEGER NOT NULL DEFAULT 0,
    latency_max_us INTEGER NOT NULL DEFAULT 0,
    latency_bucket_0 INTEGER NOT NULL DEFAULT 0,
    latency_bucket_1 INTEGER NOT NULL DEFAULT 0,
    latency_bucket_2 INTEGER NOT NULL DEFAULT 0,
    latency_bucket_3 INTEGER NOT NULL DEFAULT 0,
    latency_bucket_4 INTEGER NOT NULL DEFAULT 0,
    latency_bucket_5 INTEGER NOT NULL DEFAULT 0,
    latency_bucket_6 INTEGER NOT NULL DEFAULT 0,
    latency_bucket_7 INTEGER NOT NULL DEFAULT 0,
    latency_bucket_8 INTEGER NOT NULL DEFAULT 0,
    latency_bucket_9 INTEGER NOT NULL DEFAULT 0,
    latency_bucket_10 INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (bucket_start, route)
);
CREATE INDEX IF NOT EXISTS idx_stat_request_buckets_time
    ON stat_request_buckets(bucket_start);

CREATE TABLE IF NOT EXISTS stat_domain_buckets (
    bucket_start INTEGER NOT NULL,
    domain TEXT NOT NULL,
    requests INTEGER NOT NULL DEFAULT 0,
    errors INTEGER NOT NULL DEFAULT 0,
    request_bytes INTEGER NOT NULL DEFAULT 0,
    response_bytes INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (bucket_start, domain)
);
CREATE INDEX IF NOT EXISTS idx_stat_domain_buckets_time
    ON stat_domain_buckets(bucket_start);
CREATE INDEX IF NOT EXISTS idx_stat_domain_buckets_domain_time
    ON stat_domain_buckets(domain, bucket_start);

CREATE TABLE IF NOT EXISTS stat_resource_samples (
    bucket_start INTEGER PRIMARY KEY,
    sample_count INTEGER NOT NULL DEFAULT 0,
    system_cpu_sum REAL NOT NULL DEFAULT 0,
    system_cpu_count INTEGER NOT NULL DEFAULT 0,
    system_cpu_peak REAL NOT NULL DEFAULT 0,
    system_memory_sum REAL NOT NULL DEFAULT 0,
    system_memory_count INTEGER NOT NULL DEFAULT 0,
    system_memory_peak REAL NOT NULL DEFAULT 0,
    process_cpu_sum REAL NOT NULL DEFAULT 0,
    process_cpu_count INTEGER NOT NULL DEFAULT 0,
    process_cpu_peak REAL NOT NULL DEFAULT 0,
    process_memory_sum REAL NOT NULL DEFAULT 0,
    process_memory_count INTEGER NOT NULL DEFAULT 0,
    process_memory_peak REAL NOT NULL DEFAULT 0,
    process_memory_percent_sum REAL NOT NULL DEFAULT 0,
    process_memory_percent_count INTEGER NOT NULL DEFAULT 0,
    process_memory_percent_peak REAL NOT NULL DEFAULT 0,
    system_network_rx_bytes INTEGER NOT NULL DEFAULT 0,
    system_network_tx_bytes INTEGER NOT NULL DEFAULT 0,
    system_network_rx_available INTEGER NOT NULL DEFAULT 0,
    system_network_tx_available INTEGER NOT NULL DEFAULT 0,
    system_disk_read_bytes INTEGER NOT NULL DEFAULT 0,
    system_disk_write_bytes INTEGER NOT NULL DEFAULT 0,
    system_disk_read_available INTEGER NOT NULL DEFAULT 0,
    system_disk_write_available INTEGER NOT NULL DEFAULT 0,
    process_disk_read_bytes INTEGER NOT NULL DEFAULT 0,
    process_disk_write_bytes INTEGER NOT NULL DEFAULT 0,
    process_disk_read_available INTEGER NOT NULL DEFAULT 0,
    process_disk_write_available INTEGER NOT NULL DEFAULT 0,
    gc_cycles INTEGER NOT NULL DEFAULT 0,
    gc_forced_cycles INTEGER NOT NULL DEFAULT 0,
    gc_pause_total_ns INTEGER NOT NULL DEFAULT 0,
    gc_pause_max_ns INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_0 INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_1 INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_2 INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_3 INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_4 INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_5 INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_6 INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_7 INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_8 INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_9 INTEGER NOT NULL DEFAULT 0,
    gc_pause_bucket_10 INTEGER NOT NULL DEFAULT 0,
    gc_pressure_sum REAL NOT NULL DEFAULT 0,
    gc_pressure_count INTEGER NOT NULL DEFAULT 0,
    gc_pressure_peak REAL NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_stat_resource_samples_time
    ON stat_resource_samples(bucket_start);
`

type historyWriteTask struct {
	batch       *historyBatch
	cleanupUnix int64
	done        chan error
}

// HistoryStore is a minute-bucket SQLite store with one serialized writer.
// Request handlers only update the in-memory overlay; the writer receives an
// aggregated batch every flush interval, keeping lock contention out of the
// gateway request path.
type HistoryStore struct {
	database *gorm.DB
	config   svr_config.SequenceConfig

	overlayMu sync.RWMutex
	overlay   *historyBatch

	queue      chan historyWriteTask
	stop       chan struct{}
	writerDone chan struct{}
	flushMu    sync.Mutex
	closed     atomic.Bool
	closeOnce  sync.Once
	closeErr   error
}

// NewHistoryStore creates the fixed-table store. A nil store is returned when
// sequence history is disabled or no database is available.
func NewHistoryStore(database *gorm.DB, cfg svr_config.SequenceConfig) (*HistoryStore, error) {
	if database == nil || !cfg.Enabled {
		return nil, nil
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = HistoryRetentionDays
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 3600
	}
	if err := ensureHistorySchema(database); err != nil {
		return nil, err
	}
	store := &HistoryStore{
		database:   database,
		config:     cfg,
		overlay:    newHistoryBatch(),
		queue:      make(chan historyWriteTask, 8),
		stop:       make(chan struct{}),
		writerDone: make(chan struct{}),
	}
	go store.runWriter()
	return store, nil
}

func ensureHistorySchema(database *gorm.DB) error {
	for _, statement := range splitSchemaStatements(historySchema) {
		if err := database.Exec(statement).Error; err != nil {
			return fmt.Errorf("create history schema: %w", err)
		}
	}
	if err := ensureResourceGCSchema(database); err != nil {
		return err
	}
	return nil
}

func ensureResourceGCSchema(database *gorm.DB) error {
	type columnInfo struct {
		Name string `gorm:"column:name"`
	}
	var columns []columnInfo
	if err := database.Raw("PRAGMA table_info(stat_resource_samples)").Scan(&columns).Error; err != nil {
		return fmt.Errorf("inspect resource history schema: %w", err)
	}
	existing := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		existing[column.Name] = struct{}{}
	}
	definitions := map[string]string{
		"gc_cycles":          "INTEGER NOT NULL DEFAULT 0",
		"gc_forced_cycles":   "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_total_ns":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_max_ns":    "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_0":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_1":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_2":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_3":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_4":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_5":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_6":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_7":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_8":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_9":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pause_bucket_10": "INTEGER NOT NULL DEFAULT 0",
		"gc_pressure_sum":    "REAL NOT NULL DEFAULT 0",
		"gc_pressure_count":  "INTEGER NOT NULL DEFAULT 0",
		"gc_pressure_peak":   "REAL NOT NULL DEFAULT 0",
	}
	for name, definition := range definitions {
		if _, ok := existing[name]; ok {
			continue
		}
		if err := database.Exec(fmt.Sprintf("ALTER TABLE stat_resource_samples ADD COLUMN %s %s", name, definition)).Error; err != nil {
			return fmt.Errorf("add resource history column %s: %w", name, err)
		}
	}
	return nil
}

func splitSchemaStatements(schema string) []string {
	statements := make([]string, 0, 8)
	current := ""
	for _, line := range splitLines(schema) {
		current += line
		current += "\n"
		if len(trimSpace(line)) > 0 && endsWithSemicolon(line) {
			statements = append(statements, current)
			current = ""
		}
	}
	if trimSpace(current) != "" {
		statements = append(statements, current)
	}
	return statements
}

// These tiny helpers keep schema parsing independent of strings.Fields so SQL
// comments/indentation remain harmless and the schema stays readable.
func splitLines(value string) []string {
	lines := make([]string, 0, 64)
	start := 0
	for index := 0; index < len(value); index++ {
		if value[index] == '\n' {
			lines = append(lines, value[start:index])
			start = index + 1
		}
	}
	if start < len(value) {
		lines = append(lines, value[start:])
	}
	return lines
}

func trimSpace(value string) string {
	start, end := 0, len(value)
	for start < end && (value[start] == ' ' || value[start] == '\t' || value[start] == '\r' || value[start] == '\n') {
		start++
	}
	for end > start && (value[end-1] == ' ' || value[end-1] == '\t' || value[end-1] == '\r' || value[end-1] == '\n') {
		end--
	}
	return value[start:end]
}

func endsWithSemicolon(value string) bool {
	value = trimSpace(value)
	return len(value) > 0 && value[len(value)-1] == ';'
}

func (s *HistoryStore) runWriter() {
	defer close(s.writerDone)
	for {
		select {
		case task := <-s.queue:
			err := s.executeTask(task)
			task.done <- err
		case <-s.stop:
			return
		}
	}
}

func (s *HistoryStore) executeTask(task historyWriteTask) error {
	if task.batch != nil && !task.batch.empty() {
		if err := s.writeBatch(task.batch); err != nil {
			return err
		}
	}
	if task.cleanupUnix > 0 {
		if err := s.database.Exec("DELETE FROM stat_request_buckets WHERE bucket_start < ?", task.cleanupUnix).Error; err != nil {
			return err
		}
		if err := s.database.Exec("DELETE FROM stat_domain_buckets WHERE bucket_start < ?", task.cleanupUnix).Error; err != nil {
			return err
		}
		if err := s.database.Exec("DELETE FROM stat_resource_samples WHERE bucket_start < ?", task.cleanupUnix).Error; err != nil {
			return err
		}
		// INCREMENTAL auto_vacuum is intentionally bounded so cleanup never
		// monopolizes the SQLite writer for a large historical file.
		if err := s.database.Exec("PRAGMA incremental_vacuum(64)").Error; err != nil {
			logger.GetLogger().Debug().Err(err).Msg("stat history incremental vacuum failed")
		}
	}
	return nil
}

func (s *HistoryStore) writeBatch(batch *historyBatch) error {
	return s.database.Transaction(func(tx *gorm.DB) error {
		for key, value := range batch.Requests {
			args := []interface{}{
				key.BucketStart, string(key.Route), value.Requests, value.Errors,
				value.Status1xx, value.Status2xx, value.Status3xx, value.Status4xx, value.Status5xx,
				value.RequestBytes, value.ResponseBytes, value.LatencySumUS, value.LatencyMaxUS,
			}
			for _, count := range value.LatencyBuckets {
				args = append(args, count)
			}
			if err := tx.Exec(`
INSERT INTO stat_request_buckets (
  bucket_start, route, requests, errors, status_1xx, status_2xx, status_3xx,
  status_4xx, status_5xx, request_bytes, response_bytes, latency_sum_us,
  latency_max_us, latency_bucket_0, latency_bucket_1, latency_bucket_2,
  latency_bucket_3, latency_bucket_4, latency_bucket_5, latency_bucket_6,
  latency_bucket_7, latency_bucket_8, latency_bucket_9, latency_bucket_10
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(bucket_start, route) DO UPDATE SET
  requests = stat_request_buckets.requests + excluded.requests,
  errors = stat_request_buckets.errors + excluded.errors,
  status_1xx = stat_request_buckets.status_1xx + excluded.status_1xx,
  status_2xx = stat_request_buckets.status_2xx + excluded.status_2xx,
  status_3xx = stat_request_buckets.status_3xx + excluded.status_3xx,
  status_4xx = stat_request_buckets.status_4xx + excluded.status_4xx,
  status_5xx = stat_request_buckets.status_5xx + excluded.status_5xx,
  request_bytes = stat_request_buckets.request_bytes + excluded.request_bytes,
  response_bytes = stat_request_buckets.response_bytes + excluded.response_bytes,
  latency_sum_us = stat_request_buckets.latency_sum_us + excluded.latency_sum_us,
  latency_max_us = MAX(stat_request_buckets.latency_max_us, excluded.latency_max_us),
  latency_bucket_0 = stat_request_buckets.latency_bucket_0 + excluded.latency_bucket_0,
  latency_bucket_1 = stat_request_buckets.latency_bucket_1 + excluded.latency_bucket_1,
  latency_bucket_2 = stat_request_buckets.latency_bucket_2 + excluded.latency_bucket_2,
  latency_bucket_3 = stat_request_buckets.latency_bucket_3 + excluded.latency_bucket_3,
  latency_bucket_4 = stat_request_buckets.latency_bucket_4 + excluded.latency_bucket_4,
  latency_bucket_5 = stat_request_buckets.latency_bucket_5 + excluded.latency_bucket_5,
  latency_bucket_6 = stat_request_buckets.latency_bucket_6 + excluded.latency_bucket_6,
  latency_bucket_7 = stat_request_buckets.latency_bucket_7 + excluded.latency_bucket_7,
  latency_bucket_8 = stat_request_buckets.latency_bucket_8 + excluded.latency_bucket_8,
  latency_bucket_9 = stat_request_buckets.latency_bucket_9 + excluded.latency_bucket_9,
  latency_bucket_10 = stat_request_buckets.latency_bucket_10 + excluded.latency_bucket_10`, args...).Error; err != nil {
				return err
			}
		}
		for key, value := range batch.Domains {
			if err := tx.Exec(`
INSERT INTO stat_domain_buckets (bucket_start, domain, requests, errors, request_bytes, response_bytes)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(bucket_start, domain) DO UPDATE SET
  requests = stat_domain_buckets.requests + excluded.requests,
  errors = stat_domain_buckets.errors + excluded.errors,
  request_bytes = stat_domain_buckets.request_bytes + excluded.request_bytes,
  response_bytes = stat_domain_buckets.response_bytes + excluded.response_bytes`,
				key.BucketStart, key.Domain, value.Requests, value.Errors, value.RequestBytes, value.ResponseBytes).Error; err != nil {
				return err
			}
		}
		for bucket, value := range batch.Resources {
			if err := tx.Exec(`
INSERT INTO stat_resource_samples (
  bucket_start, sample_count, system_cpu_sum, system_cpu_count, system_cpu_peak,
  system_memory_sum, system_memory_count, system_memory_peak, process_cpu_sum,
  process_cpu_count, process_cpu_peak, process_memory_sum, process_memory_count,
  process_memory_peak, process_memory_percent_sum, process_memory_percent_count,
  process_memory_percent_peak, system_network_rx_bytes, system_network_tx_bytes,
  system_network_rx_available, system_network_tx_available, system_disk_read_bytes,
  system_disk_write_bytes, system_disk_read_available, system_disk_write_available,
  process_disk_read_bytes, process_disk_write_bytes, process_disk_read_available,
  process_disk_write_available, gc_cycles, gc_forced_cycles, gc_pause_total_ns,
  gc_pause_max_ns, gc_pause_bucket_0, gc_pause_bucket_1, gc_pause_bucket_2,
  gc_pause_bucket_3, gc_pause_bucket_4, gc_pause_bucket_5, gc_pause_bucket_6,
  gc_pause_bucket_7, gc_pause_bucket_8, gc_pause_bucket_9, gc_pause_bucket_10,
  gc_pressure_sum, gc_pressure_count, gc_pressure_peak
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(bucket_start) DO UPDATE SET
  sample_count = stat_resource_samples.sample_count + excluded.sample_count,
  system_cpu_sum = stat_resource_samples.system_cpu_sum + excluded.system_cpu_sum,
  system_cpu_count = stat_resource_samples.system_cpu_count + excluded.system_cpu_count,
  system_cpu_peak = MAX(stat_resource_samples.system_cpu_peak, excluded.system_cpu_peak),
  system_memory_sum = stat_resource_samples.system_memory_sum + excluded.system_memory_sum,
  system_memory_count = stat_resource_samples.system_memory_count + excluded.system_memory_count,
  system_memory_peak = MAX(stat_resource_samples.system_memory_peak, excluded.system_memory_peak),
  process_cpu_sum = stat_resource_samples.process_cpu_sum + excluded.process_cpu_sum,
  process_cpu_count = stat_resource_samples.process_cpu_count + excluded.process_cpu_count,
  process_cpu_peak = MAX(stat_resource_samples.process_cpu_peak, excluded.process_cpu_peak),
  process_memory_sum = stat_resource_samples.process_memory_sum + excluded.process_memory_sum,
  process_memory_count = stat_resource_samples.process_memory_count + excluded.process_memory_count,
  process_memory_peak = MAX(stat_resource_samples.process_memory_peak, excluded.process_memory_peak),
  process_memory_percent_sum = stat_resource_samples.process_memory_percent_sum + excluded.process_memory_percent_sum,
  process_memory_percent_count = stat_resource_samples.process_memory_percent_count + excluded.process_memory_percent_count,
  process_memory_percent_peak = MAX(stat_resource_samples.process_memory_percent_peak, excluded.process_memory_percent_peak),
  system_network_rx_bytes = stat_resource_samples.system_network_rx_bytes + excluded.system_network_rx_bytes,
  system_network_tx_bytes = stat_resource_samples.system_network_tx_bytes + excluded.system_network_tx_bytes,
  system_network_rx_available = MAX(stat_resource_samples.system_network_rx_available, excluded.system_network_rx_available),
  system_network_tx_available = MAX(stat_resource_samples.system_network_tx_available, excluded.system_network_tx_available),
  system_disk_read_bytes = stat_resource_samples.system_disk_read_bytes + excluded.system_disk_read_bytes,
  system_disk_write_bytes = stat_resource_samples.system_disk_write_bytes + excluded.system_disk_write_bytes,
  system_disk_read_available = MAX(stat_resource_samples.system_disk_read_available, excluded.system_disk_read_available),
  system_disk_write_available = MAX(stat_resource_samples.system_disk_write_available, excluded.system_disk_write_available),
  process_disk_read_bytes = stat_resource_samples.process_disk_read_bytes + excluded.process_disk_read_bytes,
  process_disk_write_bytes = stat_resource_samples.process_disk_write_bytes + excluded.process_disk_write_bytes,
  process_disk_read_available = MAX(stat_resource_samples.process_disk_read_available, excluded.process_disk_read_available),
  process_disk_write_available = MAX(stat_resource_samples.process_disk_write_available, excluded.process_disk_write_available),
  gc_cycles = stat_resource_samples.gc_cycles + excluded.gc_cycles,
  gc_forced_cycles = stat_resource_samples.gc_forced_cycles + excluded.gc_forced_cycles,
  gc_pause_total_ns = stat_resource_samples.gc_pause_total_ns + excluded.gc_pause_total_ns,
  gc_pause_max_ns = MAX(stat_resource_samples.gc_pause_max_ns, excluded.gc_pause_max_ns),
  gc_pause_bucket_0 = stat_resource_samples.gc_pause_bucket_0 + excluded.gc_pause_bucket_0,
  gc_pause_bucket_1 = stat_resource_samples.gc_pause_bucket_1 + excluded.gc_pause_bucket_1,
  gc_pause_bucket_2 = stat_resource_samples.gc_pause_bucket_2 + excluded.gc_pause_bucket_2,
  gc_pause_bucket_3 = stat_resource_samples.gc_pause_bucket_3 + excluded.gc_pause_bucket_3,
  gc_pause_bucket_4 = stat_resource_samples.gc_pause_bucket_4 + excluded.gc_pause_bucket_4,
  gc_pause_bucket_5 = stat_resource_samples.gc_pause_bucket_5 + excluded.gc_pause_bucket_5,
  gc_pause_bucket_6 = stat_resource_samples.gc_pause_bucket_6 + excluded.gc_pause_bucket_6,
  gc_pause_bucket_7 = stat_resource_samples.gc_pause_bucket_7 + excluded.gc_pause_bucket_7,
  gc_pause_bucket_8 = stat_resource_samples.gc_pause_bucket_8 + excluded.gc_pause_bucket_8,
  gc_pause_bucket_9 = stat_resource_samples.gc_pause_bucket_9 + excluded.gc_pause_bucket_9,
  gc_pause_bucket_10 = stat_resource_samples.gc_pause_bucket_10 + excluded.gc_pause_bucket_10,
  gc_pressure_sum = stat_resource_samples.gc_pressure_sum + excluded.gc_pressure_sum,
  gc_pressure_count = stat_resource_samples.gc_pressure_count + excluded.gc_pressure_count,
  gc_pressure_peak = MAX(stat_resource_samples.gc_pressure_peak, excluded.gc_pressure_peak)`,
				bucket, value.SampleCount, value.SystemCPUSum, value.SystemCPUCount, value.SystemCPUPeak,
				value.SystemMemorySum, value.SystemMemoryCount, value.SystemMemoryPeak,
				value.ProcessCPUSum, value.ProcessCPUCount, value.ProcessCPUPeak,
				value.ProcessMemorySum, value.ProcessMemoryCount, value.ProcessMemoryPeak,
				value.ProcessMemoryPercentSum, value.ProcessMemoryPercentCount, value.ProcessMemoryPercentPeak,
				value.SystemNetworkRX, value.SystemNetworkTX, boolInt(value.SystemNetworkRXAvail), boolInt(value.SystemNetworkTXAvail),
				value.SystemDiskRead, value.SystemDiskWrite, boolInt(value.SystemDiskReadAvail), boolInt(value.SystemDiskWriteAvail),
				value.ProcessDiskRead, value.ProcessDiskWrite, boolInt(value.ProcessDiskReadAvail), boolInt(value.ProcessDiskWriteAvail),
				value.GCCycles, value.GCForcedCycles, value.GCPauseTotalNS, value.GCPauseMaxNS,
				value.GCPauseBuckets[0], value.GCPauseBuckets[1], value.GCPauseBuckets[2], value.GCPauseBuckets[3], value.GCPauseBuckets[4],
				value.GCPauseBuckets[5], value.GCPauseBuckets[6], value.GCPauseBuckets[7], value.GCPauseBuckets[8], value.GCPauseBuckets[9], value.GCPauseBuckets[10],
				value.GCPressureSum, value.GCPressureCount, value.GCPressurePeak).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

// RecordObservation adds one request to the in-memory overlay.
func (s *HistoryStore) RecordObservation(observation RequestSnapshot) {
	if s == nil || s.closed.Load() {
		return
	}
	if observation.StartedAt.IsZero() {
		observation.StartedAt = time.Now().UTC()
	}
	route := observation.Route
	if route != RouteFrontend && route != RouteBackend {
		route = RouteUnknown
	}
	statusCode := observation.StatusCode
	if statusCode == 0 {
		statusCode = httpStatusOK
	}
	latencyUS := observation.Duration.Microseconds()
	if latencyUS < 0 {
		latencyUS = 0
	}
	delta := requestBucket{
		Requests:       1,
		RequestBytes:   maxInt64(observation.RequestBytes, 0),
		ResponseBytes:  maxInt64(observation.ResponseBytes, 0),
		LatencySumUS:   latencyUS,
		LatencyMaxUS:   latencyUS,
		LatencyBuckets: [requestHistogramSize]int64{},
	}
	delta.LatencyBuckets[latencyHistogramIndex(latencyUS)] = 1
	if statusCode >= 400 {
		delta.Errors = 1
	}
	switch statusCode / 100 {
	case 1:
		delta.Status1xx = 1
	case 2:
		delta.Status2xx = 1
	case 3:
		delta.Status3xx = 1
	case 4:
		delta.Status4xx = 1
	case 5:
		delta.Status5xx = 1
	default:
		// Non-standard status codes remain part of the request total/error
		// count but do not get forced into a misleading status class.
	}

	s.overlayMu.Lock()
	if s.closed.Load() {
		s.overlayMu.Unlock()
		return
	}
	key := requestBucketKey{BucketStart: bucketStart(observation.StartedAt), Route: route}
	current := s.overlay.Requests[key]
	mergeRequestBucket(&current, delta)
	s.overlay.Requests[key] = current
	if domain := NormalizeDomain(observation.Domain); domain != "" {
		domainKey := domainBucketKey{BucketStart: key.BucketStart, Domain: domain}
		domainCurrent := s.overlay.Domains[domainKey]
		domainCurrent.Requests++
		domainCurrent.RequestBytes += delta.RequestBytes
		domainCurrent.ResponseBytes += delta.ResponseBytes
		if delta.Errors > 0 {
			domainCurrent.Errors++
		}
		s.overlay.Domains[domainKey] = domainCurrent
	}
	s.overlayMu.Unlock()
}

const httpStatusOK = 200

func maxInt64(value, floor int64) int64 {
	if value < floor {
		return floor
	}
	return value
}

func mergeRequestBucket(dst *requestBucket, src requestBucket) {
	dst.Requests += src.Requests
	dst.Errors += src.Errors
	dst.Status1xx += src.Status1xx
	dst.Status2xx += src.Status2xx
	dst.Status3xx += src.Status3xx
	dst.Status4xx += src.Status4xx
	dst.Status5xx += src.Status5xx
	dst.RequestBytes += src.RequestBytes
	dst.ResponseBytes += src.ResponseBytes
	dst.LatencySumUS += src.LatencySumUS
	if src.LatencyMaxUS > dst.LatencyMaxUS {
		dst.LatencyMaxUS = src.LatencyMaxUS
	}
	for index := range dst.LatencyBuckets {
		dst.LatencyBuckets[index] += src.LatencyBuckets[index]
	}
}

// RecordResource adds one five-second resource reading to its minute bucket.
func (s *HistoryStore) RecordResource(reading ResourceReading) {
	if s == nil || s.closed.Load() {
		return
	}
	when := reading.Timestamp
	if when.IsZero() {
		when = time.Now().UTC()
	}
	sample := resourceBucket{SampleCount: 1}
	if reading.SystemCPUPercent != nil {
		sample.SystemCPUCount = 1
		sample.SystemCPUSum = *reading.SystemCPUPercent
		sample.SystemCPUPeak = *reading.SystemCPUPercent
	}
	if reading.SystemMemoryPercent != nil {
		sample.SystemMemoryCount = 1
		sample.SystemMemorySum = *reading.SystemMemoryPercent
		sample.SystemMemoryPeak = *reading.SystemMemoryPercent
	}
	if reading.ProcessCPUPercent != nil {
		sample.ProcessCPUCount = 1
		sample.ProcessCPUSum = *reading.ProcessCPUPercent
		sample.ProcessCPUPeak = *reading.ProcessCPUPercent
	}
	if reading.ProcessMemoryBytes != nil {
		sample.ProcessMemoryCount = 1
		sample.ProcessMemorySum = float64(*reading.ProcessMemoryBytes)
		sample.ProcessMemoryPeak = float64(*reading.ProcessMemoryBytes)
	}
	if reading.ProcessMemoryPercent != nil {
		sample.ProcessMemoryPercentCount = 1
		sample.ProcessMemoryPercentSum = *reading.ProcessMemoryPercent
		sample.ProcessMemoryPercentPeak = *reading.ProcessMemoryPercent
	}
	if reading.SystemNetworkRXBytes != nil {
		sample.SystemNetworkRXAvail = true
		sample.SystemNetworkRX = maxInt64(*reading.SystemNetworkRXBytes, 0)
	}
	if reading.SystemNetworkTXBytes != nil {
		sample.SystemNetworkTXAvail = true
		sample.SystemNetworkTX = maxInt64(*reading.SystemNetworkTXBytes, 0)
	}
	if reading.SystemDiskReadBytes != nil {
		sample.SystemDiskReadAvail = true
		sample.SystemDiskRead = maxInt64(*reading.SystemDiskReadBytes, 0)
	}
	if reading.SystemDiskWriteBytes != nil {
		sample.SystemDiskWriteAvail = true
		sample.SystemDiskWrite = maxInt64(*reading.SystemDiskWriteBytes, 0)
	}
	if reading.ProcessDiskReadBytes != nil {
		sample.ProcessDiskReadAvail = true
		sample.ProcessDiskRead = maxInt64(*reading.ProcessDiskReadBytes, 0)
	}
	if reading.ProcessDiskWriteBytes != nil {
		sample.ProcessDiskWriteAvail = true
		sample.ProcessDiskWrite = maxInt64(*reading.ProcessDiskWriteBytes, 0)
	}
	if reading.GCCycles != nil {
		sample.GCCycles = maxInt64(*reading.GCCycles, 0)
	}
	if reading.GCForcedCycles != nil {
		sample.GCForcedCycles = maxInt64(*reading.GCForcedCycles, 0)
	}
	if reading.GCPauseTotalNS != nil {
		sample.GCPauseTotalNS = maxInt64(*reading.GCPauseTotalNS, 0)
	}
	if reading.GCPauseMaxNS != nil {
		sample.GCPauseMaxNS = maxInt64(*reading.GCPauseMaxNS, 0)
	}
	if reading.GCCycles != nil || reading.GCPauseTotalNS != nil || reading.GCPauseMaxNS != nil {
		copy(sample.GCPauseBuckets[:], reading.GCPauseBuckets[:])
	}
	if reading.GCPressurePercent != nil {
		pressure := *reading.GCPressurePercent
		if pressure < 0 {
			pressure = 0
		}
		sample.GCPressureSum = pressure
		sample.GCPressureCount = 1
		sample.GCPressurePeak = pressure
	}

	s.overlayMu.Lock()
	if s.closed.Load() {
		s.overlayMu.Unlock()
		return
	}
	key := bucketStart(when)
	current := s.overlay.Resources[key]
	mergeResourceBucket(&current, sample)
	s.overlay.Resources[key] = current
	s.overlayMu.Unlock()
}

func mergeResourceBucket(dst *resourceBucket, src resourceBucket) {
	dst.SampleCount += src.SampleCount
	dst.SystemCPUSum += src.SystemCPUSum
	dst.SystemCPUCount += src.SystemCPUCount
	if src.SystemCPUPeak > dst.SystemCPUPeak {
		dst.SystemCPUPeak = src.SystemCPUPeak
	}
	dst.SystemMemorySum += src.SystemMemorySum
	dst.SystemMemoryCount += src.SystemMemoryCount
	if src.SystemMemoryPeak > dst.SystemMemoryPeak {
		dst.SystemMemoryPeak = src.SystemMemoryPeak
	}
	dst.ProcessCPUSum += src.ProcessCPUSum
	dst.ProcessCPUCount += src.ProcessCPUCount
	if src.ProcessCPUPeak > dst.ProcessCPUPeak {
		dst.ProcessCPUPeak = src.ProcessCPUPeak
	}
	dst.ProcessMemorySum += src.ProcessMemorySum
	dst.ProcessMemoryCount += src.ProcessMemoryCount
	if src.ProcessMemoryPeak > dst.ProcessMemoryPeak {
		dst.ProcessMemoryPeak = src.ProcessMemoryPeak
	}
	dst.ProcessMemoryPercentSum += src.ProcessMemoryPercentSum
	dst.ProcessMemoryPercentCount += src.ProcessMemoryPercentCount
	if src.ProcessMemoryPercentPeak > dst.ProcessMemoryPercentPeak {
		dst.ProcessMemoryPercentPeak = src.ProcessMemoryPercentPeak
	}
	dst.SystemNetworkRX += src.SystemNetworkRX
	dst.SystemNetworkTX += src.SystemNetworkTX
	dst.SystemNetworkRXAvail = dst.SystemNetworkRXAvail || src.SystemNetworkRXAvail
	dst.SystemNetworkTXAvail = dst.SystemNetworkTXAvail || src.SystemNetworkTXAvail
	dst.SystemDiskRead += src.SystemDiskRead
	dst.SystemDiskWrite += src.SystemDiskWrite
	dst.SystemDiskReadAvail = dst.SystemDiskReadAvail || src.SystemDiskReadAvail
	dst.SystemDiskWriteAvail = dst.SystemDiskWriteAvail || src.SystemDiskWriteAvail
	dst.ProcessDiskRead += src.ProcessDiskRead
	dst.ProcessDiskWrite += src.ProcessDiskWrite
	dst.ProcessDiskReadAvail = dst.ProcessDiskReadAvail || src.ProcessDiskReadAvail
	dst.ProcessDiskWriteAvail = dst.ProcessDiskWriteAvail || src.ProcessDiskWriteAvail
	mergeGCBucket(&dst.gcBucket, src.gcBucket)
}

func mergeGCBucket(dst *gcBucket, src gcBucket) {
	dst.GCCycles += src.GCCycles
	dst.GCForcedCycles += src.GCForcedCycles
	dst.GCPauseTotalNS += src.GCPauseTotalNS
	if src.GCPauseMaxNS > dst.GCPauseMaxNS {
		dst.GCPauseMaxNS = src.GCPauseMaxNS
	}
	for index := range dst.GCPauseBuckets {
		dst.GCPauseBuckets[index] += src.GCPauseBuckets[index]
	}
	dst.GCPressureSum += src.GCPressureSum
	dst.GCPressureCount += src.GCPressureCount
	if src.GCPressurePeak > dst.GCPressurePeak {
		dst.GCPressurePeak = src.GCPressurePeak
	}
}

// Flush synchronously commits the current overlay through the single writer.
func (s *HistoryStore) Flush() error {
	if s == nil || s.closed.Load() {
		return nil
	}
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	return s.flushLocked()
}

func (s *HistoryStore) flushLocked() error {
	// Rotate the overlay before enqueueing the write. New observations can
	// continue accumulating in a fresh map while SQLite commits the snapshot,
	// so a successful flush never has to subtract counters or reconstruct a
	// latency maximum from a partially changed bucket.
	s.overlayMu.Lock()
	batch := s.overlay
	if batch.empty() {
		s.overlayMu.Unlock()
		return nil
	}
	s.overlay = newHistoryBatch()
	s.overlayMu.Unlock()
	done := make(chan error, 1)
	s.queue <- historyWriteTask{batch: batch, done: done}
	err := <-done
	if err != nil {
		s.overlayMu.Lock()
		mergeHistoryBatch(s.overlay, batch)
		s.overlayMu.Unlock()
		return err
	}
	return nil
}

// CleanupExpired flushes pending data and deletes buckets older than the
// configured rolling retention window.
func (s *HistoryStore) CleanupExpired(now time.Time) error {
	if s == nil || s.closed.Load() {
		return nil
	}
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	if err := s.flushLocked(); err != nil {
		return err
	}
	cutoff := now.UTC().Add(-time.Duration(s.config.RetentionDays) * 24 * time.Hour)
	cutoffUnix := bucketStart(cutoff)
	done := make(chan error, 1)
	s.queue <- historyWriteTask{cleanupUnix: cutoffUnix, done: done}
	return <-done
}

// Close flushes the overlay, drains the writer and leaves old sequence tables
// untouched. The shared DB itself is closed by StatManager after this call.
func (s *HistoryStore) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		s.flushMu.Lock()
		// Prevent a request that passed the fast-path check before shutdown from
		// being added to the fresh overlay after the final flush snapshot.
		s.closed.Store(true)
		s.closeErr = s.flushLocked()
		close(s.stop)
		<-s.writerDone
		s.flushMu.Unlock()
	})
	return s.closeErr
}

type requestBucketRow struct {
	BucketStart     int64  `gorm:"column:bucket_start"`
	Route           string `gorm:"column:route"`
	Requests        int64  `gorm:"column:requests"`
	Errors          int64  `gorm:"column:errors"`
	Status1xx       int64  `gorm:"column:status_1xx"`
	Status2xx       int64  `gorm:"column:status_2xx"`
	Status3xx       int64  `gorm:"column:status_3xx"`
	Status4xx       int64  `gorm:"column:status_4xx"`
	Status5xx       int64  `gorm:"column:status_5xx"`
	RequestBytes    int64  `gorm:"column:request_bytes"`
	ResponseBytes   int64  `gorm:"column:response_bytes"`
	LatencySumUS    int64  `gorm:"column:latency_sum_us"`
	LatencyMaxUS    int64  `gorm:"column:latency_max_us"`
	LatencyBucket0  int64  `gorm:"column:latency_bucket_0"`
	LatencyBucket1  int64  `gorm:"column:latency_bucket_1"`
	LatencyBucket2  int64  `gorm:"column:latency_bucket_2"`
	LatencyBucket3  int64  `gorm:"column:latency_bucket_3"`
	LatencyBucket4  int64  `gorm:"column:latency_bucket_4"`
	LatencyBucket5  int64  `gorm:"column:latency_bucket_5"`
	LatencyBucket6  int64  `gorm:"column:latency_bucket_6"`
	LatencyBucket7  int64  `gorm:"column:latency_bucket_7"`
	LatencyBucket8  int64  `gorm:"column:latency_bucket_8"`
	LatencyBucket9  int64  `gorm:"column:latency_bucket_9"`
	LatencyBucket10 int64  `gorm:"column:latency_bucket_10"`
}

func (r requestBucketRow) bucket() requestBucket {
	return requestBucket{
		Requests: r.Requests, Errors: r.Errors,
		Status1xx: r.Status1xx, Status2xx: r.Status2xx, Status3xx: r.Status3xx,
		Status4xx: r.Status4xx, Status5xx: r.Status5xx,
		RequestBytes: r.RequestBytes, ResponseBytes: r.ResponseBytes,
		LatencySumUS: r.LatencySumUS, LatencyMaxUS: r.LatencyMaxUS,
		LatencyBuckets: [requestHistogramSize]int64{
			r.LatencyBucket0, r.LatencyBucket1, r.LatencyBucket2, r.LatencyBucket3,
			r.LatencyBucket4, r.LatencyBucket5, r.LatencyBucket6, r.LatencyBucket7,
			r.LatencyBucket8, r.LatencyBucket9, r.LatencyBucket10,
		},
	}
}

type domainBucketRow struct {
	BucketStart   int64  `gorm:"column:bucket_start"`
	Domain        string `gorm:"column:domain"`
	Requests      int64  `gorm:"column:requests"`
	Errors        int64  `gorm:"column:errors"`
	RequestBytes  int64  `gorm:"column:request_bytes"`
	ResponseBytes int64  `gorm:"column:response_bytes"`
}

type resourceBucketRow struct {
	BucketStart               int64   `gorm:"column:bucket_start"`
	SampleCount               int64   `gorm:"column:sample_count"`
	SystemCPUSum              float64 `gorm:"column:system_cpu_sum"`
	SystemCPUCount            int64   `gorm:"column:system_cpu_count"`
	SystemCPUPeak             float64 `gorm:"column:system_cpu_peak"`
	SystemMemorySum           float64 `gorm:"column:system_memory_sum"`
	SystemMemoryCount         int64   `gorm:"column:system_memory_count"`
	SystemMemoryPeak          float64 `gorm:"column:system_memory_peak"`
	ProcessCPUSum             float64 `gorm:"column:process_cpu_sum"`
	ProcessCPUCount           int64   `gorm:"column:process_cpu_count"`
	ProcessCPUPeak            float64 `gorm:"column:process_cpu_peak"`
	ProcessMemorySum          float64 `gorm:"column:process_memory_sum"`
	ProcessMemoryCount        int64   `gorm:"column:process_memory_count"`
	ProcessMemoryPeak         float64 `gorm:"column:process_memory_peak"`
	ProcessMemoryPercentSum   float64 `gorm:"column:process_memory_percent_sum"`
	ProcessMemoryPercentCount int64   `gorm:"column:process_memory_percent_count"`
	ProcessMemoryPercentPeak  float64 `gorm:"column:process_memory_percent_peak"`
	SystemNetworkRX           int64   `gorm:"column:system_network_rx_bytes"`
	SystemNetworkTX           int64   `gorm:"column:system_network_tx_bytes"`
	SystemNetworkRXAvailable  int64   `gorm:"column:system_network_rx_available"`
	SystemNetworkTXAvailable  int64   `gorm:"column:system_network_tx_available"`
	SystemDiskRead            int64   `gorm:"column:system_disk_read_bytes"`
	SystemDiskWrite           int64   `gorm:"column:system_disk_write_bytes"`
	SystemDiskReadAvailable   int64   `gorm:"column:system_disk_read_available"`
	SystemDiskWriteAvailable  int64   `gorm:"column:system_disk_write_available"`
	ProcessDiskRead           int64   `gorm:"column:process_disk_read_bytes"`
	ProcessDiskWrite          int64   `gorm:"column:process_disk_write_bytes"`
	ProcessDiskReadAvailable  int64   `gorm:"column:process_disk_read_available"`
	ProcessDiskWriteAvailable int64   `gorm:"column:process_disk_write_available"`
	GCCycles                  int64   `gorm:"column:gc_cycles"`
	GCForcedCycles            int64   `gorm:"column:gc_forced_cycles"`
	GCPauseTotalNS            int64   `gorm:"column:gc_pause_total_ns"`
	GCPauseMaxNS              int64   `gorm:"column:gc_pause_max_ns"`
	GCPauseBucket0            int64   `gorm:"column:gc_pause_bucket_0"`
	GCPauseBucket1            int64   `gorm:"column:gc_pause_bucket_1"`
	GCPauseBucket2            int64   `gorm:"column:gc_pause_bucket_2"`
	GCPauseBucket3            int64   `gorm:"column:gc_pause_bucket_3"`
	GCPauseBucket4            int64   `gorm:"column:gc_pause_bucket_4"`
	GCPauseBucket5            int64   `gorm:"column:gc_pause_bucket_5"`
	GCPauseBucket6            int64   `gorm:"column:gc_pause_bucket_6"`
	GCPauseBucket7            int64   `gorm:"column:gc_pause_bucket_7"`
	GCPauseBucket8            int64   `gorm:"column:gc_pause_bucket_8"`
	GCPauseBucket9            int64   `gorm:"column:gc_pause_bucket_9"`
	GCPauseBucket10           int64   `gorm:"column:gc_pause_bucket_10"`
	GCPressureSum             float64 `gorm:"column:gc_pressure_sum"`
	GCPressureCount           int64   `gorm:"column:gc_pressure_count"`
	GCPressurePeak            float64 `gorm:"column:gc_pressure_peak"`
}

func (r resourceBucketRow) bucket() resourceBucket {
	return resourceBucket{
		SampleCount:  r.SampleCount,
		SystemCPUSum: r.SystemCPUSum, SystemCPUCount: r.SystemCPUCount, SystemCPUPeak: r.SystemCPUPeak,
		SystemMemorySum: r.SystemMemorySum, SystemMemoryCount: r.SystemMemoryCount, SystemMemoryPeak: r.SystemMemoryPeak,
		ProcessCPUSum: r.ProcessCPUSum, ProcessCPUCount: r.ProcessCPUCount, ProcessCPUPeak: r.ProcessCPUPeak,
		ProcessMemorySum: r.ProcessMemorySum, ProcessMemoryCount: r.ProcessMemoryCount, ProcessMemoryPeak: r.ProcessMemoryPeak,
		ProcessMemoryPercentSum: r.ProcessMemoryPercentSum, ProcessMemoryPercentCount: r.ProcessMemoryPercentCount, ProcessMemoryPercentPeak: r.ProcessMemoryPercentPeak,
		SystemNetworkRX: r.SystemNetworkRX, SystemNetworkTX: r.SystemNetworkTX,
		SystemNetworkRXAvail: r.SystemNetworkRXAvailable > 0, SystemNetworkTXAvail: r.SystemNetworkTXAvailable > 0,
		SystemDiskRead: r.SystemDiskRead, SystemDiskWrite: r.SystemDiskWrite,
		SystemDiskReadAvail: r.SystemDiskReadAvailable > 0, SystemDiskWriteAvail: r.SystemDiskWriteAvailable > 0,
		ProcessDiskRead: r.ProcessDiskRead, ProcessDiskWrite: r.ProcessDiskWrite,
		ProcessDiskReadAvail: r.ProcessDiskReadAvailable > 0, ProcessDiskWriteAvail: r.ProcessDiskWriteAvailable > 0,
		gcBucket: gcBucket{
			GCCycles: r.GCCycles, GCForcedCycles: r.GCForcedCycles, GCPauseTotalNS: r.GCPauseTotalNS, GCPauseMaxNS: r.GCPauseMaxNS,
			GCPauseBuckets: [requestHistogramSize]int64{
				r.GCPauseBucket0, r.GCPauseBucket1, r.GCPauseBucket2, r.GCPauseBucket3, r.GCPauseBucket4, r.GCPauseBucket5,
				r.GCPauseBucket6, r.GCPauseBucket7, r.GCPauseBucket8, r.GCPauseBucket9, r.GCPauseBucket10,
			},
			GCPressureSum: r.GCPressureSum, GCPressureCount: r.GCPressureCount, GCPressurePeak: r.GCPressurePeak,
		},
	}
}

type HistoryRows struct {
	Requests  []requestBucketRow
	Domains   []domainBucketRow
	Resources []resourceBucketRow
}

// QueryRows returns raw minute buckets and includes unflushed overlay data.
func (s *HistoryStore) QueryRows(from, to time.Time, domain string) (HistoryRows, error) {
	if s == nil || s.closed.Load() {
		return HistoryRows{}, nil
	}
	// A flush rotates the overlay before its writer commits the snapshot. Hold
	// the same mutex across the database reads and overlay merge so a query can
	// never observe the gap between those two operations.
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	if s.closed.Load() {
		return HistoryRows{}, nil
	}
	fromUnix := bucketStart(from)
	toUnix := bucketStart(to)
	if toUnix < fromUnix {
		fromUnix, toUnix = toUnix, fromUnix
	}
	rows := HistoryRows{}
	if err := s.database.Raw("SELECT * FROM stat_request_buckets WHERE bucket_start >= ? AND bucket_start <= ? ORDER BY bucket_start, route", fromUnix, toUnix).Scan(&rows.Requests).Error; err != nil {
		return HistoryRows{}, err
	}
	if domain == "" {
		if err := s.database.Raw("SELECT * FROM stat_domain_buckets WHERE bucket_start >= ? AND bucket_start <= ? ORDER BY bucket_start, domain", fromUnix, toUnix).Scan(&rows.Domains).Error; err != nil {
			return HistoryRows{}, err
		}
	} else {
		if err := s.database.Raw("SELECT * FROM stat_domain_buckets WHERE bucket_start >= ? AND bucket_start <= ? AND domain = ? ORDER BY bucket_start", fromUnix, toUnix, NormalizeDomain(domain)).Scan(&rows.Domains).Error; err != nil {
			return HistoryRows{}, err
		}
	}
	if err := s.database.Raw("SELECT * FROM stat_resource_samples WHERE bucket_start >= ? AND bucket_start <= ? ORDER BY bucket_start", fromUnix, toUnix).Scan(&rows.Resources).Error; err != nil {
		return HistoryRows{}, err
	}

	s.overlayMu.RLock()
	defer s.overlayMu.RUnlock()
	for key, value := range s.overlay.Requests {
		if key.BucketStart < fromUnix || key.BucketStart > toUnix {
			continue
		}
		rows.Requests = append(rows.Requests, requestBucketRowFromOverlay(key, value))
	}
	for key, value := range s.overlay.Domains {
		if key.BucketStart < fromUnix || key.BucketStart > toUnix {
			continue
		}
		if domain != "" && key.Domain != NormalizeDomain(domain) {
			continue
		}
		rows.Domains = append(rows.Domains, domainBucketRow{BucketStart: key.BucketStart, Domain: key.Domain, Requests: value.Requests, Errors: value.Errors, RequestBytes: value.RequestBytes, ResponseBytes: value.ResponseBytes})
	}
	for bucket, value := range s.overlay.Resources {
		if bucket < fromUnix || bucket > toUnix {
			continue
		}
		rows.Resources = append(rows.Resources, resourceBucketRowFromOverlay(bucket, value))
	}
	return rows, nil
}

func requestBucketRowFromOverlay(key requestBucketKey, value requestBucket) requestBucketRow {
	return requestBucketRow{
		BucketStart: key.BucketStart, Route: string(key.Route), Requests: value.Requests, Errors: value.Errors,
		Status1xx: value.Status1xx, Status2xx: value.Status2xx, Status3xx: value.Status3xx, Status4xx: value.Status4xx, Status5xx: value.Status5xx,
		RequestBytes: value.RequestBytes, ResponseBytes: value.ResponseBytes, LatencySumUS: value.LatencySumUS, LatencyMaxUS: value.LatencyMaxUS,
		LatencyBucket0: value.LatencyBuckets[0], LatencyBucket1: value.LatencyBuckets[1], LatencyBucket2: value.LatencyBuckets[2], LatencyBucket3: value.LatencyBuckets[3], LatencyBucket4: value.LatencyBuckets[4], LatencyBucket5: value.LatencyBuckets[5], LatencyBucket6: value.LatencyBuckets[6], LatencyBucket7: value.LatencyBuckets[7], LatencyBucket8: value.LatencyBuckets[8], LatencyBucket9: value.LatencyBuckets[9], LatencyBucket10: value.LatencyBuckets[10],
	}
}

func resourceBucketRowFromOverlay(bucket int64, value resourceBucket) resourceBucketRow {
	return resourceBucketRow{
		BucketStart: bucket, SampleCount: value.SampleCount,
		SystemCPUSum: value.SystemCPUSum, SystemCPUCount: value.SystemCPUCount, SystemCPUPeak: value.SystemCPUPeak,
		SystemMemorySum: value.SystemMemorySum, SystemMemoryCount: value.SystemMemoryCount, SystemMemoryPeak: value.SystemMemoryPeak,
		ProcessCPUSum: value.ProcessCPUSum, ProcessCPUCount: value.ProcessCPUCount, ProcessCPUPeak: value.ProcessCPUPeak,
		ProcessMemorySum: value.ProcessMemorySum, ProcessMemoryCount: value.ProcessMemoryCount, ProcessMemoryPeak: value.ProcessMemoryPeak,
		ProcessMemoryPercentSum: value.ProcessMemoryPercentSum, ProcessMemoryPercentCount: value.ProcessMemoryPercentCount, ProcessMemoryPercentPeak: value.ProcessMemoryPercentPeak,
		SystemNetworkRX: value.SystemNetworkRX, SystemNetworkTX: value.SystemNetworkTX, SystemNetworkRXAvailable: boolInt(value.SystemNetworkRXAvail), SystemNetworkTXAvailable: boolInt(value.SystemNetworkTXAvail),
		SystemDiskRead: value.SystemDiskRead, SystemDiskWrite: value.SystemDiskWrite, SystemDiskReadAvailable: boolInt(value.SystemDiskReadAvail), SystemDiskWriteAvailable: boolInt(value.SystemDiskWriteAvail),
		ProcessDiskRead: value.ProcessDiskRead, ProcessDiskWrite: value.ProcessDiskWrite, ProcessDiskReadAvailable: boolInt(value.ProcessDiskReadAvail), ProcessDiskWriteAvailable: boolInt(value.ProcessDiskWriteAvail),
		GCCycles: value.GCCycles, GCForcedCycles: value.GCForcedCycles, GCPauseTotalNS: value.GCPauseTotalNS, GCPauseMaxNS: value.GCPauseMaxNS,
		GCPauseBucket0: value.GCPauseBuckets[0], GCPauseBucket1: value.GCPauseBuckets[1], GCPauseBucket2: value.GCPauseBuckets[2], GCPauseBucket3: value.GCPauseBuckets[3],
		GCPauseBucket4: value.GCPauseBuckets[4], GCPauseBucket5: value.GCPauseBuckets[5], GCPauseBucket6: value.GCPauseBuckets[6], GCPauseBucket7: value.GCPauseBuckets[7],
		GCPauseBucket8: value.GCPauseBuckets[8], GCPauseBucket9: value.GCPauseBuckets[9], GCPauseBucket10: value.GCPauseBuckets[10],
		GCPressureSum: value.GCPressureSum, GCPressureCount: value.GCPressureCount, GCPressurePeak: value.GCPressurePeak,
	}
}

// RetentionDays exposes the effective configured window to the API layer.
func (s *HistoryStore) RetentionDays() int {
	if s == nil || s.config.RetentionDays <= 0 {
		return HistoryRetentionDays
	}
	return s.config.RetentionDays
}

// Database returns the underlying database for integration tests and admin
// diagnostics; callers must not use it for history writes.
func (s *HistoryStore) Database() *gorm.DB {
	if s == nil {
		return db.GetDB()
	}
	return s.database
}
