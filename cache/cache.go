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

// Package cache implements a image cache.
package cache

import "github.com/willnorris/go-imageproxy/data"

// Cache provides a cache for image metadata and transformed variants of the
// image.
type Cache interface {
	// Get retrieves the cached Image for the provided image URL.
	Get(string) (image *data.Image, ok bool)

	// Put caches the provided Image.
	Save(*data.Image)

	// Delete deletes the cached Image and all variants for the image at the specified URL.
	Delete(string)
}
