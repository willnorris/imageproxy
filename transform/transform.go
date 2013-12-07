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

// Package transform handles image transformation such as resizing.
package transform

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"reflect"

	"github.com/disintegration/imaging"
	"github.com/willnorris/go-imageproxy/data"
)

var emptyOptions = new(data.Options)

// Transform the provided image.
func Transform(img data.Image, opt *data.Options) (*data.Image, error) {
	if opt == nil || reflect.DeepEqual(opt, emptyOptions) {
		// bail if no transformation was requested
		return &img, nil
	}

	if opt.Width == 0 && opt.Height == 0 {
		// TODO(willnorris): Currently, only resize related options are
		// supported, so bail if no sizes are specified.  Remove this
		// check if we ever support non-resizing transformations.
		return &img, nil
	}

	// decode image
	m, format, err := image.Decode(bytes.NewReader(img.Bytes))
	if err != nil {
		return nil, err
	}

	// convert percentage width and height values to absolute values
	var h, w int
	if opt.Width > 0 && opt.Width < 1 {
		w = int(float64(m.Bounds().Max.X-m.Bounds().Min.X) * opt.Width)
	} else {
		w = int(opt.Width)
	}
	if opt.Height > 0 && opt.Height < 1 {
		h = int(float64(m.Bounds().Max.Y-m.Bounds().Min.Y) * opt.Height)
	} else {
		h = int(opt.Height)
	}

	// resize
	if opt.Fit {
		m = imaging.Fit(m, w, h, imaging.Lanczos)
	} else {
		if opt.Width == 0 || opt.Height == 0 {
			m = imaging.Resize(m, w, h, imaging.Lanczos)
		} else {
			m = imaging.Thumbnail(m, w, h, imaging.Lanczos)
		}
	}

	// rotate
	switch opt.Rotate {
	case 90:
		m = imaging.Rotate90(m)
		break
	case 180:
		m = imaging.Rotate180(m)
		break
	case 270:
		m = imaging.Rotate270(m)
		break
	}

	// encode image
	buf := new(bytes.Buffer)
	switch format {
	case "gif":
		gif.Encode(buf, m, nil)
		break
	case "jpeg":
		jpeg.Encode(buf, m, nil)
		break
	case "png":
		png.Encode(buf, m)
		break
	}

	img.Bytes = buf.Bytes()
	return &img, nil
}
