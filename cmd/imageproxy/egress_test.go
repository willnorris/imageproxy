// Copyright 2026 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDenyPrivateAddress(t *testing.T) {
	tests := []struct {
		address string
		blocked bool
	}{
		{"8.8.8.8:443", false},
		{"1.1.1.1:80", false},
		{"[2001:4860:4860::8888]:443", false},
		{"[2606:4700:4700::1111]:443", false},
		{"[::ffff:8.8.8.8]:80", false},   // IPv4-mapped for 8.8.8.8
		{"[64:ff9b::808:808]:80", false}, // NAT64 for 8.8.8.8
		{"[2002:808:808::1]:80", false},  // 6to4 for 8.8.8.8

		{"10.1.2.3:80", true},
		{"172.16.0.1:80", true},
		{"192.168.1.1:80", true},
		{"127.0.0.1:6379", true},
		{"100.64.0.1:80", true},
		{"169.254.169.254:80", true},
		{"0.0.0.0:80", true},
		{"192.0.0.1:80", true},
		{"198.18.0.1:80", true},
		{"224.0.0.1:80", true},
		{"255.255.255.255:80", true},

		{"[::1]:80", true},
		{"[::]:80", true},
		{"[fd00:ec2::254]:80", true},                        // AWS IPv6 metadata
		{"[fd20:ce::254]:80", true},                         // GCP IPv6 metadata
		{"[fe80::1]:80", true},                              // link-local
		{"[fe80::1%eth0]:80", true},                         // zone id
		{"[::1%lo]:80", true},                               // zone id on loopback
		{"[2001:4860:4860::8888%eth0]:80", true},            // zone id on a public address
		{"[fec0::1]:80", true},                              // deprecated site-local
		{"[ff02::1]:80", true},                              // multicast
		{"[::ffff:127.0.0.1]:80", true},                     // IPv4-mapped
		{"[::ffff:a9fe:a9fe]:80", true},                     // IPv4-mapped, hex
		{"[::a01:304]:80", true},                            // IPv4-compatible for 10.1.3.4
		{"[::ffff:0:a01:304]:80", true},                     // IPv4-translated for 10.1.3.4
		{"[64:ff9b::a9fe:a9fe]:80", true},                   // NAT64 for 169.254.169.254
		{"[64:ff9b:1::a01:304]:80", true},                   // local-use NAT64
		{"[2002:a01:304::1]:80", true},                      // 6to4 for 10.1.3.4
		{"[2002:a9fe:a9fe::1]:80", true},                    // 6to4 for 169.254.169.254
		{"[2001:0:4136:e378:8000:63bf:3fff:fdd2]:80", true}, // Teredo
		{"[2001:db8::1]:80", true},                          // documentation
	}
	for _, tt := range tests {
		err := denyPrivateAddress("tcp", tt.address, nil)
		if tt.blocked && !errors.Is(err, errAddressNotAllowed) {
			t.Errorf("denyPrivateAddress(%q) = %v, want errAddressNotAllowed", tt.address, err)
		}
		if !tt.blocked && err != nil {
			t.Errorf("denyPrivateAddress(%q) = %v, want nil", tt.address, err)
		}
	}
}

func TestNewTransport(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer origin.Close()
	port := origin.Listener.Addr().(*net.TCPAddr).Port
	// literal IP, and a hostname that resolves to a loopback address
	urls := []string{origin.URL, fmt.Sprintf("http://localhost:%d/", port)}

	allowed := &http.Client{Transport: newTransport(nil)}
	denied := &http.Client{Transport: newTransport(denyPrivateAddress)}
	for _, url := range urls {
		resp, err := allowed.Get(url)
		if err != nil {
			t.Fatalf("GET %s without control: %v", url, err)
		}
		resp.Body.Close()

		if resp, err := denied.Get(url); !errors.Is(err, errAddressNotAllowed) {
			if err == nil {
				resp.Body.Close()
			}
			t.Errorf("GET %s = %v, want errAddressNotAllowed", url, err)
		}
	}
}
