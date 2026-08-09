// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"net/url"
	"os"

	"github.com/gomodule/redigo/redis"
	rediscache "github.com/gregjones/httpcache/redis"
	"willnorris.com/go/imageproxy/cache"
)

func init() {
	cache.RegisterCache("redis", buildCache)
}

func buildCache(config *url.URL) (cache.Cache, error) {
	conn, err := redis.DialURL(config.String(), redis.DialPassword(os.Getenv("REDIS_PASSWORD")))
	if err != nil {
		return nil, err
	}
	return rediscache.NewWithClient(conn), nil
}
