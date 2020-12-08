// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"bytes"
	"encoding/base64"
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
	"golang.org/x/image/bmp"
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
func newImage(w, h int, pixels ...color.Color) image.Image {
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	if len(pixels) == 1 {
		draw.Draw(m, m.Bounds(), &image.Uniform{pixels[0]}, image.Point{}, draw.Src)
	} else {
		for i, p := range pixels {
			m.Set(i%w, i/w, p)
		}
	}
	return m
}

func TestResizeParams(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 64, 128))
	tests := []struct {
		opt    Options
		w, h   int
		resize bool
	}{
		{Options{Width: 0.5}, 32, 0, true},
		{Options{Height: 0.5}, 0, 64, true},
		{Options{Width: 0.5, Height: 0.5}, 32, 64, true},
		{Options{Width: 100, Height: 200}, 0, 0, false},
		{Options{Width: 100, Height: 200, ScaleUp: true}, 100, 200, true},
		{Options{Width: 64}, 0, 0, false},
		{Options{Height: 128}, 0, 0, false},
	}
	for _, tt := range tests {
		w, h, resize := resizeParams(src, tt.opt)
		if w != tt.w || h != tt.h || resize != tt.resize {
			t.Errorf("resizeParams(%v) returned (%d,%d,%t), want (%d,%d,%t)", tt.opt, w, h, resize, tt.w, tt.h, tt.resize)
		}
	}
}

func TestCropParams(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 64, 128))
	tests := []struct {
		opt            Options
		x0, y0, x1, y1 int
	}{
		{Options{CropWidth: 10, CropHeight: 0}, 0, 0, 10, 128},
		{Options{CropWidth: 0, CropHeight: 10}, 0, 0, 64, 10},
		{Options{CropWidth: -1, CropHeight: -1}, 0, 0, 64, 128},
		{Options{CropWidth: 50, CropHeight: 100}, 0, 0, 50, 100},
		{Options{CropWidth: 100, CropHeight: 100}, 0, 0, 64, 100},
		{Options{CropX: 50, CropY: 100}, 50, 100, 64, 128},
		{Options{CropX: 50, CropY: 100, CropWidth: 100, CropHeight: 150}, 50, 100, 64, 128},
		{Options{CropX: -50, CropY: -50}, 14, 78, 64, 128},
		{Options{CropY: 0.5, CropWidth: 0.5}, 0, 64, 32, 128},
		{Options{Width: 10, Height: 10, SmartCrop: true}, 0, 0, 64, 64},
	}
	for _, tt := range tests {
		want := image.Rect(tt.x0, tt.y0, tt.x1, tt.y1)
		got := cropParams(src, tt.opt)
		if !got.Eq(want) {
			t.Errorf("cropParams(%v) returned %v, want %v", tt.opt, got, want)
		}
	}
}

func TestTransform(t *testing.T) {
	src := newImage(2, 2, red, green, blue, yellow)

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, src); err != nil {
		t.Errorf("error encoding reference image: %v", err)
	}

	tests := []struct {
		name        string
		encode      func(io.Writer, image.Image) error
		exactOutput bool // whether input and output should match exactly
	}{
		{"bmp", func(w io.Writer, m image.Image) error { return bmp.Encode(w, m) }, true},
		{"gif", func(w io.Writer, m image.Image) error { return gif.Encode(w, m, nil) }, true},
		{"jpeg", func(w io.Writer, m image.Image) error { return jpeg.Encode(w, m, nil) }, false},
		{"png", func(w io.Writer, m image.Image) error { return png.Encode(w, m) }, true},
	}

	for _, tt := range tests {
		buf := new(bytes.Buffer)
		if err := tt.encode(buf, src); err != nil {
			t.Errorf("error encoding image: %v", err)
		}
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

func TestTranform_ImageTooLarge(t *testing.T) {
	// lottapixel.jpg (https://hackerone.com/reports/390)
	lottaPixel := `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`

	lg, err := base64.StdEncoding.Strict().Strict().DecodeString(lottaPixel)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = Transform(lg, Options{Width: 1}); err == nil {
		t.Errorf("Transform with large image did not return expected error")
	}
}

func TestTransform_InvalidFormat(t *testing.T) {
	src := newImage(2, 2, red, green, blue, yellow)
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, src); err != nil {
		t.Errorf("error encoding reference image: %v", err)
	}

	_, err := Transform(buf.Bytes(), Options{Format: "invalid"})
	if err == nil {
		t.Errorf("Transform with invalid format did not return expected error")
	}
}

// Test that each of the eight EXIF orientations is applied to the transformed
// image appropriately.
func TestTransform_EXIF(t *testing.T) {
	ref := newImage(2, 2, red, green, blue, yellow)

	// reference image encoded as TIF, with each of the 8 EXIF orientations
	// applied in reverse and the EXIF tag set. When orientation is
	// applied, each should display as the ref image.
	tests := []string{
		"SUkqAAgAAAAOAAABAwABAAAAAgAAAAEBAwABAAAAAgAAAAIBAwAEAAAAtgAAAAMBAwABAAAACAAAAAYBAwABAAAAAgAAABEBBAABAAAAzgAAABIBAwABAAAAAQAAABUBAwABAAAABAAAABYBAwABAAAAAgAAABcBBAABAAAAGQAAABoBBQABAAAAvgAAABsBBQABAAAAxgAAACgBAwABAAAAAgAAAFIBAwABAAAAAgAAAAAAAAAIAAgACAAIAEgAAAABAAAASAAAAAEAAAB4nPrPwPAfDBn+////n+E/IAAA//9DzAj4AA==", // Orientation=1
		"SUkqAAgAAAAOAAABAwABAAAAAgAAAAEBAwABAAAAAgAAAAIBAwAEAAAAtgAAAAMBAwABAAAACAAAAAYBAwABAAAAAgAAABEBBAABAAAAzgAAABIBAwABAAAAAgAAABUBAwABAAAABAAAABYBAwABAAAAAgAAABcBBAABAAAAGQAAABoBBQABAAAAvgAAABsBBQABAAAAxgAAACgBAwABAAAAAgAAAFIBAwABAAAAAgAAAAAAAAAIAAgACAAIAEgAAAABAAAASAAAAAEAAAB4nGL4z/D/PwPD////GcAUIAAA//9HyAj4AA==", // Orientation=2
		"SUkqAAgAAAAOAAABAwABAAAAAgAAAAEBAwABAAAAAgAAAAIBAwAEAAAAtgAAAAMBAwABAAAACAAAAAYBAwABAAAAAgAAABEBBAABAAAAzgAAABIBAwABAAAAAwAAABUBAwABAAAABAAAABYBAwABAAAAAgAAABcBBAABAAAAFwAAABoBBQABAAAAvgAAABsBBQABAAAAxgAAACgBAwABAAAAAgAAAFIBAwABAAAAAgAAAAAAAAAIAAgACAAIAEgAAAABAAAASAAAAAEAAAB4nPr/n+E/AwOY/A9iAAIAAP//T8AI+AA=",     // Orientation=3
		"SUkqAAgAAAAOAAABAwABAAAAAgAAAAEBAwABAAAAAgAAAAIBAwAEAAAAtgAAAAMBAwABAAAACAAAAAYBAwABAAAAAgAAABEBBAABAAAAzgAAABIBAwABAAAABAAAABUBAwABAAAABAAAABYBAwABAAAAAgAAABcBBAABAAAAGgAAABoBBQABAAAAvgAAABsBBQABAAAAxgAAACgBAwABAAAAAgAAAFIBAwABAAAAAgAAAAAAAAAIAAgACAAIAEgAAAABAAAASAAAAAEAAAB4nGJg+P///3+G//8ZGP6DICAAAP//S8QI+A==", // Orientation=4
		"SUkqAAgAAAAOAAABAwABAAAAAgAAAAEBAwABAAAAAgAAAAIBAwAEAAAAtgAAAAMBAwABAAAACAAAAAYBAwABAAAAAgAAABEBBAABAAAAzgAAABIBAwABAAAABQAAABUBAwABAAAABAAAABYBAwABAAAAAgAAABcBBAABAAAAGAAAABoBBQABAAAAvgAAABsBBQABAAAAxgAAACgBAwABAAAAAgAAAFIBAwABAAAAAgAAAAAAAAAIAAgACAAIAEgAAAABAAAASAAAAAEAAAB4nPrPwABC/xn+M/wHkYAAAAD//0PMCPg=",     // Orientation=5
		"SUkqAAgAAAAOAAABAwABAAAAAgAAAAEBAwABAAAAAgAAAAIBAwAEAAAAtgAAAAMBAwABAAAACAAAAAYBAwABAAAAAgAAABEBBAABAAAAzgAAABIBAwABAAAABgAAABUBAwABAAAABAAAABYBAwABAAAAAgAAABcBBAABAAAAGAAAABoBBQABAAAAvgAAABsBBQABAAAAxgAAACgBAwABAAAAAgAAAFIBAwABAAAAAgAAAAAAAAAIAAgACAAIAEgAAAABAAAASAAAAAEAAAB4nGL4z/D/PwgzMIDQf0AAAAD//0vECPg=",     // Orientation=6
		"SUkqAAgAAAAOAAABAwABAAAAAgAAAAEBAwABAAAAAgAAAAIBAwAEAAAAtgAAAAMBAwABAAAACAAAAAYBAwABAAAAAgAAABEBBAABAAAAzgAAABIBAwABAAAABwAAABUBAwABAAAABAAAABYBAwABAAAAAgAAABcBBAABAAAAFgAAABoBBQABAAAAvgAAABsBBQABAAAAxgAAACgBAwABAAAAAgAAAFIBAwABAAAAAgAAAAAAAAAIAAgACAAIAEgAAAABAAAASAAAAAEAAAB4nPr/nwECGf7/BxGAAAAA//9PwAj4",         // Orientation=7
		"SUkqAAgAAAAOAAABAwABAAAAAgAAAAEBAwABAAAAAgAAAAIBAwAEAAAAtgAAAAMBAwABAAAACAAAAAYBAwABAAAAAgAAABEBBAABAAAAzgAAABIBAwABAAAACAAAABUBAwABAAAABAAAABYBAwABAAAAAgAAABcBBAABAAAAFQAAABoBBQABAAAAvgAAABsBBQABAAAAxgAAACgBAwABAAAAAgAAAFIBAwABAAAAAgAAAAAAAAAIAAgACAAIAEgAAAABAAAASAAAAAEAAAB4nGJg+P//P4QAQ0AAAAD//0fICPgA",         // Orientation=8
	}

	for _, src := range tests {
		in, err := base64.StdEncoding.DecodeString(src)
		if err != nil {
			t.Errorf("error decoding source: %v", err)
		}
		out, err := Transform(in, Options{Height: -1, Width: -1, Format: "tiff"})
		if err != nil {
			t.Errorf("Transform(%q) returned error: %v", src, err)
		}
		d, _, err := image.Decode(bytes.NewReader(out))
		if err != nil {
			t.Errorf("error decoding transformed image: %v", err)
		}

		// construct new image with same colors as decoded image for easy comparison
		got := newImage(2, 2, d.At(0, 0), d.At(1, 0), d.At(0, 1), d.At(1, 1))
		if want := ref; !reflect.DeepEqual(got, want) {
			t.Errorf("Transform(%v) returned image %#v, want %#v", src, got, want)
		}
	}
}

// Test that EXIF orientation and any additional transforms don't conflict.
// This is tested with orientation=7, which involves both a rotation and a
// flip, combined with an additional rotation transform.
func TestTransform_EXIF_Rotate(t *testing.T) {
	// base64-encoded TIF image (2x2 yellow green blue red) with EXIF
	// orientation=7. When orientation applied, displays as (2x2 red green
	// blue yellow).
	src := "SUkqAAgAAAAOAAABAwABAAAAAgAAAAEBAwABAAAAAgAAAAIBAwAEAAAAtgAAAAMBAwABAAAACAAAAAYBAwABAAAAAgAAABEBBAABAAAAzgAAABIBAwABAAAABwAAABUBAwABAAAABAAAABYBAwABAAAAAgAAABcBBAABAAAAFgAAABoBBQABAAAAvgAAABsBBQABAAAAxgAAACgBAwABAAAAAgAAAFIBAwABAAAAAgAAAAAAAAAIAAgACAAIAEgAAAABAAAASAAAAAEAAAB4nPr/nwECGf7/BxGAAAAA//9PwAj4"

	in, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		t.Errorf("error decoding source: %v", err)
	}
	out, err := Transform(in, Options{Rotate: 90, Format: "tiff"})
	if err != nil {
		t.Errorf("Transform(%q) returned error: %v", src, err)
	}
	d, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Errorf("error decoding transformed image: %v", err)
	}

	// construct new image with same colors as decoded image for easy comparison
	got := newImage(2, 2, d.At(0, 0), d.At(1, 0), d.At(0, 1), d.At(1, 1))
	want := newImage(2, 2, green, yellow, red, blue)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Transform(%v) returned image %#v, want %#v", src, got, want)
	}
}

func TestTransformImage(t *testing.T) {
	// ref is a 2x2 reference image containing four colors
	ref := newImage(2, 2, red, green, blue, yellow)

	// cropRef is a 4x4 image with four colors, each in 2x2 quarter
	cropRef := newImage(4, 4, red, red, green, green, red, red, green, green, blue, blue, yellow, yellow, blue, blue, yellow, yellow)

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
		{ref, Options{Rotate: 360}, ref},
		{ref, Options{Rotate: 90}, newImage(2, 2, green, yellow, red, blue)},
		{ref, Options{Rotate: 180}, newImage(2, 2, yellow, blue, green, red)},
		{ref, Options{Rotate: 270}, newImage(2, 2, blue, red, yellow, green)},
		{ref, Options{Rotate: 630}, newImage(2, 2, blue, red, yellow, green)},
		{ref, Options{Rotate: -90}, newImage(2, 2, blue, red, yellow, green)},

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
		{
			ref,
			Options{Rotate: 90, FlipHorizontal: true},
			newImage(2, 2, yellow, green, blue, red),
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
			newImage(1, 2, blue, red),
		},

		// crop
		{ // quarter ((0, 0), (2, 2)) -> red
			cropRef,
			Options{CropHeight: 2, CropWidth: 2},
			newImage(2, 2, red, red, red, red),
		},
		{ // quarter ((2, 0), (4, 2)) -> green
			cropRef,
			Options{CropHeight: 2, CropWidth: 2, CropX: 2},
			newImage(2, 2, green, green, green, green),
		},
		{ // quarter ((0, 2), (2, 4)) -> blue
			cropRef,
			Options{CropHeight: 2, CropWidth: 2, CropX: 0, CropY: 2},
			newImage(2, 2, blue, blue, blue, blue),
		},
		{ // quarter ((2, 2), (4, 4)) -> yellow
			cropRef,
			Options{CropHeight: 2, CropWidth: 2, CropX: 2, CropY: 2},
			newImage(2, 2, yellow, yellow, yellow, yellow),
		},

		// percentage-based resize in addition to rectangular crop
		{
			newImage(12, 12, red),
			Options{Width: 0.5, Height: 0.5, CropWidth: 8, CropHeight: 8},
			newImage(6, 6, red),
		},
	}

	for _, tt := range tests {
		if got := transformImage(tt.src, tt.opt); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("transformImage(%v, %v) returned image %#v, want %#v", tt.src, tt.opt, got, tt.want)
		}
	}
}

func TestTrimEdges(t *testing.T) {
	x := color.NRGBA{255, 255, 255, 255}
	o := color.NRGBA{0, 0, 0, 255}

	tests := []struct {
		name string
		src  image.Image // source image to transform
		want image.Image // expected transformed image
	}{
		{
			name: "empty",
			src:  newImage(0, 0),
			want: newImage(0, 0), // same as src
		},
		{
			name: "solid",
			src:  newImage(8, 8, x),
			want: newImage(8, 8, x), // same as src
		},
		{
			name: "square",
			src: newImage(4, 4,
				x, x, x, x,
				x, o, o, x,
				x, o, o, x,
				x, x, x, x,
			),
			want: newImage(2, 2,
				o, o,
				o, o,
			),
		},
		{
			name: "diamond",
			src: newImage(5, 5,
				x, x, x, x, x,
				x, x, o, x, x,
				x, o, o, o, x,
				x, x, o, x, x,
				x, x, x, x, x,
			),
			want: newImage(3, 3,
				x, o, x,
				o, o, o,
				x, o, x,
			),
		},
		{
			name: "irregular",
			src: newImage(5, 5,
				x, o, x, x, x,
				x, o, o, x, x,
				x, o, o, x, x,
				x, x, x, x, x,
				x, x, x, x, x,
			),
			want: newImage(2, 3,
				o, x,
				o, o,
				o, o,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimEdges(tt.src)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("trimEdges() returned image %#v, want %#v", got, tt.want)
			}
		})
	}
}
