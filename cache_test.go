// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import "testing"

func TestNopCache(t *testing.T) {
	data, ok := NopCache.Get("foo")
	if data != nil {
		t.Errorf("NopCache.Get returned non-nil data")
	}
	if ok != false {
		t.Errorf("NopCache.Get returned ok = true, should always be false.")
	}

	// nothing to test on these methods other than to verify they exist
	NopCache.Set("", []byte{})
	NopCache.Delete("")
}
