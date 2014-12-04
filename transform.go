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

package imageproxy

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"

	"github.com/disintegration/imaging"
)

// compression quality of resized jpegs
const jpegQuality = 95

// Transform the provided image.  img should contain the raw bytes of an
// encoded image in one of the supported formats (gif, jpeg, or png).  The
// bytes of a similarly encoded image is returned.
func Transform(img []byte, opt Options) ([]byte, error) {
	if opt == emptyOptions {
		// bail if no transformation was requested
		return img, nil
	}

	// decode image
	m, format, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		return nil, err
	}

	m = transformImage(m, opt)

	// encode image
	buf := new(bytes.Buffer)
	switch format {
	case "gif":
		gif.Encode(buf, m, nil)
	case "jpeg":
		jpeg.Encode(buf, m, &jpeg.Options{Quality: jpegQuality})
	case "png":
		png.Encode(buf, m)
	}

	return buf.Bytes(), nil
}

// transformImage modifies the image m based on the transformations specified
// in opt.
func transformImage(m image.Image, opt Options) image.Image {
	// convert percentage width and height values to absolute values
	imgW := m.Bounds().Max.X - m.Bounds().Min.X
	imgH := m.Bounds().Max.Y - m.Bounds().Min.Y
	var w, h int
	if 0 < opt.Width && opt.Width < 1 {
		w = int(float64(imgW) * opt.Width)
	} else if opt.Width < 0 {
		w = 0
	} else {
		w = int(opt.Width)
	}
	if 0 < opt.Height && opt.Height < 1 {
		h = int(float64(imgH) * opt.Height)
	} else if opt.Height < 0 {
		h = 0
	} else {
		h = int(opt.Height)
	}

	// never resize larger than the original image
	if w > imgW {
		w = imgW
	}
	if h > imgH {
		h = imgH
	}

	// resize
	if w != 0 || h != 0 {
		if opt.Fit {
			m = imaging.Fit(m, w, h, imaging.Lanczos)
		} else {
			if w == 0 || h == 0 {
				m = imaging.Resize(m, w, h, imaging.Lanczos)
			} else {
				m = imaging.Thumbnail(m, w, h, imaging.Lanczos)
			}
		}
	}

	// flip
	if opt.FlipVertical {
		m = imaging.FlipV(m)
	}
	if opt.FlipHorizontal {
		m = imaging.FlipH(m)
	}

	// rotate
	switch opt.Rotate {
	case 90:
		m = imaging.Rotate90(m)
	case 180:
		m = imaging.Rotate180(m)
	case 270:
		m = imaging.Rotate270(m)
	}

	return m
}
