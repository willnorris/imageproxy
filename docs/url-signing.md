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

## Language Examples

Here are examples of calculating signatures in a variety of languages.  These
demonstrate the HMAC-SHA256 bits, but not the option canonicalization.

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
        mac := hmac.New(sha256.New, []byte(os.Args[1]))
        mac.Write([]byte(os.Args[2]))
        want := mac.Sum(nil)
        fmt.Println("result: ",base64.URLEncoding.EncodeToString(want))
}
```

```shell
$ go run main.go "test" "https://www.google.fr/images/srpr/logo11w.png"
result:  RYifAJRfbhsitJeOrDNxWURCCkPsVR4ihCPXNv-ePbA=
```

### OpenSSL

```shell
$ echo -n "https://www.google.fr/images/srpr/logo11w.png" | openssl dgst -sha256 -hmac "test" -binary|base64| tr '/+' '_-'
RYifAJRfbhsitJeOrDNxWURCCkPsVR4ihCPXNv-ePbA=
```

### Java

```java
import org.apache.commons.codec.binary.Base64;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;

class EncodeUrl {

  public static String encode(String key, String data) throws Exception {
    Mac sha256_HMAC = Mac.getInstance("HmacSHA256");
    SecretKeySpec secret_key = new SecretKeySpec(key.getBytes(), "HmacSHA256");
    sha256_HMAC.init(secret_key);

    return Base64.encodeBase64URLSafeString(sha256_HMAC.doFinal(data.getBytes()));
  }

  public static void main(String [] args) throws Exception {
    System.out.println(encode(args[0], args[1]));
  }

}
```

```shell
$ java -cp commons-codec-1.10.jar:. EncodeUrl test https://www.google.fr/images/srpr/logo11w.png
RYifAJRfbhsitJeOrDNxWURCCkPsVR4ihCPXNv-ePbA
```

### Ruby

```ruby
require 'openssl'
require 'base64'

key = "test"
data = "https://www.google.fr/images/srpr/logo11w.png"
puts Base64.urlsafe_encode64(OpenSSL::HMAC.digest(OpenSSL::Digest.new('sha256'), key, data)).strip()
```

```shell
% ruby sign.rb
RYifAJRfbhsitJeOrDNxWURCCkPsVR4ihCPXNv-ePbA=
```

### Python

```python
import hmac
import hashlib
import base64

key = 'secret key'
data = 'https://octodex.github.com/images/codercat.jpg'
print base64.urlsafe_b64encode(hmac.new(key, msg=data, digestmod=hashlib.sha256).digest()) 
```

### JavaScript

```javascript
import crypto from 'crypto';
import URLSafeBase64 from 'urlsafe-base64';

let key = 'secret key';
let data = 'https://octodex.github.com/images/codercat.jpg';
console.log(URLSafeBase64.encode(crypto.createHmac('sha256', key).update(data).digest()));
```

### PHP

````php
<?php
$key = 'test';
$data = "https://www.google.fr/images/srpr/logo11w.png";
echo strtr(base64_encode(hash_hmac('sha256', $data, $key, 1)), '/+' , '_-');
````

````shell
$ php ex.php
RYifAJRfbhsitJeOrDNxWURCCkPsVR4ihCPXNv-ePbA=
````
