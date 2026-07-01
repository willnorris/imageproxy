package png

import (
	"bytes"
	"image"

	"github.com/gen2brain/avif"
	ipimage "willnorris.com/go/imageproxy/image"
	"willnorris.com/go/imageproxy/options"
)

const (
	// AVIF's quality scale is more aggressive than JPEG, so a lower
	// number yields comparable perceived quality at a much smaller size.
	defaultQuality = 55

	// AVIF encode speed 0(slow)–10(fast). 8 keeps CPU sane on modest servers;
	// every variant is encoded once and then served from cache.
	encodeSpeed = 8
)

func init() {
	ipimage.RegisterEncoder("avif", encoder{})
}

type encoder struct{}

func (encoder) Encode(m image.Image, opt options.Options) ([]byte, error) {
	quality := opt.Quality
	if quality == 0 {
		quality = defaultQuality
	}
	buf := new(bytes.Buffer)
	err := avif.Encode(buf, m, avif.Options{Quality: quality, Speed: encodeSpeed})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (encoder) ContentType() string {
	return "image/avif"
}
