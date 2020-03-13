// Package twotier provides a wrapper for two httpcache.Cache instances,
// allowing you to use both a small and fast cache for popular objects and
// fall back to a larger and slower cache for less popular ones.
package twotier

import (
	"github.com/gregjones/httpcache"
)

// TwoTier creates a two-tiered cache out of two httpcache.Cache instances.
// Reads are favored from first, and writes affect both first and second.
type TwoTier struct {
	first  httpcache.Cache
	second httpcache.Cache
}

// New creates a TwoTier. Both first and second must be non-nil.
func New(first httpcache.Cache, second httpcache.Cache) *TwoTier {
	if first == nil || second == nil || first == second {
		return nil
	}
	return &TwoTier{first: first, second: second}
}

// Get returns the []byte representation of a cached response and a bool set
// to true if the key was found.  It tries the first tier cache, and if
// that's not successful, copies the result from the second tier into the
// first tier.
func (c *TwoTier) Get(key string) ([]byte, bool) {
	if value, ok := c.first.Get(key); ok {
		return value, true
	}

	value, ok := c.second.Get(key)
	if !ok {
		return nil, false
	}

	c.first.Set(key, value)

	return value, true
}

// Set stores the []byte representation of a response for a given key into
// the second tier cache, and deletes the cache entry from the first tier
// cache.
func (c *TwoTier) Set(key string, value []byte) {
	c.second.Set(key, value)
	c.first.Delete(key)
}

// Delete removes the value associated with a key from both the first and
// second tier caches.
func (c *TwoTier) Delete(key string) {
	c.second.Delete(key)
	c.first.Delete(key)
}
