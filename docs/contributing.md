# Contributing to imageproxy

## Types of contributions

Simple bug fixes for existing functionality are always welcome.  In many cases,
it may be helpful to include a reproducible sample case that demonstrates the
bug being fixed.

For new functionality, it's general best to open an issue first to discuss it.

## Reporting issues

Bugs, feature requests, and development-related questions should be directed to
the [GitHub issue tracker](https://github.com/willnorris/imageproxy/issues).
If reporting a bug, please try and provide as much context as possible such as
what version of imageproxy you're running, what configuration options, specific
remote URLs that exhibit issues, and anything else that might be relevant to
the bug.  For feature requests, please explain what you're trying to do, and
how the requested feature would help you do that.

Security related bugs can either be reported in the issue tracker, or if they
are more sensitive, emailed to <will@willnorris.com>.

## Code Style and Tests

Go code should follow general best practices, such as using go fmt, go lint, and
go vet (this is enforced by our continuous integration setup).  Tests should
always be included where possible, especially for bug fixes in order to prevent
regressions.
