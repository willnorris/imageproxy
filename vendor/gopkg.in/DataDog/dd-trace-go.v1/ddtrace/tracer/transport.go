// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package tracer

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/version"
)

var (
	// We copy the transport to avoid using the default one, as it might be
	// augmented with tracing and we don't want these calls to be recorded.
	// See https://golang.org/pkg/net/http/#DefaultTransport .
	defaultRoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
)

const (
	defaultHostname    = "localhost"
	defaultPort        = "8126"
	defaultAddress     = defaultHostname + ":" + defaultPort
	defaultHTTPTimeout = time.Second             // defines the current timeout before giving up with the send process
	traceCountHeader   = "X-Datadog-Trace-Count" // header containing the number of traces in the payload
)

// transport is an interface for span submission to the agent.
type transport interface {
	// send sends the payload p to the agent using the transport set up.
	// It returns a non-nil response body when no error occurred.
	send(p *payload) (body io.ReadCloser, err error)
}

// newTransport returns a new Transport implementation that sends traces to a
// trace agent running on the given hostname and port, using a given
// http.RoundTripper. If the zero values for hostname and port are provided,
// the default values will be used ("localhost" for hostname, and "8126" for
// port). If roundTripper is nil, a default is used.
//
// In general, using this method is only necessary if you have a trace agent
// running on a non-default port, if it's located on another machine, or when
// otherwise needing to customize the transport layer, for instance when using
// a unix domain socket.
func newTransport(addr string, roundTripper http.RoundTripper) transport {
	if roundTripper == nil {
		roundTripper = defaultRoundTripper
	}
	return newHTTPTransport(addr, roundTripper)
}

// newDefaultTransport return a default transport for this tracing client
func newDefaultTransport() transport {
	return newHTTPTransport(defaultAddress, defaultRoundTripper)
}

type httpTransport struct {
	traceURL string            // the delivery URL for traces
	client   *http.Client      // the HTTP client used in the POST
	headers  map[string]string // the Transport headers
}

// newHTTPTransport returns an httpTransport for the given endpoint
func newHTTPTransport(addr string, roundTripper http.RoundTripper) *httpTransport {
	// initialize the default EncoderPool with Encoder headers
	defaultHeaders := map[string]string{
		"Datadog-Meta-Lang":             "go",
		"Datadog-Meta-Lang-Version":     strings.TrimPrefix(runtime.Version(), "go"),
		"Datadog-Meta-Lang-Interpreter": runtime.Compiler + "-" + runtime.GOARCH + "-" + runtime.GOOS,
		"Datadog-Meta-Tracer-Version":   version.Tag,
		"Content-Type":                  "application/msgpack",
	}
	f, err := os.Open("/proc/self/cgroup")
	if err == nil {
		if id, ok := readContainerID(f); ok {
			defaultHeaders["Datadog-Container-ID"] = id
		}
		f.Close()
	}
	return &httpTransport{
		traceURL: fmt.Sprintf("http://%s/v0.4/traces", resolveAddr(addr)),
		client: &http.Client{
			Transport: roundTripper,
			Timeout:   defaultHTTPTimeout,
		},
		headers: defaultHeaders,
	}
}

var (
	// expLine matches a line in the /proc/self/cgroup file. It has a submatch for the last element (path), which contains the container ID.
	expLine = regexp.MustCompile(`^\d+:[^:]*:(.+)$`)
	// expContainerID matches contained IDs and sources. Source: https://github.com/Qard/container-info/blob/master/index.js
	expContainerID = regexp.MustCompile(`([0-9a-f]{8}[-_][0-9a-f]{4}[-_][0-9a-f]{4}[-_][0-9a-f]{4}[-_][0-9a-f]{12}|[0-9a-f]{64})(?:.scope)?$`)
)

// readContainerID finds the first container ID reading from r and returns it.
func readContainerID(r io.Reader) (id string, ok bool) {
	scn := bufio.NewScanner(r)
	for scn.Scan() {
		path := expLine.FindStringSubmatch(scn.Text())
		if len(path) != 2 {
			// invalid entry, continue
			continue
		}
		if id := expContainerID.FindString(path[1]); id != "" {
			return id, true
		}
	}
	return "", false
}

func (t *httpTransport) send(p *payload) (body io.ReadCloser, err error) {
	req, err := http.NewRequest("POST", t.traceURL, p)
	if err != nil {
		return nil, fmt.Errorf("cannot create http request: %v", err)
	}
	for header, value := range t.headers {
		req.Header.Set(header, value)
	}
	req.Header.Set(traceCountHeader, strconv.Itoa(p.itemCount()))
	req.Header.Set("Content-Length", strconv.Itoa(p.size()))
	response, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	p.waitClose()
	if code := response.StatusCode; code >= 400 {
		// error, check the body for context information and
		// return a nice error.
		msg := make([]byte, 1000)
		n, _ := response.Body.Read(msg)
		response.Body.Close()
		txt := http.StatusText(code)
		if n > 0 {
			return nil, fmt.Errorf("%s (Status: %s)", msg[:n], txt)
		}
		return nil, fmt.Errorf("%s", txt)
	}
	return response.Body, nil
}

// resolveAddr resolves the given agent address and fills in any missing host
// and port using the defaults. Some environment variable settings will
// take precedence over configuration.
func resolveAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// no port in addr
		host = addr
	}
	if host == "" {
		host = defaultHostname
	}
	if port == "" {
		port = defaultPort
	}
	if v := os.Getenv("DD_AGENT_HOST"); v != "" {
		host = v
	}
	if v := os.Getenv("DD_TRACE_AGENT_PORT"); v != "" {
		port = v
	}
	return fmt.Sprintf("%s:%s", host, port)
}
