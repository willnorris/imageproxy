# go-imageproxy #

go-imageproxy is a caching image proxy server written in golang.  It supports
dynamic image resizing and URL whitelisting.

This project was inspired by, and is designed to be an alternative to,
WordPress's [photon service][photon].  Photon is a great free service, but is
limited to sites hosted on WordPress.com, or that use the [Jetpack
plugin][jetpack].  If you don't want to use Jetpack, then you're asked to use a
different service.  If you're looking for an alternative hosted service, I'd
recommend [resize.ly][], [embed.ly][], or [cloudinary][].  I decided to try
building my own for fun.

[photon]: http://developer.wordpress.com/docs/photon/
[jetpack]: http://jetpack.me/
[resize.ly]: https://resize.ly/
[embed.ly]: http://embed.ly/display
[cloudinary]: http://cloudinary.com/


## URL Structure ##

go-imageproxy URLs are of the form `http://localhost/{options}/{remote_url}`.

### Options ###

Currently, the options path segment follows the same structure as
[resize.ly][].  You can specify:

 - square crop - one number, which is used as both the height and width.
   (example: `500`)
 - rectangular crop - two numbers, separated by an 'x', resizes to the exact
   dimensions (width listed first).  (example: `250x125`)
 - auto height (preserves aspect ratio) - one number to the left of the 'x',
   resizes to a specific width, adjusting the height to preserve the
   aspect ration (example: `160x`)
 - auto width (preserves aspect ratio) - one number to the right of the 'x',
   resizes to a specific height, adjusting the width to preserve the
   aspect ration (example: `x200`)

### Remote URL ###

The URL of the original image to load is specified as the remainder of the
path, without any encoding.  For example,
`http://localhost/200/https://willnorris.com/logo.jpg`.

In order to [optimize caching][], it is recommended that URLs not contain query
strings.

[optimize caching]: http://www.stevesouders.com/blog/2008/08/23/revving-filenames-dont-use-querystring/


## License ##

This application is distributed under the BSD-style license found in the
[LICENSE](./LICENSE) file.
