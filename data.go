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

// Package data provides common shared data structures for imageproxy.
package imageproxy

import (
	"bytes"
	"fmt"
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

// Options specifies transformations that can be performed on a
// requested image.
type Options struct {
	Width  float64 // requested width, in pixels
	Height float64 // requested height, in pixels

	// If true, resize the image to fit in the specified dimensions.  Image
	// will not be cropped, and aspect ratio will be maintained.
	Fit bool

	// Rotate image the specified degrees counter-clockwise.  Valid values are 90, 180, 270.
	Rotate int

	FlipVertical   bool
	FlipHorizontal bool
}

var emptyOptions = Options{}

func (o Options) String() string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%vx%v", o.Width, o.Height)
	if o.Fit {
		buf.WriteString(",fit")
	}
	if o.Rotate != 0 {
		fmt.Fprintf(buf, ",r%d", o.Rotate)
	}
	if o.FlipVertical {
		buf.WriteString(",fv")
	}
	if o.FlipHorizontal {
		buf.WriteString(",fh")
	}
	return buf.String()
}

func ParseOptions(str string) Options {
	o := Options{}

	parts := strings.Split(str, ",")
	for _, part := range parts {
		if part == "fit" {
			o.Fit = true
			continue
		}
		if part == "fv" {
			o.FlipVertical = true
			continue
		}
		if part == "fh" {
			o.FlipHorizontal = true
			continue
		}

		if len(part) > 2 && part[:1] == "r" {
			o.Rotate, _ = strconv.Atoi(part[1:])
			continue
		}

		if strings.ContainsRune(part, 'x') {
			var h, w string
			size := strings.SplitN(part, "x", 2)
			w = size[0]
			if len(size) > 1 {
				h = size[1]
			} else {
				h = w
			}

			if w != "" {
				o.Width, _ = strconv.ParseFloat(w, 64)
			}
			if h != "" {
				o.Height, _ = strconv.ParseFloat(h, 64)
			}
			continue
		}

		if size, err := strconv.ParseFloat(part, 64); err == nil {
			o.Width = size
			o.Height = size
			continue
		}
	}

	return o
}

type Request struct {
	URL     *url.URL // URL of the image to proxy
	Options Options  // Image transformation to perform
}

// NewRequest parses an http.Request into an image request.
func NewRequest(r *http.Request) (*Request, error) {
	var err error
	req := new(Request)

	path := r.URL.Path[1:] // strip leading slash
	req.URL, err = url.Parse(path)
	if err != nil || !req.URL.IsAbs() {
		// first segment is likely options
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			return nil, URLError{"too few path segments", r.URL}
		}

		req.URL, err = url.Parse(parts[1])
		if err != nil {
			return nil, URLError{fmt.Sprintf("unable to parse remote URL: %v", err), r.URL}
		}

		req.Options = ParseOptions(parts[0])
	}

	if !req.URL.IsAbs() {
		return nil, URLError{"must provide absolute remote URL", r.URL}
	}

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return nil, URLError{"remote URL must have http or https URL", r.URL}
	}

	// query string is always part of the remote URL
	req.URL.RawQuery = r.URL.RawQuery
	return req, nil
}
