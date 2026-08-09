// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"willnorris.com/go/imageproxy/options"
)

type Options = options.Options

// URLError reports a malformed URL error.
type URLError struct {
	Message string
	URL     *url.URL
}

func (e URLError) Error() string {
	return fmt.Sprintf("malformed URL %q: %s", e.URL, e.Message)
}

// Request is an imageproxy request which includes a remote URL of an image to
// proxy, and an optional set of transformations to perform.
type Request struct {
	URL      *url.URL      // URL of the image to proxy
	Options  Options       // Image transformation to perform
	Original *http.Request // The original HTTP request
}

// String returns the request URL as a string, with r.Options encoded in the
// URL fragment.
func (r Request) String() string {
	u := *r.URL
	u.Fragment = r.Options.String()
	return u.String()
}

// NewRequest parses an http.Request into an imageproxy Request.  Options and
// the remote image URL are specified in the request path, formatted as:
// /{options}/{remote_url}.  Options may be omitted, so a request path may
// simply contain /{remote_url}.
//
// The remote URL may be included in plain text without any encoding,
// percent-encoded (aka URL encoded), or base64 encoded (URL safe, no padding).
//
// When no encoding is used, any URL query string is treated as part of the remote URL.
// For example, given the proxy URL of `http://localhost/x/http://example.com/?id=1`,
// the remote URL is `http://example.com/?id=1`.
//
// When percent-encoding is used, the full URL must be encoded.
// Any query string on the proxy URL is NOT included as part of the remote URL.
// Percent-encoded URLs must be absolute URLs;
// they cannot be relative URLs used with a default base URL.
//
// When base64 encoding is used, the full URL must be encoded.
// Any query string on the proxy URL is NOT included as part of the remote URL.
// Base64 encoded URLs may be relative URLs used with a default base URL.
//
// Assuming an imageproxy server running on localhost, the following are all
// valid imageproxy requests:
//
//	http://localhost/100x200/http://example.com/image.jpg
//	http://localhost/100x200,r90/http://example.com/image.jpg?foo=bar
//	http://localhost//http://example.com/image.jpg
//	http://localhost/http://example.com/image.jpg
//	http://localhost/x/http%3A%2F%2Fexample.com%2Fimage.jpg
//	http://localhost/100x200/aHR0cDovL2V4YW1wbGUuY29tL2ltYWdlLmpwZw
func NewRequest(r *http.Request, baseURL *url.URL) (*Request, error) {
	var err error
	req := &Request{Original: r}
	var enc bool // whether the remote URL was base64 or URL encoded

	path := r.URL.EscapedPath()[1:] // strip leading slash
	req.URL, enc, err = parseURL(path, baseURL)
	if err != nil || !req.URL.IsAbs() {
		// first segment should be options
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			return nil, URLError{"too few path segments", r.URL}
		}

		var err error
		req.URL, enc, err = parseURL(parts[1], baseURL)
		if err != nil {
			return nil, URLError{fmt.Sprintf("unable to parse remote URL: %v", err), r.URL}
		}

		req.Options = options.ParseOptions(parts[0])
	}

	if baseURL != nil {
		req.URL = baseURL.ResolveReference(req.URL)
	}

	if !req.URL.IsAbs() {
		return nil, URLError{"must provide absolute remote URL", r.URL}
	}

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return nil, URLError{"remote URL must have http or https scheme", r.URL}
	}

	if !enc {
		// if the remote URL was not base64 or URL encoded,
		// then the query string is part of the remote URL
		req.URL.RawQuery = r.URL.RawQuery
	}
	return req, nil
}

var reCleanedURL = regexp.MustCompile(`^(https?):/+([^/])`)
var reIsEncodedURL = regexp.MustCompile(`^(?i)https?%3A%2F`)

// parseURL parses s as a URL, handling URLs that have been munged by
// path.Clean or a webserver that collapses multiple slashes.
// The returned enc bool indicates whether the remote URL was encoded.
func parseURL(s string, baseURL *url.URL) (_ *url.URL, enc bool, _ error) {
	// Try to base64 decode the string. If it is not base64 encoded,
	// this will fail quickly on the first invalid character like ":", ".", or "/".
	// Accept the decoded string if it looks like an absolute HTTP URL,
	// or if we have a baseURL and the decoded string did not contain invalid code points.
	// This allows for values like "/path", which do successfully base64 decode,
	// but not to valid code points, to be treated as an unencoded string.
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		d := string(b)
		if strings.HasPrefix(d, "http://") || strings.HasPrefix(d, "https://") {
			enc = true
			s = d
		} else if baseURL != nil && !strings.ContainsRune(d, unicode.ReplacementChar) {
			enc = true
			s = d
		}
	}

	// If the string looks like a URL encoded absolute HTTP(S) URL, decode it.
	if reIsEncodedURL.MatchString(s) {
		if u, err := url.PathUnescape(s); err == nil {
			enc = true
			s = u
		}
	}

	s = reCleanedURL.ReplaceAllString(s, "$1://$2")
	u, err := url.Parse(s)
	return u, enc, err
}
