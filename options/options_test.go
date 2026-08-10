// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package options_test

import (
	"testing"
	"time"

	_ "willnorris.com/go/imageproxy/image/jpeg"
	_ "willnorris.com/go/imageproxy/image/png"
	"willnorris.com/go/imageproxy/options"
)

type Options = options.Options

var emptyOptions = Options{}

func TestOptions_String(t *testing.T) {
	tests := []struct {
		Options Options
		String  string
	}{
		{
			emptyOptions,
			"0x0",
		},
		{
			Options{Width: 1, Height: 2, Fit: true, Rotate: 90, FlipVertical: true, FlipHorizontal: true, Quality: 80},
			"1x2,fh,fit,fv,q80,r90",
		},
		{
			Options{Width: 0.15, Height: 1.3, Rotate: 45, Quality: 95, Signature: "c0ffee", Format: "png", ValidUntil: time.Unix(123, 0)},
			"0.15x1.3,png,q95,r45,sc0ffee,vu123",
		},
		{
			Options{Width: 0.15, Height: 1.3, CropX: 100, CropY: 200},
			"0.15x1.3,cx100,cy200",
		},
		{
			Options{ScaleUp: true, CropX: 100, CropY: 200, CropWidth: 300, CropHeight: 400, SmartCrop: true},
			"0x0,ch400,cw300,cx100,cy200,sc,scaleUp",
		},
	}

	for i, tt := range tests {
		if got, want := tt.Options.String(), tt.String; got != want {
			t.Errorf("%d. Options.String returned %v, want %v", i, got, want)
		}
	}
}

func TestParseOptions(t *testing.T) {
	tests := []struct {
		Input   string
		Options Options
	}{
		{"", emptyOptions},
		{"x", emptyOptions},
		{"r", emptyOptions},
		{"0", emptyOptions},
		{",,,,", emptyOptions},

		// size variations
		{"1x", Options{Width: 1}},
		{"x1", Options{Height: 1}},
		{"1x2", Options{Width: 1, Height: 2}},
		{"-1x-2", Options{Width: -1, Height: -2}},
		{"0.1x0.2", Options{Width: 0.1, Height: 0.2}},
		{"1", Options{Width: 1, Height: 1}},
		{"0.1", Options{Width: 0.1, Height: 0.1}},

		// additional flags
		{"fit", Options{Fit: true}},
		{"r90", Options{Rotate: 90}},
		{"fv", Options{FlipVertical: true}},
		{"fh", Options{FlipHorizontal: true}},
		{"jpeg", Options{Format: "jpeg"}},

		// duplicate flags (last one wins)
		{"1x2,3x4", Options{Width: 3, Height: 4}},
		{"1x2,3", Options{Width: 3, Height: 3}},
		{"1x2,0x3", Options{Width: 0, Height: 3}},
		{"1x,x2", Options{Width: 1, Height: 2}},
		{"r90,r270", Options{Rotate: 270}},
		{"jpeg,png", Options{Format: "png"}},

		// mix of valid and invalid flags
		{"FOO,1,BAR,r90,BAZ", Options{Width: 1, Height: 1, Rotate: 90}},

		// flags, in different orders
		{"q70,1x2,fit,r90,fv,fh,sc0ffee,png", Options{Width: 1, Height: 2, Fit: true, Rotate: 90, FlipVertical: true, FlipHorizontal: true, Quality: 70, Signature: "c0ffee", Format: "png"}},
		{"r90,fh,sc0ffee,png,q90,1x2,fv,fit", Options{Width: 1, Height: 2, Fit: true, Rotate: 90, FlipVertical: true, FlipHorizontal: true, Quality: 90, Signature: "c0ffee", Format: "png"}},
		{"cx100,cw300,1x2,cy200,ch400,sc,scaleUp,vu1234567890", Options{Width: 1, Height: 2, ScaleUp: true, CropX: 100, CropY: 200, CropWidth: 300, CropHeight: 400, SmartCrop: true, ValidUntil: time.Unix(1234567890, 0)}},
	}

	for _, tt := range tests {
		if got, want := options.ParseOptions(tt.Input), tt.Options; got != want {
			t.Errorf("ParseOptions(%q) returned %#v, want %#v", tt.Input, got, want)
		}
	}
}
