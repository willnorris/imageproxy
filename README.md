# imageproxy [![Build Status](https://travis-ci.org/willnorris/imageproxy.svg?branch=master)](https://travis-ci.org/willnorris/imageproxy) [![GoDoc](https://godoc.org/willnorris.com/go/imageproxy?status.svg)](https://godoc.org/willnorris.com/go/imageproxy) [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg?style=flat)](LICENSE)

imageproxy is a caching image proxy server written in golang.  It supports
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

imageproxy URLs are of the form `http://localhost/{options}/{remote_url}`.

### Options ###

Options are specified as a comma delimited list of parameters, which can be
supplied in any order.  Duplicate parameters overwrite previous values.

The format is a superset of [resize.ly's options](https://resize.ly/#demo).

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
preserved; imageproxy will never stretch the original image.

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

The `r{degrees}` option will rotate the image the specified number of degrees,
counter-clockwise.  Valid degrees values are `90`, `180`, and `270`.  Images
are rotated **after** being resized.

#### Flip ####

The `fv` option will flip the image vertically.  The `fh` option will flip the
image horizontally.  Images are flipped **after** being resized and rotated.

#### Quality ####

The `q{percentage}` option can be used to specify the output quality (JPEG
only).  If not specified, the default value of `95` is used.

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

[small-things]: https://willnorris.com/2013/12/small-things.jpg

Options | Meaning                                  | Image
--------|------------------------------------------|------
200x    | 200px wide, proportional height          | <a href="https://willnorris.com/api/imageproxy/200x/https://willnorris.com/2013/12/small-things.jpg"><img src="https://willnorris.com/api/imageproxy/200x/https://willnorris.com/2013/12/small-things.jpg" alt="200x"></a>
0.15x   | 15% original width, proportional height  | <a href="https://willnorris.com/api/imageproxy/0.15x/https://willnorris.com/2013/12/small-things.jpg"><img src="https://willnorris.com/api/imageproxy/0.15x/https://willnorris.com/2013/12/small-things.jpg" alt="0.15x"></a>
x100    | 100px tall, proportional width           | <a href="https://willnorris.com/api/imageproxy/x100/https://willnorris.com/2013/12/small-things.jpg"><img src="https://willnorris.com/api/imageproxy/x100/https://willnorris.com/2013/12/small-things.jpg" alt="x100"></a>
100x150 | 100 by 150 pixels, cropping as needed    | <a href="https://willnorris.com/api/imageproxy/100x150/https://willnorris.com/2013/12/small-things.jpg"><img src="https://willnorris.com/api/imageproxy/100x150/https://willnorris.com/2013/12/small-things.jpg" alt="100x150"></a>
100     | 100px square, cropping as needed         | <a href="https://willnorris.com/api/imageproxy/100/https://willnorris.com/2013/12/small-things.jpg"><img src="https://willnorris.com/api/imageproxy/100/https://willnorris.com/2013/12/small-things.jpg" alt="100"></a>
150,fit | scale to fit 150px square, no cropping   | <a href="https://willnorris.com/api/imageproxy/150,fit/https://willnorris.com/2013/12/small-things.jpg"><img src="https://willnorris.com/api/imageproxy/150,fit/https://willnorris.com/2013/12/small-things.jpg" alt="150,fit"></a>
100,r90 | 100px square, rotated 90 degrees         | <a href="https://willnorris.com/api/imageproxy/100,r90/https://willnorris.com/2013/12/small-things.jpg"><img src="https://willnorris.com/api/imageproxy/100,r90/https://willnorris.com/2013/12/small-things.jpg" alt="100,r90"></a>
100,fv,fh | 100px square, flipped horizontal and vertical | <a href="https://willnorris.com/api/imageproxy/100,fv,fh/https://willnorris.com/2013/12/small-things.jpg"><img src="https://willnorris.com/api/imageproxy/100,fv,fh/https://willnorris.com/2013/12/small-things.jpg" alt="100,fv,fh"></a>
200x,q60 | 200px wide, proportional height, 60% quality | <a href="https://willnorris.com/api/imageproxy/200x,q60/https://willnorris.com/2013/12/small-things.jpg"><img src="https://willnorris.com/api/imageproxy/200x,q60/https://willnorris.com/2013/12/small-things.jpg" alt="200x,q60"></a>


## Getting Started ##

Install the package using:

    go get willnorris.com/go/imageproxy/cmd/imageproxy

Once installed, ensure `$GOPATH/bin` is in your `$PATH`, then run the proxy using:

    imageproxy

This will start the proxy on port 8080, without any caching and with no host
whitelist (meaning any remote URL can be proxied).  Test this by navigating to
<http://localhost:8080/500/https://octodex.github.com/images/codercat.jpg> and
you should see a 500px square coder octocat.

### Disk cache ###

By default, the imageproxy command uses an in-memory cache that will grow
unbounded.  To cache images on disk instead, include the `cacheDir` flag:

    imageproxy -cacheDir /tmp/imageproxy

Reload the [codercat URL](http://localhost:8080/500/https://octodex.github.com/images/codercat.jpg),
and then inspect the contents of `/tmp/imageproxy`.  There should be two files
there, one for the original full-size codercat image, and one for the resized
500px version.

### Host whitelist ###

You can limit the remote hosts that the proxy will fetch images from using the
`whitelist` flag.  This is useful, for example, for locking the proxy down to
your own hosts to prevent others from abusing it.  Of course if you want to
support fetching from any host, leave off the whitelist flag.  Try it out by
running:

    imageproxy -whitelist example.com

Reload the [codercat URL](http://localhost:8080/500/https://octodex.github.com/images/codercat.jpg),
and you should now get an error message.  You can specify multiple hosts as a
comma separated list, or prefix a host value with `*.` to allow all sub-domains
as well.

Run `imageproxy -help` for a complete list of flags the command accepts.  If
you want to use a different caching implementation, it's probably easiest to
just make a copy of `cmd/imageproxy/main.go` and customize it to fit your
needs... it's a very simple command.

### Default Base URL ###

Typically, remote images to be proxied are specified as absolute URLs.
However, if you commonly proxy images from a single source, you can provide a
base URL and then specify remote images relative to that base.  Try it out by running:

    imageproxy -baseURL https://octodex.github.com/

Then load the codercat image, specified as a URL relative to that base:
<http://localhost:8080/500/images/codercat.jpg>.  Note that this is not an
effective method to mask the true source of the images being proxied; it is
trivial to discover the base URL being used.  Even when a base URL is
specified, you can always provide the absolute URL of the image to be proxied.


## Deploying ##

You can build and deploy imageproxy using any standard go toolchain, but here's
how I do it.

I use [goxc](https://github.com/laher/goxc) to build and deploy to an Ubuntu
server.  I have a `$GOPATH/willnorris.com/go/imageproxy/.goxc.local.json` file
which limits builds to 64-bit linux:

``` json
 {
   "ConfigVersion": "0.9",
   "BuildConstraints": "linux,amd64"
 }
```

I then run `goxc` which compiles the static binary and creates a deb package at
`build/0.2.1/imageproxy_0.2.1_amd64.deb` (or whatever the current version is).
I copy this file to my server and install it using `sudo dpkg -i
imageproxy_0.2.1_amd64.deb`, which is installed to `/usr/bin/imageproxy`.

Ubuntu uses upstart to manage services, so I copy
[`etc/imageproxy.conf`](etc/imageproxy.conf) to `/etc/init/imageproxy.conf` on
my server and start it using `sudo service imageproxy start`.  You will
certainly want to modify that upstart script to suit your desired
configuration.


## License ##

This application is distributed under the Apache 2.0 license found in the
[LICENSE](./LICENSE) file.
