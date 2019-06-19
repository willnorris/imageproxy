// Package lrucache provides a byte-size-limited implementation of
// httpcache.Cache that stores data in memory.
package lrucache

import (
	"container/list"
	"sync"
	"time"
)

// LruCache is a thread-safe, in-memory httpcache.Cache that evicts the
// least recently used entries from memory when either MaxSize (in bytes)
// limit would be exceeded or (if set) the entries are older than MaxAge (in
// seconds).  Use the New constructor to create one.
type LruCache struct {
	MaxSize int64
	MaxAge  int64

	mu    sync.Mutex
	cache map[string]*list.Element
	lru   *list.List // Front is least-recent
	size  int64
}

// New creates an LruCache that will restrict itself to maxSize bytes of
// memory.  If maxAge > 0, entries will also be expired after maxAge
// seconds.
func New(maxSize int64, maxAge int64) *LruCache {
	c := &LruCache{
		MaxSize: maxSize,
		MaxAge:  maxAge,
		lru:     list.New(),
		cache:   make(map[string]*list.Element),
	}

	return c
}

// Get returns the []byte representation of a cached response and a bool
// set to true if the key was found.
func (c *LruCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()

	le, ok := c.cache[key]
	if !ok {
		c.mu.Unlock() // Avoiding defer overhead
		return nil, false
	}

	if c.MaxAge > 0 && le.Value.(*entry).expires <= time.Now().Unix() {
		c.deleteElement(le)
		c.maybeDeleteOldest()

		c.mu.Unlock() // Avoiding defer overhead
		return nil, false
	}

	c.lru.MoveToBack(le)
	value := le.Value.(*entry).value

	c.mu.Unlock() // Avoiding defer overhead
	return value, true
}

// Set stores the []byte representation of a response for a given key.
func (c *LruCache) Set(key string, value []byte) {
	c.mu.Lock()

	expires := int64(0)
	if c.MaxAge > 0 {
		expires = time.Now().Unix() + c.MaxAge
	}

	if le, ok := c.cache[key]; ok {
		c.lru.MoveToBack(le)
		e := le.Value.(*entry)
		c.size += int64(len(value)) - int64(len(e.value))
		e.value = value
		e.expires = expires
	} else {
		e := &entry{key: key, value: value, expires: expires}
		c.cache[key] = c.lru.PushBack(e)
		c.size += e.size()
	}

	c.maybeDeleteOldest()

	c.mu.Unlock()
}

// Delete removes the value associated with a key.
func (c *LruCache) Delete(key string) {
	c.mu.Lock()

	if le, ok := c.cache[key]; ok {
		c.deleteElement(le)
	}

	c.mu.Unlock()
}

// Size returns the estimated current memory usage of LruCache.
func (c *LruCache) Size() int64 {
	c.mu.Lock()
	size := c.size
	c.mu.Unlock()

	return size
}

func (c *LruCache) maybeDeleteOldest() {
	for c.size > c.MaxSize {
		le := c.lru.Front()
		if le == nil {
			panic("LruCache: non-zero size but empty lru")
		}
		c.deleteElement(le)
	}

	if c.MaxAge > 0 {
		now := time.Now().Unix()
		for le := c.lru.Front(); le != nil && le.Value.(*entry).expires <= now; le = c.lru.Front() {
			c.deleteElement(le)
		}
	}
}

func (c *LruCache) deleteElement(le *list.Element) {
	c.lru.Remove(le)
	e := le.Value.(*entry)
	delete(c.cache, e.key)
	c.size -= e.size()
}

// Rough estimate of map + entry object + string + byte slice overheads in bytes.
const entryOverhead = 168

type entry struct {
	key     string
	value   []byte
	expires int64
}

func (e *entry) size() int64 {
	return entryOverhead + int64(len(e.key)) + int64(len(e.value))
}
