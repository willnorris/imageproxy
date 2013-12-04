package cache

import "github.com/willnorris/go-imageproxy/data"

// MemoryCache provides an in-memory Cache implementation.
type MemoryCache struct {
	images map[string]*data.Image
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		make(map[string]*data.Image),
	}
}

func (c MemoryCache) Get(u string) (*data.Image, bool) {
	image, ok := c.images[u]
	return image, ok
}

func (c MemoryCache) Save(image *data.Image) {
	c.images[image.URL] = image
}

func (c MemoryCache) Delete(u string) {
	delete(c.images, u)
}
