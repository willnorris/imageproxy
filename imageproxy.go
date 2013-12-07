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

	"github.com/willnorris/go-imageproxy/cache"
	"github.com/willnorris/go-imageproxy/proxy"
)

var addr = flag.String("addr", "localhost:8080", "TCP address to listen on")
var whitelist = flag.String("whitelist", "", "comma separated list of allowed remote hosts")

func main() {
	flag.Parse()

	fmt.Printf("go-imageproxy listening on %s\n", *addr)

	p := proxy.NewProxy(nil)
	p.Cache = cache.NewMemoryCache()
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
