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

// Package proxy provides the image proxy.
package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/gregjones/httpcache"
)

// Proxy serves image requests.
type Proxy struct {
	Client *http.Client // client used to fetch remote URLs
	Cache  Cache

	// Whitelist specifies a list of remote hosts that images can be proxied from.  An empty list means all hosts are allowed.
	Whitelist []string

	MaxWidth  int
	MaxHeight int
}

// NewProxy constructs a new proxy.  The provided http Client will be used to
// fetch remote URLs.  If nil is provided, http.DefaultClient will be used.
func NewProxy(transport http.RoundTripper, cache Cache) *Proxy {
	if transport == nil {
		transport = http.DefaultTransport
	}
	if cache == nil {
		cache = NopCache
	}

	client := new(http.Client)
	client.Transport = &httpcache.Transport{
		Transport:           &TransformingTransport{transport, client},
		Cache:               cache,
		MarkCachedResponses: true,
	}

	return &Proxy{
		Client: client,
		Cache:  cache,
	}
}

// ServeHTTP handles image requests.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, err := NewRequest(r)
	if err != nil {
		glog.Errorf("invalid request URL: %v", err)
		http.Error(w, fmt.Sprintf("invalid request URL: %v", err), http.StatusBadRequest)
		return
	}

	if p.MaxWidth > 0 && int(req.Options.Width) > p.MaxWidth {
		req.Options.Width = float64(p.MaxWidth)
	}
	if p.MaxHeight > 0 && int(req.Options.Height) > p.MaxHeight {
		req.Options.Height = float64(p.MaxHeight)
	}

	if !p.allowed(req.URL) {
		glog.Errorf("remote URL is not for an allowed host: %v", req.URL.Host)
		http.Error(w, fmt.Sprintf("remote URL is not for an allowed host: %v", req.URL.Host), http.StatusBadRequest)
		return
	}

	u := req.URL.String()
	if req.Options != nil && !reflect.DeepEqual(req.Options, emptyOptions) {
		u += "#" + req.Options.String()
	}
	resp, err := p.Client.Get(u)
	if err != nil {
		glog.Errorf("error fetching remote image: %v", err)
		http.Error(w, fmt.Sprintf("Error fetching remote image: %v", err), http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Remote URL %q returned status: %v", req.URL, resp.Status), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Last-Modified", resp.Header.Get("Last-Modified"))
	w.Header().Add("Expires", resp.Header.Get("Expires"))
	w.Header().Add("Etag", resp.Header.Get("Etag"))

	if is304 := check304(w, r, resp); is304 {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Add("Content-Length", resp.Header.Get("Content-Length"))
	defer resp.Body.Close()
	io.Copy(w, resp.Body)
}

// allowed returns whether the specified URL is on the whitelist of remote hosts.
func (p *Proxy) allowed(u *url.URL) bool {
	if len(p.Whitelist) == 0 {
		return true
	}

	for _, host := range p.Whitelist {
		if u.Host == host {
			return true
		}
	}

	return false
}

func check304(w http.ResponseWriter, req *http.Request, resp *http.Response) bool {
	etag := resp.Header.Get("Etag")
	if etag != "" && etag == req.Header.Get("If-None-Match") {
		return true
	}

	lastModified, err := time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))
	if err != nil {
		return false
	}
	ifModSince, err := time.Parse(time.RFC1123, req.Header.Get("If-Modified-Since"))
	if err != nil {
		return false
	}
	if lastModified.Before(ifModSince) {
		return true
	}

	return false
}

// TransformingTransport is an implementation of http.RoundTripper that
// optionally transforms images using the options specified in the request URL
// fragment.
type TransformingTransport struct {
	// Transport is used to satisfy non-transform requests (those that do not include a URL fragment)
	Transport http.RoundTripper

	// Client is used to fetch images to be resized.
	Client *http.Client
}

// RoundTrip implements http.RoundTripper.
func (t *TransformingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Fragment == "" {
		// normal requests pass through
		glog.Infof("fetching remote URL: %v", req.URL)
		return t.Transport.RoundTrip(req)
	}

	u := *req.URL
	u.Fragment = ""
	resp, err := t.Client.Get(u.String())

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	opt := ParseOptions(req.URL.Fragment)
	img, err := Transform(b, opt)
	if err != nil {
		img = b
	}

	// replay response with transformed image and updated content length
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s %s\n", resp.Proto, resp.Status)
	resp.Header.WriteSubset(buf, map[string]bool{"Content-Length": true})
	fmt.Fprintf(buf, "Content-Length: %d\n\n", len(img))
	buf.Write(img)

	return http.ReadResponse(bufio.NewReader(buf), req)
}
