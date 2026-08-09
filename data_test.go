// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"net/http"
	"net/url"
	"testing"
)

var emptyOptions = Options{}

// Test that request URLs are properly parsed into Options and RemoteURL.  This
// test verifies that invalid remote URLs throw errors, and that valid
// combinations of Options and URL are accept.  This does not exhaustively test
// the various Options that can be specified; see TestParseOptions for that.
func TestNewRequest(t *testing.T) {
	tests := []struct {
		URL         string  // input URL to parse as an imageproxy request
		RemoteURL   string  // expected URL of remote image parsed from input
		Options     Options // expected options parsed from input
		ExpectError bool    // whether an error is expected from NewRequest
	}{
		// invalid URLs
		{"http://localhost/", "", emptyOptions, true},
		{"http://localhost/1/", "", emptyOptions, true},
		{"http://localhost//example.com/foo", "", emptyOptions, true},
		{"http://localhost//ftp://example.com/foo", "", emptyOptions, true},

		// invalid options.  These won't return errors, but will not fully parse the options
		{
			"http://localhost/s/http://example.com/",
			"http://example.com/", emptyOptions, false,
		},
		{
			"http://localhost/1xs/http://example.com/",
			"http://example.com/", Options{Width: 1}, false,
		},

		// valid URLs
		{
			"http://localhost/http://example.com/foo",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost//http://example.com/foo",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost//https://example.com/foo",
			"https://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost/1x2/http://example.com/foo",
			"http://example.com/foo", Options{Width: 1, Height: 2}, false,
		},
		{
			"http://localhost//http://example.com/foo?bar",
			"http://example.com/foo?bar", emptyOptions, false,
		},
		{
			"http://localhost/http:/example.com/foo",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost/http:///example.com/foo",
			"http://example.com/foo", emptyOptions, false,
		},
		// base64 encoded paths
		{
			"http://localhost/aHR0cDovL2V4YW1wbGUuY29tL2Zvbw",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost//aHR0cDovL2V4YW1wbGUuY29tL2Zvbw",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost/x/aHR0cDovL2V4YW1wbGUuY29tL2Zvbw",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost/x/aHR0cHM6Ly9leGFtcGxlLmNvbS9mb28_YmFy",
			"https://example.com/foo?bar", emptyOptions, false,
		},
		{
			"http://localhost/x/aHR0cHM6Ly9leGFtcGxlLmNvbS9mb28_YmFy?baz",
			"https://example.com/foo?bar", emptyOptions, false,
		},
		{ // escaped path
			"http://localhost/http://example.com/%2C",
			"http://example.com/%2C", emptyOptions, false,
		},
		// percent encoded cases
		{
			"http://localhost/1x2/http%3A%2F%2Fexample.com%2Ffoo",
			"http://example.com/foo", Options{Width: 1, Height: 2}, false,
		},
		{
			"http://localhost/1x2/http%3A%2F%2Fexample.com%2Fhttp%2Fstuff",
			"http://example.com/http/stuff", Options{Width: 1, Height: 2}, false,
		},
		{
			"http://localhost/http%3A%2F%2Fexample.com%2Ffoo",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost/HTTP%3a%2f%2fexample.com%2Ffoo",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost/http%3A%2Fexample.com%2Ffoo",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost/http%3A%2F%2F%2Fexample.com%2Ffoo",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost//http%3A%2F%2Fexample.com%2Ffoo",
			"http://example.com/foo", emptyOptions, false,
		},
		{
			"http://localhost/http%3A%2F%2Fexample.com%2Ffoo%3Ftest%3D1%26test%3D2",
			"http://example.com/foo?test=1&test=2", emptyOptions, false,
		},
		{
			"http://localhost/1x2/http%3A%2F%2Fexample.com%2Ffoo%3Ftest%3D1%26test%3D2",
			"http://example.com/foo?test=1&test=2", Options{Width: 1, Height: 2}, false,
		},
	}

	for _, tt := range tests {
		req, err := http.NewRequest("GET", tt.URL, nil)
		if err != nil {
			t.Errorf("http.NewRequest(%q) returned error: %v", tt.URL, err)
			continue
		}

		r, err := NewRequest(req, nil)
		if tt.ExpectError {
			if err == nil {
				t.Errorf("NewRequest(%v) did not return expected error", req)
			}
			continue
		} else if err != nil {
			t.Errorf("NewRequest(%v) return unexpected error: %v", req, err)
			continue
		}

		if got, want := r.URL.String(), tt.RemoteURL; got != want {
			t.Errorf("NewRequest(%q) request URL = %v, want %v", tt.URL, got, want)
		}
		if got, want := r.Options, tt.Options; got != want {
			t.Errorf("NewRequest(%q) request options = %v, want %v", tt.URL, got, want)
		}
	}
}

func TestNewRequest_BaseURL(t *testing.T) {
	base, _ := url.Parse("https://example.com/")

	tests := []struct {
		path string
		want string
	}{
		{
			path: "/x/path",
			want: "https://example.com/path#0x0",
		},
		{ // Chinese characters 已然
			path: "/x/5bey54S2",
			want: "https://example.com/%E5%B7%B2%E7%84%B6#0x0",
		},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", tt.path, nil)
		r, err := NewRequest(req, base)
		if err != nil {
			t.Errorf("NewRequest(%v, %v) returned unexpected error: %v", req, base, err)
		}

		if got := r.String(); got != tt.want {
			t.Errorf("NewRequest(%v, %v) returned %q, want %q", req, base, got, tt.want)
		}
	}
}
