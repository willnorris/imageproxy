package imageproxy

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

// syntheticProfile builds an ICC blob of n bytes whose header declares the
// given data color space.
func syntheticProfile(n int, space string) []byte {
	p := make([]byte, n)
	if n >= 20 {
		copy(p[16:20], space)
	}
	for i := 20; i < n; i++ {
		p[i] = byte(i % 251)
	}
	return p
}

func TestICCRoundTrip(t *testing.T) {
	// a minimal JPEG skeleton: SOI, a DQT-ish segment, SOS, EOI
	base := []byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x04, 0xAA, 0xBB, 0xFF, 0xDA, 0x00, 0x02, 0xFF, 0xD9}

	for _, size := range []int{1, iccHeaderSize, maxICCChunk, maxICCChunk + 1, maxICCProfileSize} {
		profile := syntheticProfile(size, "RGB ")
		embedded := embedICCProfile(base, profile)
		got := extractICCProfile(embedded)
		if !bytes.Equal(got, profile) {
			t.Errorf("size %d: round trip lost the profile (got %d bytes, want %d)", size, len(got), len(profile))
		}
		if !bytes.HasSuffix(embedded, base[2:]) {
			t.Errorf("size %d: original segments were not preserved", size)
		}
	}
}

func TestExtractICCProfileRejectsMalformed(t *testing.T) {
	tests := map[string][]byte{
		"not a jpeg":      {0x00, 0x01, 0x02},
		"truncated":       {0xFF, 0xD8, 0xFF, 0xE2, 0x00},
		"length past end": {0xFF, 0xD8, 0xFF, 0xE2, 0xFF, 0xFF, 0x01},
		"no icc segment":  {0xFF, 0xD8, 0xFF, 0xDA, 0x00, 0x02, 0xFF, 0xD9},
	}
	for name, img := range tests {
		if got := extractICCProfile(img); got != nil {
			t.Errorf("%s: expected no profile, got %d bytes", name, len(got))
		}
	}
}

func TestExtractICCProfileRejectsIncompleteChunkSet(t *testing.T) {
	base := []byte{0xFF, 0xD8, 0xFF, 0xDA, 0x00, 0x02, 0xFF, 0xD9}
	full := embedICCProfile(base, syntheticProfile(maxICCChunk+1, "RGB "))

	// drop the first APP2 segment, leaving chunk 2 claiming a set of 2
	partial := append([]byte{0xFF, 0xD8}, full[2+4+iccChunkHeaderLen+maxICCChunk:]...)
	if got := extractICCProfile(partial); got != nil {
		t.Errorf("expected no profile from an incomplete chunk set, got %d bytes", len(got))
	}
}

func TestExtractICCProfileRejectsOversizedProfile(t *testing.T) {
	base := []byte{0xFF, 0xD8, 0xFF, 0xDA, 0x00, 0x02, 0xFF, 0xD9}

	if got := extractICCProfile(embedICCProfile(base, syntheticProfile(maxICCProfileSize+1, "RGB "))); got != nil {
		t.Errorf("expected an oversized profile to be refused before reassembly, got %d bytes", len(got))
	}
	if got := extractICCProfile(embedICCProfile(base, syntheticProfile(maxICCProfileSize, "RGB "))); len(got) != maxICCProfileSize {
		t.Errorf("a profile exactly at the cap should survive, got %d bytes", len(got))
	}
}

func TestPreservesSourceColorSpace(t *testing.T) {
	rgb := image.NewRGBA(image.Rect(0, 0, 1, 1))
	tests := []struct {
		name string
		m    image.Image
		p    []byte
		want bool
	}{
		{"rgb profile on rgb pixels", rgb, syntheticProfile(512, "RGB "), true},
		{"cmyk profile", rgb, syntheticProfile(512, "CMYK"), false},
		{"gray source", image.NewGray(image.Rect(0, 0, 1, 1)), syntheticProfile(512, "RGB "), false},
		{"cmyk source", image.NewCMYK(image.Rect(0, 0, 1, 1)), syntheticProfile(512, "RGB "), false},
		{"oversized profile", rgb, syntheticProfile(maxICCProfileSize+1, "RGB "), false},
		{"short of a full icc header", rgb, syntheticProfile(iccHeaderSize-1, "RGB "), false},
		{"exactly one icc header", rgb, syntheticProfile(iccHeaderSize, "RGB "), true},
		{"no profile", rgb, nil, false},
		{"runt profile", rgb, []byte{1, 2, 3}, false},
	}
	for _, tt := range tests {
		if got := preservesSourceColorSpace(tt.m, tt.p); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTransformPassthroughICC(t *testing.T) {
	profile := syntheticProfile(256, "RGB ")

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 32, 32)), nil); err != nil {
		t.Fatal(err)
	}
	img := embedICCProfile(buf.Bytes(), profile)
	if got := extractICCProfile(img); !bytes.Equal(got, profile) {
		t.Fatalf("fixture lost its profile before Transform ran")
	}

	off, err := Transform(img, Options{Width: 16})
	if err != nil {
		t.Fatal(err)
	}
	if got := extractICCProfile(off); got != nil {
		t.Errorf("PassthroughICC off: expected no profile, got %d bytes", len(got))
	}

	on, err := Transform(img, Options{Width: 16, PassthroughICC: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := extractICCProfile(on); !bytes.Equal(got, profile) {
		t.Errorf("PassthroughICC on: got %d profile bytes, want %d", len(got), len(profile))
	}
}
