// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package disk

import (
	"net/url"

	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
	"willnorris.com/go/imageproxy/cache"
)

func init() {
	cache.RegisterCache("file", diskCache)
}

func diskCache(config *url.URL) (cache.Cache, error) {
	path := config.Path
	d := diskv.New(diskv.Options{
		BasePath: path,

		// For file "c0ffee", store file as "c0/ff/c0ffee"
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
	return diskcache.NewWithDiskv(d), nil
}
