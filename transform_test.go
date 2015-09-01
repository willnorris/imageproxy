package imageproxy

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"reflect"
	"testing"

	"github.com/disintegration/imaging"
)

var (
	red    = color.NRGBA{255, 0, 0, 255}
	green  = color.NRGBA{0, 255, 0, 255}
	blue   = color.NRGBA{0, 0, 255, 255}
	yellow = color.NRGBA{255, 255, 0, 255}
)

// newImage creates a new NRGBA image with the specified dimensions and pixel
// color data.  If the length of pixels is 1, the entire image is filled with
// that color.
func newImage(w, h int, pixels ...color.NRGBA) image.Image {
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	if len(pixels) == 1 {
		draw.Draw(m, m.Bounds(), &image.Uniform{pixels[0]}, image.ZP, draw.Src)
	} else {
		for i, p := range pixels {
			m.Set(i%w, i/w, p)
		}
	}
	return m
}

func TestTransform(t *testing.T) {
	src := newImage(2, 2, red, green, blue, yellow)

	buf := new(bytes.Buffer)
	png.Encode(buf, src)

	tests := []struct {
		name        string
		encode      func(io.Writer, image.Image)
		exactOutput bool // whether input and output should match exactly
	}{
		{"gif", func(w io.Writer, m image.Image) { gif.Encode(w, m, nil) }, true},
		{"jpeg", func(w io.Writer, m image.Image) { jpeg.Encode(w, m, nil) }, false},
		{"png", func(w io.Writer, m image.Image) { png.Encode(w, m) }, true},
	}

	for _, tt := range tests {
		buf := new(bytes.Buffer)
		tt.encode(buf, src)
		in := buf.Bytes()

		out, err := Transform(in, emptyOptions)
		if err != nil {
			t.Errorf("Transform with encoder %s returned unexpected error: %v", tt.name, err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Errorf("Transform with with encoder %s with empty options returned modified result", tt.name)
		}

		out, err = Transform(in, Options{Width: -1, Height: -1})
		if err != nil {
			t.Errorf("Transform with encoder %s returned unexpected error: %v", tt.name, err)
		}
		if len(out) == 0 {
			t.Errorf("Transform with encoder %s returned empty bytes", tt.name)
		}
		if tt.exactOutput && !reflect.DeepEqual(in, out) {
			t.Errorf("Transform with encoder %s with noop Options returned modified result", tt.name)
		}
	}

	if _, err := Transform([]byte{}, Options{Width: 1}); err == nil {
		t.Errorf("Transform with invalid image input did not return expected err")
	}
}

func TestTransformImage(t *testing.T) {
	// ref is a 2x2 reference image containing four colors
	ref := newImage(2, 2, red, green, blue, yellow)

	// use simpler filter while testing that won't skew colors
	resampleFilter = imaging.Box

	tests := []struct {
		src  image.Image // source image to transform
		opt  Options     // options to apply during transform
		want image.Image // expected transformed image
	}{
		// no transformation
		{ref, emptyOptions, ref},

		// rotations
		{ref, Options{Rotate: 45}, ref}, // invalid rotation is a noop
		{ref, Options{Rotate: 90}, newImage(2, 2, green, yellow, red, blue)},
		{ref, Options{Rotate: 180}, newImage(2, 2, yellow, blue, green, red)},
		{ref, Options{Rotate: 270}, newImage(2, 2, blue, red, yellow, green)},

		// flips
		{
			ref,
			Options{FlipHorizontal: true},
			newImage(2, 2, green, red, yellow, blue),
		},
		{
			ref,
			Options{FlipVertical: true},
			newImage(2, 2, blue, yellow, red, green),
		},
		{
			ref,
			Options{FlipHorizontal: true, FlipVertical: true},
			newImage(2, 2, yellow, blue, green, red),
		},

		// resizing
		{ // can't resize larger than original image
			ref,
			Options{Width: 100, Height: 100},
			ref,
		},
		{ // can resize larger than original image
			ref,
			Options{Width: 4, Height: 4, ScaleUp: true},
			newImage(4, 4, red, red, green, green, red, red, green, green, blue, blue, yellow, yellow, blue, blue, yellow, yellow),
		},
		{ // invalid values
			ref,
			Options{Width: -1, Height: -1},
			ref,
		},
		{ // absolute values
			newImage(100, 100, red),
			Options{Width: 1, Height: 1},
			newImage(1, 1, red),
		},
		{ // percentage values
			newImage(100, 100, red),
			Options{Width: 0.50, Height: 0.25},
			newImage(50, 25, red),
		},
		{ // only width specified, proportional height
			newImage(100, 50, red),
			Options{Width: 50},
			newImage(50, 25, red),
		},
		{ // only height specified, proportional width
			newImage(100, 50, red),
			Options{Height: 25},
			newImage(50, 25, red),
		},
		{ // resize in one dimenstion, with cropping
			newImage(4, 2, red, red, blue, blue, red, red, blue, blue),
			Options{Width: 4, Height: 1},
			newImage(4, 1, red, red, blue, blue),
		},
		{ // resize in two dimensions, with cropping
			newImage(4, 2, red, red, blue, blue, red, red, blue, blue),
			Options{Width: 2, Height: 2},
			newImage(2, 2, red, blue, red, blue),
		},
		{ // resize in two dimensions, fit option prevents cropping
			newImage(4, 2, red, red, blue, blue, red, red, blue, blue),
			Options{Width: 2, Height: 2, Fit: true},
			newImage(2, 1, red, blue),
		},
		{ // scale image explicitly
			newImage(4, 2, red, red, blue, blue, red, red, blue, blue),
			Options{Width: 2, Height: 1},
			newImage(2, 1, red, blue),
		},

		// combinations of options
		{
			newImage(4, 2, red, red, blue, blue, red, red, blue, blue),
			Options{Width: 2, Height: 1, Fit: true, FlipHorizontal: true, Rotate: 90},
			newImage(1, 2, red, blue),
		},
	}

	for _, tt := range tests {
		if got := transformImage(tt.src, tt.opt); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("trasformImage(%v, %v) returned image %#v, want %#v", tt.src, tt.opt, got, tt.want)
		}
	}
}
