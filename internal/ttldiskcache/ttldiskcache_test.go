// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package ttldiskcache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTTLDiskCache(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "ttldiskcache_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create cache with 1 second TTL
	cache := New(tempDir, 1*time.Second)

	// Test basic set and get
	t.Run("Basic Set and Get", func(t *testing.T) {
		key := "test-key"
		data := []byte("test-data")

		cache.Set(key, data)
		got, exists := cache.Get(key)
		if !exists {
			t.Error("expected data to exist in cache")
		}
		if string(got) != string(data) {
			t.Errorf("got %q, want %q", got, data)
		}
	})

	// Test expiration
	t.Run("Expiration", func(t *testing.T) {
		key := "expiring-key"
		data := []byte("expiring-data")

		cache.Set(key, data)
		time.Sleep(2 * time.Second) // Wait for TTL to expire

		_, exists := cache.Get(key)
		if exists {
			t.Error("expected data to be expired")
		}

		// Give the async deletion some time to complete
		time.Sleep(100 * time.Millisecond)

		// Verify metadata file is cleaned up
		metaPath := filepath.Join(tempDir, "_metadata", key+".meta")
		if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
			t.Error("expected metadata file to be deleted")
		}
	})

	// Test deletion
	t.Run("Delete", func(t *testing.T) {
		key := "delete-key"
		data := []byte("delete-data")

		cache.Set(key, data)
		cache.Delete(key)

		_, exists := cache.Get(key)
		if exists {
			t.Error("expected data to be deleted")
		}

		// Verify metadata file is cleaned up
		metaPath := filepath.Join(tempDir, "_metadata", key+".meta")
		if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
			t.Error("expected metadata file to be deleted")
		}
	})

	// Test cleanup of expired entries
	t.Run("Cleanup Expired", func(t *testing.T) {
		keys := []string{"expire1", "expire2", "valid"}
		for _, key := range keys {
			cache.Set(key, []byte(key+"-data"))
		}

		time.Sleep(2 * time.Second) // Wait for TTL to expire

		// Add one more valid entry
		cache.Set("valid", []byte("valid-data"))

		cache.CleanupExpired()

		// Check expired entries are removed
		for _, key := range keys[:2] { // expire1 and expire2
			_, exists := cache.Get(key)
			if exists {
				t.Errorf("expected %s to be cleaned up", key)
			}
		}

		// Check valid entry still exists
		_, exists := cache.Get("valid")
		if !exists {
			t.Error("expected valid entry to still exist")
		}
	})
}

func TestTTLDiskCacheConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ttldiskcache_concurrent_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache := New(tempDir, 1*time.Second)

	// Test concurrent access
	t.Run("Concurrent Access", func(t *testing.T) {
		const goroutines = 10
		done := make(chan bool)

		for i := 0; i < goroutines; i++ {
			go func(id int) {
				key := "concurrent-key"
				data := []byte("concurrent-data")

				// Perform multiple operations
				cache.Set(key, data)
				cache.Get(key)
				cache.Delete(key)

				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < goroutines; i++ {
			<-done
		}
	})
}
