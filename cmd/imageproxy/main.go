// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

// imageproxy starts an HTTP server that proxies requests for remote images.
package main

import (
	"willnorris.com/go/imageproxy/cmd"

	// imageproxy cache plugins
	_ "willnorris.com/go/imageproxy/cache/azure"
	_ "willnorris.com/go/imageproxy/cache/disk"
	_ "willnorris.com/go/imageproxy/cache/gcs"
	_ "willnorris.com/go/imageproxy/cache/memory"
	_ "willnorris.com/go/imageproxy/cache/redis"
	_ "willnorris.com/go/imageproxy/cache/s3"

	// imageproxy image encoding plugins
	_ "willnorris.com/go/imageproxy/image/bmp"
	_ "willnorris.com/go/imageproxy/image/gif"
	_ "willnorris.com/go/imageproxy/image/jpeg"
	_ "willnorris.com/go/imageproxy/image/png"
	_ "willnorris.com/go/imageproxy/image/tiff"

	_ "golang.org/x/image/webp" // decoder only
)

func main() {
	cmd.Main()
}
