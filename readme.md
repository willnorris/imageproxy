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

Options are specified as a comma delimited list of parameters, the first of
which always specifies image size.  The format is a superset of [resize.ly's
options](https://resize.ly/#demo).

#### Size ####

The size option takes the general form `{width}x{height}`, where width and
height are numbers.  Integer values greater than 1 are interpreted as exact
pixel values.  Floats between 0 and 1 are interpreted as percentages of the
original image size.  If either value is omitted or set to 0, it will be
automatically set to preserve the aspect ratio based on the other dimension.
If a single number is provided (with no "x" separator), it will be used for
both height and width.

#### Crop Mode ####

Depending on the options specified, an image may be cropped to fit the
requested size.  In all cases, the original aspect ratio of the image will be
preserved; go-imageproxy will never stretch the original image.

When no explicit crop mode is specified, the following rules are followed:

 - If both width and height values are specified, the image will be scaled to
   fill the space, cropping if necessary to fit the exact dimension.

 - If only one of the width or height values is specified, the image will be
   resized to fit the specified dimension, scaling the other dimension as
   needed to maintain the aspect ratio.

If the `fit` option is specified together with a width and height value, the
image will be resized to fit within a containing box of the specified size.  As
always, the original aspect ratio will be preserved. Specifying the `fit`
option with only one of either width or height does the same thing as if `fit`
had not been specified.

#### Rotate ####

The `r={degrees}` option will rotate the image the specified number of degrees,
counter-clockwise.  Valid degrees values are `90`, `180`, and `270`.  Images
are rotated **after** being resized.

#### Flip ####

The `fv` option will flip the image vertically.  The `fh` option will flip the
image horizontally.  Images are flipped **after** being resized and rotated.

### Remote URL ###

The URL of the original image to load is specified as the remainder of the
path, without any encoding.  For example,
`http://localhost/200/https://willnorris.com/logo.jpg`.

In order to [optimize caching][], it is recommended that URLs not contain query
strings.

[optimize caching]: http://www.stevesouders.com/blog/2008/08/23/revving-filenames-dont-use-querystring/

### Examples ###

The following live examples demonstrate setting different options on [this
source image][small-things], which measures 1024 by 678 pixels.

[small-things]: https://willnorris.com/content/uploads/2013/12/small-things.jpg

Options | Meaning                                  | Image
--------|------------------------------------------|------
200x    | 200px wide, proportional height          | ![200x](https://s.wjn.me/200x/https://willnorris.com/content/uploads/2013/12/small-things.jpg)
0.15x   | 15% original width, proportional height  | ![0.15x](https://s.wjn.me/0.15x/https://willnorris.com/content/uploads/2013/12/small-things.jpg)
x100    | 100px tall, proportional width           | ![x100](https://s.wjn.me/x100/https://willnorris.com/content/uploads/2013/12/small-things.jpg)
100x150 | 100 by 150 pixels, cropping as needed    | ![100x150](https://s.wjn.me/100x150/https://willnorris.com/content/uploads/2013/12/small-things.jpg)
100     | 100px square, cropping as needed         | ![100](https://s.wjn.me/100/https://willnorris.com/content/uploads/2013/12/small-things.jpg)
150,fit | scale to fit 150px square, no cropping   | ![150,fit](https://s.wjn.me/150,fit/https://willnorris.com/content/uploads/2013/12/small-things.jpg)
100,r=90| 100px square, rotated 90 degrees         | ![100,r=90](https://s.wjn.me/100,r=90/https://willnorris.com/content/uploads/2013/12/small-things.jpg)
100,fv,fh | 100px square, flipped horizontal and vertical | ![100,fv,fh](https://s.wjn.me/100,fv,fh/https://willnorris.com/content/uploads/2013/12/small-things.jpg)


## License ##

This application is distributed under the Apache 2.0 license found in the
[LICENSE](./LICENSE) file.
