package httpcache

import (
	"net/http"
	"strings"
)

type CacheControl map[string]string

func ParseCacheControl(headers http.Header) CacheControl {
	cc := CacheControl{}
	ccHeader := headers.Get("Cache-Control")
	for _, part := range strings.Split(ccHeader, ",") {
		part = strings.Trim(part, " ")
		if part == "" {
			continue
		}
		if strings.ContainsRune(part, '=') {
			keyval := strings.Split(part, "=")
			cc[strings.Trim(keyval[0], " ")] = strings.Trim(keyval[1], ",")
		} else {
			cc[part] = ""
		}
	}
	return cc
}

func (cc CacheControl) String() string {
	parts := make([]string, 0, len(cc))
	for k, v := range cc {
		if v == "" {
			parts = append(parts, k)
		} else {
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, ", ")
}
