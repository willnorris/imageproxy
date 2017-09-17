// Copyright 2013 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// imageproxy starts an HTTP server that proxies requests for remote images.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PaulARoy/azurestoragecache"
	"github.com/die-net/lrucache"
	"github.com/diegomarangoni/gcscache"
	"github.com/garyburd/redigo/redis"
	"github.com/gregjones/httpcache/diskcache"
	rediscache "github.com/gregjones/httpcache/redis"
	"github.com/peterbourgon/diskv"
	"willnorris.com/go/imageproxy"
	"willnorris.com/go/imageproxy/internal/s3cache"
)

const defaultMemorySize = 100

var addr = flag.String("addr", "localhost:8080", "TCP address to listen on")
var whitelist = flag.String("whitelist", "", "comma separated list of allowed remote hosts")
var referrers = flag.String("referrers", "", "comma separated list of allowed referring hosts")
var baseURL = flag.String("baseURL", "", "default base URL for relative remote URLs")
var cache = flag.String("cache", "", "location to cache images (see https://github.com/willnorris/imageproxy#cache)")
var signatureKey = flag.String("signatureKey", "", "HMAC key used in calculating request signatures")
var scaleUp = flag.Bool("scaleUp", false, "allow images to scale beyond their original dimensions")
var timeout = flag.Duration("timeout", 0, "time limit for requests served by this proxy")
var verbose = flag.Bool("verbose", false, "print verbose logging messages")
var version = flag.Bool("version", false, "Deprecated: this flag does nothing")

func main() {
	flag.Parse()

	c, err := parseCache()
	if err != nil {
		log.Fatal(err)
	}

	p := imageproxy.NewProxy(nil, c)
	if *whitelist != "" {
		p.Whitelist = strings.Split(*whitelist, ",")
	}
	if *referrers != "" {
		p.Referrers = strings.Split(*referrers, ",")
	}
	if *signatureKey != "" {
		key := []byte(*signatureKey)
		if strings.HasPrefix(*signatureKey, "@") {
			file := strings.TrimPrefix(*signatureKey, "@")
			var err error
			key, err = ioutil.ReadFile(file)
			if err != nil {
				log.Fatalf("error reading signature file: %v", err)
			}
		}
		p.SignatureKey = key
	}
	if *baseURL != "" {
		var err error
		p.DefaultBaseURL, err = url.Parse(*baseURL)
		if err != nil {
			log.Fatalf("error parsing baseURL: %v", err)
		}
	}

	p.Timeout = *timeout
	p.ScaleUp = *scaleUp
	p.Verbose = *verbose

	server := &http.Server{
		Addr:    *addr,
		Handler: p,
	}

	fmt.Printf("imageproxy listening on %s\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}

// parseCache parses the cache-related flags and returns the specified Cache implementation.
func parseCache() (imageproxy.Cache, error) {
	if *cache == "" {
		return nil, nil
	}

	if *cache == "memory" {
		*cache = fmt.Sprintf("memory:%d", defaultMemorySize)
	}

	u, err := url.Parse(*cache)
	if err != nil {
		return nil, fmt.Errorf("error parsing cache flag: %v", err)
	}

	switch u.Scheme {
	case "azure":
		return azurestoragecache.New("", "", u.Host)
	case "gcs":
		return gcscache.New(u.String()), nil
	case "memory":
		return lruCache(u.Opaque)
	case "redis":
		conn, err := redis.DialURL(u.String(), redis.DialPassword(os.Getenv("REDIS_PASSWORD")))
		if err != nil {
			return nil, err
		}
		return rediscache.NewWithClient(conn), nil
	case "s3":
		return s3cache.New(u.String())
	case "file":
		fallthrough
	default:
		return diskCache(u.Path), nil
	}
}

// lruCache creates an LRU Cache with the specified options of the form
// "maxSize:maxAge".  maxSize is specified in megabytes, maxAge is a duration.
func lruCache(options string) (*lrucache.LruCache, error) {
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

func diskCache(path string) *diskcache.Cache {
	d := diskv.New(diskv.Options{
		BasePath: path,

		// For file "c0ffee", store file as "c0/ff/c0ffee"
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
	return diskcache.NewWithDiskv(d)
}
