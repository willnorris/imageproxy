// Copyright 2013 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"reflect"
	"testing"
)

func TestOptions_String(t *testing.T) {
	tests := []struct {
		Options *Options
		String  string
	}{
		{emptyOptions, "0x0"},
		{
			&Options{1, 2, true, 90, true, true},
			"1x2,fit,r90,fv,fh",
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
		Options *Options
	}{
		{"", emptyOptions},
		{"x", emptyOptions},
		{"0", emptyOptions},

		// size variations
		{"1x", &Options{Width: 1}},
		{"x1", &Options{Height: 1}},
		{"1x2", &Options{Width: 1, Height: 2}},
		{"0.1x0.2", &Options{Width: 0.1, Height: 0.2}},

		// additional flags
		{",fit", &Options{Fit: true}},
		{",r90", &Options{Rotate: 90}},
		{",fv", &Options{FlipVertical: true}},
		{",fh", &Options{FlipHorizontal: true}},

		{"1x2,fit,r90,fv,fh", &Options{1, 2, true, 90, true, true}},
	}

	for i, tt := range tests {
		if got, want := ParseOptions(tt.Input), tt.Options; !reflect.DeepEqual(got, want) {
			t.Errorf("%d. ParseOptions returned %#v, want %#v", i, got, want)
		}
	}
}
