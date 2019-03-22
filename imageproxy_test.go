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

package imageproxy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

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
			dst:  http.Header{},
			src:  http.Header{"A": []string{"a"}},
			keys: nil,
			want: http.Header{"A": []string{"a"}},
		},
		{
			dst:  http.Header{"A": []string{"a"}},
			src:  http.Header{"B": []string{"b"}},
			keys: nil,
			want: http.Header{"A": []string{"a"}, "B": []string{"b"}},
		},
		{
			dst:  http.Header{"A": []string{"a"}},
			src:  http.Header{"B": []string{"b"}, "C": []string{"c"}},
			keys: []string{"B"},
			want: http.Header{"A": []string{"a"}, "B": []string{"b"}},
		},
		{
			dst:  http.Header{"A": []string{"a1"}},
			src:  http.Header{"A": []string{"a2"}},
			keys: nil,
			want: http.Header{"A": []string{"a1", "a2"}},
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
	allowHosts := []string{"good"}
	key := []byte("c0ffee")

	genRequest := func(headers map[string]string) *http.Request {
		req := &http.Request{Header: make(http.Header)}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		return req
	}

	tests := []struct {
		url        string
		options    Options
		allowHosts []string
		denyHosts  []string
		referrers  []string
		key        []byte
		request    *http.Request
		allowed    bool
	}{
		// no allowHosts or signature key
		{"http://test/image", emptyOptions, nil, nil, nil, nil, nil, true},

		// allowHosts
		{"http://good/image", emptyOptions, allowHosts, nil, nil, nil, nil, true},
		{"http://bad/image", emptyOptions, allowHosts, nil, nil, nil, nil, false},

		// referrer
		{"http://test/image", emptyOptions, nil, nil, allowHosts, nil, genRequest(map[string]string{"Referer": "http://good/foo"}), true},
		{"http://test/image", emptyOptions, nil, nil, allowHosts, nil, genRequest(map[string]string{"Referer": "http://bad/foo"}), false},
		{"http://test/image", emptyOptions, nil, nil, allowHosts, nil, genRequest(map[string]string{"Referer": "MALFORMED!!"}), false},
		{"http://test/image", emptyOptions, nil, nil, allowHosts, nil, genRequest(map[string]string{}), false},

		// signature key
		{"http://test/image", Options{Signature: "NDx5zZHx7QfE8E-ijowRreq6CJJBZjwiRfOVk_mkfQQ="}, nil, nil, nil, key, nil, true},
		{"http://test/image", Options{Signature: "deadbeef"}, nil, nil, nil, key, nil, false},
		{"http://test/image", emptyOptions, nil, nil, nil, key, nil, false},

		// allowHosts and signature
		{"http://good/image", emptyOptions, allowHosts, nil, nil, key, nil, true},
		{"http://bad/image", Options{Signature: "gWivrPhXBbsYEwpmWAKjbJEiAEgZwbXbltg95O2tgNI="}, nil, nil, nil, key, nil, true},
		{"http://bad/image", emptyOptions, allowHosts, nil, nil, key, nil, false},

		// deny requests that match denyHosts, even if signature is valid or also matches allowHosts
		{"http://test/image", emptyOptions, nil, []string{"test"}, nil, nil, nil, false},
		{"http://test/image", emptyOptions, []string{"test"}, []string{"test"}, nil, nil, nil, false},
		{"http://test/image", Options{Signature: "NDx5zZHx7QfE8E-ijowRreq6CJJBZjwiRfOVk_mkfQQ="}, nil, []string{"test"}, nil, key, nil, false},
	}

	for _, tt := range tests {
		p := NewProxy(nil, nil)
		p.AllowHosts = tt.allowHosts
		p.DenyHosts = tt.denyHosts
		p.SignatureKey = tt.key
		p.Referrers = tt.referrers

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
	}

	for _, tt := range tests {
		u, err := url.Parse(tt.url)
		if err != nil {
			t.Errorf("error parsing url %q: %v", tt.url, err)
		}
		req := &Request{u, tt.options, &http.Request{}}
		if got, want := validSignature(key, req), tt.valid; got != want {
			t.Errorf("validSignature(%v, %q) returned %v, want %v", key, u, got, want)
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
type testTransport struct{}

func (t testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
		png.Encode(img, m)

		raw = fmt.Sprintf("HTTP/1.1 200 OK\nContent-Length: %d\nContent-Type: image/png\n\n%s", len(img.Bytes()), img.Bytes())
	default:
		raw = "HTTP/1.1 404 Not Found\n\n"
	}

	buf := bufio.NewReader(bytes.NewBufferString(raw))
	return http.ReadResponse(buf, req)
}

func TestProxy_ServeHTTP(t *testing.T) {
	p := &Proxy{
		Client: &http.Client{
			Transport: testTransport{},
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
			Transport: testTransport{},
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

func TestTransformingTransport(t *testing.T) {
	client := new(http.Client)
	tr := &TransformingTransport{
		Transport:     testTransport{},
		CachingClient: client,
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
