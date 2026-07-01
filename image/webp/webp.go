package png

import (
	"bytes"
	"image"

	"github.com/gen2brain/webp" // encoder (libwebp via WASM)
	_ "golang.org/x/image/webp" // decoder only
	ipimage "willnorris.com/go/imageproxy/image"
	"willnorris.com/go/imageproxy/options"
)

var defaultQuality = 80

func init() {
	ipimage.RegisterEncoder("webp", encoder{})
}

type encoder struct{}

func (encoder) Encode(m image.Image, opt options.Options) ([]byte, error) {
	quality := opt.Quality
	if quality == 0 {
		quality = defaultQuality
	}
	buf := new(bytes.Buffer)
	err := webp.Encode(buf, m, webp.Options{Quality: quality, Method: 4})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (encoder) ContentType() string {
	return "image/webp"
}
