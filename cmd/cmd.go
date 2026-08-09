// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

// Package cmd contains common code for building a command line tool to start an imageproxy instance.
//
// Custom imageproxy binaries can import the desired set of modules,
// then call cmd.Main() to parse flags and start the server.
package cmd

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"willnorris.com/go/imageproxy"
	ipcache "willnorris.com/go/imageproxy/cache"
	"willnorris.com/go/imageproxy/third_party/envy"

	// always include disk and memory cache plugins
	_ "willnorris.com/go/imageproxy/cache/disk"
	_ "willnorris.com/go/imageproxy/cache/memory"
)

var addr = flag.String("addr", "localhost:8080", "address to listen on, either a TCP address or a Unix domain socket path prefixed with unix:")
var allowHosts = flag.String("allowHosts", "", "comma separated list of allowed remote hosts")
var denyHosts = flag.String("denyHosts", "", "comma separated list of denied remote hosts")
var referrers = flag.String("referrers", "", "comma separated list of allowed referring hosts")
var includeReferer = flag.Bool("includeReferer", false, "include referer header in remote requests")
var followRedirects = flag.Bool("followRedirects", true, "follow redirects")
var baseURL = flag.String("baseURL", "", "default base URL for relative remote URLs")
var passRequestHeaders = flag.String("passRequestHeaders", "", "comma separatetd list of request headers to pass to remote server")
var passResponseHeaders = flag.String("passResponseHeaders", "Cache-Control,Last-Modified,Expires,Etag,Link", "comma separated list of response headers to pass from remote server")
var cache ipcache.TieredCache
var signatureKeys signatureKeyList
var scaleUp = flag.Bool("scaleUp", false, "allow images to scale beyond their original dimensions")
var timeout = flag.Duration("timeout", 0, "time limit for requests served by this proxy")
var verbose = flag.Bool("verbose", false, "print verbose logging messages")
var _ = flag.Bool("version", false, "Deprecated: this flag does nothing")
var contentTypes = flag.String("contentTypes", "image/*", "comma separated list of allowed content types")
var userAgent = flag.String("userAgent", "willnorris/imageproxy", "specify the user-agent used by imageproxy when fetching images from origin website")
var minCacheDuration = flag.Duration("minCacheDuration", 0, "minimum duration to cache remote images")
var forceCache = flag.Bool("forceCache", false, "Ignore no-store and private directives in responses")

func init() {
	flag.Var(&cache, "cache", "location to cache images (see https://github.com/willnorris/imageproxy#cache)")
	flag.Var(&signatureKeys, "signatureKey", "HMAC key used in calculating request signatures")
}

func Main() {
	envy.Parse("IMAGEPROXY")
	flag.Parse()

	p := imageproxy.NewProxy(nil, cache.Cache)
	if *allowHosts != "" {
		p.AllowHosts = strings.Split(*allowHosts, ",")
	}
	if *denyHosts != "" {
		p.DenyHosts = strings.Split(*denyHosts, ",")
	}
	if *referrers != "" {
		p.Referrers = strings.Split(*referrers, ",")
	}
	if *contentTypes != "" {
		p.ContentTypes = strings.Split(*contentTypes, ",")
	}
	if *passRequestHeaders != "" {
		p.PassRequestHeaders = strings.Split(*passRequestHeaders, ",")
	}
	if *passResponseHeaders != "" {
		p.PassResponseHeaders = strings.Split(*passResponseHeaders, ",")
	} else {
		// set to a non-nil empty slice to pass no headers.
		p.PassResponseHeaders = []string{}
	}
	p.SignatureKeys = signatureKeys
	if *baseURL != "" {
		var err error
		p.DefaultBaseURL, err = url.Parse(*baseURL)
		if err != nil {
			log.Fatalf("error parsing baseURL: %v", err)
		}
	}

	p.IncludeReferer = *includeReferer
	p.FollowRedirects = *followRedirects
	p.Timeout = *timeout
	p.ScaleUp = *scaleUp
	p.Verbose = *verbose
	p.UserAgent = *userAgent
	p.MinimumCacheDuration = *minCacheDuration
	p.ForceCache = *forceCache

	var ln net.Listener
	var err error

	if path, ok := strings.CutPrefix(*addr, "unix:"); ok {
		ln, err = net.Listen("unix", path)
	} else {
		ln, err = net.Listen("tcp", *addr)
	}
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	server := &http.Server{
		Addr:    *addr,
		Handler: p,

		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Printf("imageproxy listening on %s\n", *addr)
	log.Fatal(server.Serve(ln))
}

type signatureKeyList [][]byte

func (skl *signatureKeyList) String() string {
	return fmt.Sprint(*skl)
}

func (skl *signatureKeyList) Set(value string) error {
	for _, v := range strings.Fields(value) {
		key := []byte(v)
		if strings.HasPrefix(v, "@") {
			file := strings.TrimPrefix(v, "@")
			var err error
			key, err = os.ReadFile(file)
			if err != nil {
				log.Fatalf("error reading signature file: %v", err)
			}
		}
		*skl = append(*skl, key)
	}
	return nil
}
