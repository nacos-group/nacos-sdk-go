package rpc

import (
	"context"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// TestResolveGrpcAddressIntegration tests actual gRPC connectivity using resolveGrpcAddress.
// It requires a local Nacos server running on 127.0.0.1:9848 (gRPC port).
// Set NACOS_INTEGRATION_TEST=1 to enable these tests.
func TestResolveGrpcAddressIntegration(t *testing.T) {
	if os.Getenv("NACOS_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test: set NACOS_INTEGRATION_TEST=1 to enable")
	}

	tests := []struct {
		name     string
		serverIp string
		port     uint64
		wantOK   bool
	}{
		// Should connect successfully
		{"IPv4 direct", "127.0.0.1", 9848, true},
		{"localhost domain", "localhost", 9848, true},
		{"http prefix IPv4", "http://127.0.0.1", 9848, true},
		{"http prefix localhost", "http://localhost", 9848, true},
		{"bracketed IPv6 loopback", "[::1]", 9848, true},
		{"bare IPv6 loopback", "::1", 9848, true},

		// Should fail to connect (unreachable or invalid)
		{"unreachable IP", "192.168.255.255", 9848, false},
		{"non-existent domain", "this-host-does-not-exist.invalid", 9848, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := resolveGrpcAddress(tt.serverIp, tt.port)
			t.Logf("resolveGrpcAddress(%q, %d) = %q", tt.serverIp, tt.port, address)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn, err := grpc.NewClient(address,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				if tt.wantOK {
					t.Fatalf("grpc.NewClient failed: %v", err)
				}
				return
			}
			defer conn.Close()

			conn.Connect()

			connected := false
			for {
				state := conn.GetState()
				if state == connectivity.Ready {
					connected = true
					break
				}
				if state == connectivity.TransientFailure || state == connectivity.Shutdown {
					break
				}
				if !conn.WaitForStateChange(ctx, state) {
					break
				}
			}

			if tt.wantOK && !connected {
				t.Errorf("Expected successful connection for %q, but failed (target=%q)", tt.serverIp, address)
			}
			if !tt.wantOK && connected {
				t.Errorf("Expected connection failure for %q, but succeeded (target=%q)", tt.serverIp, address)
			}
		})
	}
}
