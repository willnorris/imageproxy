// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

// Package ttldiskcache provides a disk cache implementation with TTL support
package ttldiskcache

import (
	"bytes"
	"encoding/gob"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
)

// cacheEntry represents a cached item with TTL
type cacheEntry struct {
	Data       []byte    `json:"data"`
	ExpiryTime time.Time `json:"expiry_time"`
}

// TTLDiskCache wraps the standard disk cache with TTL support
type TTLDiskCache struct {
	*diskcache.Cache
	ttl         time.Duration
	metadataDir string
	mu          sync.RWMutex
}

// New creates a new TTLDiskCache with the specified base path and TTL
func New(basePath string, ttl time.Duration) *TTLDiskCache {
	d := diskv.New(diskv.Options{
		BasePath: basePath,
		// For file "c0ffee", store file as "c0/ff/c0ffee"
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})

	metadataDir := filepath.Join(basePath, "_metadata")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		log.Printf("error creating metadata directory: %v", err)
	}

	return &TTLDiskCache{
		Cache:       diskcache.NewWithDiskv(d),
		ttl:         ttl,
		metadataDir: metadataDir,
	}
}

// Get retrieves data from the cache if it exists and hasn't expired
func (c *TTLDiskCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, err := c.loadMetadata(key)
	if err != nil {
		return nil, false
	}

	if !entry.ExpiryTime.IsZero() && time.Now().After(entry.ExpiryTime) {
		// Data has expired, delete it
		go c.Delete(key)
		return nil, false
	}

	return entry.Data, true
}

// Set stores data in the cache with the configured TTL
func (c *TTLDiskCache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := cacheEntry{
		Data:       data,
		ExpiryTime: time.Now().Add(c.ttl),
	}

	if err := c.saveMetadata(key, entry); err != nil {
		log.Printf("error saving cache metadata: %v", err)
		return
	}

	c.Cache.Set(key, data)
}

// Delete removes data from both the cache and its metadata
func (c *TTLDiskCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Cache.Delete(key)
	c.deleteMetadata(key)
}

func (c *TTLDiskCache) metadataPath(key string) string {
	return filepath.Join(c.metadataDir, key+".meta")
}

func (c *TTLDiskCache) saveMetadata(key string, entry cacheEntry) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(entry); err != nil {
		return err
	}

	return os.WriteFile(c.metadataPath(key), buf.Bytes(), 0644)
}

func (c *TTLDiskCache) loadMetadata(key string) (cacheEntry, error) {
	var entry cacheEntry

	data, err := os.ReadFile(c.metadataPath(key))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("error reading cache metadata: %v", err)
		}
		return entry, err
	}

	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&entry); err != nil {
		log.Printf("error decoding cache metadata: %v", err)
		return entry, err
	}

	return entry, nil
}

func (c *TTLDiskCache) deleteMetadata(key string) {
	if err := os.Remove(c.metadataPath(key)); err != nil && !os.IsNotExist(err) {
		log.Printf("error deleting cache metadata: %v", err)
	}
}

// CleanupExpired removes all expired entries from the cache
func (c *TTLDiskCache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.metadataDir)
	if err != nil {
		log.Printf("error reading metadata directory: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		key := entry.Name()
		if filepath.Ext(key) != ".meta" {
			continue
		}

		key = key[:len(key)-5] // remove .meta extension
		metadata, err := c.loadMetadata(key)
		if err != nil {
			continue
		}

		if !metadata.ExpiryTime.IsZero() && time.Now().After(metadata.ExpiryTime) {
			c.Cache.Delete(key)
			c.deleteMetadata(key)
		}
	}
}
