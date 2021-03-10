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
