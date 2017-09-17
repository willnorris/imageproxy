TwoTier [![Build Status](https://travis-ci.org/die-net/lrucache.svg?branch=master)](https://travis-ci.org/die-net/lrucache)
========

TwoTier is an [httpcache.Cache](https://github.com/gregjones/httpcache) implementation that wraps two other httpcache.Cache instances,
allowing you to use both a small and fast cache (such as an in-memory [LruCache](https://github.com/die-net/lrucache) or [memcache](https://github.com/gregjones/httpcache/tree/master/memcache)) for popular objects and
fall back to a larger and slower cache (such as [s3cache](https://github.com/sourcegraph/s3cache)) for less popular ones.

While TwoTier passes Set and Delete operations to both tiers, it can't make strong guarantees that the contents of both caches will always remain in sync. If you are caching URLs that don't change often or don't mind that you sometimes get different versions of the same URL's contents, this is probably fine. When using LruCache as the first-tier cache, you can limit how long it can disagree with the second-tier cache by setting its MaxAge parameter to the maximum time you are comfortable with them disagreeing.

See the godoc API documentation for [TwoTier](https://godoc.org/github.com/die-net/lrucache/twotier) or [LruCache](https://godoc.org/github.com/die-net/lrucache).

There is a test-suite included that has close to 100% test coverage on TwoTier's relatively simple functionality.

License
-------

Copyright 2016 Aaron Hopkins and contributors

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at: http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
