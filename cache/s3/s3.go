// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package s3

import (
	"net/url"

	"willnorris.com/go/imageproxy/cache"
	"willnorris.com/go/imageproxy/internal/s3cache"
)

func init() {
	cache.RegisterCache("s3", buildCache)
}

func buildCache(config *url.URL) (cache.Cache, error) {
	return s3cache.New(config.String())
}
