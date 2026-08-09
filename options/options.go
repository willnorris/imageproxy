// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	optFit             = "fit"
	optFlipVertical    = "fv"
	optFlipHorizontal  = "fh"
	optFormatJPEG      = "jpeg"
	optFormatPNG       = "png"
	optFormatTIFF      = "tiff"
	optRotatePrefix    = "r"
	optQualityPrefix   = "q"
	optSignaturePrefix = "s"
	optSizeDelimiter   = "x"
	optScaleUp         = "scaleUp"
	optCropX           = "cx"
	optCropY           = "cy"
	optCropWidth       = "cw"
	optCropHeight      = "ch"
	optSmartCrop       = "sc"
	optTrim            = "trim"
	optValidUntil      = "vu"
)

// URLError reports a malformed URL error.
type URLError struct {
	Message string
	URL     *url.URL
}

func (e URLError) Error() string {
	return fmt.Sprintf("malformed URL %q: %s", e.URL, e.Message)
}

// Options specifies transformations to be performed on the requested image.
type Options struct {
	// See ParseOptions for interpretation of Width and Height values
	Width  float64
	Height float64

	// If true, resize the image to fit in the specified dimensions.  Image
	// will not be cropped, and aspect ratio will be maintained.
	Fit bool

	// Rotate image the specified degrees counter-clockwise.  Valid values
	// are 90, 180, 270.
	Rotate int

	FlipVertical   bool
	FlipHorizontal bool

	// Quality of output image
	Quality int

	// HMAC Signature for signed requests.
	Signature string

	// Allow image to scale beyond its original dimensions.  This value
	// will always be overwritten by the value of Proxy.ScaleUp.
	ScaleUp bool

	// Desired image format. Valid values are "jpeg", "png", "tiff".
	Format string

	// Crop rectangle params
	CropX      float64
	CropY      float64
	CropWidth  float64
	CropHeight float64

	// Automatically find good crop points based on image content.
	SmartCrop bool

	// If true, automatically trim pixels of the same color around the edges
	Trim bool

	// If non-zero, the URL is valid until this time.
	ValidUntil time.Time
}

func (o Options) String() string {
	opts := []string{fmt.Sprintf("%v%s%v", o.Width, optSizeDelimiter, o.Height)}
	if o.Fit {
		opts = append(opts, optFit)
	}
	if o.Rotate != 0 {
		opts = append(opts, fmt.Sprintf("%s%d", optRotatePrefix, o.Rotate))
	}
	if o.FlipVertical {
		opts = append(opts, optFlipVertical)
	}
	if o.FlipHorizontal {
		opts = append(opts, optFlipHorizontal)
	}
	if o.Quality != 0 {
		opts = append(opts, fmt.Sprintf("%s%d", optQualityPrefix, o.Quality))
	}
	if o.Signature != "" {
		opts = append(opts, fmt.Sprintf("%s%s", optSignaturePrefix, o.Signature))
	}
	if o.ScaleUp {
		opts = append(opts, optScaleUp)
	}
	if o.Format != "" {
		opts = append(opts, o.Format)
	}
	if o.CropX != 0 {
		opts = append(opts, fmt.Sprintf("%s%v", optCropX, o.CropX))
	}
	if o.CropY != 0 {
		opts = append(opts, fmt.Sprintf("%s%v", optCropY, o.CropY))
	}
	if o.CropWidth != 0 {
		opts = append(opts, fmt.Sprintf("%s%v", optCropWidth, o.CropWidth))
	}
	if o.CropHeight != 0 {
		opts = append(opts, fmt.Sprintf("%s%v", optCropHeight, o.CropHeight))
	}
	if o.SmartCrop {
		opts = append(opts, optSmartCrop)
	}
	if o.Trim {
		opts = append(opts, optTrim)
	}
	if !o.ValidUntil.IsZero() {
		opts = append(opts, fmt.Sprintf("%s%d", optValidUntil, o.ValidUntil.Unix()))
	}

	sort.Strings(opts)

	return strings.Join(opts, ",")
}

// Transform returns whether o includes transformation options.  Some fields
// are not Transform related at all (like Signature), and others only apply in
// the presence of other fields (like Fit).  A non-empty Format value is
// assumed to involve a transformation.
func (o Options) Transform() bool {
	return o.Width != 0 || o.Height != 0 || o.Rotate != 0 || o.FlipHorizontal || o.FlipVertical || o.Quality != 0 || o.Format != "" || o.CropX != 0 || o.CropY != 0 || o.CropWidth != 0 || o.CropHeight != 0 || o.Trim
}

// ParseOptions parses str as a list of comma separated transformation options.
// The options can be specified in in order, with duplicate options overwriting
// previous values.
//
// # Rectangle Crop
//
// There are four options controlling rectangle crop:
//
//	cx{x}      - X coordinate of top left rectangle corner (default: 0)
//	cy{y}      - Y coordinate of top left rectangle corner (default: 0)
//	cw{width}  - rectangle width (default: image width)
//	ch{height} - rectangle height (default: image height)
//
// For all options, integer values are interpreted as exact pixel values and
// floats between 0 and 1 are interpreted as percentages of the original image
// size. Negative values for cx and cy are measured from the right and bottom
// edges of the image, respectively.
//
// If the crop width or height exceed the width or height of the image, the
// crop width or height will be adjusted, preserving the specified cx and cy
// values.  Rectangular crop is applied before any other transformations.
//
// # Smart Crop
//
// The "sc" option will perform a content-aware smart crop to fit the
// requested image width and height dimensions (see Size and Cropping below).
// The smart crop option will override any requested rectangular crop.
//
// # Size and Cropping
//
// The size option takes the general form "{width}x{height}", where width and
// height are numbers. Integer values greater than 1 are interpreted as exact
// pixel values. Floats between 0 and 1 are interpreted as percentages of the
// original image size. If either value is omitted or set to 0, it will be
// automatically set to preserve the aspect ratio based on the other dimension.
// If a single number is provided (with no "x" separator), it will be used for
// both height and width.
//
// Depending on the size options specified, an image may be cropped to fit the
// requested size. In all cases, the original aspect ratio of the image will be
// preserved; imageproxy will never stretch the original image.
//
// When no explicit crop mode is specified, the following rules are followed:
//
// - If both width and height values are specified, the image will be scaled to
// fill the space, cropping if necessary to fit the exact dimension.
//
// - If only one of the width or height values is specified, the image will be
// resized to fit the specified dimension, scaling the other dimension as
// needed to maintain the aspect ratio.
//
// If the "fit" option is specified together with a width and height value, the
// image will be resized to fit within a containing box of the specified size.
// As always, the original aspect ratio will be preserved. Specifying the "fit"
// option with only one of either width or height does the same thing as if
// "fit" had not been specified.
//
// # Rotation and Flips
//
// The "r{degrees}" option will rotate the image the specified number of
// degrees, counter-clockwise. Valid degrees values are 90, 180, and 270.
//
// The "fv" option will flip the image vertically. The "fh" option will flip
// the image horizontally. Images are flipped after being rotated.
//
// # Quality
//
// The "q{qualityPercentage}" option can be used to specify the quality of the
// output file (JPEG only). If not specified, the default value of "95" is used.
//
// # Format
//
// The "jpeg", "png", and "tiff" options can be used to specify the desired
// image format of the proxied image.
//
// # Signature
//
// The "s{signature}" option specifies an optional base64 encoded HMAC used to
// sign the remote URL in the request.  The HMAC key used to verify signatures is
// provided to the imageproxy server on startup.
//
// See https://github.com/willnorris/imageproxy/blob/master/docs/url-signing.md
// for examples of generating signatures.
//
// # Trim
//
// The "trim" option will automatically trim pixels of the same color around
// the edges of the image.  This is useful for removing borders from images
// that have been resized or cropped.  The trim option is applied before other
// options such as cropping or resizing.
//
// # Valid Until
//
// The "vu{unixtime}" option specifies a Unix timestamp at which the request URL is no longer valid.
// For example, "vu1800000000" would mean the URL is valid until 2027-01-15T08:00:00Z.
//
// Examples
//
//	0x0         - no resizing
//	200x        - 200 pixels wide, proportional height
//	x0.15       - 15% original height, proportional width
//	100x150     - 100 by 150 pixels, cropping as needed
//	100         - 100 pixels square, cropping as needed
//	150,fit     - scale to fit 150 pixels square, no cropping
//	100,r90     - 100 pixels square, rotated 90 degrees
//	100,fv,fh   - 100 pixels square, flipped horizontal and vertical
//	200x,q60    - 200 pixels wide, proportional height, 60% quality
//	200x,png    - 200 pixels wide, converted to PNG format
//	cw100,ch100 - crop image to 100px square, starting at (0,0)
//	cx10,cy20,cw100,ch200 - crop image starting at (10,20) is 100px wide and 200px tall
func ParseOptions(str string) Options {
	var options Options

	for _, opt := range strings.Split(str, ",") {
		switch {
		case len(opt) == 0: // do nothing
		case opt == optFit:
			options.Fit = true
		case opt == optFlipVertical:
			options.FlipVertical = true
		case opt == optFlipHorizontal:
			options.FlipHorizontal = true
		case opt == optScaleUp: // this option is intentionally not documented above
			options.ScaleUp = true
		case opt == optFormatJPEG, opt == optFormatPNG, opt == optFormatTIFF:
			options.Format = opt
		case opt == optSmartCrop:
			options.SmartCrop = true
		case opt == optTrim:
			options.Trim = true
		case strings.HasPrefix(opt, optRotatePrefix):
			value := strings.TrimPrefix(opt, optRotatePrefix)
			options.Rotate, _ = strconv.Atoi(value)
		case strings.HasPrefix(opt, optQualityPrefix):
			value := strings.TrimPrefix(opt, optQualityPrefix)
			options.Quality, _ = strconv.Atoi(value)
		case strings.HasPrefix(opt, optSignaturePrefix):
			options.Signature = strings.TrimPrefix(opt, optSignaturePrefix)
		case strings.HasPrefix(opt, optCropX):
			value := strings.TrimPrefix(opt, optCropX)
			options.CropX, _ = strconv.ParseFloat(value, 64)
		case strings.HasPrefix(opt, optCropY):
			value := strings.TrimPrefix(opt, optCropY)
			options.CropY, _ = strconv.ParseFloat(value, 64)
		case strings.HasPrefix(opt, optCropWidth):
			value := strings.TrimPrefix(opt, optCropWidth)
			options.CropWidth, _ = strconv.ParseFloat(value, 64)
		case strings.HasPrefix(opt, optCropHeight):
			value := strings.TrimPrefix(opt, optCropHeight)
			options.CropHeight, _ = strconv.ParseFloat(value, 64)
		case strings.HasPrefix(opt, optValidUntil):
			value := strings.TrimPrefix(opt, optValidUntil)
			if v, _ := strconv.ParseInt(value, 10, 64); v > 0 {
				options.ValidUntil = time.Unix(v, 0)
			}
		case strings.Contains(opt, optSizeDelimiter):
			size := strings.SplitN(opt, optSizeDelimiter, 2)
			if w := size[0]; w != "" {
				options.Width, _ = strconv.ParseFloat(w, 64)
			}
			if h := size[1]; h != "" {
				options.Height, _ = strconv.ParseFloat(h, 64)
			}
		default:
			if size, err := strconv.ParseFloat(opt, 64); err == nil {
				options.Width = size
				options.Height = size
			}
		}
	}

	return options
}
