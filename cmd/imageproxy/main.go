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
var cacheDir = flag.String("cacheDir", "", "directory to use for file cache. can also be s3://s3-us-west-2.amazonaws.com/my-bucket")
var cacheSize = flag.Uint64("cacheSize", 100, "maximum size of file cache (in MB)")
var signatureKey = flag.String("signatureKey", "", "HMAC key used in calculating request signatures")
var scaleUp = flag.Bool("scaleUp", false, "allow images to scale beyond their original dimensions")
var version = flag.Bool("version", false, "print version information")

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%v\nBuild: %v\n", VERSION, BUILD_DATE)
		return
	}

	var c httpcache.Cache
	if *cacheDir != "" {
		_, err := os.Stat(*cacheDir)
		if err == nil {
			d := diskv.New(diskv.Options{
				BasePath:     *cacheDir,
				CacheSizeMax: *cacheSize * 1024 * 1024,
			})
			c = diskcache.NewWithDiskv(d)
		} else if strings.HasPrefix(*cacheDir, "s3://") {
			fmt.Printf("Using s3 as a cache backend: %v\n", *cacheDir)
			bucket_url := strings.Replace(*cacheDir, "s3://", "https://", 1)
			c = s3cache.New(bucket_url)
		} else {
			fmt.Printf("Unknown directory or s3 bucket")
			return
		}
	} else if *cacheSize != 0 {
		c = httpcache.NewMemoryCache()
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

	server := &http.Server{
		Addr:    *addr,
		Handler: p,
	}

	fmt.Printf("imageproxy (version %v) listening on %s\n", VERSION, server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
