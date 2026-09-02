// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"bytes"
	"encoding/binary"
	"image"
)

// ICC_PROFILE marker segments are identified by this null-terminated string,
// followed by a 1-based chunk number and the total chunk count (ITU-T T.872 B.4).
const iccIdentifier = "ICC_PROFILE\x00"

const (
	iccChunkHeaderLen = len(iccIdentifier) + 2 // identifier + seq + count
	// A JPEG marker segment payload is limited by its 16-bit length field,
	// which counts itself.
	maxSegmentPayload = 0xFFFF - 2
	maxICCChunk       = maxSegmentPayload - iccChunkHeaderLen

	// Profiles larger than this are dropped rather than attached: a 500 KB
	// LUT profile on an 80px thumbnail costs far more than the color it
	// preserves.
	maxICCProfileSize = 64 << 10

	// Length of the ICC profile header (ICC.1:2010 section 7.2).
	iccHeaderSize = 128
)

// extractICCProfile returns the ICC profile embedded in the APP2 segments of
// the JPEG in img, or nil if it carries none. Chunks are reassembled in the
// order declared by their sequence numbers.
func extractICCProfile(img []byte) []byte {
	chunks := map[byte][]byte{}
	var count byte

	forEachJPEGSegment(img, func(marker byte, payload []byte) {
		if marker != 0xE2 || len(payload) <= iccChunkHeaderLen {
			return
		}
		if !bytes.HasPrefix(payload, []byte(iccIdentifier)) {
			return
		}
		seq := payload[len(iccIdentifier)]
		total := payload[len(iccIdentifier)+1]
		if seq == 0 || total == 0 || seq > total {
			return
		}
		if count == 0 {
			count = total
		} else if total != count {
			return // inconsistent chunk set; refuse to guess
		}
		chunks[seq] = payload[iccChunkHeaderLen:]
	})

	if count == 0 || len(chunks) != int(count) {
		return nil
	}

	// Chunks are still sub-slices of img at this point. A crafted source can
	// declare 255 of them, so measure before allocating rather than build a
	// ~16 MB profile for preservesSourceColorSpace to throw away.
	size := 0
	for _, c := range chunks {
		size += len(c)
	}
	if size > maxICCProfileSize {
		return nil
	}

	profile := make([]byte, 0, size)
	for i := byte(1); i <= count; i++ {
		profile = append(profile, chunks[i]...)
	}
	return profile
}

// embedICCProfile returns the JPEG in img with profile written into APP2
// segments immediately after the SOI marker. The rest of img is carried over
// untouched, so the caller must only pass a JPEG that has no ICC segments of
// its own — two chunk sets would leave the profile unreadable. Output from
// Go's jpeg encoder, which writes no APP segments at all, satisfies that.
func embedICCProfile(img, profile []byte) []byte {
	if len(profile) == 0 || len(img) < 2 {
		return img
	}

	chunkCount := (len(profile) + maxICCChunk - 1) / maxICCChunk
	if chunkCount > 255 {
		return img
	}

	out := make([]byte, 0, len(img)+len(profile)+chunkCount*(iccChunkHeaderLen+4))
	out = append(out, img[:2]...) // SOI

	for i := 0; i < chunkCount; i++ {
		chunk := profile[i*maxICCChunk:]
		if len(chunk) > maxICCChunk {
			chunk = chunk[:maxICCChunk]
		}
		out = append(out, 0xFF, 0xE2)
		out = binary.BigEndian.AppendUint16(out, uint16(2+iccChunkHeaderLen+len(chunk)))
		out = append(out, iccIdentifier...)
		out = append(out, byte(i+1), byte(chunkCount))
		out = append(out, chunk...)
	}

	return append(out, img[2:]...)
}

// forEachJPEGSegment walks the marker segments of a JPEG, calling fn for each
// one that carries a payload. It stops at the start of scan, after which the
// entropy-coded data begins.
func forEachJPEGSegment(img []byte, fn func(marker byte, payload []byte)) {
	if len(img) < 2 || img[0] != 0xFF || img[1] != 0xD8 {
		return
	}

	for i := 2; i+1 < len(img); {
		if img[i] != 0xFF {
			return // not at a marker boundary; give up rather than resync
		}
		marker := img[i+1]
		i += 2

		switch {
		case marker == 0xFF:
			i-- // fill byte; the next byte may be the marker
			continue
		case marker == 0x01, marker >= 0xD0 && marker <= 0xD9:
			continue // standalone markers carry no payload
		case marker == 0xDA:
			return // start of scan
		}

		if i+2 > len(img) {
			return
		}
		length := int(binary.BigEndian.Uint16(img[i:]))
		if length < 2 || i+length > len(img) {
			return
		}
		fn(marker, img[i+2:i+length])
		i += length
	}
}

// preservesSourceColorSpace reports whether the pixels decoded from a source
// image still live in the color space its embedded profile describes, so that
// re-attaching the profile to the output is correct rather than merely present.
//
// Go's JPEG decoder converts YCbCr to RGB with the fixed JPEG matrix, which is
// color-space agnostic — the resulting triplets stay in the source's primaries.
// A CMYK or YCCK source is different: the decoder converts it to RGB with a
// naive inversion, so the CMYK profile no longer describes the result.
func preservesSourceColorSpace(m image.Image, profile []byte) bool {
	// Every ICC profile opens with a 128-byte header, so a shorter blob is
	// malformed however plausible its color-space field looks.
	if len(profile) < iccHeaderSize || len(profile) > maxICCProfileSize {
		return false
	}
	// Bytes 16..20 of the ICC header hold the data color space signature.
	if !bytes.Equal(profile[16:20], []byte("RGB ")) {
		return false
	}
	switch m.(type) {
	case *image.CMYK, *image.Gray, *image.Gray16:
		return false
	}
	return true
}
