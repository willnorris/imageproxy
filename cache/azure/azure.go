// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"net/url"

	"github.com/PaulARoy/azurestoragecache"
	"willnorris.com/go/imageproxy/cache"
)

func init() {
	cache.RegisterCache("azure", buildCache)
}

func buildCache(config *url.URL) (cache.Cache, error) {
	return azurestoragecache.New("", "", config.Host)
}
