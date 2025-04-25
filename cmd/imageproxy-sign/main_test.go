// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/url"
	"os"
	"reflect"
	"testing"
)

var key = "secret"

func TestSign(t *testing.T) {
	s := "http://example.com/image.jpg#0x0"

	got, err := sign(key, s, false)
	if err != nil {
		t.Errorf("sign(%q, %q, false) returned error: %v", key, s, err)
	}
	want := []byte{0xc3, 0x4c, 0x45, 0xb5, 0x75, 0x84, 0x76, 0xdf, 0xd9, 0x6b, 0x12, 0xa4, 0x84, 0x8f, 0x37, 0xc6, 0x2d, 0x8b, 0x8d, 0x77, 0xda, 0x6, 0xf8, 0xb5, 0x10, 0xc9, 0x96, 0x3c, 0x6e, 0x13, 0xda, 0xf0}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sign(%q, %q, true) returned %v, want %v", key, s, got, want)
	}
}

func TestSign_URLOnly(t *testing.T) {
	s := "http://example.com/image.jpg#0x0"

	got, err := sign(key, s, true)
	if err != nil {
		t.Errorf("sign(%q, %q, true) returned error: %v", key, s, err)
	}
	want := []byte{0x93, 0xea, 0x5d, 0x23, 0x68, 0xa0, 0xfc, 0x50, 0x8e, 0x91, 0x7, 0xbf, 0x3e, 0xb3, 0x1f, 0x49, 0xf7, 0x1d, 0x81, 0xf1, 0x74, 0xfe, 0x25, 0x36, 0xfc, 0x74, 0xf8, 0x81, 0x15, 0xf5, 0x58, 0x40}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sign(%q, %q, true) returned %v, want %v", key, s, got, want)
	}
}

func TestSign_Errors(t *testing.T) {
	var err error

	tests := []struct {
		key, url string
	}{
		{"", ""},
		{"", "%"},
		{"@/does/not/exist", "s"},
	}

	for _, tt := range tests {
		_, err = sign(tt.key, tt.url, false)
		if err == nil {
			t.Errorf("sign(%q, %q, false) did not return expected error", tt.key, tt.url)
		}
	}
}

func TestParseKey(t *testing.T) {
	k, err := parseKey(key)
	got := string(k)
	if err != nil {
		t.Errorf("parseKey(%q) returned error: %v", key, err)
	}
	if want := key; got != want {
		t.Errorf("parseKey(%q) returned %v, want %v", key, got, want)
	}
}

func TestParseKey_FilePath(t *testing.T) {
	f, err := os.CreateTemp("", "key")
	if err != nil {
		t.Errorf("error creating temp file: %v", err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	if _, err := f.WriteString(key); err != nil {
		t.Errorf("error writing to temp file: %v", err)
	}
	path := "@" + f.Name()
	k, err := parseKey(path)
	got := string(k)
	if err != nil {
		t.Errorf("parseKey(%q) returned error: %v", path, err)
	}
	if want := key; got != want {
		t.Errorf("parseKey(%q) returned %v, want %v", path, got, want)
	}
}

func TestParseURL(t *testing.T) {
	tests := []struct {
		input, output string
	}{
		{"/", "/#0x0"},

		// imageproxy URLs
		{"http://localhost:8080//http://example.com/", "http://example.com/#0x0"},
		{"http://localhost:8080/10,r90,jpeg/http://example.com/", "http://example.com/#10x10,jpeg,r90"},

		// remote URLs, with and without options
		{"http://example.com/", "http://example.com/#0x0"},
		{"http://example.com/#r90,jpeg", "http://example.com/#0x0,jpeg,r90"},

		// ensure signature values are stripped
		{"http://localhost:8080/sc0ffee/http://example.com/", "http://example.com/#0x0"},
		{"http://example.com/#sc0ffee", "http://example.com/#0x0"},
	}

	for _, tt := range tests {
		want, _ := url.Parse(tt.output)
		got := parseURL(tt.input)
		if got.String() != want.String() {
			t.Errorf("parseURL(%q) returned %q, want %q", tt.input, got, want)
		}
	}
}
