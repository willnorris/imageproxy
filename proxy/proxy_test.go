package proxy

import (
	"net/http"
	"testing"
)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		URL         string
		RemoteURL   string
		Width       int
		Height      int
		ExpectError bool
	}{
		// invalid URLs
		{
			"http://localhost/", "", 0, 0, true,
		},
		{
			"http://localhost/1/", "", 0, 0, true,
		},
		{
			"http://localhost//example.com/foo", "", 0, 0, true,
		},
		{
			"http://localhost//ftp://example.com/foo", "", 0, 0, true,
		},
		{
			"http://localhost/s/http://example.com/", "", 0, 0, true,
		},
		{
			"http://localhost/1xs/http://example.com/", "", 0, 0, true,
		},

		// valid URLs
		{
			"http://localhost//http://example.com/foo",
			"http://example.com/foo", 0, 0, false,
		},
		{
			"http://localhost//https://example.com/foo",
			"https://example.com/foo", 0, 0, false,
		},
		{
			"http://localhost//http://example.com/foo?bar",
			"http://example.com/foo?bar", 0, 0, false,
		},

		// size variations
		{
			"http://localhost/x/http://example.com/",
			"http://example.com/", 0, 0, false,
		},
		{
			"http://localhost/0/http://example.com/",
			"http://example.com/", 0, 0, false,
		},
		{
			"http://localhost/1x/http://example.com/",
			"http://example.com/", 1, 0, false,
		},
		{
			"http://localhost/x1/http://example.com/",
			"http://example.com/", 0, 1, false,
		},
		{
			"http://localhost/1x2/http://example.com/",
			"http://example.com/", 1, 2, false,
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
		if tt.Height != r.Height {
			t.Errorf("%d. Request Height = %v, want %v", i, r.Height, tt.Height)
		}
		if tt.Width != r.Width {
			t.Errorf("%d. Request Width = %v, want %v", i, r.Width, tt.Width)
		}
	}
}
