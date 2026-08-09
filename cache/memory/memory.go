// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/die-net/lrucache"
	"willnorris.com/go/imageproxy/cache"
)

func init() {
	cache.RegisterCache("memory", lruCache)
}

const defaultMemorySize = 100

// lruCache creates an LRU Cache with the specified options of the form
// "maxSize:maxAge".  maxSize is specified in megabytes, maxAge is a duration.
func lruCache(config *url.URL) (cache.Cache, error) {
	options := config.Opaque
	if options == "" {
		options = strconv.Itoa(defaultMemorySize)
	}
	parts := strings.SplitN(options, ":", 2)
	size, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}

	var age time.Duration
	if len(parts) > 1 {
		age, err = time.ParseDuration(parts[1])
		if err != nil {
			return nil, err
		}
	}

	return lrucache.New(size*1e6, int64(age.Seconds())), nil
}
