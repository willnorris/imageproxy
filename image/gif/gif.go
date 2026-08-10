package gif

import (
	"bytes"
	"image"
	"image/gif"
	"io"

	"willnorris.com/go/gifresize"
	ipimage "willnorris.com/go/imageproxy/image"
	"willnorris.com/go/imageproxy/options"
)

func init() {
	ipimage.RegisterEncoder("gif", encoder{})
}

type encoder struct{}

func (encoder) Encode(m image.Image, opt options.Options) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := gif.Encode(buf, m, nil)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (encoder) RawEncode(r io.Reader, opt options.Options, tf ipimage.TransformFunc) ([]byte, error) {
	fn := func(img image.Image) image.Image {
		return tf(img, opt)
	}

	buf := new(bytes.Buffer)
	err := gifresize.Process(buf, r, fn)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
