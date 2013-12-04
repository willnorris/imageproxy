// Package proxy provides the image proxy.
package proxy

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/willnorris/go-imageproxy/cache"
	"github.com/willnorris/go-imageproxy/data"
)

// URLError reports a malformed URL error.
type URLError struct {
	Message string
	URL     *url.URL
}

func (e URLError) Error() string {
	return fmt.Sprintf("malformed URL %q: %s", e.URL, e.Message)
}

// NewRequest parses an http.Request into an image request.
func NewRequest(r *http.Request) (*data.Request, error) {
	path := strings.SplitN(r.URL.Path, "/", 3)
	if len(path) != 3 {
		return nil, URLError{"too few path segments", r.URL}
	}

	var err error
	req := new(data.Request)

	req.URL, err = url.Parse(path[2])
	if err != nil {
		return nil, URLError{
			fmt.Sprintf("unable to parse remote URL: %v", err),
			r.URL,
		}
	}

	if !req.URL.IsAbs() {
		return nil, URLError{"must provide absolute remote URL", r.URL}
	}

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return nil, URLError{"remote URL must have http or https URL", r.URL}
	}

	// query string is always part of the remote URL
	req.URL.RawQuery = r.URL.RawQuery
	req.Transform, err = data.ParseTransform(path[1])
	if err != nil {
		return nil, URLError{err.Error(), r.URL}
	}

	return req, nil
}

// Proxy serves image requests.
type Proxy struct {
	Client *http.Client // client used to fetch remote URLs
	Cache  cache.Cache

	// Whitelist specifies a list of remote hosts that images can be proxied from.  An empty list means all hosts are allowed.
	Whitelist []string
}

// NewProxy constructs a new proxy.  The provided http Client will be used to
// fetch remote URLs.  If nil is provided, http.DefaultClient will be used.
func NewProxy(client *http.Client) *Proxy {
	if client == nil {
		client = http.DefaultClient
	}
	return &Proxy{Client: client, Cache: cache.NopCache}
}

// ServeHTTP handles image requests.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, err := NewRequest(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request URL: %v", err.Error()), http.StatusBadRequest)
		return
	}

	u := req.URL.String()
	glog.Infof("request for image: %v", u)

	if !p.allowed(req.URL) {
		http.Error(w, fmt.Sprintf("remote URL is not for an allowed host: %v", req.URL.Host), http.StatusForbidden)
		return
	}

	image, ok := p.Cache.Get(u)
	if !ok {
		glog.Infof("image not cached")
		image, err = p.fetchRemoteImage(u, nil)
		if err != nil {
			glog.Errorf("errorf fetching remote image: %v", err)
		}
		p.Cache.Save(image)
	} else if time.Now().After(image.Expires) {
		glog.Infof("cached image expired")
		image, err = p.fetchRemoteImage(u, image)
		if err != nil {
			glog.Errorf("errorf fetching remote image: %v", err)
		}
		p.Cache.Save(image)
	} else {
		glog.Infof("serving from cache")
	}

	w.Header().Add("Content-Length", strconv.Itoa(len(image.Bytes)))
	w.Header().Add("Expires", image.Expires.Format(time.RFC1123))
	w.Write(image.Bytes)
}

func (p *Proxy) fetchRemoteImage(u string, cached *data.Image) (*data.Image, error) {
	glog.Infof("fetching remote image: %s", u)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	if cached != nil && cached.Etag != "" {
		req.Header.Add("If-None-Match", cached.Etag)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotModified {
		glog.Infof("remote image not modified (304 response)")
		cached.Expires = parseExpires(resp)
		return cached, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("HTTP status not OK: %v", resp.Status))
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &data.Image{
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
