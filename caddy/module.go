// Package caddy provides ImageProxy as a Caddy module.
package caddy

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	caddy "github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
	"go.uber.org/zap"
	"willnorris.com/go/imageproxy"
)

func init() {
	caddy.RegisterModule(ImageProxy{})
	httpcaddyfile.RegisterHandlerDirective("imageproxy", parseCaddyfile)
}

type ImageProxy struct {
	Cache string `json:"cache,omitempty"`

	DefaultBaseURL string `json:"default_base_url,omitempty"`

	AllowHosts   []string `json:"allow_hosts,omitempty"`
	DenyHosts    []string `json:"deny_hosts,omitempty"`
	Referrers    []string `json:"referrers,omitempty"`
	ContentTypes []string `json:"content_types,omitempty"`

	SignatureKeys []string `json:"signature_keys,omitempty"`
	Verbose       bool     `json:"verbose,omitempty"`

	logger *zap.Logger
	proxy  *imageproxy.Proxy
}

// interface guard
var (
	_ caddyhttp.MiddlewareHandler = (*ImageProxy)(nil)
)

// CaddyModule returns the Caddy module information.
func (ImageProxy) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.imageproxy",
		New: func() caddy.Module { return new(ImageProxy) },
	}
}

func (p *ImageProxy) Provision(ctx caddy.Context) error {
	p.logger = ctx.Logger()
	cache, _ := parseCache(p.Cache)
	p.proxy = imageproxy.NewProxy(nil, cache)
	p.proxy.DefaultBaseURL, _ = url.Parse(p.DefaultBaseURL)
	p.proxy.AllowHosts = p.AllowHosts
	p.proxy.DenyHosts = p.DenyHosts
	p.proxy.Referrers = p.Referrers
	p.proxy.ContentTypes = p.ContentTypes
	if len(p.proxy.ContentTypes) == 0 {
		p.proxy.ContentTypes = []string{"image/*"}
	}
	for _, key := range p.SignatureKeys {
		p.proxy.SignatureKeys = append(p.proxy.SignatureKeys, []byte(key))
	}
	p.proxy.Logger = zap.NewStdLog(p.logger)
	p.proxy.Verbose = p.Verbose
	p.proxy.FollowRedirects = true
	return nil
}

func (p *ImageProxy) ServeHTTP(w http.ResponseWriter, r *http.Request, _ caddyhttp.Handler) error {
	p.proxy.ServeHTTP(w, r)
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	p := new(ImageProxy)

	h.Next() // consume the directive name
	for nesting := h.Nesting(); h.NextBlock(nesting); {
		switch h.Val() {
		case "cache":
			if !h.NextArg() {
				return nil, h.ArgErr()
			}
			p.Cache = h.Val()
		case "default_base_url":
			if !h.NextArg() {
				return nil, h.ArgErr()
			}
			p.DefaultBaseURL = h.Val()
		case "allow_hosts":
			if !h.NextArg() {
				return nil, h.ArgErr()
			}
			p.AllowHosts = append(p.AllowHosts, strings.Split(h.Val(), ",")...)
		case "deny_hosts":
			if !h.NextArg() {
				return nil, h.ArgErr()
			}
			p.DenyHosts = append(p.DenyHosts, strings.Split(h.Val(), ",")...)
		case "referrers":
			if !h.NextArg() {
				return nil, h.ArgErr()
			}
			p.Referrers = append(p.Referrers, strings.Split(h.Val(), ",")...)
		case "content_types":
			if !h.NextArg() {
				return nil, h.ArgErr()
			}
			p.ContentTypes = append(p.ContentTypes, strings.Split(h.Val(), ",")...)
		case "signature_key":
			if !h.NextArg() {
				return nil, h.ArgErr()
			}
			p.SignatureKeys = append(p.SignatureKeys, h.Val())
		case "verbose":
			if !h.NextArg() {
				return nil, h.ArgErr()
			}
			p.Verbose, _ = strconv.ParseBool(h.Val())
		}
	}
	return p, nil
}

// parseCache parses c returns the specified Cache implementation.
func parseCache(c string) (imageproxy.Cache, error) {
	const defaultMemorySize = 100

	if c == "" {
		return nil, nil
	}

	if c == "memory" {
		c = fmt.Sprintf("memory:%d", defaultMemorySize)
	}

	u, err := url.Parse(c)
	if err != nil {
		return nil, fmt.Errorf("error parsing cache flag: %w", err)
	}

	switch u.Scheme {
	case "file":
		return diskCache(u.Path), nil
	default:
		return diskCache(c), nil
	}
}

func diskCache(path string) *diskcache.Cache {
	d := diskv.New(diskv.Options{
		BasePath: path,

		// For file "c0ffee", store file as "c0/ff/c0ffee"
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
	return diskcache.NewWithDiskv(d)
}
