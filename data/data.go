// Package data provides common shared data structures for go-imageproxy.
package data

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Transform specifies transformations that can be performed on a
// requested image.
type Transform struct {
	Width  int `json:"width"`  // requested width, in pixels
	Height int `json:"height"` // requested height, in pixels
}

func (o Transform) String() string {
	return fmt.Sprintf("%dx%d", o.Width, o.Height)
}

func ParseTransform(str string) (*Transform, error) {
	t := new(Transform)
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
	URL       *url.URL   // URL of the image to proxy
	Transform *Transform // Image transformation to perform
}
