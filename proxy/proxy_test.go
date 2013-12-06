package proxy

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/willnorris/go-imageproxy/data"
)

var emptyOptions = new(data.Options)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		URL         string
		RemoteURL   string
		Options     *data.Options
		ExpectError bool
	}{
		// invalid URLs
		{
			"http://localhost/", "", nil, true,
		},
		{
			"http://localhost/1/", "", nil, true,
		},
		{
			"http://localhost//example.com/foo", "", nil, true,
		},
		{
			"http://localhost//ftp://example.com/foo", "", nil, true,
		},

		// invalid options.  These won't return errors, but will not fully parse the options
		{
			"http://localhost/s/http://example.com/",
			"http://example.com/", emptyOptions, false,
		},
		{
			"http://localhost/1xs/http://example.com/",
			"http://example.com/", &data.Options{Width: 1}, false,
		},

		// valid URLs
		{
			"http://localhost/http://example.com/foo",
			"http://example.com/foo", nil, false,
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
			"http://localhost//http://example.com/foo?bar",
			"http://example.com/foo?bar", emptyOptions, false,
		},

		// size variations
		{
			"http://localhost/x/http://example.com/",
			"http://example.com/", emptyOptions, false,
		},
		{
			"http://localhost/0/http://example.com/",
			"http://example.com/", emptyOptions, false,
		},
		{
			"http://localhost/1x/http://example.com/",
			"http://example.com/", &data.Options{1, 0, false}, false,
		},
		{
			"http://localhost/x1/http://example.com/",
			"http://example.com/", &data.Options{0, 1, false}, false,
		},
		{
			"http://localhost/1x2/http://example.com/",
			"http://example.com/", &data.Options{1, 2, false}, false,
		},
		{
			"http://localhost/,fit/http://example.com/",
			"http://example.com/", &data.Options{0, 0, true}, false,
		},
		{
			"http://localhost/1x2,fit/http://example.com/",
			"http://example.com/", &data.Options{1, 2, true}, false,
		},
		{
			"http://localhost/0.1x0.2,fit/http://example.com/",
			"http://example.com/", &data.Options{0.1, 0.2, true}, false,
		},
	}

	for i, tt := range tests {
		req, err := http.NewRequest("GET", tt.URL, nil)
		if err != nil {
			t.Errorf("%d. Error parsing request: %v", i, err)
			continue
		}

		r, err := NewRequest(req)
		if tt.ExpectError {
			if err == nil {
				t.Errorf("%d. Expected parsing error", i)
			}
			continue
		} else if err != nil {
			t.Errorf("%d. Error parsing request: %v", i, err)
			continue
		}

		if got := r.URL.String(); tt.RemoteURL != got {
			t.Errorf("%d. Request URL = %v, want %v", i, got, tt.RemoteURL)
		}
		if !reflect.DeepEqual(tt.Options, r.Options) {
			t.Errorf("%d. Request Options = %v, want %v", i, r.Options, tt.Options)
		}
	}
}
