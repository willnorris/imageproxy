// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

// imageproxy starts an HTTP server that proxies requests for remote images.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PaulARoy/azurestoragecache"
	"github.com/die-net/lrucache"
	"github.com/die-net/lrucache/twotier"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/gregjones/httpcache/diskcache"
	rediscache "github.com/gregjones/httpcache/redis"
	"github.com/peterbourgon/diskv"
	"willnorris.com/go/imageproxy"
	"willnorris.com/go/imageproxy/internal/gcscache"
	"willnorris.com/go/imageproxy/internal/s3cache"
	"willnorris.com/go/imageproxy/third_party/envy"
)

const defaultMemorySize = 100

var addr = flag.String("addr", "localhost:8080", "TCP address to listen on")
var allowHosts = flag.String("allowHosts", "", "comma separated list of allowed remote hosts")
var denyHosts = flag.String("denyHosts", "", "comma separated list of denied remote hosts")
var referrers = flag.String("referrers", "", "comma separated list of allowed referring hosts")
var includeReferer = flag.Bool("includeReferer", false, "include referer header in remote requests")
var followRedirects = flag.Bool("followRedirects", true, "follow redirects")
var baseURL = flag.String("baseURL", "", "default base URL for relative remote URLs")
var passRequestHeaders = flag.String("passRequestHeaders", "", "comma separatetd list of request headers to pass to remote server")
var cache tieredCache
var signatureKeys signatureKeyList
var scaleUp = flag.Bool("scaleUp", false, "allow images to scale beyond their original dimensions")
var timeout = flag.Duration("timeout", 0, "time limit for requests served by this proxy")
var verbose = flag.Bool("verbose", false, "print verbose logging messages")
var _ = flag.Bool("version", false, "Deprecated: this flag does nothing")
var contentTypes = flag.String("contentTypes", "image/*", "comma separated list of allowed content types")
var userAgent = flag.String("userAgent", "willnorris/imageproxy", "specify the user-agent used by imageproxy when fetching images from origin website")

func init() {
	flag.Var(&cache, "cache", "location to cache images (see https://github.com/willnorris/imageproxy#cache)")
	flag.Var(&signatureKeys, "signatureKey", "HMAC key used in calculating request signatures")
}

func main() {
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

	r := mux.NewRouter().SkipClean(true).UseEncodedPath()
	r.PathPrefix("/").Handler(p)

	server := &http.Server{
		Addr:    *addr,
		Handler: r,

		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Printf("imageproxy listening on %s\n", server.Addr)
	log.Fatal(server.ListenAndServe())
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

// tieredCache allows specifying multiple caches via flags, which will create
// tiered caches using the twotier package.
type tieredCache struct {
	imageproxy.Cache
}

func (tc *tieredCache) String() string {
	return fmt.Sprint(*tc)
}

func (tc *tieredCache) Set(value string) error {
	for _, v := range strings.Fields(value) {
		c, err := parseCache(v)
		if err != nil {
			return err
		}

		if tc.Cache == nil {
			tc.Cache = c
		} else {
			tc.Cache = twotier.New(tc.Cache, c)
		}
	}
	return nil
}

// parseCache parses c returns the specified Cache implementation.
func parseCache(c string) (imageproxy.Cache, error) {
	if c == "" {
		return nil, nil
	}

	if c == "memory" {
		c = fmt.Sprintf("memory:%d", defaultMemorySize)
	}

	u, err := url.Parse(c)
	if err != nil {
		return nil, fmt.Errorf("error parsing cache flag: %w", err)
	}

	switch u.Scheme {
	case "azure":
		return azurestoragecache.New("", "", u.Host)
	case "gcs":
		return gcscache.New(u.Host, strings.TrimPrefix(u.Path, "/"))
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
		return diskCache(u.Path), nil
	default:
		return diskCache(c), nil
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
