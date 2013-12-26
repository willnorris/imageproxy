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
			"http://example.com/", &Options{1, 0, false, 0, false, false}, false,
		},
		{
			"http://localhost/x1/http://example.com/",
			"http://example.com/", &Options{0, 1, false, 0, false, false}, false,
		},
		{
			"http://localhost/1x2/http://example.com/",
			"http://example.com/", &Options{1, 2, false, 0, false, false}, false,
		},
		{
			"http://localhost/0.1x0.2/http://example.com/",
			"http://example.com/", &Options{0.1, 0.2, false, 0, false, false}, false,
		},
		{
			"http://localhost/,fit/http://example.com/",
			"http://example.com/", &Options{0, 0, true, 0, false, false}, false,
		},
		{
			"http://localhost/1x2,fit,r90,fv,fh/http://example.com/",
			"http://example.com/", &Options{1, 2, true, 90, true, true}, false,
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
