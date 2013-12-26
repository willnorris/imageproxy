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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/gregjones/httpcache"
)

// URLError reports a malformed URL error.
type URLError struct {
	Message string
	URL     *url.URL
}

func (e URLError) Error() string {
	return fmt.Sprintf("malformed URL %q: %s", e.URL, e.Message)
}

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
func NewProxy(client *http.Client, cache Cache) *Proxy {
	if client == nil {
		client = http.DefaultClient
	}
	if cache == nil {
		cache = NopCache
	}

	return &Proxy{
		Client: &http.Client{
			Transport: &httpcache.Transport{
				Transport:           client.Transport,
				Cache:               cache,
				MarkCachedResponses: true,
			},
		},
		Cache: cache,
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

	u := req.URL.String()
	glog.Infof("request for image: %v", u)

	if !p.allowed(req.URL) {
		glog.Errorf("remote URL is not for an allowed host: %v", req.URL.Host)
		http.Error(w, fmt.Sprintf("remote URL is not for an allowed host: %v", req.URL.Host), http.StatusBadRequest)
		return
	}

	image, err := p.fetchRemoteImage(u)
	if err != nil {
		glog.Errorf("error fetching remote image: %v", err)
		http.Error(w, fmt.Sprintf("Error fetching remote image: %v", err), http.StatusInternalServerError)
		return
	}

	b, _ := Transform(image.Bytes, req.Options)
	image.Bytes = b

	w.Header().Add("Content-Length", strconv.Itoa(len(image.Bytes)))
	w.Header().Add("Expires", image.Expires.Format(time.RFC1123))
	w.Write(image.Bytes)
}

func (p *Proxy) fetchRemoteImage(u string) (*Image, error) {
	resp, err := p.Client.Get(u)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("HTTP status not OK: %v", resp.Status))
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &Image{
		URL:     u,
		Expires: parseExpires(resp),
		Etag:    resp.Header.Get("Etag"),
		Bytes:   b,
	}, nil
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

func parseExpires(resp *http.Response) time.Time {
	exp := resp.Header.Get("Expires")
	if exp == "" {
		return time.Now()
	}

	t, err := time.Parse(time.RFC1123, exp)
	if err != nil {
		return time.Now()
	}

	return t
}
