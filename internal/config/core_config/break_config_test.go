package core_config

import "testing"

func TestBreakConfigDefaultsWhenUnset(t *testing.T) {
	cfg := BreakConfig{}.Effective()
	if !cfg.IsEnabled() {
		t.Fatal("unset breaker should be enabled")
	}
	if cfg.MaxError != DefaultBreakMaxError || cfg.Reset != DefaultBreakResetSeconds || cfg.Bucket != DefaultBreakBucket {
		t.Fatalf("defaults = %+v", cfg)
	}
	if cfg.MinRequests != DefaultBreakMinRequests || cfg.HalfOpenMaxRequests != DefaultBreakHalfOpenMaxRequests || cfg.SuccessThreshold != DefaultBreakSuccessThreshold {
		t.Fatalf("probe defaults = %+v", cfg)
	}
	if cfg.FailureRatio != 0 {
		t.Fatalf("failure ratio = %v, want 0", cfg.FailureRatio)
	}
	if len(cfg.TripStatusCodes) != 4 || cfg.TripStatusCodes[0] != 500 || cfg.TripStatusCodes[3] != 504 {
		t.Fatalf("status codes = %v", cfg.TripStatusCodes)
	}

	cfg.TripStatusCodes[0] = 1
	again := BreakConfig{}.Effective()
	if again.TripStatusCodes[0] != 500 {
		t.Fatal("default status codes were shared and mutated")
	}
}

func TestBreakConfigNormalizesInvalidValues(t *testing.T) {
	enabled := false
	cfg := BreakConfig{
		Enabled:             &enabled,
		MaxError:            -5,
		Reset:               -1,
		Bucket:              0,
		FailureRatio:        2,
		MinRequests:         -1,
		HalfOpenMaxRequests: -3,
		SuccessThreshold:    0,
	}
	cfg.Normalize()
	if cfg.IsEnabled() {
		t.Fatal("explicit false was enabled")
	}
	if cfg.MaxError != DefaultBreakMaxError || cfg.Reset != DefaultBreakResetSeconds || cfg.Bucket != DefaultBreakBucket {
		t.Fatalf("normalized limits = %+v", cfg)
	}
	if cfg.FailureRatio != 1 || cfg.HalfOpenMaxRequests != DefaultBreakHalfOpenMaxRequests {
		t.Fatalf("normalized ratios = %+v", cfg)
	}
}

func TestDefaultBreakConfigIsEnabled(t *testing.T) {
	cfg := DefaultBreakConfig()
	if !cfg.IsEnabled() || cfg.Reset <= 0 || cfg.MaxError <= 0 || cfg.Bucket <= 0 {
		t.Fatalf("default config = %+v", cfg)
	}
}
