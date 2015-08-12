# imageproxy [![Build Status](https://travis-ci.org/willnorris/imageproxy.svg?branch=master)](https://travis-ci.org/willnorris/imageproxy) [![GoDoc](https://godoc.org/willnorris.com/go/imageproxy?status.svg)](https://godoc.org/willnorris.com/go/imageproxy) [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg?style=flat)](LICENSE)

imageproxy is a caching image proxy server written in go.  It features:

 - basic image adjustments like resizing, cropping, and rotation
 - access control using host whitelists or request signing (HMAC-SHA256)
 - support for jpeg, png, and gif image formats (including animated gifs)
 - on-disk caching, respecting the cache headers of the original images
 - easy deployment, since it's pure go

Personally, I use it primarily to dynamically resize images hosted on my own
site (read more in [this post][]).  But you can also enable request signing and
use it as an SSL proxy for remote images, similar to [atmos/camo][] but with
additional image adjustment options.

[this post]: https://willnorris.com/2014/01/a-self-hosted-alternative-to-jetpacks-photon-service
[atmos/camo]: https://github.com/atmos/camo


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

#### Signature ####

The `s{signature}` option specifies an optional base64 encoded HMAC used to
sign the remote URL in the request.  The HMAC key used to verify signatures is
provided to the imageproxy server on startup.

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

Transformation also works on animated gifs.  Here is [this source
image][material-animation] resized to 200px square and rotated 270 degrees:

[material-animation]: https://willnorris.com/2015/05/material-animations.gif

<a href="https://willnorris.com/api/imageproxy/200,r270/https://willnorris.com/2015/05/material-animations.gif"><img src="https://willnorris.com/api/imageproxy/200,r270/https://willnorris.com/2015/05/material-animations.gif" alt="200,r270"></a>


## Getting Started ##

Install the package using:

    go get willnorris.com/go/imageproxy/cmd/imageproxy

(Note that go1.2 and earlier may have trouble fetching the package with `go
get`).

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

Reload the [codercat URL][], and then inspect the contents of
`/tmp/imageproxy`.  There should be two files there, one for the original
full-size codercat image, and one for the resized 500px version.

[codercat URL]: http://localhost:8080/500/https://octodex.github.com/images/codercat.jpg

### Host whitelist ###

You can limit the remote hosts that the proxy will fetch images from using the
`whitelist` flag.  This is useful, for example, for locking the proxy down to
your own hosts to prevent others from abusing it.  Of course if you want to
support fetching from any host, leave off the whitelist flag.  Try it out by
running:

    imageproxy -whitelist example.com

Reload the [codercat URL][], and you should now get an error message.  You can
specify multiple hosts as a comma separated list, or prefix a host value with
`*.` to allow all sub-domains as well.

### Signed Requests ###

Instead of a host whitelist, you can require that requests be signed.  This is
useful in preventing abuse when you don't have just a static list of hosts you
want to allow.  Signatures are generated using HMAC-SHA256 against the remote
URL, and url-safe base64 encoding the result:

    base64urlencode(hmac.New(sha256, <key>).digest(<remote_url>))

The HMAC key is specified using the `signatureKey` flag.  If this flag
begins with an "@", the remainder of the value is interpreted as a file on disk
which contains the HMAC key.

Try it out by running:

    imageproxy -signatureKey "secret key"

Reload the [codercat URL][], and you should see an error message.  Now load a
[signed codercat URL][] and verify that it loads properly.

[signed codercat URL]: http://localhost:8080/500,sXyMwWKIC5JPCtlYOQ2f4yMBTqpjtUsfI67Sp7huXIYY=/https://octodex.github.com/images/codercat.jpg

Some simple code samples for generating signatures in various languages can be
found starting in [this comment](https://github.com/willnorris/imageproxy/issues/11#issuecomment-101428470).

If both a whiltelist and signatureKey are specified, requests can match either.
In other words, requests that match one of the whitelisted hosts don't
necessarily need to be signed, though they can be.


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

### Scaling beyond original size ###

By default, the imageproxy won't scale images beyond their original size. However, you can use the `scaleUp` command-line flag to allow this to happen:

    imageproxy -scaleUp true

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

## Docker ##

A docker image is available at [`willnorris/imageproxy`](https://registry.hub.docker.com/u/willnorris/imageproxy/dockerfile/).

You can run it by
```
docker run -p 8080:8080 willnorris/imageproxy -addr 0.0.0.0:8080
```

## License ##

imageproxy is copyright Google, but is not an official Google product.  It is
available under the [Apache 2.0 License](./LICENSE).
