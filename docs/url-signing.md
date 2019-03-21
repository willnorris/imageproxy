# How to generate signed requests

## Go

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

## OpenSSL

```shell
$ echo -n "https://www.google.fr/images/srpr/logo11w.png" | openssl dgst -sha256 -hmac "test" -binary|base64| tr '/+' '_-'
RYifAJRfbhsitJeOrDNxWURCCkPsVR4ihCPXNv-ePbA=
```

## Java

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

## Ruby

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

## Python

```python
import hmac
import hashlib
import base64

key = 'secret key'
data = 'https://octodex.github.com/images/codercat.jpg'
print base64.urlsafe_b64encode(hmac.new(key, msg=data, digestmod=hashlib.sha256).digest()) 
```

## JavaScript

```javascript
import crypto from 'crypto';
import URLSafeBase64 from 'urlsafe-base64';

let key = 'secret key';
let data = 'https://octodex.github.com/images/codercat.jpg';
console.log(URLSafeBase64.encode(crypto.createHmac('sha256', key).update(data).digest()));
```

## PHP

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