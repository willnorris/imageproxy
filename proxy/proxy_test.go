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

package proxy

import (
	"net/http"
	"reflect"
	"testing"
)

// Test that request URLs are properly parsed into Options and RemoteURL.  This
// test verifies that invalid remote URLs throw errors, and that valid
// combinations of Options and URL are accept.  This does not exhaustively test
// the various Options that can be specified; see TestParseOptions for that.
func TestNewRequest(t *testing.T) {
	tests := []struct {
		URL         string
		RemoteURL   string
		Options     *Options
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
			"http://example.com/", &Options{Width: 1}, false,
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
			"http://localhost/1x2/http://example.com/foo",
			"http://example.com/foo", &Options{Width: 1, Height: 2}, false,
		},
		{
			"http://localhost//http://example.com/foo?bar",
			"http://example.com/foo?bar", emptyOptions, false,
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
