package png

import (
	"bytes"
	"image"
	"image/png"

	ipimage "willnorris.com/go/imageproxy/image"
	"willnorris.com/go/imageproxy/options"
)

func init() {
	ipimage.RegisterEncoder("png", encoder{})
}

type encoder struct{}

func (encoder) Encode(m image.Image, opt options.Options) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, m)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
