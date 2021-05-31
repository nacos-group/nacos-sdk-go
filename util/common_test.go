package util

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	if err := SetFilterNetNumberAndMask([]string{
		"10.0.0.0/8",     // RFC 1918 IPv4 private network address
		"100.64.0.0/10",  // RFC 6598 IPv4 shared address space
		"127.0.0.0/8",    // RFC 1122 IPv4 loopback address
		"169.254.0.0/16", // RFC 3927 IPv4 link local address
		"172.16.0.0/12",  // RFC 1918 IPv4 private network address
		"192.0.0.0/24",   // RFC 6890 IPv4 IANA address
		"192.0.2.0/24",   // RFC 5737 IPv4 documentation address
		"192.168.0.0/16", // RFC 1918 IPv4 private network address
	}...); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		ip      string
		private bool
	}{
		// IPv4 private addresses
		{"10.0.0.1", true},    // private network address
		{"100.64.0.1", true},  // shared address space
		{"172.16.0.1", true},  // private network address
		{"192.168.0.1", true}, // private network address
		{"192.0.0.1", true},   // IANA address
		{"192.0.2.1", true},   // documentation address
		{"127.0.0.1", true},   // loopback address
		{"127.1.0.1", true},   // loopback address
		{"169.254.0.1", true}, // link local address

		// IPv4 public addresses
		{"1.2.3.4", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("%s is not a valid ip address", tt.ip)
			}
			if got, want := isFilteredIP(ip), tt.private; got != want {
				t.Fatalf("got %v for %v want %v", got, ip, want)
			}
		})
	}
}
