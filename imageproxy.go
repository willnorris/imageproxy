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

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/willnorris/go-imageproxy/proxy"
)

// goxc values
var (
	// VERSION is the version string for go-imageproxy.
	VERSION string
	// BUILD_DATE is the timestamp of when go-imageproxy was built.
	BUILD_DATE string
)

var addr = flag.String("addr", "localhost:8080", "TCP address to listen on")
var whitelist = flag.String("whitelist", "", "comma separated list of allowed remote hosts")
var cacheDir = flag.String("cacheDir", "", "directory to use for file cache")
var version = flag.Bool("version", false, "print version information")

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%v\nBuild: %v\n", VERSION, BUILD_DATE)
		return
	}

	fmt.Printf("go-imageproxy (version %v) listening on %s\n", VERSION, *addr)

	var c httpcache.Cache
	if *cacheDir != "" {
		c = diskcache.New(*cacheDir)
	} else {
		c = httpcache.NewMemoryCache()
	}

	p := proxy.NewProxy(nil, c)
	p.MaxWidth = 2000
	p.MaxHeight = 2000
	if *whitelist != "" {
		p.Whitelist = strings.Split(*whitelist, ",")
	}
	server := &http.Server{
		Addr:    *addr,
		Handler: p,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
