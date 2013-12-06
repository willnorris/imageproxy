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
		return &img, nil
	}

	// decode image
	m, format, err := image.Decode(bytes.NewReader(img.Bytes))
	if err != nil {
		return nil, err
	}

	// resize
	if opt.Width != 0 || opt.Height != 0 {
		m = imaging.Fit(m, opt.Width, opt.Height, imaging.Lanczos)
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
