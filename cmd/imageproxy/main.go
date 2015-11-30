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
	"os"
	"os/signal"
	"syscall"

	"github.com/sitano/glog"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
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
var filetypes = flag.String("filetypes", "", "comma separated list of allowed file types")
var referrers = flag.String("referrers", "", "comma separated list of allowed referring hosts")
var baseURL = flag.String("baseURL", "", "default base URL for relative remote URLs")
var fileCache = flag.Bool("fileCache", false, "use file cache on disk")
var memCache = flag.Bool("memCache", false, "use memory cache")
var cacheDir = flag.String("cacheDir", "", "directory to use for file cache")
var cacheSize = flag.Uint64("cacheSize", 100, "maximum size of file cache (in MB)")
var signatureKey = flag.String("signatureKey", "", "HMAC key used in calculating request signatures")
var scaleUp = flag.Bool("scaleUp", false, "allow images to scale beyond their original dimensions")
var version = flag.Bool("version", false, "print version information")

func main() {
	flag.Parse()

	//set sighup handler for log rotation
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGHUP)

	//rotate log
	go func() {
		for {
			<-signalChannel
			glog.Rotate()
		}
	}()

	if *version {
		fmt.Printf("%v\nBuild: %v\n", VERSION, BUILD_DATE)
		return
	}

	var c httpcache.Cache
	if *fileCache {
		if *cacheDir != "" {
			d := diskv.New(diskv.Options{
				BasePath:     *cacheDir,
				CacheSizeMax: *cacheSize * 1024 * 1024,
			})
			c = diskcache.NewWithDiskv(d)
		} else {
			log.Fatalf("cacheDir option is mandatory for fileCache use")
		}
	} else if *memCache {
		c = httpcache.NewMemoryCache()
	} else {
		c = nil
	}

	p := imageproxy.NewProxy(nil, c)

	if *whitelist != "" {
		p.Whitelist = strings.Split(*whitelist, ",")
	}

	if *filetypes != "" {
		p.Filetypes = strings.Split(*filetypes, ",")
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
