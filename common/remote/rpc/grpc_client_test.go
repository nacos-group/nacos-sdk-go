package rpc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
)

func TestResolveGrpcAddress(t *testing.T) {
	tests := []struct {
		name     string
		serverIp string
		port     uint64
		want     string
	}{
		// IPv4 cases
		{
			name:     "IPv4 address",
			serverIp: "127.0.0.1",
			port:     9848,
			want:     "passthrough:///127.0.0.1:9848",
		},
		{
			name:     "IPv4 private address",
			serverIp: "10.0.0.1",
			port:     9848,
			want:     "passthrough:///10.0.0.1:9848",
		},
		{
			name:     "IPv4 with different port",
			serverIp: "192.168.1.100",
			port:     19848,
			want:     "passthrough:///192.168.1.100:19848",
		},

		// IPv6 cases
		{
			name:     "bare IPv6 loopback",
			serverIp: "::1",
			port:     9848,
			want:     "passthrough:///[::1]:9848",
		},
		{
			name:     "bracketed IPv6 loopback",
			serverIp: "[::1]",
			port:     9848,
			want:     "passthrough:///[::1]:9848",
		},
		{
			name:     "bare IPv6 full address",
			serverIp: "2001:db8::1",
			port:     9848,
			want:     "passthrough:///[2001:db8::1]:9848",
		},
		{
			name:     "bracketed IPv6 full address",
			serverIp: "[2001:db8::1]",
			port:     9848,
			want:     "passthrough:///[2001:db8::1]:9848",
		},
		{
			name:     "IPv4-mapped IPv6",
			serverIp: "::ffff:127.0.0.1",
			port:     9848,
			want:     "passthrough:///[::ffff:127.0.0.1]:9848",
		},
		{
			name:     "bracketed IPv4-mapped IPv6",
			serverIp: "[::ffff:127.0.0.1]",
			port:     9848,
			want:     "passthrough:///[::ffff:127.0.0.1]:9848",
		},

		// Domain name cases
		{
			name:     "localhost domain",
			serverIp: "localhost",
			port:     9848,
			want:     "dns:///localhost:9848",
		},
		{
			name:     "FQDN domain",
			serverIp: "nacos.example.com",
			port:     9848,
			want:     "dns:///nacos.example.com:9848",
		},
		{
			name:     "Kubernetes service domain",
			serverIp: "nacos-server.nacos.svc.cluster.local",
			port:     9848,
			want:     "dns:///nacos-server.nacos.svc.cluster.local:9848",
		},

		// http:// prefix cases (user misconfiguration)
		{
			name:     "http prefix with IPv4",
			serverIp: "http://127.0.0.1",
			port:     9848,
			want:     "passthrough:///127.0.0.1:9848",
		},
		{
			name:     "http prefix with domain",
			serverIp: "http://nacos.example.com",
			port:     9848,
			want:     "dns:///nacos.example.com:9848",
		},
		{
			name:     "http prefix with localhost",
			serverIp: "http://localhost",
			port:     9848,
			want:     "dns:///localhost:9848",
		},
		{
			name:     "https prefix with IPv4",
			serverIp: "https://10.0.0.1",
			port:     9848,
			want:     "passthrough:///10.0.0.1:9848",
		},
		{
			name:     "https prefix with domain",
			serverIp: "https://nacos.example.com",
			port:     9848,
			want:     "dns:///nacos.example.com:9848",
		},
		{
			name:     "http prefix with bracketed IPv6",
			serverIp: "http://[::1]",
			port:     9848,
			want:     "passthrough:///[::1]:9848",
		},

		// Edge cases
		{
			name:     "empty string defaults to dns",
			serverIp: "",
			port:     9848,
			want:     "dns:///:9848",
		},
		{
			name:     "port zero",
			serverIp: "127.0.0.1",
			port:     0,
			want:     "passthrough:///127.0.0.1:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveGrpcAddress(tt.serverIp, tt.port)
			if got != tt.want {
				t.Errorf("resolveGrpcAddress(%q, %d) = %q, want %q", tt.serverIp, tt.port, got, tt.want)
			}
		})
	}
}

// generateTestCerts creates a self-signed CA cert and a client cert/key pair for testing.
// Returns paths to: caFile, certFile, keyFile
func generateTestCerts(t *testing.T, dir string) (string, string, string) {
	t.Helper()

	// Generate CA key and cert
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Test CA"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caFile := filepath.Join(dir, "ca.pem")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	if err := os.WriteFile(caFile, caPEM, 0644); err != nil {
		t.Fatal(err)
	}

	// Generate client key and cert signed by CA
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{Organization: []string{"Test Client"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	certFile := filepath.Join(dir, "client.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER})
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		t.Fatal(err)
	}

	keyFile := filepath.Join(dir, "client-key.pem")
	keyDER, err := x509.MarshalECPrivateKey(clientKey)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyFile, keyPEM, 0644); err != nil {
		t.Fatal(err)
	}

	return caFile, certFile, keyFile
}

func TestGetTLSCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	caFile, certFile, keyFile := generateTestCerts(t, tmpDir)

	// Create an invalid PEM file (not valid certificate data)
	invalidPEMFile := filepath.Join(tmpDir, "invalid.pem")
	if err := os.WriteFile(invalidPEMFile, []byte("not a valid PEM"), 0644); err != nil {
		t.Fatal(err)
	}

	serverInfo := ServerInfo{serverIp: "127.0.0.1", serverPort: 8848}

	tests := []struct {
		name      string
		tlsConfig *constant.TLSConfig
		wantErr   bool
		errMsg    string
	}{
		{
			name: "no CA, no client cert, TrustAll=true",
			tlsConfig: &constant.TLSConfig{
				Enable:   true,
				TrustAll: true,
			},
			wantErr: false,
		},
		{
			name: "no CA, no client cert, TrustAll=false (use system CA pool)",
			tlsConfig: &constant.TLSConfig{
				Enable:   true,
				TrustAll: false,
			},
			wantErr: false,
		},
		{
			name: "valid CA file",
			tlsConfig: &constant.TLSConfig{
				Enable: true,
				CaFile: caFile,
			},
			wantErr: false,
		},
		{
			name: "valid CA + valid client cert",
			tlsConfig: &constant.TLSConfig{
				Enable:   true,
				CaFile:   caFile,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
			wantErr: false,
		},
		{
			name: "non-existent CA file",
			tlsConfig: &constant.TLSConfig{
				Enable: true,
				CaFile: "/non/existent/ca.pem",
			},
			wantErr: true,
			errMsg:  "failed to read CA file",
		},
		{
			name: "invalid PEM in CA file",
			tlsConfig: &constant.TLSConfig{
				Enable: true,
				CaFile: invalidPEMFile,
			},
			wantErr: true,
			errMsg:  "failed to parse CA certificates",
		},
		{
			name: "non-existent client cert file",
			tlsConfig: &constant.TLSConfig{
				Enable:   true,
				CertFile: "/non/existent/cert.pem",
				KeyFile:  keyFile,
			},
			wantErr: true,
			errMsg:  "failed to load client certificate",
		},
		{
			name: "non-existent client key file",
			tlsConfig: &constant.TLSConfig{
				Enable:   true,
				CertFile: certFile,
				KeyFile:  "/non/existent/key.pem",
			},
			wantErr: true,
			errMsg:  "failed to load client certificate",
		},
		{
			name: "mismatched cert and key",
			tlsConfig: &constant.TLSConfig{
				Enable:   true,
				CertFile: caFile,
				KeyFile:  keyFile,
			},
			wantErr: true,
			errMsg:  "failed to load client certificate",
		},
		{
			name: "ServerNameOverride set",
			tlsConfig: &constant.TLSConfig{
				Enable:             true,
				TrustAll:           true,
				ServerNameOverride: "nacos.example.com",
			},
			wantErr: false,
		},
		{
			name: "only CertFile without KeyFile (should skip client cert)",
			tlsConfig: &constant.TLSConfig{
				Enable:   true,
				TrustAll: true,
				CertFile: certFile,
			},
			wantErr: false,
		},
		{
			name: "only KeyFile without CertFile (should skip client cert)",
			tlsConfig: &constant.TLSConfig{
				Enable:   true,
				TrustAll: true,
				KeyFile:  keyFile,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := getTLSCredentials(tt.tlsConfig, serverInfo)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsStr(err.Error(), tt.errMsg) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errMsg)
				}
				if creds != nil {
					t.Error("expected nil credentials on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if creds == nil {
					t.Error("expected non-nil credentials")
				}
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
