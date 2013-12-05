// Package data provides common shared data structures for go-imageproxy.
package data

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Options specifies transformations that can be performed on a
// requested image.
type Options struct {
	Width  int // requested width, in pixels
	Height int // requested height, in pixels
}

func (o Options) String() string {
	return fmt.Sprintf("%dx%d", o.Width, o.Height)
}

func ParseOptions(str string) (*Options, error) {
	t := new(Options)
	var err error
	var h, w string

	size := strings.SplitN(str, "x", 2)
	w = size[0]
	if len(size) > 1 {
		h = size[1]
	} else {
		h = w
	}

	if w != "" {
		t.Width, err = strconv.Atoi(w)
		if err != nil {
			return nil, errors.New("width must be an int")
		}
	}
	if h != "" {
		t.Height, err = strconv.Atoi(h)
		if err != nil {
			return nil, errors.New("height must be an int")
		}
	}

	return t, nil
}

type Request struct {
	URL     *url.URL // URL of the image to proxy
	Options *Options // Image transformation to perform
}

// Image represents a remote image that is being proxied.  It tracks where
// the image was originally retrieved from and how long the image can be cached.
type Image struct {
	// URL of original remote image.
	URL string

	// Expires is the cache expiration time for the original image, as
	// returned by the remote server.
	Expires time.Time

	// Etag returned from server when fetching image.
	Etag string

	// Bytes contains the actual image.
	Bytes []byte
}
