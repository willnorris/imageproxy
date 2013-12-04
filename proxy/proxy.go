// Package proxy provides the image proxy.
package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error fetching remote image: %v", err.Error()), http.StatusInternalServerError)
	}
}
