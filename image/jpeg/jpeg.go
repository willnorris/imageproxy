package jpeg

import (
	"bytes"
	"image"
	"image/jpeg"

	ipimage "willnorris.com/go/imageproxy/image"
	"willnorris.com/go/imageproxy/options"
)

// default compression quality of resized jpegs
const defaultQuality = 95

func init() {
	ipimage.RegisterEncoder("jpeg", encoder{})
}

type encoder struct{}

func (encoder) Encode(m image.Image, opt options.Options) ([]byte, error) {
	quality := opt.Quality
	if quality == 0 {
		quality = defaultQuality
	}

	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, m, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
