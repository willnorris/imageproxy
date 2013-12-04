package cache

import "github.com/willnorris/go-imageproxy/data"

// NopCache provides a no-op cache implementation that doesn't actually cache anything.
var NopCache = new(nopCache)

type nopCache struct{}

func (c nopCache) Get(u string) (*data.Image, bool) { return nil, false }
func (c nopCache) Save(image *data.Image)           {}
func (c nopCache) Delete(u string)                  {}
