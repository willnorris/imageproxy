// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/die-net/lrucache"
	"github.com/google/uuid"
	"github.com/gregjones/httpcache"
)

func TestPeekContentType(t *testing.T) {
	// 1 pixel png image, base64 encoded
	b, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAEUlEQVR4nGJiYGBgAAQAAP//AA8AA/6P688AAAAASUVORK5CYII=")
	got := peekContentType(bufio.NewReader(bytes.NewReader(b)))
	if want := "image/png"; got != want {
		t.Errorf("peekContentType returned %v, want %v", got, want)
	}

	// single zero byte
	got = peekContentType(bufio.NewReader(bytes.NewReader([]byte{0x0})))
	if want := "application/octet-stream"; got != want {
		t.Errorf("peekContentType returned %v, want %v", got, want)
	}
}

func TestCopyHeader(t *testing.T) {
	tests := []struct {
		dst, src http.Header
		keys     []string
		want     http.Header
	}{
		// empty
		{http.Header{}, http.Header{}, nil, http.Header{}},
		{http.Header{}, http.Header{}, []string{}, http.Header{}},
		{http.Header{}, http.Header{}, []string{"A"}, http.Header{}},

		// nothing to copy
		{
			dst:  http.Header{"A": []string{"a1"}},
			src:  http.Header{},
			keys: nil,
			want: http.Header{"A": []string{"a1"}},
		},
		{
			dst:  http.Header{},
			src:  http.Header{"A": []string{"a"}},
			keys: []string{"B"},
			want: http.Header{},
		},

		// copy headers
		{
			dst:  http.Header{"A": []string{"a"}},
			src:  http.Header{"B": []string{"b"}, "C": []string{"c"}},
			keys: []string{"B"},
			want: http.Header{"A": []string{"a"}, "B": []string{"b"}},
		},
	}

	for _, tt := range tests {
		// copy dst map
		got := make(http.Header)
		for k, v := range tt.dst {
			got[k] = v
		}

		copyHeader(got, tt.src, tt.keys...)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("copyHeader(%v, %v, %v) returned %v, want %v", tt.dst, tt.src, tt.keys, got, tt.want)
		}
	}
}

func TestAllowed(t *testing.T) {
	good := []string{"good"}
	key := [][]byte{
		[]byte("c0ffee"),
	}
	multipleKey := [][]byte{
		[]byte("c0ffee"),
		[]byte("beer"),
	}

	genRequest := func(headers map[string]string) *http.Request {
		req := &http.Request{Header: make(http.Header)}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		return req
	}

	now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		url        string
		now        time.Time
		options    Options
		allowHosts []string
		denyHosts  []string
		referrers  []string
		keys       [][]byte
		request    *http.Request
		allowed    bool
	}{
		// no allowHosts or signature key
		{url: "http://test/image", allowed: true},

		// allowHosts
		{url: "http://good/image", allowHosts: good, allowed: true},
		{url: "http://bad/image", allowHosts: good, allowed: false},

		// referrer
		{url: "http://test/image", referrers: good, request: genRequest(map[string]string{"Referer": "http://good/foo"}), allowed: true},
		{url: "http://test/image", referrers: good, request: genRequest(map[string]string{"Referer": "http://bad/foo"}), allowed: false},
		{url: "http://test/image", referrers: good, request: genRequest(map[string]string{"Referer": "MALFORMED!!"}), allowed: false},
		{url: "http://test/image", referrers: good, request: genRequest(map[string]string{}), allowed: false},

		// signature key
		{url: "http://test/image", options: Options{Signature: "NDx5zZHx7QfE8E-ijowRreq6CJJBZjwiRfOVk_mkfQQ="}, keys: key, allowed: true},
		{url: "http://test/image", options: Options{Signature: "NDx5zZHx7QfE8E-ijowRreq6CJJBZjwiRfOVk_mkfQQ="}, keys: multipleKey, allowed: true}, // signed with key "c0ffee"
		{url: "http://test/image", options: Options{Signature: "FWIawYV4SEyI4zKJMeGugM-eJM1eI_jXPEQ20ZgRe4A="}, keys: multipleKey, allowed: true}, // signed with key "beer"
		{url: "http://test/image", options: Options{Signature: "deadbeef"}, keys: key, allowed: false},
		{url: "http://test/image", options: Options{Signature: "deadbeef"}, keys: multipleKey, allowed: false},
		{url: "http://test/image", keys: key, allowed: false},

		// allowHosts and signature
		{url: "http://good/image", allowHosts: good, keys: key, allowed: true},
		{url: "http://bad/image", options: Options{Signature: "gWivrPhXBbsYEwpmWAKjbJEiAEgZwbXbltg95O2tgNI="}, keys: key, allowed: true},
		{url: "http://bad/image", allowHosts: good, keys: key, allowed: false},

		// deny requests that match denyHosts, even if signature is valid or also matches allowHosts
		{url: "http://test/image", denyHosts: []string{"test"}, allowed: false},
		{url: "http://test:3000/image", denyHosts: []string{"test"}, allowed: false},
		{url: "http://test/image", allowHosts: []string{"test"}, denyHosts: []string{"test"}, allowed: false},
		{url: "http://test/image", options: Options{Signature: "NDx5zZHx7QfE8E-ijowRreq6CJJBZjwiRfOVk_mkfQQ="}, denyHosts: []string{"test"}, keys: key, allowed: false},
		{url: "http://127.0.0.1/image", denyHosts: []string{"127.0.0.0/8"}, allowed: false},
		{url: "http://127.0.0.1:3000/image", denyHosts: []string{"127.0.0.0/8"}, allowed: false},

		// valid until options
		{url: "http://test/image", now: now, options: Options{ValidUntil: now.Add(time.Second)}, allowed: true},
		{url: "http://test/image", now: now, options: Options{ValidUntil: now.Add(-time.Second)}, allowed: false},
		{url: "http://test/image", now: now, options: Options{ValidUntil: now}, allowed: false},
	}

	for _, tt := range tests {
		p := NewProxy(nil, nil)
		p.AllowHosts = tt.allowHosts
		p.DenyHosts = tt.denyHosts
		p.SignatureKeys = tt.keys
		p.Referrers = tt.referrers
		p.timeNow = tt.now

		u, err := url.Parse(tt.url)
		if err != nil {
			t.Errorf("error parsing url %q: %v", tt.url, err)
		}
		req := &Request{u, tt.options, tt.request}
		if got, want := p.allowed(req), tt.allowed; (got == nil) != want {
			t.Errorf("allowed(%q) returned %v, want %v.\nTest struct: %#v", req, got, want, tt)
		}
	}
}

func TestHostMatches(t *testing.T) {
	hosts := []string{"a.test", "*.b.test", "*c.test"}

	tests := []struct {
		url   string
		valid bool
	}{
		{"http://a.test/image", true},
		{"http://x.a.test/image", false},

		{"http://b.test/image", true},
		{"http://x.b.test/image", true},
		{"http://x.y.b.test/image", true},

		{"http://c.test/image", false},
		{"http://xc.test/image", false},
		{"/image", false},
	}

	for _, tt := range tests {
		u, err := url.Parse(tt.url)
		if err != nil {
			t.Errorf("error parsing url %q: %v", tt.url, err)
		}
		if got, want := hostMatches(hosts, u), tt.valid; got != want {
			t.Errorf("hostMatches(%v, %q) returned %v, want %v", hosts, u, got, want)
		}
	}
}

func TestReferrerMatches(t *testing.T) {
	hosts := []string{"a.test"}

	tests := []struct {
		referrer string
		valid    bool
	}{
		{"", false},
		{"%", false},
		{"http://a.test/", true},
		{"http://b.test/", false},
	}

	for _, tt := range tests {
		r, _ := http.NewRequest("GET", "/", nil)
		r.Header.Set("Referer", tt.referrer)
		if got, want := referrerMatches(hosts, r), tt.valid; got != want {
			t.Errorf("referrerMatches(%v, %v) returned %v, want %v", hosts, r, got, want)
		}
	}
}

func TestValidSignature(t *testing.T) {
	key := []byte("c0ffee")

	tests := []struct {
		url     string
		options Options
		valid   bool
	}{
		{"http://test/image", Options{Signature: "NDx5zZHx7QfE8E-ijowRreq6CJJBZjwiRfOVk_mkfQQ="}, true},
		{"http://test/image", Options{Signature: "NDx5zZHx7QfE8E-ijowRreq6CJJBZjwiRfOVk_mkfQQ"}, true},
		{"http://test/image", emptyOptions, false},
		// url-only signature with options
		{"http://test/image", Options{Signature: "NDx5zZHx7QfE8E-ijowRreq6CJJBZjwiRfOVk_mkfQQ", Rotate: 90}, true},
		// signature calculated from url plus options
		{"http://test/image", Options{Signature: "ZGTzEm32o4iZ7qcChls3EVYaWyrDd9u0etySo0-WkF8=", Rotate: 90}, true},
		// invalid base64 encoded signature
		{"http://test/image", Options{Signature: "!!"}, false},
	}

	for _, tt := range tests {
		u, err := url.Parse(tt.url)
		if err != nil {
			t.Errorf("error parsing url %q: %v", tt.url, err)
		}
		req := &Request{u, tt.options, &http.Request{}}
		if got, want := validSignature(key, req), tt.valid; got != want {
			t.Errorf("validSignature(%v, %v) returned %v, want %v", key, req, got, want)
		}
	}
}

func TestShould304(t *testing.T) {
	tests := []struct {
		req, resp string
		is304     bool
	}{
		{ // etag match
			"GET / HTTP/1.1\nIf-None-Match: \"v\"\n\n",
			"HTTP/1.1 200 OK\nEtag: \"v\"\n\n",
			true,
		},
		{ // last-modified before
			"GET / HTTP/1.1\nIf-Modified-Since: Sun, 02 Jan 2000 00:00:00 GMT\n\n",
			"HTTP/1.1 200 OK\nLast-Modified: Sat, 01 Jan 2000 00:00:00 GMT\n\n",
			true,
		},
		{ // last-modified match
			"GET / HTTP/1.1\nIf-Modified-Since: Sat, 01 Jan 2000 00:00:00 GMT\n\n",
			"HTTP/1.1 200 OK\nLast-Modified: Sat, 01 Jan 2000 00:00:00 GMT\n\n",
			true,
		},

		// mismatches
		{
			"GET / HTTP/1.1\n\n",
			"HTTP/1.1 200 OK\n\n",
			false,
		},
		{
			"GET / HTTP/1.1\n\n",
			"HTTP/1.1 200 OK\nEtag: \"v\"\n\n",
			false,
		},
		{
			"GET / HTTP/1.1\nIf-None-Match: \"v\"\n\n",
			"HTTP/1.1 200 OK\n\n",
			false,
		},
		{
			"GET / HTTP/1.1\nIf-None-Match: \"a\"\n\n",
			"HTTP/1.1 200 OK\nEtag: \"b\"\n\n",
			false,
		},
		{ // last-modified match
			"GET / HTTP/1.1\n\n",
			"HTTP/1.1 200 OK\nLast-Modified: Sat, 01 Jan 2000 00:00:00 GMT\n\n",
			false,
		},
		{ // last-modified match
			"GET / HTTP/1.1\nIf-Modified-Since: Sun, 02 Jan 2000 00:00:00 GMT\n\n",
			"HTTP/1.1 200 OK\n\n",
			false,
		},
		{ // last-modified match
			"GET / HTTP/1.1\nIf-Modified-Since: Fri, 31 Dec 1999 00:00:00 GMT\n\n",
			"HTTP/1.1 200 OK\nLast-Modified: Sat, 01 Jan 2000 00:00:00 GMT\n\n",
			false,
		},
	}

	for _, tt := range tests {
		buf := bufio.NewReader(strings.NewReader(tt.req))
		req, err := http.ReadRequest(buf)
		if err != nil {
			t.Errorf("http.ReadRequest(%q) returned error: %v", tt.req, err)
		}

		buf = bufio.NewReader(strings.NewReader(tt.resp))
		resp, err := http.ReadResponse(buf, req)
		if err != nil {
			t.Errorf("http.ReadResponse(%q) returned error: %v", tt.resp, err)
		}

		if got, want := should304(req, resp), tt.is304; got != want {
			t.Errorf("should304(%q, %q) returned: %v, want %v", tt.req, tt.resp, got, want)
		}
	}
}

// testTransport is an http.RoundTripper that returns certained canned
// responses for particular requests.
type testTransport struct {
	replyNotModified bool
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var raw string

	switch req.URL.Path {
	case "/plain":
		raw = "HTTP/1.1 200 OK\n\n"
	case "/error":
		return nil, errors.New("http protocol error")
	case "/nocontent":
		raw = "HTTP/1.1 204 No Content\n\n"
	case "/etag":
		raw = "HTTP/1.1 200 OK\nEtag: \"tag\"\n\n"
	case "/png":
		m := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		img := new(bytes.Buffer)
		_ = png.Encode(img, m)

		raw = fmt.Sprintf("HTTP/1.1 200 OK\nContent-Length: %d\nContent-Type: image/png\n\n%s", len(img.Bytes()), img.Bytes())
	case "/redirect-to-notmodified":
		parts := []string{
			"HTTP/1.1 303\nLocation: http://notmodified.test/notmodified?X-Security-Token=",
			uuid.NewString(),
			"#_=_\nCache-Control: no-store\n\n",
		}
		raw = strings.Join(parts, "")
	case "/notmodified":
		if t.replyNotModified {
			raw = "HTTP/1.1 304 Not modified\nEtag: \"abcdef\"\n\n"
		} else {
			raw = "HTTP/1.1 200 OK\nEtag: \"abcdef\"\n\nOriginal response\n"
		}
	default:
		redirectRegexp := regexp.MustCompile(`/redirects-(\d+)`)
		if redirectRegexp.MatchString(req.URL.Path) {
			redirectsLeft, _ := strconv.ParseUint(redirectRegexp.FindStringSubmatch(req.URL.Path)[1], 10, 8)
			if redirectsLeft == 0 {
				raw = "HTTP/1.1 200 OK\n\n"
			} else {
				raw = fmt.Sprintf("HTTP/1.1 302\nLocation: /http://redirect.test/redirects-%d\n\n", redirectsLeft-1)
			}
		} else {
			raw = "HTTP/1.1 404 Not Found\n\n"
		}
	}

	buf := bufio.NewReader(bytes.NewBufferString(raw))
	return http.ReadResponse(buf, req)
}

func TestProxy_UpdateCacheHeaders(t *testing.T) {
	date := "Mon, 02 Jan 2006 15:04:05 MST"
	exp := "Mon, 02 Jan 2006 16:04:05 MST"

	tests := []struct {
		name        string
		minDuration time.Duration
		forceCache  bool
		headers     http.Header
		want        http.Header
	}{
		{
			name:    "zero",
			headers: http.Header{},
			want:    http.Header{},
		},
		{
			name: "no min duration",
			headers: http.Header{
				"Date":          {date},
				"Expires":       {exp},
				"Cache-Control": {"max-age=600"},
			},
			want: http.Header{
				"Date":          {date},
				"Expires":       {exp},
				"Cache-Control": {"max-age=600"},
			},
		},
		{
			name:        "min duration, no header",
			minDuration: 30 * time.Second,
			headers:     http.Header{},
			want: http.Header{
				"Cache-Control": {"max-age=30"},
			},
		},
		{
			name:        "cache control exceeds min duration",
			minDuration: 30 * time.Second,
			headers: http.Header{
				"Cache-Control": {"max-age=600"},
			},
			want: http.Header{
				"Cache-Control": {"max-age=600"},
			},
		},
		{
			name:        "cache control exceeds min duration, expires",
			minDuration: 30 * time.Second,
			headers: http.Header{
				"Date":          {date},
				"Expires":       {exp},
				"Cache-Control": {"max-age=86400"},
			},
			want: http.Header{
				"Date":          {date},
				"Cache-Control": {"max-age=86400"},
			},
		},
		{
			name:        "min duration exceeds cache control",
			minDuration: 1 * time.Hour,
			headers: http.Header{
				"Cache-Control": {"max-age=600"},
			},
			want: http.Header{
				"Cache-Control": {"max-age=3600"},
			},
		},
		{
			name:        "min duration exceeds cache control, expires",
			minDuration: 2 * time.Hour,
			headers: http.Header{
				"Date":          {date},
				"Expires":       {exp},
				"Cache-Control": {"max-age=600"},
			},
			want: http.Header{
				"Date":          {date},
				"Cache-Control": {"max-age=7200"},
			},
		},
		{
			name:        "expires exceeds min duration, cache control",
			minDuration: 30 * time.Minute,
			headers: http.Header{
				"Date":          {date},
				"Expires":       {exp},
				"Cache-Control": {"max-age=600"},
			},
			want: http.Header{
				"Date":          {date},
				"Cache-Control": {"max-age=3600"},
			},
		},
		{
			name: "respect no-store",
			headers: http.Header{
				"Cache-Control": {"max-age=600, no-store"},
			},
			want: http.Header{
				"Cache-Control": {"max-age=600, no-store"},
			},
		},
		{
			name: "respect private",
			headers: http.Header{
				"Cache-Control": {"max-age=600, private"},
			},
			want: http.Header{
				"Cache-Control": {"max-age=600, no-store, private"},
			},
		},
		{
			name:       "force cache, normalize directives",
			forceCache: true,
			headers: http.Header{
				"Cache-Control": {"MAX-AGE=600, no-store, private"},
			},
			want: http.Header{
				"Cache-Control": {"max-age=600"},
			},
		},
		{
			name:        "force cache with min duration",
			minDuration: 1 * time.Hour,
			forceCache:  true,
			headers: http.Header{
				"Cache-Control": {"max-age=600, private, no-store"},
			},
			want: http.Header{
				"Cache-Control": {"max-age=3600"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Proxy{
				MinimumCacheDuration: tt.minDuration,
				ForceCache:           tt.forceCache,
			}
			hdr := maps.Clone(tt.headers)
			p.updateCacheHeaders(hdr)

			if !reflect.DeepEqual(hdr, tt.want) {
				t.Errorf("updateCacheHeaders(%v) returned %v, want %v", tt.headers, hdr, tt.want)
			}
		})
	}
}

func TestProxy_ServeHTTP(t *testing.T) {
	p := &Proxy{
		Client: &http.Client{
			Transport: &testTransport{},
		},
		AllowHosts:   []string{"good.test"},
		ContentTypes: []string{"image/*"},
	}

	tests := []struct {
		url  string // request URL
		code int    // expected response status code
	}{
		{"/favicon.ico", http.StatusOK},
		{"//foo", http.StatusBadRequest},                            // invalid request URL
		{"/http://bad.test/", http.StatusForbidden},                 // Disallowed host
		{"/http://good.test/error", http.StatusInternalServerError}, // HTTP protocol error
		{"/http://good.test/nocontent", http.StatusNoContent},       // non-OK response
		{"/100/http://good.test/png", http.StatusOK},
		{"/100/http://good.test/plain", http.StatusForbidden}, // non-image response

		// health-check URLs
		{"/", http.StatusOK},
		{"/health-check", http.StatusOK},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", "http://localhost"+tt.url, nil)
		resp := httptest.NewRecorder()
		p.ServeHTTP(resp, req)

		if got, want := resp.Code, tt.code; got != want {
			t.Errorf("ServeHTTP(%v) returned status %d, want %d", req, got, want)
		}
	}
}

// test that 304 Not Modified responses are returned properly.
func TestProxy_ServeHTTP_is304(t *testing.T) {
	p := &Proxy{
		Client: &http.Client{
			Transport: &testTransport{},
		},
	}

	req, _ := http.NewRequest("GET", "http://localhost/http://good.test/etag", nil)
	req.Header.Add("If-None-Match", `"tag"`)
	resp := httptest.NewRecorder()
	p.ServeHTTP(resp, req)

	if got, want := resp.Code, http.StatusNotModified; got != want {
		t.Errorf("ServeHTTP(%v) returned status %d, want %d", req, got, want)
	}
	if got, want := resp.Header().Get("Etag"), `"tag"`; got != want {
		t.Errorf("ServeHTTP(%v) returned etag header %v, want %v", req, got, want)
	}
}

func TestProxy_ServeHTTP_cached304(t *testing.T) {
	cache := lrucache.New(1024*1024*8, 0)
	client := new(http.Client)
	tt := testTransport{}
	client.Transport = &httpcache.Transport{
		Transport: &TransformingTransport{
			Transport:     &tt,
			CachingClient: client,
		},
		Cache:               cache,
		MarkCachedResponses: true,
	}

	p := &Proxy{
		Client:          client,
		FollowRedirects: true,
	}

	// prime the cache
	req := httptest.NewRequest("GET", "http://localhost//http://good.test/redirect-to-notmodified", nil)
	recorder := httptest.NewRecorder()
	p.ServeHTTP(recorder, req)

	resp := recorder.Result()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("ServeHTTP(%v) returned status %d, want %d", req, got, want)
	}
	if _, found := cache.Get("http://good.test/redirect-to-notmodified#0x0"); !found {
		t.Errorf("Response to http://good.test/redirect-to-notmodified#0x0 should be cached")
	}

	// now make the same request again, but this time make sure the server responds with a 304
	tt.replyNotModified = true
	req = httptest.NewRequest("GET", "http://localhost//http://good.test/redirect-to-notmodified", nil)
	recorder = httptest.NewRecorder()
	p.ServeHTTP(recorder, req)

	resp = recorder.Result()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("ServeHTTP(%v) returned status %d, want %d", req, got, want)
	}

	if recorder.Body.String() != "Original response\n" {
		t.Errorf("Response isn't what we expected: %v", recorder.Body.String())
	}
}

func TestProxy_ServeHTTP_maxRedirects(t *testing.T) {
	p := &Proxy{
		Client: &http.Client{
			Transport: &testTransport{},
		},
		FollowRedirects: true,
	}

	tests := []struct {
		url  string
		code int
	}{
		{"/http://redirect.test/redirects-0", http.StatusOK},
		{"/http://redirect.test/redirects-2", http.StatusOK},
		{"/http://redirect.test/redirects-11", http.StatusInternalServerError}, // too many redirects
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", "http://localhost"+tt.url, nil)
		resp := httptest.NewRecorder()
		p.ServeHTTP(resp, req)

		if got, want := resp.Code, tt.code; got != want {
			t.Errorf("ServeHTTP(%v) returned status %d, want %d", req, got, want)
		}
	}
}

func TestProxy_log(t *testing.T) {
	var b strings.Builder

	p := &Proxy{
		Logger: log.New(&b, "", 0),
	}
	p.log("Test")

	if got, want := b.String(), "Test\n"; got != want {
		t.Errorf("log wrote %s, want %s", got, want)
	}

	b.Reset()
	p.logf("Test %v", 123)

	if got, want := b.String(), "Test 123\n"; got != want {
		t.Errorf("logf wrote %s, want %s", got, want)
	}
}

func TestProxy_log_default(t *testing.T) {
	var b strings.Builder

	defer func(flags int) {
		log.SetOutput(os.Stderr)
		log.SetFlags(flags)
	}(log.Flags())

	log.SetOutput(&b)
	log.SetFlags(0)

	p := &Proxy{}
	p.log("Test")

	if got, want := b.String(), "Test\n"; got != want {
		t.Errorf("log wrote %s, want %s", got, want)
	}

	b.Reset()
	p.logf("Test %v", 123)

	if got, want := b.String(), "Test 123\n"; got != want {
		t.Errorf("logf wrote %s, want %s", got, want)
	}
}

func TestTransformingTransport(t *testing.T) {
	client := new(http.Client)
	tr := &TransformingTransport{
		Transport:     &testTransport{},
		CachingClient: client,
		limiter:       make(chan struct{}, 1),
	}
	client.Transport = tr

	tests := []struct {
		url         string
		code        int
		expectError bool
	}{
		{"http://good.test/png#1", http.StatusOK, false},
		{"http://good.test/error#1", http.StatusInternalServerError, true},
		// TODO: test more than just status code... verify that image
		// is actually transformed and returned properly and that
		// non-image responses are returned as-is
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", tt.url, nil)

		resp, err := tr.RoundTrip(req)
		if err != nil {
			if !tt.expectError {
				t.Errorf("RoundTrip(%v) returned unexpected error: %v", tt.url, err)
			}
			continue
		} else if tt.expectError {
			t.Errorf("RoundTrip(%v) did not return expected error", tt.url)
		}
		if got, want := resp.StatusCode, tt.code; got != want {
			t.Errorf("RoundTrip(%v) returned status code %d, want %d", tt.url, got, want)
		}
	}
}

func TestContentTypeMatches(t *testing.T) {
	tests := []struct {
		patterns    []string
		contentType string
		valid       bool
	}{
		// no patterns
		{nil, "", true},
		{nil, "text/plain", true},
		{[]string{}, "", true},
		{[]string{}, "text/plain", true},

		// empty pattern
		{[]string{""}, "", true},
		{[]string{""}, "text/plain", false},

		// exact match
		{[]string{"text/plain"}, "", false},
		{[]string{"text/plain"}, "text", false},
		{[]string{"text/plain"}, "text/html", false},
		{[]string{"text/plain"}, "text/plain", true},
		{[]string{"text/plain"}, "text/plaintext", false},
		{[]string{"text/plain"}, "text/plain+foo", false},

		// wildcard match
		{[]string{"text/*"}, "", false},
		{[]string{"text/*"}, "text", false},
		{[]string{"text/*"}, "text/html", true},
		{[]string{"text/*"}, "text/plain", true},
		{[]string{"text/*"}, "image/jpeg", false},

		{[]string{"image/svg*"}, "image/svg", true},
		{[]string{"image/svg*"}, "image/svg+html", true},

		// complete wildcard does not match
		{[]string{"*"}, "text/foobar", false},

		// multiple patterns
		{[]string{"text/*", "image/*"}, "image/jpeg", true},
	}
	for _, tt := range tests {
		got := contentTypeMatches(tt.patterns, tt.contentType)
		if want := tt.valid; got != want {
			t.Errorf("contentTypeMatches(%q, %q) returned %v, want %v", tt.patterns, tt.contentType, got, want)
		}
	}
}
