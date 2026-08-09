// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/die-net/lrucache/twotier"
)

// The Cache interface defines a cache for storing arbitrary data.  The
// interface is designed to align with httpcache.Cache.
type Cache interface {
	// Get retrieves the cached data for the provided key.
	Get(key string) (data []byte, ok bool)

	// Set caches the provided data.
	Set(key string, data []byte)

	// Delete deletes the cached data at the specified key.
	Delete(key string)
}

// CacheBuilder is a func that builds a Cache implementation based on a configuration URL.
type CacheBuilder func(config *url.URL) (Cache, error)

// RegisterCache registers a CacheBuilder for a given scheme.
// If another func is already registered for the given scheme, this panics.
//
// This is not a stable API.
func RegisterCache(scheme string, fn CacheBuilder) {
	scheme = strings.ToLower(scheme)
	mu.Lock()
	defer mu.Unlock()
	if _, ok := caches[scheme]; ok {
		panic("cache scheme " + scheme + " is already registered")
	}
	caches[scheme] = fn
}

// lookupCacheBuilder returns the CacheBuilder for the provided scheme, if one is registered.
func lookupCacheBuilder(scheme string) (fn CacheBuilder, ok bool) {
	mu.Lock()
	defer mu.Unlock()
	c, ok := caches[scheme]
	return c, ok
}

var (
	mu     sync.Mutex // protects caches
	caches = map[string]CacheBuilder{}
)

// TieredCache allows specifying multiple caches via flags, which will create
// tiered caches using the twotier package.
type TieredCache struct {
	Cache
}

func (tc *TieredCache) String() string {
	return fmt.Sprint(*tc)
}

func (tc *TieredCache) Set(value string) error {
	for _, v := range strings.Fields(value) {
		c, err := parseCache(v)
		if err != nil {
			return err
		}

		if tc.Cache == nil {
			tc.Cache = c
		} else {
			tc.Cache = twotier.New(tc.Cache, c)
		}
	}
	return nil
}

// parseCache parses c returns the specified Cache implementation.
func parseCache(c string) (Cache, error) {
	if c == "" {
		return nil, nil
	}

	// if just "memory" is specified, force it into a URL shape
	// so that it will be passed to the relevant CacherFunc.
	if c == "memory" {
		c += ":"
	}

	u, err := url.Parse(c)
	if err != nil {
		return nil, fmt.Errorf("error parsing cache flag: %w", err)
	}

	// if no scheme, treat input as a file path
	if u.Scheme == "" {
		u.Scheme = "file"
	}

	if c, ok := lookupCacheBuilder(u.Scheme); ok {
		return c(u)
	}

	return nil, fmt.Errorf("unknown cache option: %q", c)
}

// NopCache provides a no-op cache implementation that doesn't actually cache anything.
var NopCache = new(nopCache)

type nopCache struct{}

func (c nopCache) Get(string) ([]byte, bool) { return nil, false }
func (c nopCache) Set(string, []byte)        {}
func (c nopCache) Delete(string)             {}
