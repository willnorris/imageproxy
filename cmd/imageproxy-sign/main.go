// The imageproxy-sign tool creates signature values for a provided URL and
// signing key.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"willnorris.com/go/imageproxy"
)

var key = flag.String("key", "@/etc/imageproxy.key", "signing key, or file containing key prefixed with '@'")
var urlOnly = flag.Bool("url", false, "only sign the URL value, do not include options")

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("imageproxy-sign url [key]")
		os.Exit(1)
	}

	u := parseURL(flag.Arg(0))
	if u == nil {
		fmt.Printf("unable to parse URL: %v\n", flag.Arg(0))
		os.Exit(1)
	}
	if *urlOnly {
		u.Fragment = ""
	}

	k, err := parseKey(*key)
	if err != nil {
		fmt.Printf("error parsing key: %v", err)
		os.Exit(1)
	}

	mac := hmac.New(sha256.New, []byte(k))
	mac.Write([]byte(u.String()))
	sig := mac.Sum(nil)

	fmt.Printf("url: %v\n", u)
	fmt.Printf("signature: %v\n", base64.URLEncoding.EncodeToString(sig))
}

func parseKey(s string) ([]byte, error) {
	if strings.HasPrefix(s, "@") {
		return ioutil.ReadFile(s[1:])
	}
	return []byte(s), nil
}

// parseURL parses s as either an imageproxy request URL or a remote URL with
// options in the URL fragment.  Any existing signature values are stripped,
// and the final remote URL returned with remaining options in the fragment.
func parseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if s == "" || err != nil {
		return nil
	}

	// first try to parse this as an imageproxy URL, containing
	// transformation options and the remote URL embedded
	if r, err := imageproxy.NewRequest(&http.Request{URL: u}, nil); err == nil {
		r.Options.Signature = ""
		r.URL.Fragment = r.Options.String()
		return r.URL
	}

	// second, we assume that this is the remote URL itself. If a fragment
	// is present, treat it as an option string.
	opt := imageproxy.ParseOptions(u.Fragment)
	opt.Signature = ""
	u.Fragment = opt.String()
	return u
}
