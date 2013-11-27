// Package proxy provides the image proxy.
package proxy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// URLError reports a malformed URL error.
type URLError struct {
	Message string
	URL     *url.URL
}

func (e URLError) Error() string {
	return fmt.Sprintf("malformed URL %q: %s", e.URL, e.Message)
}

// Request is a request for an image.
type Request struct {
	URL    *url.URL // URL of the image to proxy
	Width  int      // requested width, in pixels
	Height int      // requested height, in pixels
}

// NewRequest parses an http.Request into an image request.
func NewRequest(r *http.Request) (*Request, error) {
	path := strings.SplitN(r.URL.Path, "/", 3)
	if len(path) != 3 {
		return nil, URLError{"too few path segments", r.URL}
	}

	var err error
	req := new(Request)

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

	var h, w string
	size := strings.SplitN(path[1], "x", 2)
	w = size[0]
	if len(size) > 1 {
		h = size[1]
	} else {
		h = w
	}

	if w != "" {
		req.Width, err = strconv.Atoi(w)
		if err != nil {
			return nil, URLError{"width must be an int", r.URL}
		}
	}
	if h != "" {
		req.Height, err = strconv.Atoi(h)
		if err != nil {
			return nil, URLError{"height must be an int", r.URL}
		}
	}

	return req, nil
}

// Proxy serves image requests.
type Proxy struct {
	Client *http.Client // client used to fetch remote URLs
}

// NewProxy constructs a new proxy.  The provided http Client will be used to
// fetch remote URLs.  If nil is provided, http.DefaultClient will be used.
func NewProxy(client *http.Client) *Proxy {
	if client == nil {
		client = http.DefaultClient
	}
	return &Proxy{Client: client}
}

// ServeHTTP handles image requests.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, err := NewRequest(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request URL: %v", err.Error()), http.StatusBadRequest)
		return
	}
	resp, err := p.Client.Get(req.URL.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("error fetching remote image: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("error fetching remote image: %v", resp.Status), resp.StatusCode)
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error fetching remote image: %v", err.Error()), http.StatusInternalServerError)
	}
	w.Write(data)
}
