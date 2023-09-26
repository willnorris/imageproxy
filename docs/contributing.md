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

## Contributor License Agreement

Even though this is effectively a personal project of mine, it is still governed
by Google's Contributor License Agreement because of my employment there.  You
(or your employer) retain the copyright to your contribution; the CLA simply
gives permission to use and redistribute your contributions as part of the
project.  Head over to <https://cla.developers.google.com/> to see your current
agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different Google project), you probably don't need to do
it again.
