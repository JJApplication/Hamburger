package stat

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"Hamburger/internal/config/svr_config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestHistoryStore(t *testing.T) (*HistoryStore, *gorm.DB, string) {
	t.Helper()
	databasePath := filepath.Join(t.TempDir(), "stat.db")
	database, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	store, err := NewHistoryStore(database, svr_config.SequenceConfig{
		Enabled: true, Interval: 60, RetentionDays: 30, FlushInterval: 5, CleanupInterval: 3600,
	})
	if err != nil {
		t.Fatalf("create history store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
		if sqlDB, dbErr := database.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	return store, database, databasePath
}

func TestHistoryStoreCurrentBucketAndStatusClassification(t *testing.T) {
	store, database, _ := newTestHistoryStore(t)
	now := time.Date(2026, time.August, 28, 12, 34, 17, 0, time.UTC)
	store.RecordObservation(RequestSnapshot{
		StartedAt: now, Duration: 120 * time.Millisecond, Domain: "HTTPS://Example.COM:443/path",
		Route: RouteFrontend, StatusCode: 404, RequestBytes: 11, ResponseBytes: 23,
	})
	store.RecordObservation(RequestSnapshot{
		StartedAt: now.Add(3 * time.Second), Duration: 2 * time.Second, Domain: "example.com.",
		Route: RouteBackend, StatusCode: 500, RequestBytes: 5, ResponseBytes: 7,
	})

	rows, err := store.QueryRows(now.Add(-time.Minute), now.Add(time.Minute), "")
	if err != nil {
		t.Fatalf("query current overlay: %v", err)
	}
	if len(rows.Requests) != 2 || len(rows.Domains) != 1 {
		t.Fatalf("current rows = requests %d domains %d, want requests 2 domains 1", len(rows.Requests), len(rows.Domains))
	}
	if rows.Domains[0].Domain != "example.com" || rows.Domains[0].Requests != 2 || rows.Domains[0].Errors != 2 {
		t.Fatalf("domain row = %+v", rows.Domains[0])
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	rows, err = store.QueryRows(now.Add(-time.Minute), now.Add(time.Minute), "example.com")
	if err != nil {
		t.Fatalf("query flushed rows: %v", err)
	}
	if len(rows.Requests) != 2 || len(rows.Domains) != 1 {
		t.Fatalf("flushed rows = requests %d domains %d", len(rows.Requests), len(rows.Domains))
	}

	var requestCount int64
	if err := database.Table(requestBucketTable).Where("bucket_start = ?", bucketStart(now)).Count(&requestCount).Error; err != nil {
		t.Fatalf("count request rows: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("request row count = %d, want 2", requestCount)
	}
}

func TestHistoryStoreConcurrentRecordingAndIncrementalUpsert(t *testing.T) {
	store, _, _ := newTestHistoryStore(t)
	now := time.Date(2026, time.August, 28, 12, 34, 17, 0, time.UTC)
	const workers = 12
	const requestsPerWorker = 25
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index := 0; index < requestsPerWorker; index++ {
				store.RecordObservation(RequestSnapshot{
					StartedAt: now, Duration: 10 * time.Millisecond, Domain: "api.example.com",
					Route: RouteBackend, StatusCode: 200, RequestBytes: 2, ResponseBytes: 4,
				})
			}
		}()
	}
	wait.Wait()
	if err := store.Flush(); err != nil {
		t.Fatalf("flush concurrent batch: %v", err)
	}
	store.RecordObservation(RequestSnapshot{
		StartedAt: now, Duration: 20 * time.Millisecond, Domain: "api.example.com",
		Route: RouteBackend, StatusCode: 200, RequestBytes: 3, ResponseBytes: 5,
	})
	if err := store.Flush(); err != nil {
		t.Fatalf("flush upsert batch: %v", err)
	}

	rows, err := store.QueryRows(now.Add(-time.Minute), now.Add(time.Minute), "api.example.com")
	if err != nil {
		t.Fatalf("query concurrent rows: %v", err)
	}
	if len(rows.Requests) != 1 || rows.Requests[0].Requests != workers*requestsPerWorker+1 {
		t.Fatalf("request rows = %+v", rows.Requests)
	}
	if rows.Requests[0].RequestBytes != workers*requestsPerWorker*2+3 || rows.Requests[0].ResponseBytes != workers*requestsPerWorker*4+5 {
		t.Fatalf("byte totals = request %d response %d", rows.Requests[0].RequestBytes, rows.Requests[0].ResponseBytes)
	}
}

func TestHistoryStoreRestartRecovery(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "stat.db")
	database, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	store, err := NewHistoryStore(database, svr_config.SequenceConfig{Enabled: true, RetentionDays: 30})
	if err != nil {
		t.Fatalf("create history store: %v", err)
	}
	now := time.Date(2026, time.August, 28, 12, 34, 17, 0, time.UTC)
	store.RecordObservation(RequestSnapshot{StartedAt: now, Duration: time.Millisecond, Domain: "recover.example", Route: RouteFrontend, StatusCode: 200})
	if err := store.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	reopened, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	store, err = NewHistoryStore(reopened, svr_config.SequenceConfig{Enabled: true, RetentionDays: 30})
	if err != nil {
		t.Fatalf("create recovered store: %v", err)
	}
	defer func() {
		_ = store.Close()
		if sqlDB, dbErr := reopened.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	}()
	rows, err := store.QueryRows(now.Add(-time.Minute), now.Add(time.Minute), "recover.example")
	if err != nil {
		t.Fatalf("query recovered rows: %v", err)
	}
	if len(rows.Requests) != 1 || rows.Requests[0].Requests != 1 {
		t.Fatalf("recovered requests = %+v", rows.Requests)
	}
}

func TestHistoryStoreCleanupRetentionBoundary(t *testing.T) {
	store, database, _ := newTestHistoryStore(t)
	now := time.Date(2026, time.August, 28, 12, 34, 17, 0, time.UTC)
	cutoff := bucketStart(now.Add(-30 * 24 * time.Hour))
	for _, bucket := range []int64{cutoff - HistoryBucketSeconds, cutoff} {
		if err := database.Exec("INSERT INTO stat_domain_buckets (bucket_start, domain, requests) VALUES (?, ?, ?)", bucket, "boundary.example", 1).Error; err != nil {
			t.Fatalf("insert cleanup fixture: %v", err)
		}
	}
	if err := store.CleanupExpired(now); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	var count int64
	if err := database.Table(domainBucketTable).Where("domain = ?", "boundary.example").Count(&count).Error; err != nil {
		t.Fatalf("count retained rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("retained boundary rows = %d, want 1", count)
	}
}
