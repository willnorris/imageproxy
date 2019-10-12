# Plugin Design Doc

**Status:** idea phase, with no immediate timeline for implementation

## Objective

Rearchitect imageproxy to use a plugin-based system for most features like
transformations, security, and caching. This should reduce build times and
binary sizes in the common case, and provide a mechanism for users to easily
add custom features that would not be added to core for various reasons.

## Background

I created imageproxy to [scratch a personal itch](https://wjn.me/b/J_), I
needed a simple way to dynamically resize images for my personal website. I
published it as an open source projects because that's what I do, and I'm happy
to see others finding it useful for their needs as well.

But inevitably, with more users came requests for additional features because
people have different use cases and requirements. Some of these requests were
relatively minor, and I was happy to add them. But one of the more common
requests was to support different caching backends. Personally, I still use the
on-disk cache, but many people wanted to use redis or a cloud provider like
AWS, Azure, or GCP. For a long time I was resistant to adding support for
these, mainly out of concern for inflating build times and binary sizes. I did
eventually relent, and
[#49](https://github.com/willnorris/imageproxy/issues/49) tracked adding
support for the most common backends.

Unfortunately my concerns proved true, and build times are *significantly*
slower (TODO: add concrete numbers) now because of all the additional cloud
SDKs that get compiled in. I don't personally care too much about binary size,
since I'm not running in a constrained environment, but these build times are
really wearing on me. Additionally, there are a number of outstanding pull
requests for relatively obscure features that I don't really want to have to
support in the main project. And quite honestly, there are a number of obscure
features that did get merged in over the years that I kinda wish I could rip
back out.

### Plugin support in Go

TODO: talk about options like
 - RPC (https://github.com/hashicorp/go-plugin)
 - pkg/plugin (https://golang.org/pkg/plugin/)
 - embedded interpreter (https://github.com/robertkrimen/otto)
 - custom binaries (https://github.com/mholt/caddy,
   https://caddy.community/t/59)

Spoiler: I'm planning on following the Caddy approach and using custom
binaries.

## Design

I plan to model imageproxy after Caddy, moving all key functionality into
separate plugins that register themselves with the server, and which all
compile to a single statically-linked binary.  The core project will provide a
great number of plugins to cover all of the existing functionality.  I also
expect I'll be much more open to adding plugins for features I may not care as
much about personally. Of course, users can also write their own plugins and
link them in without needing to contribute them to core if they don't want to.

I anticipate providing two or three build configurations in core:
 - **full** - include all the plugins that are part of core (except where they
   may conflict)
 - **minimal** - some set of minimal features that only includes basic caching
   options, limited transformation options, etc
 - **my personal config** - I'll also definitely have a build that I use
   personally on my site. I may decide to just make that the "minimal" build
   and perhaps call it something different, rather than have a third
   configuration.

Custom configurations beyond what is provided by core can be done by creating a
minimal main package that imports the plugins you care about and calling some
kind of bootstrap method (similar to [what Caddy now
does](https://caddy.community/t/59)).

### Types of plugins

(Initially in no particular order, just capturing thoughts. Lots to do here in
thinking through the use cases and what kind of plugin API we really need to
provide.)

See also issues and PRs with [label:plugins][].

[label:plugins]: https://github.com/willnorris/imageproxy/issues?q=label:plugins

#### Caching backend

This is one of the most common feature requests, and is also one of the worst
offender for inflating build times and binary sizes because of the size of the
dependencies that are typically required.  The minimal imageproxy build would
probably only include the in-memory and on-disk caches. Anything that talked to
an external store (redis, cloud providers, etc) would be pulled out.

#### Transformation engine

Today, imageproxy only performs transformations which can be done with pure Go
libraries. There have been a number of requests (or at least questions) to use
something like [vips](https://github.com/DAddYE/vips) or
[imagemagick](https://github.com/gographics/imagick), which are both C
libraries. They provide more options, and (likely) better performance, at the
cost of complexity and loss of portability in using cgo. These would likely
replace the entire transformation engine in imageproxy, so I don't know how
they would interact with other plugins that merely extend the main engine (they
probably wouldn't be able to interact at all).

#### Transformation options

Today, imageproxy performs minimal transformations, mostly around resizing,
cropping, and rotation.  It doesn't support any kind of filters, brightness or
contrast adjustment, etc. There are go libraries for them, they're just outside
the scope of what I originally intended imageproxy for.  But I'd be happy to
have plugins that do that kind of thing. These plugins would need to be able to
hook into the option parsing engine so that they could register their URL
options.

#### Image format support

There have been a number of requests for imge format support that require cgo
libraries:

 - **webp encoding** - needs cgo
   [#114](https://github.com/willnorris/imageproxy/issues/114)
 - **progressive jpegs** - probably needs cgo?
   [#77](https://github.com/willnorris/imageproxy/issues/77)
 - **gif to mp4** - maybe doable in pure go, but probably belongs in a plugin
   [#136](https://github.com/willnorris/imageproxy/issues/136)
 - **HEIF** - formate used by newer iPhones
   ([HEIF](https://en.wikipedia.org/wiki/High_Efficiency_Image_File_Format))

#### Option parsing

Today, options are specified as the first component in the URL path, but
[#66](https://github.com/willnorris/imageproxy/pull/66) proposes optionally
moving that to a query parameter (for a good reason, actually). Maybe putting
that in core is okay? Maybe it belongs in a plugin, in which case we'd need to
expose an API for replacing the option parsing code entirely.

#### Security options

Some people want to add a host blacklist
[#85](https://github.com/willnorris/imageproxy/pull/85), refusal to process
non-image files [#53](https://github.com/willnorris/imageproxy/issues/53)
[#119](https://github.com/willnorris/imageproxy/pull/119). I don't think there
is an issue for it, but an early fork of the project added request signing that
was compatible with nginx's [secure link
module](https://nginx.org/en/docs/http/ngx_http_secure_link_module.html).

### Registering Plugins

Plugins are loaded simply by importing their package.  They should have an
`init` func that calls `imageproxy.RegisterPlugin`:

``` go
type Plugin struct {
}

func RegisterPlugin(name string, plugin Plugin)
```

Plugins hook into various extension points of imageproxy by implementing
appropriate interfaces.  A single plugin can hook into multiple parts of
imageproxy by implementing multiple interfaces.

For example, two possible interfaces for security related plugins:

``` go
// A RequestAuthorizer determines if a request is authorized to be processed.
// Requests are processed before the remote resource is retrieved.
type RequestAuthorizer interface {
    // Authorize returns an error if the request should not
    // be processed further (for example, it doesn't have a
    // valid signature, is not for a whitelisted host, etc).
    AuthorizeRequest(req *http.Request) error
}

// A ResponseAuthorizer determines if a response from a remote server
// is authorized to be returned.
type ResponseAuthorizer interface {
    // AuthorizeResponse returns an error if a response should not be
    // returned to a client (for example, it is not for an image
    // resource, etc).
    AuthorizeResponse(res http.Response) error
}
```

A hypothetical interface for plugins that transform images:

``` go
// An ImageTransformer transforms an image.
type ImageTransformer interface {
   // TransformImage based on the provided options and return the result.
   TransformImage(m image.Image, opt Options) image.Image
}
```

Plugins are additionally responsible for registering any additional command
line flags they wish to expose to the user, as well as storing any global state
that would previously have been stored on the Proxy struct.
