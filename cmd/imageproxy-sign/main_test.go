package main

import (
	"net/url"
	"testing"
)

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
