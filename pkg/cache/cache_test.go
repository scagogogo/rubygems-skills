package cache

import (
	"strconv"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	// Create a cache with 100ms expiration and 200ms cleanup interval
	cache := NewMemoryCache(100*time.Millisecond, 200*time.Millisecond)
	defer cache.Close()

	// Test Set and Get
	t.Run("Set and Get", func(t *testing.T) {
		cache.Set("key1", "value1")
		cache.Set("key2", 2)
		cache.Set("key3", struct{ Name string }{"test"})

		// Check that values are correct
		if val, found := cache.Get("key1"); !found || val.(string) != "value1" {
			t.Errorf("Expected key1=value1, got %v, found=%v", val, found)
		}

		if val, found := cache.Get("key2"); !found || val.(int) != 2 {
			t.Errorf("Expected key2=2, got %v, found=%v", val, found)
		}

		if val, found := cache.Get("key3"); !found || val.(struct{ Name string }).Name != "test" {
			t.Errorf("Expected key3.Name=test, got %v, found=%v", val, found)
		}

		// Check non-existent key
		if _, found := cache.Get("not_exists"); found {
			t.Error("Expected not_exists to not be found")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		cache.Set("key_to_delete", "value")
		if _, found := cache.Get("key_to_delete"); !found {
			t.Error("Expected key_to_delete to be found before deletion")
		}

		cache.Delete("key_to_delete")
		if _, found := cache.Get("key_to_delete"); found {
			t.Error("Expected key_to_delete to not be found after deletion")
		}
	})

	// Test expiration
	t.Run("Expiration", func(t *testing.T) {
		cache.SetWithExpiration("expire_key", "value", 50*time.Millisecond)
		if _, found := cache.Get("expire_key"); !found {
			t.Error("Expected expire_key to be found before expiration")
		}

		// Wait for the item to expire
		time.Sleep(100 * time.Millisecond)
		if _, found := cache.Get("expire_key"); found {
			t.Error("Expected expire_key to not be found after expiration")
		}
	})

	// Test count
	t.Run("Count", func(t *testing.T) {
		cache.Clear()
		cache.Set("key1", "value1")
		cache.Set("key2", "value2")
		if count := cache.Count(); count != 2 {
			t.Errorf("Expected count=2, got %d", count)
		}

		cache.Set("key3", "value3")
		if count := cache.Count(); count != 3 {
			t.Errorf("Expected count=3, got %d", count)
		}

		cache.Delete("key1")
		if count := cache.Count(); count != 2 {
			t.Errorf("Expected count=2 after deletion, got %d", count)
		}

		cache.Clear()
		if count := cache.Count(); count != 0 {
			t.Errorf("Expected count=0 after clear, got %d", count)
		}
	})

	// Test auto cleanup
	t.Run("Auto Cleanup", func(t *testing.T) {
		cleanupCache := NewMemoryCache(50*time.Millisecond, 100*time.Millisecond)
		defer cleanupCache.Close()

		cleanupCache.Set("key1", "value1")
		cleanupCache.Set("key2", "value2")

		// Wait for auto cleanup
		time.Sleep(200 * time.Millisecond)

		if _, found := cleanupCache.Get("key1"); found {
			t.Error("Expected key1 to be automatically cleaned up")
		}

		if _, found := cleanupCache.Get("key2"); found {
			t.Error("Expected key2 to be automatically cleaned up")
		}
	})

	// Test never-expiring cache items
	t.Run("Never Expire", func(t *testing.T) {
		cache.Clear()
		cache.SetWithExpiration("never_expire", "value", -1)

		// Wait for the normal expiration time
		time.Sleep(150 * time.Millisecond)

		// Verify the item still exists
		if val, found := cache.Get("never_expire"); !found || val.(string) != "value" {
			t.Errorf("Expected never_expire to still exist with value='value', got %v, found=%v", val, found)
		}
	})

	// Test cache override
	t.Run("Cache Override", func(t *testing.T) {
		cache.Clear()
		cache.Set("override_key", "original")

		// Verify the original value
		if val, found := cache.Get("override_key"); !found || val.(string) != "original" {
			t.Errorf("Expected override_key=original, got %v, found=%v", val, found)
		}

		// Override the value
		cache.Set("override_key", "updated")

		// Verify the updated value
		if val, found := cache.Get("override_key"); !found || val.(string) != "updated" {
			t.Errorf("Expected override_key=updated, got %v, found=%v", val, found)
		}
	})
}

// Test default values when creating a cache
func TestNewMemoryCache(t *testing.T) {
	// Test default expiration
	t.Run("Default Expiration", func(t *testing.T) {
		cache := NewMemoryCache(0, 0)
		defer cache.Close()

		// Default expiration should be 1 hour
		cache.Set("key", "value")

		// Should be able to get the value normally
		if val, found := cache.Get("key"); !found || val.(string) != "value" {
			t.Errorf("Expected key=value with default expiration, got %v, found=%v", val, found)
		}
	})

	// Test no cleanup interval
	t.Run("No Cleanup Interval", func(t *testing.T) {
		cache := NewMemoryCache(50*time.Millisecond, 0)
		defer cache.Close()

		cache.Set("key", "value")

		// Wait for the item to expire
		time.Sleep(100 * time.Millisecond)

		// Even though the item has expired, without auto cleanup, Get still checks the expiration time
		if _, found := cache.Get("key"); found {
			t.Error("Expected expired key to not be found even without cleanup")
		}
	})
}

// Test multiple concurrent cleanups
func TestMultipleCleanupRoutines(t *testing.T) {
	cache := NewMemoryCache(50*time.Millisecond, 20*time.Millisecond)

	// Add some items
	for i := 0; i < 5; i++ {
		cache.Set(strconv.Itoa(i), i)
	}

	// Wait for a while to let the cleanup routine run multiple times
	time.Sleep(150 * time.Millisecond)

	// Close the cache
	cache.Close()

	// After multiple cleanups, all items should be expired and deleted
	if count := cache.Count(); count != 0 {
		t.Errorf("Expected all items to be cleaned up, but found %d items", count)
	}
}

// Test close and repeated close
func TestClose(t *testing.T) {
	cache := NewMemoryCache(100*time.Millisecond, 200*time.Millisecond)
	cache.Set("key", "value")

	// Normal close
	cache.Close()

	// Verify it can still be used, but the cleanup goroutine has stopped
	cache.Set("key2", "value2")
	val, found := cache.Get("key2")
	if !found || val.(string) != "value2" {
		t.Error("Cache should still be usable after close")
	}

	// Closing again should not cause issues
	cache.Close()
}

// TestSetWithExpirationZeroDurationUsesDefault covers the d == 0 branch in
// SetWithExpiration, which falls back to the cache's default expiration.
func TestSetWithExpirationZeroDurationUsesDefault(t *testing.T) {
	cache := NewMemoryCache(50*time.Millisecond, 0)
	defer cache.Close()

	cache.SetWithExpiration("k", "v", 0)
	if val, found := cache.Get("k"); !found || val.(string) != "v" {
		t.Errorf("expected k=v with default expiration, got %v found=%v", val, found)
	}

	// Wait beyond the default expiration; the item should expire.
	time.Sleep(100 * time.Millisecond)
	if _, found := cache.Get("k"); found {
		t.Error("expected k to expire after default TTL")
	}
}

// Test cache expiration
func TestExpiredItemRemoval(t *testing.T) {
	cache := NewMemoryCache(50*time.Millisecond, 0)
	defer cache.Close()

	// Add some items that expire soon
	cache.Set("expire1", "value1")
	cache.Set("expire2", "value2")
	cache.SetWithExpiration("never_expire", "value3", -1)

	// Wait for some items to expire
	time.Sleep(100 * time.Millisecond)

	// Verify that expired items are no longer accessible
	if _, found := cache.Get("expire1"); found {
		t.Error("Expected expire1 to be expired")
	}

	if _, found := cache.Get("expire2"); found {
		t.Error("Expected expire2 to be expired")
	}

	// Verify the never-expiring item still exists
	if val, found := cache.Get("never_expire"); !found || val.(string) != "value3" {
		t.Errorf("Expected never_expire to still exist, got %v, found=%v", val, found)
	}
}
