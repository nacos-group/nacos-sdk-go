package rpc

import "testing"

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
