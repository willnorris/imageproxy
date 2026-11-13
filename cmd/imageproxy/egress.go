// Copyright 2026 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

var errAddressNotAllowed = errors.New("remote address is not allowed")

// nonPublicIPv4 are the IANA special-purpose IPv4 ranges that are not
// globally reachable, plus multicast and reserved space. All other IPv4
// addresses are public.
var nonPublicIPv4 = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),       // "this network"
	netip.MustParsePrefix("10.0.0.0/8"),      // private
	netip.MustParsePrefix("100.64.0.0/10"),   // carrier-grade NAT
	netip.MustParsePrefix("127.0.0.0/8"),     // loopback
	netip.MustParsePrefix("169.254.0.0/16"),  // link-local, includes cloud metadata
	netip.MustParsePrefix("172.16.0.0/12"),   // private
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // documentation
	netip.MustParsePrefix("192.88.99.0/24"),  // deprecated 6to4 relay
	netip.MustParsePrefix("192.168.0.0/16"),  // private
	netip.MustParsePrefix("198.18.0.0/15"),   // benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // documentation
	netip.MustParsePrefix("203.0.113.0/24"),  // documentation
	netip.MustParsePrefix("224.0.0.0/4"),     // multicast
	netip.MustParsePrefix("240.0.0.0/4"),     // reserved, includes broadcast
}

// globalUnicastIPv6 is the only IPv6 range allocated for public unicast.
// Everything outside it (loopback, unique local, link-local, site-local,
// multicast, IPv4-compatible and -translated forms...) is refused.
var globalUnicastIPv6 = netip.MustParsePrefix("2000::/3")

// nonPublicIPv6 are the ranges inside 2000::/3 that are not globally
// reachable.
var nonPublicIPv6 = []netip.Prefix{
	netip.MustParsePrefix("2001::/23"),     // IETF protocol assignments, includes Teredo
	netip.MustParsePrefix("2001:db8::/32"), // documentation
	netip.MustParsePrefix("3fff::/20"),     // documentation
}

// IPv6 prefixes that embed an IPv4 address and route to it: NAT64 in the last
// 4 bytes, 6to4 in bytes 2-5. The embedded address is checked instead.
var (
	nat64Prefix = netip.MustParsePrefix("64:ff9b::/96")
	sixToFour   = netip.MustParsePrefix("2002::/16")
)

// isPublic reports whether ip is a publicly routable unicast address.
// Anything not known to be public is refused.
func isPublic(ip netip.Addr) bool {
	if ip.Zone() != "" { // scoped addresses (fe80::1%eth0) are never public
		return false
	}
	ip = ip.Unmap()
	b := ip.As16()
	switch {
	case nat64Prefix.Contains(ip):
		ip = netip.AddrFrom4([4]byte(b[12:16]))
	case sixToFour.Contains(ip):
		ip = netip.AddrFrom4([4]byte(b[2:6]))
	}
	if ip.Is4() {
		return !inAny(nonPublicIPv4, ip)
	}
	return globalUnicastIPv6.Contains(ip) && !inAny(nonPublicIPv6, ip)
}

func inAny(prefixes []netip.Prefix, ip netip.Addr) bool {
	for _, p := range prefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// denyPrivateAddress is a net.Dialer Control function that refuses to connect
// to non-public addresses. It runs after DNS resolution on the address
// actually dialed, so hostnames resolving to private IPs, DNS rebinding and
// redirects are all covered.
func denyPrivateAddress(_, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return err
	}
	if !isPublic(ap.Addr()) {
		return fmt.Errorf("%w: %s", errAddressNotAllowed, ap.Addr())
	}
	return nil
}

// newTransport returns a transport with the same settings as
// http.DefaultTransport, dialing through control. It has no proxy: a proxy
// would be the address checked, not the origin. It also replaces the default
// AIA transport, whose TLS dialer and issuer certificate fetches would bypass
// control, so origins serving an incomplete certificate chain will fail.
func newTransport(control func(network, address string, c syscall.RawConn) error) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   control,
	}
	return &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
