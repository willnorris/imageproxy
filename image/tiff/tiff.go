package tiff

import (
	"bytes"
	"image"

	"golang.org/x/image/tiff"
	ipimage "willnorris.com/go/imageproxy/image"
	"willnorris.com/go/imageproxy/options"
)

func init() {
	ipimage.RegisterEncoder("tiff", encoder{})
}

type encoder struct{}

func (encoder) Encode(m image.Image, opt options.Options) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := tiff.Encode(buf, m, &tiff.Options{
		Compression: tiff.Deflate,
		Predictor:   true,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
