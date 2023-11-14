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
	largeImage := []byte{255, 216, 255, 224, 0, 16, 74, 70, 73, 70, 0, 1, 1, 1, 0, 22, 0, 22, 0, 0, 255, 219, 0, 67, 0, 2, 2, 2, 2, 2, 1, 2, 2, 2, 2, 3, 2, 2, 3, 3, 6, 4, 3, 3, 3, 3, 7, 5, 5, 4, 6, 8, 7, 9, 8, 8, 7, 8, 8, 9, 10, 13, 11, 9, 10, 12, 10, 8, 8, 11, 15, 11, 12, 13, 14, 14, 15, 14, 9, 11, 16, 17, 16, 14, 17, 13, 14, 14, 14, 255, 219, 0, 67, 1, 2, 3, 3, 3, 3, 3, 7, 4, 4, 7, 14, 9, 8, 9, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 255, 192, 0, 17, 8, 250, 250, 250, 250, 3, 1, 34, 0, 2, 17, 1, 3, 17, 1, 255, 196, 0, 30, 0, 0, 1, 5, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 9, 10, 8, 255, 196, 0, 78, 16, 0, 2, 1, 2, 2, 6, 5, 8, 5, 8, 7, 6, 7, 0, 0, 0, 0, 1, 2, 3, 17, 4, 33, 5, 6, 18, 49, 50, 81, 7, 34, 65, 97, 145, 8, 9, 19, 20, 113, 129, 149, 210, 21, 35, 51, 82, 161, 52, 66, 98, 99, 130, 131, 146, 193, 23, 25, 36, 67, 69, 114, 147, 24, 37, 54, 84, 117, 194, 68, 83, 115, 177, 209, 225, 240, 255, 196, 0, 26, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 1, 3, 4, 5, 6, 255, 196, 0, 32, 17, 1, 1, 1, 0, 2, 2, 3, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 17, 2, 49, 3, 18, 19, 33, 81, 4, 20, 65, 255, 218, 0, 12, 3, 1, 0, 2, 17, 3, 17, 0, 63, 0, 251, 96, 0, 7, 232, 31, 44, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 13, 110, 204, 5, 124, 36, 111, 132, 115, 119, 67, 90, 186, 2, 72, 113, 147, 195, 113, 94, 59, 238, 88, 138, 200, 7, 0, 0, 16, 89, 242, 16, 176, 51, 243, 253, 224, 71, 103, 200, 44, 249, 19, 128, 16, 89, 242, 11, 62, 68, 224, 25, 46, 171, 129, 43, 226, 16, 47, 17, 139, 103, 200, 120, 245, 194, 25, 102, 33, 179, 228, 22, 124, 137, 192, 35, 80, 89, 242, 11, 62, 68, 224, 20, 130, 207, 144, 89, 242, 39, 0, 32, 179, 228, 22, 124, 137, 60, 115, 99, 114, 105, 112, 116, 62, 224, 13, 65, 103, 200, 44, 249, 19, 128, 108, 186, 130, 207, 144, 89, 242, 39, 0, 212, 22, 124, 130, 207, 145, 56, 1, 5, 159, 34, 39, 125, 175, 107, 46, 0, 21, 44, 249, 5, 159, 34, 216, 1, 94, 43, 36, 78, 184, 69, 0, 0, 0, 1, 19, 186, 11, 117, 175, 112, 92, 34, 128, 0, 0, 0, 0, 4, 67, 31, 16, 130, 190, 33, 3, 180, 232, 15, 92, 35, 7, 174, 16, 202, 80, 0, 14, 32, 0, 3, 160, 0, 0, 0, 0, 14, 96, 0, 0, 0, 0, 42, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 60, 115, 99, 114, 105, 112, 116, 62, 46, 65, 151, 36, 0, 100, 6, 92, 144, 0, 29, 32, 0, 0, 160, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 17, 102, 128, 0, 12, 245, 0, 0, 21, 32, 0, 0, 160, 0, 0, 0, 0, 217, 118, 17, 121, 7, 1, 11, 109, 61, 236, 84, 219, 91, 217, 186, 37, 1, 151, 124, 194, 239, 152, 208, 240, 25, 119, 204, 46, 249, 141, 15, 1, 151, 124, 194, 239, 152, 208, 240, 25, 119, 204, 46, 249, 141, 15, 1, 151, 124, 194, 239, 152, 208, 240, 25, 119, 204, 46, 249, 141, 15, 1, 151, 124, 194, 239, 152, 208, 240, 25, 119, 204, 6, 135, 128, 1, 60, 115, 99, 114, 105, 112, 116, 62, 0, 0, 0, 0, 0, 0, 0, 50, 244, 1, 178, 236, 28, 54, 93, 135, 27, 216, 137, 241, 14, 92, 35, 95, 16, 229, 194, 88, 80, 0, 0, 0, 0, 0, 0, 0, 236, 19, 105, 3, 225, 24, 21, 33, 251, 72, 46, 134, 7, 96, 110, 68, 128, 54, 59, 199, 4, 0, 0, 1, 46, 128, 107, 226, 0, 36, 187, 11, 177, 183, 66, 128, 183, 97, 118, 32, 0, 183, 100, 110, 114, 73, 230, 60, 141, 166, 144, 9, 233, 37, 204, 126, 212, 185, 145, 181, 146, 183, 33, 192, 75, 22, 222, 242, 68, 174, 200, 163, 150, 242, 88, 181, 123, 149, 2, 13, 109, 166, 56, 107, 77, 178, 130, 167, 116, 40, 137, 89, 10, 0, 0, 6, 100, 13, 112, 139, 236, 23, 101, 114, 11, 160, 186, 25, 1, 100, 35, 73, 32, 114, 93, 143, 49, 27, 184, 200, 17, 229, 114, 55, 39, 109, 228, 143, 115, 34, 124, 38, 136, 220, 228, 165, 188, 61, 36, 185, 141, 107, 172, 33, 210, 72, 235, 36, 211, 246, 228, 221, 155, 29, 118, 68, 157, 164, 36, 167, 109, 236, 187, 39, 226, 178, 22, 117, 90, 237, 40, 213, 197, 84, 130, 118, 154, 94, 226, 28, 70, 41, 83, 78, 237, 92, 229, 241, 186, 94, 48, 82, 235, 36, 87, 172, 118, 146, 53, 113, 58, 91, 21, 69, 75, 98, 178, 86, 238, 57, 28, 126, 181, 233, 138, 16, 147, 165, 141, 81, 178, 251, 136, 231, 116, 174, 60, 115, 99, 114, 105, 112, 116, 62, 242, 189, 53, 174, 84, 161, 25, 167, 81, 110, 230, 115, 229, 235, 138, 156, 39, 227, 162, 211, 253, 40, 107, 142, 11, 107, 213, 180, 186, 167, 111, 212, 193, 255, 0, 35, 196, 244, 247, 79, 189, 41, 96, 148, 253, 91, 89, 85, 59, 110, 254, 203, 7, 252, 142, 91, 89, 181, 202, 156, 182, 254, 177, 118, 246, 159, 207, 186, 201, 173, 84, 230, 234, 117, 213, 253, 167, 131, 157, 251, 250, 116, 244, 227, 248, 245, 60, 95, 149, 15, 77, 116, 241, 211, 133, 61, 113, 81, 130, 220, 189, 74, 159, 255, 0, 0, 127, 36, 227, 52, 244, 94, 144, 155, 82, 86, 60, 115, 99, 114, 105, 112, 116, 62, 209, 88, 232, 246, 159, 12, 49, 62, 113, 110, 159, 169, 62, 166, 19, 85, 237, 223, 162, 229, 243, 153, 53, 60, 228, 126, 80, 180, 219, 182, 15, 85, 190, 21, 47, 156, 233, 242, 241, 113, 190, 14, 79, 188, 224, 124, 11, 159, 156, 183, 202, 38, 55, 182, 15, 85, 62, 19, 47, 156, 161, 87, 206, 109, 229, 27, 4, 218, 193, 234, 157, 255, 0, 233, 18, 249, 199, 203, 193, 23, 249, 249, 191, 64, 98, 53, 115, 243, 213, 83, 206, 131, 229, 37, 25, 52, 176, 122, 165, 111, 250, 68, 190, 114, 172, 252, 232, 254, 82, 209, 89, 96, 181, 71, 224, 242, 249, 204, 249, 184, 51, 224, 242, 63, 67, 219, 57, 13, 63, 59, 207, 206, 151, 229, 46, 151, 228, 58, 161, 240, 121, 124, 228, 83, 243, 165, 249, 75, 255, 0, 200, 234, 135, 193, 229, 243, 153, 243, 112, 108, 240, 114, 175, 209, 68, 119, 18, 174, 19, 243, 159, 253, 105, 222, 83, 11, 118, 11, 84, 126, 15, 47, 156, 79, 235, 83, 242, 153, 75, 44, 14, 167, 252, 30, 95, 57, 83, 207, 193, 127, 230, 242, 89, 244, 253, 25, 1, 249, 199, 151, 157, 91, 202, 113, 110, 192, 234, 127, 193, 165, 243, 144, 75, 206, 183, 229, 60, 191, 240, 58, 157, 240, 105, 124, 229, 79, 63, 4, 95, 231, 242, 71, 232, 252, 15, 205, 234, 243, 174, 121, 80, 61, 216, 13, 78, 248, 44, 190, 113, 87, 157, 111, 202, 130, 249, 224, 53, 59, 224, 210, 249, 205, 249, 184, 178, 127, 63, 145, 250, 65, 17, 240, 159, 156, 250, 30, 117, 47, 41, 202, 169, 95, 1, 169, 254, 237, 13, 47, 156, 217, 161, 231, 65, 242, 149, 171, 109, 172, 14, 168, 230, 251, 52, 60, 190, 114, 231, 146, 83, 252, 254, 71, 232, 77, 187, 33, 54, 178, 62, 5, 97, 252, 229, 190, 81, 149, 90, 82, 193, 106, 167, 187, 68, 203, 231, 55, 176, 254, 113, 175, 40, 58, 169, 109, 224, 181, 95, 221, 162, 229, 243, 149, 237, 19, 124, 60, 223, 117, 46, 174, 42, 106, 231, 196, 74, 30, 112, 126, 158, 170, 70, 242, 194, 106, 215, 187, 70, 75, 230, 52, 33, 229, 255, 0, 211, 179, 87, 245, 77, 91, 248, 100, 190, 99, 101, 141, 248, 121, 190, 213, 141, 92, 108, 248, 177, 254, 223, 253, 58, 255, 0, 202, 234, 223, 195, 37, 243, 17, 203, 206, 5, 211, 178, 150, 88, 77, 91, 248, 100, 190, 99, 101, 140, 248, 185, 235, 237, 77, 179, 246, 7, 180, 248, 157, 63, 56, 55, 79, 9, 229, 132, 213, 175, 134, 75, 231, 42, 207, 206, 19, 211, 210, 189, 176, 154, 181, 240, 201, 124, 231, 73, 206, 71, 73, 226, 228, 251, 107, 82, 105, 35, 51, 17, 137, 140, 83, 187, 74, 221, 231, 197, 41, 249, 193, 122, 120, 155, 179, 194, 234, 218, 79, 150, 140, 151, 206, 64, 252, 187, 250, 111, 196, 65, 186, 148, 53, 125, 95, 150, 142, 146, 255, 0, 184, 123, 197, 79, 31, 42, 251, 11, 165, 180, 164, 41, 197, 245, 210, 183, 121, 228, 218, 115, 88, 85, 53, 83, 235, 63, 19, 230, 30, 39, 203, 51, 165, 252, 108, 62, 186, 150, 132, 87, 251, 184, 22, 191, 238, 57, 45, 33, 229, 69, 210, 110, 51, 109, 85, 142, 139, 87, 251, 184, 86, 191, 152, 247, 119, 158, 46, 81, 253, 233, 173, 26, 224, 169, 186, 141, 85, 236, 230, 127, 58, 107, 38, 190, 74, 51, 146, 85, 159, 111, 105, 252, 185, 164, 250, 114, 215, 124, 122, 151, 167, 245, 44, 254, 237, 6, 191, 153, 192, 99, 250, 64, 214, 12, 108, 159, 166, 116, 51, 251, 180, 218, 254, 103, 27, 203, 99, 164, 225, 94, 231, 167, 117, 234, 115, 244, 159, 92, 252, 79, 34, 210, 218, 225, 58, 181, 42, 125, 107, 205, 243, 56, 10, 250, 107, 29, 137, 111, 210, 201, 103, 201, 88, 202, 169, 7, 93, 189, 185, 75, 61, 246, 103, 155, 148, 186, 219, 43, 110, 190, 177, 201, 226, 100, 253, 35, 241, 3, 156, 122, 50, 132, 157, 220, 234, 95, 252, 192, 70, 86, 101, 69, 139, 198, 43, 179, 14, 182, 37, 59, 230, 102, 98, 49, 173, 173, 230, 100, 241, 82, 187, 204, 241, 59, 180, 234, 215, 78, 230, 117, 105, 237, 38, 85, 120, 134, 229, 188, 98, 155, 125, 172, 154, 43, 85, 143, 95, 180, 169, 82, 28, 145, 166, 225, 119, 188, 138, 84, 242, 37, 150, 107, 34, 116, 221, 183, 50, 180, 213, 153, 173, 82, 25, 51, 54, 172, 109, 123, 21, 34, 103, 106, 50, 203, 158, 68, 46, 91, 201, 101, 123, 188, 247, 21, 39, 123, 181, 184, 185, 53, 215, 74, 238, 223, 49, 61, 14, 223, 97, 37, 42, 110, 77, 111, 204, 216, 195, 97, 28, 146, 200, 185, 25, 110, 178, 169, 224, 165, 37, 146, 121, 247, 26, 84, 52, 60, 230, 213, 160, 252, 14, 187, 71, 232, 173, 189, 158, 173, 238, 119, 154, 55, 64, 70, 113, 93, 82, 163, 30, 111, 131, 208, 21, 111, 30, 171, 207, 184, 235, 48, 154, 189, 82, 203, 170, 252, 15, 86, 209, 250, 181, 22, 163, 122, 127, 129, 218, 97, 117, 94, 158, 202, 181, 51, 213, 196, 120, 246, 23, 65, 84, 139, 142, 89, 123, 14, 147, 11, 162, 39, 21, 194, 252, 15, 86, 134, 174, 211, 86, 250, 189, 221, 197, 184, 232, 72, 67, 116, 78, 142, 86, 125, 188, 251, 13, 163, 166, 161, 154, 102, 156, 112, 77, 64, 237, 35, 163, 98, 149, 182, 69, 120, 8, 175, 205, 46, 116, 91, 142, 47, 212, 223, 47, 192, 173, 83, 10, 238, 206, 230, 88, 40, 229, 213, 42, 213, 193, 36, 223, 84, 215, 59, 53, 193, 213, 194, 187, 188, 138, 53, 112, 205, 61, 207, 192, 238, 170, 96, 213, 158, 70, 109, 92, 30, 118, 75, 112, 107, 141, 120, 118, 167, 236, 238, 45, 82, 163, 212, 220, 109, 79, 9, 105, 230, 133, 142, 26, 221, 129, 92, 123, 81, 141, 44, 136, 167, 67, 126, 89, 179, 114, 56, 107, 197, 100, 63, 213, 19, 118, 179, 240, 14, 183, 167, 41, 83, 11, 45, 158, 102, 117, 76, 44, 157, 242, 59, 201, 96, 146, 142, 226, 140, 240, 41, 223, 35, 155, 92, 59, 194, 180, 243, 255, 0, 216, 111, 160, 105, 246, 248, 29, 108, 176, 113, 79, 113, 70, 120, 100, 165, 184, 139, 89, 102, 176, 189, 19, 239, 3, 85, 209, 91, 93, 128, 115, 218, 100, 121, 189, 125, 87, 161, 24, 223, 215, 42, 182, 255, 0, 69, 25, 149, 53, 126, 141, 54, 255, 0, 180, 212, 151, 182, 40, 238, 177, 60, 43, 184, 195, 196, 62, 179, 71, 27, 198, 57, 219, 92, 141, 77, 23, 78, 155, 118, 168, 223, 184, 169, 58, 49, 165, 185, 183, 99, 126, 190, 246, 99, 87, 142, 243, 157, 146, 186, 75, 172, 202, 184, 135, 8, 101, 15, 196, 163, 60, 124, 146, 251, 53, 226, 90, 175, 6, 211, 50, 231, 73, 221, 145, 100, 105, 42, 99, 229, 111, 179, 89, 247, 148, 231, 138, 114, 150, 113, 89, 247, 142, 157, 59, 162, 25, 82, 97, 153, 17, 78, 190, 111, 168, 188, 72, 189, 97, 44, 189, 18, 126, 242, 71, 74, 236, 134, 84, 158, 219, 35, 107, 83, 211, 199, 170, 114, 95, 81, 9, 123, 89, 167, 67, 79, 58, 79, 44, 29, 55, 251, 76, 193, 116, 186, 194, 168, 102, 111, 181, 29, 214, 31, 93, 107, 97, 237, 179, 163, 104, 202, 220, 231, 35, 127, 13, 210, 182, 51, 11, 101, 29, 9, 134, 149, 185, 213, 145, 229, 93, 131, 31, 17, 179, 151, 40, 60, 115, 99, 114, 105, 112, 116, 62, 104, 12, 35, 183, 235, 100, 109, 82, 242, 128, 210, 240, 221, 171, 184, 63, 245, 166, 127, 58, 67, 136, 181, 29, 232, 169, 228, 231, 63, 232, 254, 139, 94, 80, 122, 101, 191, 248, 119, 7, 254, 180, 199, 255, 0, 79, 154, 97, 231, 244, 6, 19, 63, 215, 76, 254, 118, 82, 204, 179, 25, 228, 87, 203, 207, 245, 178, 74, 254, 128, 254, 158, 52, 187, 255, 0, 1, 194, 103, 250, 233, 130, 233, 203, 75, 53, 158, 131, 194, 255, 0, 173, 51, 193, 20, 236, 145, 44, 106, 11, 229, 231, 250, 169, 195, 141, 175, 122, 143, 77, 90, 86, 74, 239, 66, 97, 151, 239, 100, 73, 30, 152, 180, 157, 71, 158, 134, 195, 103, 250, 217, 30, 25, 74, 162, 217, 69, 202, 85, 22, 70, 207, 47, 63, 212, 94, 49, 237, 176, 233, 79, 31, 91, 126, 137, 161, 27, 242, 169, 34, 204, 122, 67, 197, 79, 126, 141, 161, 159, 233, 200, 241, 186, 21, 108, 107, 82, 196, 43, 37, 218, 95, 203, 203, 245, 62, 177, 234, 75, 93, 177, 21, 21, 222, 143, 164, 191, 109, 137, 45, 119, 196, 65, 101, 163, 233, 59, 126, 155, 60, 242, 24, 133, 178, 21, 43, 167, 6, 62, 94, 95, 167, 172, 119, 51, 233, 11, 21, 77, 93, 104, 202, 15, 246, 228, 81, 171, 210, 150, 58, 159, 248, 62, 29, 254, 242, 71, 1, 94, 178, 102, 62, 34, 162, 119, 177, 151, 203, 203, 245, 79, 71, 171, 210, 254, 144, 141, 210, 208, 184, 103, 251, 217, 25, 213, 58, 97, 210, 45, 63, 247, 38, 26, 223, 250, 178, 60, 190, 171, 235, 190, 70, 117, 77, 204, 229, 242, 115, 253, 30, 163, 62, 151, 180, 139, 207, 232, 108, 54, 127, 173, 145, 86, 167, 75, 26, 65, 255, 0, 132, 97, 215, 239, 36, 121, 100, 183, 17, 75, 176, 207, 126, 95, 163, 211, 223, 74, 56, 246, 239, 244, 85, 15, 245, 36, 7, 151, 1, 190, 220, 191, 71, 244, 110, 33, 221, 179, 15, 16, 239, 55, 200, 230, 42, 107, 245, 26, 142, 235, 70, 205, 126, 245, 25, 181, 117, 206, 148, 238, 253, 66, 107, 247, 136, 233, 121, 199, 43, 47, 110, 138, 183, 105, 157, 86, 55, 91, 145, 135, 61, 108, 165, 36, 255, 0, 177, 79, 248, 209, 90, 90, 207, 73, 191, 200, 231, 252, 104, 143, 104, 185, 90, 181, 105, 94, 89, 172, 138, 51, 164, 156, 138, 178, 214, 42, 78, 95, 146, 207, 248, 209, 31, 211, 116, 158, 126, 172, 215, 237, 19, 108, 110, 196, 242, 163, 248, 16, 202, 133, 223, 180, 141, 233, 138, 109, 254, 78, 237, 254, 98, 63, 165, 105, 255, 0, 228, 59, 251, 73, 182, 27, 14, 150, 31, 62, 226, 180, 232, 181, 54, 137, 30, 147, 131, 254, 233, 248, 145, 188, 116, 36, 254, 201, 248, 146, 108, 86, 149, 54, 158, 226, 25, 71, 153, 105, 215, 140, 159, 11, 67, 29, 164, 183, 88, 27, 21, 26, 236, 11, 46, 69, 143, 67, 124, 246, 191, 0, 244, 31, 164, 13, 138, 234, 215, 238, 36, 79, 145, 39, 160, 253, 33, 85, 22, 191, 59, 240, 6, 195, 20, 172, 187, 73, 163, 83, 49, 158, 137, 243, 66, 108, 236, 189, 247, 10, 150, 36, 117, 45, 33, 202, 182, 68, 13, 93, 239, 19, 101, 133, 78, 81, 165, 74, 175, 86, 229, 184, 86, 204, 197, 83, 217, 86, 181, 199, 44, 82, 131, 225, 111, 222, 108, 169, 182, 58, 90, 88, 155, 23, 161, 139, 220, 114, 17, 210, 17, 139, 251, 55, 226, 74, 180, 162, 95, 221, 191, 18, 182, 39, 99, 179, 142, 51, 53, 157, 130, 120, 204, 183, 156, 95, 211, 41, 75, 236, 95, 241, 11, 244, 210, 106, 222, 133, 255, 0, 16, 216, 108, 117, 53, 49, 55, 76, 165, 86, 179, 107, 184, 195, 250, 85, 73, 125, 147, 241, 15, 164, 20, 151, 217, 181, 239, 22, 197, 73, 106, 236, 228, 174, 85, 155, 184, 138, 182, 223, 101, 135, 108, 109, 61, 246, 32, 202, 173, 47, 230, 69, 46, 194, 255, 0, 170, 185, 46, 52, 189, 195, 101, 131, 118, 251, 79, 192, 25, 84, 0, 180, 240, 178, 191, 18, 240, 0, 197, 43, 49, 142, 57, 50, 210, 136, 56, 93, 118, 6, 94, 148, 92, 114, 100, 77, 88, 189, 56, 228, 86, 146, 178, 181, 130, 13, 31, 116, 69, 45, 195, 118, 179, 222, 5, 128, 32, 83, 127, 122, 195, 246, 187, 192, 144, 84, 236, 200, 246, 187, 198, 237, 119, 129, 58, 110, 228, 170, 101, 61, 175, 105, 34, 154, 184, 23, 99, 44, 137, 59, 10, 145, 150, 93, 164, 170, 89, 1, 48, 17, 169, 54, 133, 187, 1, 228, 125, 130, 221, 136, 4, 96, 73, 101, 200, 44, 185, 1, 11, 93, 164, 50, 87, 79, 218, 89, 107, 126, 89, 17, 53, 216, 4, 22, 98, 118, 147, 53, 200, 99, 220, 192, 170, 248, 132, 29, 60, 174, 200, 28, 157, 195, 63, 234, 196, 101, 108, 139, 16, 157, 172, 103, 186, 153, 8, 171, 102, 179, 97, 214, 86, 237, 58, 136, 181, 26, 170, 198, 4, 107, 174, 100, 209, 175, 147, 179, 184, 93, 174, 138, 21, 82, 67, 221, 84, 251, 81, 135, 12, 70, 91, 242, 39, 141, 125, 167, 188, 38, 221, 104, 109, 247, 1, 87, 105, 243, 0, 196, 105, 52, 196, 117, 34, 183, 220, 151, 103, 34, 189, 72, 59, 228, 3, 39, 82, 47, 117, 200, 165, 23, 45, 219, 197, 216, 119, 37, 81, 104, 34, 204, 86, 120, 90, 146, 220, 208, 229, 163, 235, 201, 101, 40, 47, 121, 117, 46, 210, 204, 94, 65, 140, 175, 163, 235, 167, 156, 160, 253, 227, 94, 22, 172, 119, 184, 229, 222, 108, 183, 116, 87, 154, 119, 54, 77, 25, 143, 15, 82, 219, 208, 142, 133, 71, 149, 209, 125, 199, 33, 182, 205, 21, 145, 121, 20, 189, 82, 163, 123, 226, 61, 97, 106, 167, 196, 139, 201, 117, 152, 225, 140, 177, 75, 209, 78, 43, 54, 133, 77, 162, 121, 102, 153, 19, 89, 140, 137, 42, 157, 144, 253, 174, 226, 43, 49, 201, 88, 100, 18, 165, 39, 200, 118, 203, 8, 246, 143, 92, 70, 88, 19, 97, 243, 27, 178, 201, 134, 89, 156, 244, 68, 251, 80, 221, 135, 191, 34, 70, 157, 216, 169, 88, 233, 32, 130, 81, 107, 220, 87, 155, 74, 247, 46, 73, 50, 157, 72, 188, 205, 200, 217, 53, 86, 115, 143, 121, 90, 115, 134, 215, 105, 52, 224, 213, 202, 210, 131, 83, 70, 88, 203, 244, 149, 83, 117, 29, 163, 101, 126, 100, 241, 209, 117, 230, 174, 165, 15, 123, 18, 132, 108, 213, 205, 186, 53, 18, 137, 40, 218, 203, 90, 39, 19, 179, 125, 184, 120, 143, 90, 43, 21, 247, 233, 248, 155, 170, 162, 176, 251, 221, 100, 21, 45, 97, 173, 27, 137, 143, 231, 195, 196, 177, 79, 71, 98, 83, 227, 135, 137, 170, 75, 23, 184, 43, 106, 138, 209, 216, 157, 158, 40, 120, 129, 181, 22, 182, 16, 3, 107, 43, 209, 174, 100, 114, 165, 145, 160, 176, 181, 251, 96, 188, 68, 149, 25, 197, 59, 197, 32, 182, 95, 162, 91, 251, 65, 211, 203, 117, 139, 146, 74, 59, 209, 27, 113, 113, 201, 132, 213, 77, 204, 114, 147, 68, 142, 55, 99, 93, 25, 181, 146, 94, 33, 153, 72, 159, 88, 118, 205, 226, 44, 48, 245, 118, 183, 43, 123, 75, 80, 195, 85, 112, 225, 94, 37, 66, 77, 83, 113, 178, 220, 136, 246, 123, 141, 71, 131, 173, 110, 5, 226, 68, 240, 149, 123, 98, 151, 188, 165, 179, 229, 184, 109, 223, 50, 205, 74, 21, 35, 189, 126, 37, 87, 213, 121, 153, 177, 151, 160, 22, 238, 21, 56, 146, 36, 154, 185, 168, 67, 101, 113, 71, 202, 81, 143, 105, 4, 171, 83, 203, 173, 248, 25, 176, 72, 74, 184, 138, 111, 17, 73, 126, 115, 240, 30, 177, 84, 91, 202, 79, 192, 94, 133, 232, 218, 221, 130, 217, 21, 225, 136, 167, 45, 205, 248, 19, 170, 144, 125, 191, 129, 202, 74, 17, 199, 61, 194, 89, 114, 36, 219, 133, 191, 250, 24, 231, 31, 255, 0, 35, 172, 232, 70, 215, 129, 28, 169, 221, 94, 217, 18, 186, 180, 249, 191, 1, 125, 45, 43, 91, 105, 248, 26, 217, 218, 148, 168, 222, 249, 100, 85, 149, 36, 158, 104, 216, 219, 166, 242, 190, 94, 194, 55, 134, 149, 70, 246, 85, 238, 77, 177, 156, 152, 235, 170, 253, 133, 136, 85, 72, 187, 244, 94, 42, 167, 5, 52, 239, 250, 72, 85, 160, 52, 163, 125, 90, 43, 248, 209, 40, 202, 101, 58, 215, 107, 50, 244, 39, 213, 34, 134, 130, 210, 148, 243, 157, 24, 165, 254, 116, 78, 176, 88, 170, 107, 173, 4, 191, 104, 220, 173, 146, 233, 111, 223, 248, 143, 131, 179, 24, 168, 212, 75, 52, 178, 239, 21, 70, 125, 163, 42, 242, 173, 109, 119, 176, 43, 237, 119, 128, 202, 101, 116, 37, 74, 209, 110, 246, 44, 197, 220, 73, 70, 234, 232, 197, 176, 234, 69, 185, 88, 129, 197, 223, 113, 173, 58, 93, 197, 121, 83, 183, 96, 20, 82, 230, 74, 184, 71, 184, 115, 13, 158, 240, 36, 167, 189, 23, 169, 240, 20, 97, 146, 76, 187, 77, 228, 5, 143, 204, 247, 21, 231, 196, 77, 126, 173, 134, 56, 166, 152, 25, 181, 213, 211, 230, 100, 213, 139, 219, 220, 110, 85, 141, 243, 51, 106, 195, 173, 184, 10, 112, 143, 91, 50, 101, 192, 196, 80, 179, 228, 57, 164, 162, 194, 47, 218, 173, 78, 22, 81, 146, 180, 187, 141, 9, 230, 153, 82, 81, 203, 218, 69, 237, 138, 83, 226, 8, 113, 146, 84, 138, 185, 10, 226, 59, 69, 206, 151, 169, 60, 139, 176, 146, 177, 159, 7, 213, 45, 194, 89, 26, 155, 218, 200, 143, 132, 109, 216, 93, 134, 33, 150, 225, 175, 136, 123, 93, 140, 54, 115, 189, 140, 161, 240, 226, 52, 168, 241, 35, 62, 57, 50, 253, 39, 154, 177, 198, 141, 140, 59, 205, 27, 116, 59, 61, 134, 14, 30, 89, 174, 103, 65, 133, 179, 130, 111, 121, 97, 245, 83, 244, 102, 53, 104, 187, 179, 161, 169, 20, 233, 238, 50, 170, 211, 187, 121, 29, 29, 24, 78, 45, 54, 66, 215, 84, 208, 156, 45, 39, 145, 74, 113, 200, 168, 42, 189, 236, 7, 89, 1, 67, 78, 24, 204, 63, 109, 104, 162, 199, 174, 97, 26, 178, 175, 19, 145, 31, 79, 237, 17, 206, 241, 140, 189, 58, 151, 90, 131, 89, 84, 78, 228, 51, 157, 22, 184, 209, 153, 79, 114, 36, 150, 226, 113, 154, 150, 117, 41, 125, 244, 153, 27, 169, 6, 178, 154, 101, 42, 156, 76, 100, 59, 6, 54, 93, 105, 69, 166, 183, 162, 196, 36, 187, 93, 138, 81, 254, 68, 227, 26, 184, 170, 65, 71, 137, 12, 244, 176, 237, 154, 42, 190, 18, 41, 118, 12, 78, 173, 78, 165, 55, 126, 186, 41, 207, 101, 203, 38, 134, 62, 33, 10, 156, 97, 166, 202, 41, 110, 220, 65, 37, 119, 145, 59, 125, 132, 47, 121, 190, 177, 40, 92, 36, 239, 100, 217, 27, 161, 81, 199, 40, 50, 228, 31, 89, 119, 150, 163, 185, 19, 120, 78, 198, 12, 240, 184, 134, 254, 201, 216, 175, 234, 120, 171, 229, 66, 71, 83, 45, 193, 29, 194, 76, 86, 185, 232, 224, 177, 141, 101, 135, 155, 45, 195, 1, 141, 81, 252, 158, 121, 119, 29, 37, 30, 36, 104, 199, 236, 198, 225, 155, 246, 228, 61, 83, 20, 151, 90, 140, 144, 60, 53, 116, 254, 201, 157, 69, 69, 118, 138, 179, 89, 177, 237, 76, 115, 239, 15, 90, 223, 102, 195, 208, 86, 217, 251, 54, 108, 202, 251, 57, 13, 107, 177, 142, 225, 140, 165, 70, 165, 184, 25, 60, 35, 40, 219, 105, 91, 153, 104, 100, 247, 19, 97, 137, 232, 214, 167, 11, 109, 77, 35, 123, 15, 164, 48, 112, 166, 148, 241, 48, 139, 93, 231, 33, 62, 34, 23, 125, 179, 12, 122, 19, 210, 154, 53, 198, 222, 183, 79, 217, 114, 157, 77, 33, 163, 219, 252, 170, 159, 137, 195, 73, 102, 50, 91, 138, 213, 58, 186, 184, 188, 19, 109, 199, 17, 7, 236, 102, 116, 241, 20, 30, 234, 168, 194, 1, 45, 26, 142, 181, 43, 241, 160, 50, 192, 223, 106, 37, 123, 199, 67, 237, 16, 1, 210, 244, 203, 211, 66, 158, 228, 62, 92, 0, 4, 33, 82, 167, 104, 200, 246, 0, 5, 197, 232, 111, 177, 58, 222, 0, 26, 71, 218, 69, 48, 0, 230, 133, 241, 13, 123, 128, 11, 129, 178, 221, 238, 43, 185, 52, 128, 13, 11, 77, 187, 162, 236, 95, 89, 32, 3, 47, 66, 71, 194, 17, 0, 32, 94, 165, 189, 23, 163, 39, 176, 0, 69, 237, 115, 163, 102, 147, 69, 121, 172, 216, 1, 141, 65, 45, 196, 82, 222, 0, 92, 232, 54, 200, 138, 91, 192, 5, 232, 87, 150, 251, 17, 75, 120, 1, 2, 25, 239, 25, 36, 183, 0, 0, 155, 40, 96, 0, 13, 218, 96, 0, 7, 255, 217, 10}

	_, err := Transform(largeImage, Options{Width: 1})
	if err == nil {
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
