# How to generate signed requests

Signing requests allows an imageproxy instance to proxy images from arbitrary
remote hosts, but without opening the service up for potential abuse.  When
appropriately configured, the imageproxy instance will only serve requests that
are for allowed hosts, or which have a valid signature.

Signatures can be calculated in two ways:

1. they can be calculated solely on the remote image URL, in which case any
   transformations of the image can be requested without changes to the
   signature value.  This used to be the only way to sign requests, but is no
   longer recommended since it still leaves the imageproxy instance open to
   potential abuse.

2. they can be calculated based on the combination of the remote image URL and
   the requested transformation options.

In both cases, the signature is calculated using HMAC-SHA256 and a secret key
which is provided to imageproxy on startup.  The message to be signed is the
remote URL, with the transformation options optionally set as the URL fragment,
[as documented below](#Signing-options).  The signature is url-safe base64
encoded, and [provided as an option][s-option] in the imageproxy request.

imageproxy will accept signatures for URLs with or without options
transparently.  It's up to the publisher of the signed URLs to decide which
method they use to generate the URL.

[s-option]: https://godoc.org/willnorris.com/go/imageproxy#hdr-Signature

## Signing options

Transformation options for a proxied URL are [specified as a comma separated
string][ParseOptions] of individual options, which can be supplied in any
order.  When calculating a signature, options should be put in their canonical
form, sorted in lexigraphical order (omitting the signature option itself), and
appended to the remote URL as the URL fragment.

Currently, only [size option][] has a canonical form, which is
`{width}x{height}` with the number `0` used when no value is specified.  For
example, a request that does not request any size option would still have a
canonical size value of `0x0`, indicating that no size transformation is being
performed.  If only a height of 500px is requested, the canonical form would be
`0x500`.

For example, requesting the remote URL of `http://example.com/image.jpg`,
resized to 100 pixels square, rotated 90 degrees, and converted to 75% quality
might produce an imageproxy URL similar to:

    http://localhost:8080/100,r90,q75/http://example.com/image.jpg

When calculating a signature for this request including transformation options,
the signed value would be:

    http://example.com/image.jpg#100x100,q75,r90

The `100` size option was put in its canonical form of `100x100`, and the
options are sorted, moving `q75` before `r90`.

[ParseOptions]: https://godoc.org/willnorris.com/go/imageproxy#ParseOptions
[size option]: https://godoc.org/willnorris.com/go/imageproxy#hdr-Size_and_Cropping


## Signed options example

Here is an example with signed options through each step.

Using the github codercat, our image url is `https://octodex.github.com/images/codercat.jpg` and our options are `400x400` and `q40`.

The signature key is `secretkey`

The value that goes into the Digest is `https://octodex.github.com/images/codercat.jpg#400x400,q40`

and our resulting signed key is `0sR2kjyfiF1RQRj4Jm2fFa3_6SDFqdAaDEmy1oD2U-4=`

The final url would be
`http://localhost:8080/400x400,q40,s0sR2kjyfiF1RQRj4Jm2fFa3_6SDFqdAaDEmy1oD2U-4=/https://octodex.github.com/images/codercat.jpg`



## Language Examples

Here are examples of calculating signatures in a variety of languages.  These
demonstrate the HMAC-SHA256 bits, but not the option canonicalization.  In each
example, the remote URL `https://octodex.github.com/images/codercat.jpg` is
signed using a signature key of `secretkey`.

See also the [imageproxy-sign tool](/cmd/imageproxy-sign).

### Go

main.go:
```go
package main

import (
        "os"
        "fmt"
        "crypto/hmac"
        "crypto/sha256"
        "encoding/base64"
)

func main() {
        key, url := os.Args[1], os.Args[2]
        mac := hmac.New(sha256.New, []byte(key))
        mac.Write([]byte(url))
        result := mac.Sum(nil)
        fmt.Println(base64.URLEncoding.EncodeToString(result))
}
```

```shell
$ go run sign.go "secretkey" "https://octodex.github.com/images/codercat.jpg"
cw34eyalj8YvpLpETxSIxv2k8QkLel2UAR5Cku2FzGM=
```

### OpenSSL

```shell
$ echo -n "https://octodex.github.com/images/codercat.jpg" | openssl dgst -sha256 -hmac "secretkey" -binary|base64| tr '/+' '_-'
cw34eyalj8YvpLpETxSIxv2k8QkLel2UAR5Cku2FzGM=
```

### Java

```java
import java.util.Base64;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;

class SignUrl {

  public static String sign(String key, String url) throws Exception {
    Mac sha256_HMAC = Mac.getInstance("HmacSHA256");
    SecretKeySpec secret_key = new SecretKeySpec(key.getBytes(), "HmacSHA256");
    sha256_HMAC.init(secret_key);

    return Base64.getUrlEncoder().encodeToString(sha256_HMAC.doFinal(url.getBytes()));
  }

  public static void main(String [] args) throws Exception {
    System.out.println(sign(args[0], args[1]));
  }

}
```

```shell
$ javac SignUrl.java && java SignUrl "secretkey" "https://octodex.github.com/images/codercat.jpg"
cw34eyalj8YvpLpETxSIxv2k8QkLel2UAR5Cku2FzGM=
```

### Ruby

```ruby
require 'openssl'
require 'base64'

key = ARGV[0]
url = ARGV[1]
puts Base64.urlsafe_encode64(OpenSSL::HMAC.digest(OpenSSL::Digest.new('sha256'), key, url)).strip()
```

```shell
% ruby sign.rb "secretkey" "https://octodex.github.com/images/codercat.jpg"
cw34eyalj8YvpLpETxSIxv2k8QkLel2UAR5Cku2FzGM=
```

### Python

```python
import base64
import hashlib
import hmac
import sys

key = sys.argv[1]
url = sys.argv[2]
print base64.urlsafe_b64encode(hmac.new(key, msg=url, digestmod=hashlib.sha256).digest())
```

````shell
$ python sign.py "secretkey" "https://octodex.github.com/images/codercat.jpg"
cw34eyalj8YvpLpETxSIxv2k8QkLel2UAR5Cku2FzGM=
````

### JavaScript

```javascript
const crypto = require('crypto');
const URLSafeBase64 = require('urlsafe-base64');

let key = process.argv[2];
let url = process.argv[3];
console.log(URLSafeBase64.encode(crypto.createHmac('sha256', key).update(url).digest()));
```

````shell
$ node sign.js "secretkey" "https://octodex.github.com/images/codercat.jpg"
cw34eyalj8YvpLpETxSIxv2k8QkLel2UAR5Cku2FzGM=
````

### PHP

````php
<?php

$key = $argv[1];
$url = $argv[2];
echo strtr(base64_encode(hash_hmac('sha256', $url, $key, 1)), '/+' , '_-');
````

````shell
$ php sign.php "secretkey" "https://octodex.github.com/images/codercat.jpg"
cw34eyalj8YvpLpETxSIxv2k8QkLel2UAR5Cku2FzGM=
````
