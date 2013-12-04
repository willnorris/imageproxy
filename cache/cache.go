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
