package bmp

import (
	"bytes"
	"image"

	"golang.org/x/image/bmp"
	ipimage "willnorris.com/go/imageproxy/image"
	"willnorris.com/go/imageproxy/options"
)

func init() {
	ipimage.RegisterEncoder("bmp", encoder{})
}

type encoder struct{}

func (encoder) Encode(m image.Image, opt options.Options) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := bmp.Encode(buf, m)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
