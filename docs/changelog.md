# Changelog

This file contains all notable changes to
[imageproxy](https://github.com/willnorris/imageproxy).  The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
[Unreleased]: https://github.com/willnorris/imageproxy/compare/v0.9.0...HEAD

## [0.10.0] (2020-04-02)
[0.10.0]: https://github.com/willnorris/imageproxy/compare/v0.9.0...v0.10.0

### Added
 - add support for multiple signature keys to support key rotation
   ([ef09c1b](https://github.com/willnorris/imageproxy/commit/ef09c1b), 
   [#209](https://github.com/willnorris/imageproxy/pull/209), 
   [maurociancio](https://github.com/maurociancio))
 - added option to include referer header in remote requests
   ([#216](https://github.com/willnorris/imageproxy/issues/216))
 - added basic support for recording prometheus metrics
   ([#121](https://github.com/willnorris/imageproxy/pull/121)
   [benhaan](https://github.com/benhaan))

### Fixed
 - improved content type detection for some hosts, particularly S3
   ([ea95ad9](https://github.com/willnorris/imageproxy/commit/ea95ad9),
   [shahan312](https://github.com/shahan312))
 - fix signature verification for some proxied URLs
   ([3589510](https://github.com/willnorris/imageproxy/commit/3589510),
   [#212](https://github.com/willnorris/imageproxy/issues/212),
   ([#215](https://github.com/willnorris/imageproxy/issues/215),
   thanks to [aaronpk](https://github.com/aaronpk) for helping debug and
   [fieldistor](https://github.com/fieldistor) for the suggested fix)

## [0.9.0] (2019-06-10)
[0.9.0]: https://github.com/willnorris/imageproxy/compare/v0.8.0...v0.9.0

### Added
 - allow request signatures to cover options
   ([#145](https://github.com/willnorris/imageproxy/issues/145))
 - add simple imageproxy-sign tool for calculating signatures
   ([e1558d5](https://github.com/willnorris/imageproxy/commit/e1558d5))
 - allow overriding the Logger used by Proxy
   ([#174](https://github.com/willnorris/imageproxy/pull/174),
   [hmhealey](https://github.com/hmhealey))
 - allow using environment variables for configuration
   ([50e0d11](https://github.com/willnorris/imageproxy/commit/50e0d11))
 - add support for BMP images
   ([d4ba520](https://github.com/willnorris/imageproxy/commit/d4ba520))

### Changed
 - improvements to docker image: run as non-privileged user, use go1.12
   compiler, and build imageproxy as a go module.

 - options are now sorted when converting to string.  This is a breaking change
   for anyone relying on the option order, and will additionally invalidate
   most cached values, since the option string is part of the cache key.

   Both the original remote image, as well as any transformations on that image
   are cached, but only the transformed images will be impacted by this change.
   This will result in imageproxy having to re-perform the transformations, but
   should not result in re-fetching the remote image, unless it has already
   otherwise expired.

### Fixed
 - properly include Accept header on remote URL requests
   ([#165](https://github.com/willnorris/imageproxy/issues/165),
   [6aca1e0](https://github.com/willnorris/imageproxy/commit/6aca1e0))
 - detect response content type if content-type header is missing
   ([cf54b2c](https://github.com/willnorris/imageproxy/commit/cf54b2c))

### Removed
 - removed deprecated `whitelist` flag and `Proxy.Whitelist` struct field. Use
   `allowHosts` and `Proxy.AllowHosts` instead.

## [0.8.0] (2019-03-21)
[0.8.0]: https://github.com/willnorris/imageproxy/compare/v0.7.0...v0.8.0

### Added
 - added support for restricting proxied URLs [based on Content-Type
   headers](https://github.com/willnorris/imageproxy#allowed-content-type-list)
   ([#141](https://github.com/willnorris/imageproxy/pull/141),
   [ccbrown](https://github.com/ccbrown))
 - added ability to [deny requests](https://github.com/willnorris/imageproxy#allowed-and-denied-hosts-list)
   for certain remote hosts
   ([#85](https://github.com/willnorris/imageproxy/pull/85),
   [geriljaSA](https://github.com/geriljaSA))
 - added `userAgent` flag to specify a custom user agent when fetching images
   ([#83](https://github.com/willnorris/imageproxy/pull/83),
   [huguesalary](https://github.com/huguesalary))
 - added support for [s3 compatible](https://github.com/willnorris/imageproxy#cache)
   storage providers
   ([#147](https://github.com/willnorris/imageproxy/pull/147),
   [ruledio](https://github.com/ruledio))
 - log URL when image transform fails for easier debugging
   ([#149](https://github.com/willnorris/imageproxy/pull/149),
   [daohoangson](https://github.com/daohoangson))
 - added support for building imageproxy as a [go module](https://golang.org/wiki/Modules).
   A future version will remove vendored dependencies, at which point building
   as a module will be the only supported method of building imageproxy.

### Changed
 - when a remote URL is denied, return a generic error message that does not specify exactly why it failed
   ([7e19b5c](https://github.com/willnorris/imageproxy/commit/7e19b5c))

### Deprecated
 - `whitelist` flag and `Proxy.Whitelist` struct field renamed to `allowHosts`
   and `Proxy.AllowHosts`.  Old values are still supported, but will be removed
   in a future release.

### Fixed
 - fixed tcp_mem resource leak on 304 responses
   ([#153](https://github.com/willnorris/imageproxy/pull/153),
   [Micr0mega](https://github.com/Micr0mega))

## [0.7.0] (2018-02-06)
[0.7.0]: https://github.com/willnorris/imageproxy/compare/v0.6.0...v0.7.0

### Added
 - added support for arbitrary [rectangular crops](https://godoc.org/willnorris.com/go/imageproxy#hdr-Rectangle_Crop)
   ([#90](https://github.com/willnorris/imageproxy/pull/90),
   [maciejtarnowski](https://github.com/maciejtarnowski))
 - added support for tiff images
   ([#109](https://github.com/willnorris/imageproxy/pull/109),
   [mikecx](https://github.com/mikecx))
 - added support for additional [caching backends](https://github.com/willnorris/imageproxy#cache):
    - Google Cloud Storage
      ([#106](https://github.com/willnorris/imageproxy/pull/106),
      [diegomarangoni](https://github.com/diegomarangoni))
    - Azure
      ([#79](https://github.com/willnorris/imageproxy/pull/79),
      [PaulARoy](https://github.com/PaulARoy))
    - Redis
      ([#49](https://github.com/willnorris/imageproxy/issues/49)
      [dbfc693](https://github.com/willnorris/imageproxy/commit/dbfc693))
    - Tiering multiple caches by repeating the `-cache` flag
      ([ec5b543](https://github.com/willnorris/imageproxy/commit/ec5b543))
 - added support for EXIF orientation tags
   ([#63](https://github.com/willnorris/imageproxy/issues/63),
   [67619a6](https://github.com/willnorris/imageproxy/commit/67619a6))
 - added [smart crop feature](https://godoc.org/willnorris.com/go/imageproxy#hdr-Smart_Crop)
   ([#55](https://github.com/willnorris/imageproxy/issues/55),
   [afbd254](https://github.com/willnorris/imageproxy/commit/afbd254))

### Changed
 - rotate values are normalized, such that `r-90` is the same as `r270`
   ([07c54b4](https://github.com/willnorris/imageproxy/commit/07c54b4))
 - now return `200 OK` response for requests to root `/`
   ([5ee7e28](https://github.com/willnorris/imageproxy/commit/5ee7e28))
 - switch to using official AWS Go SDK for s3 cache storage.  This is a
   breaking change for anyone using that cache implementation, since the URL
   syntax has changed.  This adds support for the newer v4 auth method, as well
   as additional s3 regions.
   ([0ee5167](https://github.com/willnorris/imageproxy/commit/0ee5167))
 - switched to standard go log library.  Added `-verbose` flag for more logging
   in-memory cache backend supports limiting the max cache size
   ([a57047f](https://github.com/willnorris/imageproxy/commit/a57047f))
 - docker image sized reduced by using scratch image and multistage build
   ([#113](https://github.com/willnorris/imageproxy/pull/113),
   [matematik7](https://github.com/matematik7))

### Removed
 - removed deprecated `cacheDir` and `cacheSize` flags

### Fixed
 - fixed interpretation of `Last-Modified` and `If-Modified-Since` headers
   ([#108](https://github.com/willnorris/imageproxy/pull/108),
   [jamesreggio](https://github.com/jamesreggio))
 - preserve original URL encoding
   ([#115](https://github.com/willnorris/imageproxy/issues/115))

## [0.6.0] (2017-08-29)
[0.6.0]: https://github.com/willnorris/imageproxy/compare/v0.5.1...v0.6.0

### Added
 - added health check endpoint
   ([#54](https://github.com/willnorris/imageproxy/pull/54),
   [immunda](https://github.com/immunda))
 - preserve Link headers from remote image
   ([#68](https://github.com/willnorris/imageproxy/pull/68),
   [xavren](https://github.com/xavren))
 - added support for per-request timeout
   ([#75](https://github.com/willnorris/imageproxy/issues/75))
 - added support for specifying output image format
   ([b9cc9df](https://github.com/willnorris/imageproxy/commit/b9cc9df))
 - added webp support (decode only)
   ([3280445](https://github.com/willnorris/imageproxy/commit/3280445))
 - added CORS support
   ([#96](https://github.com/willnorris/imageproxy/pull/96),
   [romdim](https://github.com/romdim))

### Fixed
 - improved error messages for some authorization failures
   ([27d5378](https://github.com/willnorris/imageproxy/commit/27d5378))
 - skip transformation when not needed
   ([#64](https://github.com/willnorris/imageproxy/issues/64))
 - properly handled "cleaned" remote URLs
   ([a1af9aa](https://github.com/willnorris/imageproxy/commit/a1af9aa),
   [b61992e](https://github.com/willnorris/imageproxy/commit/b61992e))

## [0.5.1] (2015-12-07)
[0.5.1]: https://github.com/willnorris/imageproxy/compare/v0.5.0...v0.5.1

### Fixed
 - fixed bug in gif resizing
   ([gifresize@104a7cd](https://github.com/willnorris/gifresize/commit/104a7cd))

## [0.5.0] (2015-12-07)
[0.5.0]: https://github.com/willnorris/imageproxy/compare/v0.4.0...v0.5.0

## Added
 - added Dockerfile
   ([#29](https://github.com/willnorris/imageproxy/pull/29),
   [sevki](https://github.com/sevki))
 - allow scaling image beyond its original size with `-scaleUp` flag
   ([#37](https://github.com/willnorris/imageproxy/pull/37),
   [runemadsen](https://github.com/runemadsen))
 - add ability to restrict HTTP referrer
   ([9213c93](https://github.com/willnorris/imageproxy/commit/9213c93),
   [connor4312](https://github.com/connor4312))
 - preserve cache-control header from remote image
   ([#43](https://github.com/willnorris/imageproxy/pull/43),
   [runemadsen](https://github.com/runemadsen))
 - add support for caching images on Amazon S3
   ([ec96fcb](https://github.com/willnorris/imageproxy/commit/ec96fcb)
   [victortrac](https://github.com/victortrac))

## Changed
 - change default cache to none, and add `-cache` flag for specifying caches.
   This deprecates the `-cacheDir` flag.
 - on-disk cache now stores files in a two-level trie.  For example, for a file
   named "c0ffee", store file as "c0/ff/c0ffee".

## Fixed
 - skip resizing if requested dimensions larger than original
   ([#46](https://github.com/willnorris/imageproxy/pull/46),
   [orian](https://github.com/orian))

## [0.4.0] (2015-05-21)
[0.4.0]: https://github.com/willnorris/imageproxy/compare/v0.3.0...v0.4.0

### Added
 - added support for animated gifs
   ([#23](https://github.com/willnorris/imageproxy/issues/23))

### Changed
 - non-200 responses from remote servers are proxied as-is

## [0.3.0] (2015-12-07)
[0.3.0]: https://github.com/willnorris/imageproxy/compare/v0.2.3...v0.3.0

### Added
 - added support for signing requests using a sha-256 HMAC.
   ([a9efefc](https://github.com/willnorris/imageproxy/commit/a9efefc))
 - more complete logging of requests and whether response is from the cache
   ([#17](https://github.com/willnorris/imageproxy/issues/17))
 - added support for a base URL for remote images.  This allows shorter relative
   URLs to be specified in requests.
   ([#15](https://github.com/willnorris/imageproxy/issues/15))

### Fixed
 - be more precise in copying over all headers from remote image response
   ([1bf0515](https://github.com/willnorris/imageproxy/commit/1bf0515))

## [0.2.3] (2015-02-20)
[0.2.3]: https://github.com/willnorris/imageproxy/compare/v0.2.2...v0.2.3

### Added
 - added quality option
   ([#13](https://github.com/willnorris/imageproxy/pull/13)
   [cubabit](https://github.com/cubabit))

## [0.2.2] (2014-12-08)
[0.2.2]: https://github.com/willnorris/imageproxy/compare/v0.2.1...v0.2.2

### Added
 - added `cacheSize` flag to command line

### Changed
 - improved documentation and error messages
 - negative width or height transformation values interpreted as 0

## [0.2.1] (2014-08-13)
[0.2.1]: https://github.com/willnorris/imageproxy/compare/v0.2.0...v0.2.1

### Changed
 - restructured package so that the command line tools is now installed from
   `willnorris.com/go/imageproxy/cmd/imageproxy`

## [0.2.0] (2014-07-02)
[0.2.0]: https://github.com/willnorris/imageproxy/compare/v0.1.0...v0.2.0

### Added
 - transformed images are cached in addition to the original image
   ([#1](https://github.com/willnorris/imageproxy/issues/1))
 - support etag and last-modified headers on incoming requests
   ([#3](https://github.com/willnorris/imageproxy/issues/3))
 - support wildcards in list of allowed hosts

### Changed
 - options can be specified in any order
 - images cannot be resized larger than their original dimensions

## [0.1.0] (2013-12-26)
[0.1.0]: https://github.com/willnorris/imageproxy/compare/5d75e8a...v0.1.0

Initial release.  Supported transformation options include:
 - width and height
 - different crop modes
 - rotation (in 90 degree increments)
 - flip (horizontal or vertical)

Images can be cached in-memory or on-disk.
