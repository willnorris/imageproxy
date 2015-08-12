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
	"strings"
	"testing"

	"github.com/gregjones/httpcache"
)

func TestAllowed(t *testing.T) {
	whitelist := []string{"good"}
	key := []byte("c0ffee")

	tests := []struct {
		url       string
		options   Options
		whitelist []string
		key       []byte
		allowed   bool
	}{
		// no whitelist or signature key
		{"http://test/image", emptyOptions, nil, nil, true},

		// whitelist
		{"http://good/image", emptyOptions, whitelist, nil, true},
		{"http://bad/image", emptyOptions, whitelist, nil, false},

		// signature key
		{"http://test/image", Options{Signature: "NDx5zZHx7QfE8E-ijowRreq6CJJBZjwiRfOVk_mkfQQ="}, nil, key, true},
		{"http://test/image", Options{Signature: "deadbeef"}, nil, key, false},
		{"http://test/image", emptyOptions, nil, key, false},

		// whitelist and signature
		{"http://good/image", emptyOptions, whitelist, key, true},
		{"http://bad/image", Options{Signature: "gWivrPhXBbsYEwpmWAKjbJEiAEgZwbXbltg95O2tgNI="}, nil, key, true},
		{"http://bad/image", emptyOptions, whitelist, key, false},
	}

	for _, tt := range tests {
		p := NewProxy(nil, nil)
		p.Whitelist = tt.whitelist
		p.SignatureKey = tt.key

		u, err := url.Parse(tt.url)
		if err != nil {
			t.Errorf("error parsing url %q: %v", tt.url, err)
		}
		req := &Request{u, tt.options}
		if got, want := p.allowed(req), tt.allowed; got != want {
			t.Errorf("allowed(%q) returned %v, want %v", req, got, want)
		}
	}
}

func TestValidHost(t *testing.T) {
	whitelist := []string{"a.test", "*.b.test", "*c.test"}

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
		if got, want := validHost(whitelist, u), tt.valid; got != want {
			t.Errorf("validHost(%v, %q) returned %v, want %v", whitelist, u, got, want)
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
		req := &Request{u, tt.options}
		if got, want := validSignature(key, req), tt.valid; got != want {
			t.Errorf("validSignature(%v, %q) returned %v, want %v", key, u, got, want)
		}
	}
}

func TestCheck304(t *testing.T) {
	tests := []struct {
		req, resp string
		is304     bool
	}{
		{ // etag match
			"GET / HTTP/1.1\nIf-None-Match: \"v\"\n\n",
			"HTTP/1.1 200 OK\nEtag: \"v\"\n\n",
			true,
		},
		{ // last-modified match
			"GET / HTTP/1.1\nIf-Modified-Since: Sun, 02 Jan 2000 00:00:00 GMT\n\n",
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

		if got, want := check304(req, resp), tt.is304; got != want {
			t.Errorf("check304(%q, %q) returned: %v, want %v", tt.req, tt.resp, got, want)
		}
	}
}

// make sure that the proxy is passed to transport in order
// to access the command line flags.
func TestProxyPointer(t *testing.T) {
	p := NewProxy(nil, nil)
	if p.Client.Transport.(*httpcache.Transport).Transport.(*TransformingTransport).Proxy != p {
		t.Errorf("Transport doesnt have proxy pointer")
	}
}

// testTransport is an http.RoundTripper that returns certained canned
// responses for particular requests.
type testTransport struct{}

func (t testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var raw string

	switch req.URL.Path {
	case "/ok":
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

		raw = fmt.Sprintf("HTTP/1.1 200 OK\nContent-Length: %d\n\n%v", len(img.Bytes()), img.Bytes())
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
		Whitelist: []string{"good.test"},
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

		{"/100/http://good.test/ok", http.StatusOK},
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
