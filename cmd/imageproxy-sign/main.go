// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

// The imageproxy-sign tool creates signature values for a provided URL and
// signing key.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"willnorris.com/go/imageproxy"
)

var signingKey = flag.String("key", "@/etc/imageproxy.key", "signing key, or file containing key prefixed with '@'")
var urlOnly = flag.Bool("url", false, "only sign the URL value, do not include options")

func main() {
	flag.Parse()
	u := flag.Arg(0)

	sig, err := sign(*signingKey, u, *urlOnly)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("url: %v\n", u)
	fmt.Printf("signature: %v\n", base64.URLEncoding.EncodeToString(sig))
}

func sign(key string, s string, urlOnly bool) ([]byte, error) {
	if s == "" {
		return nil, errors.New("imageproxy-sign url [key]")
	}

	u := parseURL(s)
	if u == nil {
		return nil, fmt.Errorf("unable to parse URL: %v", s)
	}
	if urlOnly {
		u.Fragment = ""
	}

	k, err := parseKey(key)
	if err != nil {
		return nil, fmt.Errorf("error parsing key: %w", err)
	}

	mac := hmac.New(sha256.New, k)
	if _, err := mac.Write([]byte(u.String())); err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil
}

func parseKey(s string) ([]byte, error) {
	if strings.HasPrefix(s, "@") {
		return os.ReadFile(s[1:])
	}
	return []byte(s), nil
}

// parseURL parses s as either an imageproxy request URL or a remote URL with
// options in the URL fragment.  Any existing signature values are stripped,
// and the final remote URL returned with remaining options in the fragment.
func parseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if s == "" || err != nil {
		return nil
	}

	// first try to parse this as an imageproxy URL, containing
	// transformation options and the remote URL embedded
	if r, err := imageproxy.NewRequest(&http.Request{URL: u}, nil); err == nil {
		r.Options.Signature = ""
		r.URL.Fragment = r.Options.String()
		return r.URL
	}

	// second, we assume that this is the remote URL itself. If a fragment
	// is present, treat it as an option string.
	opt := imageproxy.ParseOptions(u.Fragment)
	opt.Signature = ""
	u.Fragment = opt.String()
	return u
}
