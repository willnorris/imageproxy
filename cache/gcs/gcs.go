// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package gcs

import (
	"net/url"
	"strings"

	"willnorris.com/go/imageproxy/cache"
	"willnorris.com/go/imageproxy/internal/gcscache"
)

func init() {
	cache.RegisterCache("gcs", buildCache)
}

func buildCache(config *url.URL) (cache.Cache, error) {
	return gcscache.New(config.Host, strings.TrimPrefix(config.Path, "/"))
}
