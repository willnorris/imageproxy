// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

// image_format_test.go contains tests that require registered image encoders.
// These are separate from image_test.go to prevent import cycles.

package image_test

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

	"golang.org/x/image/bmp"

	ipimage "willnorris.com/go/imageproxy/image"

	// register encoders used in tests
	_ "willnorris.com/go/imageproxy/image/bmp"
	_ "willnorris.com/go/imageproxy/image/gif"
	_ "willnorris.com/go/imageproxy/image/jpeg"
	_ "willnorris.com/go/imageproxy/image/png"
	_ "willnorris.com/go/imageproxy/image/tiff"
)

var (
	red    = color.NRGBA{255, 0, 0, 255}
	green  = color.NRGBA{0, 255, 0, 255}
	blue   = color.NRGBA{0, 0, 255, 255}
	yellow = color.NRGBA{255, 255, 0, 255}
)

var emptyOptions = ipimage.Options{}

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

		out, err := ipimage.Transform(in, emptyOptions)
		if err != nil {
			t.Errorf("Transform with encoder %s returned unexpected error: %v", tt.name, err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Errorf("Transform with with encoder %s with empty options returned modified result", tt.name)
		}

		out, err = ipimage.Transform(in, ipimage.Options{Width: -1, Height: -1})
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

	if _, err := ipimage.Transform([]byte{}, ipimage.Options{Width: 1}); err == nil {
		t.Errorf("Transform with invalid image input did not return expected err")
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
		out, err := ipimage.Transform(in, ipimage.Options{Height: -1, Width: -1, Format: "tiff"})
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
	out, err := ipimage.Transform(in, ipimage.Options{Rotate: 90, Format: "tiff"})
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
