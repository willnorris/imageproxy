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
	"strings"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
	"sourcegraph.com/sourcegraph/s3cache"
	"willnorris.com/go/imageproxy"
)

// goxc values
var (
	// VERSION is the version string for imageproxy.
	VERSION = "HEAD"

	// BUILD_DATE is the timestamp of when imageproxy was built.
	BUILD_DATE string
)

var addr = flag.String("addr", "localhost:8080", "TCP address to listen on")
var whitelist = flag.String("whitelist", "", "comma separated list of allowed remote hosts")
var referrers = flag.String("referrers", "", "comma separated list of allowed referring hosts")
var baseURL = flag.String("baseURL", "", "default base URL for relative remote URLs")
var cache = flag.String("cache", "", "location to cache images (see https://github.com/willnorris/imageproxy#cache)")
var cacheDir = flag.String("cacheDir", "", "(Deprecated; use 'cache' instead) directory to use for file cache")
var cacheSize = flag.Uint64("cacheSize", 0, "Deprecated: this flag does nothing")
var signatureKey = flag.String("signatureKey", "", "HMAC key used in calculating request signatures")
var scaleUp = flag.Bool("scaleUp", false, "allow images to scale beyond their original dimensions")
var version = flag.Bool("version", false, "print version information")
var ioptsMode = flag.Bool("ioptsMode", false, "specify transformation options using the query string ?iopts=[options] instead of specifying options in the path")

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%v\nBuild: %v\n", VERSION, BUILD_DATE)
		return
	}

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

	p.ScaleUp = *scaleUp
	p.IoptsMode = *ioptsMode

	server := &http.Server{
		Addr:    *addr,
		Handler: p,
	}

	fmt.Printf("imageproxy (version %v) listening on %s\n", VERSION, server.Addr)
	log.Fatal(server.ListenAndServe())
}

// parseCache parses the cache-related flags and returns the specified Cache implementation.
func parseCache() (imageproxy.Cache, error) {
	if *cache == "" {
		if *cacheDir != "" {
			return diskCache(*cacheDir), nil
		}
		return nil, nil
	}

	if *cache == "memory" {
		return httpcache.NewMemoryCache(), nil
	}

	u, err := url.Parse(*cache)
	if err != nil {
		return nil, fmt.Errorf("error parsing cache flag: %v", err)
	}

	switch u.Scheme {
	case "s3":
		u.Scheme = "https"
		return s3cache.New(u.String()), nil
	case "file":
		fallthrough
	default:
		return diskCache(u.Path), nil
	}
}

func diskCache(path string) *diskcache.Cache {
	d := diskv.New(diskv.Options{
		BasePath: path,

		// For file "c0ffee", store file as "c0/ff/c0ffee"
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
	return diskcache.NewWithDiskv(d)
}
