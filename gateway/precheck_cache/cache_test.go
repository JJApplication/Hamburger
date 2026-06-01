package precheck_cache

import (
	"testing"
	"time"
)

func TestCache_SetAndIsValid(t *testing.T) {
	c, err := New(Config{
		LifeWindow:       2 * time.Minute,
		CleanupInterval:  50 * time.Millisecond,
		HardMaxCacheSize: 8,
	})
	if err != nil {
		t.Fatalf("New cache err: %v", err)
	}
	defer c.Close()

	key := "k1"
	exp := time.Now().Add(500 * time.Millisecond)
	if err := c.Set(key, exp); err != nil {
		t.Fatalf("Set err: %v", err)
	}
	if !c.IsValid(key, time.Now()) {
		t.Fatalf("expected valid before expire")
	}

	time.Sleep(650 * time.Millisecond)
	if c.IsValid(key, time.Now()) {
		t.Fatalf("expected invalid after expire")
	}
}

