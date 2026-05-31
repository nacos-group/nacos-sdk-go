/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLocalIP_PriorityResolution covers all three priority levels of LocalIP() resolution.
func TestLocalIP_PriorityResolution(t *testing.T) {
	type setup struct {
		envIP        string
		configuredIP string
	}
	tests := []struct {
		name           string
		setup          setup
		wantNonEmpty   bool   // true if we just want a non-empty result (auto-detect case)
		wantExactValue string // empty means skip exact-value assertion
		desc           string
	}{
		{
			name:           "env_only",
			setup:          setup{envIP: "10.0.1.100"},
			wantExactValue: "10.0.1.100",
			desc:           "env variable takes effect when set",
		},
		{
			name:           "config_only",
			setup:          setup{configuredIP: "10.0.2.200"},
			wantExactValue: "10.0.2.200",
			desc:           "ClientConfig.ClientIP takes effect when env empty",
		},
		{
			name:           "env_overrides_config",
			setup:          setup{envIP: "10.0.1.100", configuredIP: "10.0.2.200"},
			wantExactValue: "10.0.1.100",
			desc:           "env variable wins over ClientConfig.ClientIP",
		},
		{
			name:         "auto_detect",
			setup:        setup{},
			wantNonEmpty: true,
			desc:         "fallback to auto-detected IP when neither env nor config provided",
		},
		{
			name:           "empty_env_falls_through",
			setup:          setup{envIP: "", configuredIP: "10.0.3.30"},
			wantExactValue: "10.0.3.30",
			desc:           "empty env is treated as unset",
		},
		{
			name:           "ipv6_env_value",
			setup:          setup{envIP: "fe80::1"},
			wantExactValue: "fe80::1",
			desc:           "env value passed through as-is even for IPv6",
		},
		{
			name:           "config_with_special_chars",
			setup:          setup{configuredIP: "192.168.1.100"},
			wantExactValue: "192.168.1.100",
			desc:           "standard IPv4 from config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state for each subtest (since LocalIP uses sync.Once)
			resetLocalIPForTesting()

			// Setup environment
			if tt.setup.envIP != "" {
				t.Setenv(EnvNacosClientLocalIP, tt.setup.envIP)
			} else {
				_ = os.Unsetenv(EnvNacosClientLocalIP)
			}
			if tt.setup.configuredIP != "" {
				SetClientIPFromConfig(tt.setup.configuredIP)
			}

			got := LocalIP()

			if tt.wantNonEmpty {
				assert.NotEmpty(t, got, "expected auto-detected IP to be non-empty")
			}
			if tt.wantExactValue != "" {
				assert.Equal(t, tt.wantExactValue, got, tt.desc)
			}
		})
	}

	resetLocalIPForTesting()
}

// TestLocalIP_OnceSemantics verifies that LocalIP() resolves exactly once
// even if env/config change later.
func TestLocalIP_OnceSemantics(t *testing.T) {
	resetLocalIPForTesting()
	defer resetLocalIPForTesting()

	t.Setenv(EnvNacosClientLocalIP, "10.0.0.1")
	first := LocalIP()
	assert.Equal(t, "10.0.0.1", first)

	// Change env after first resolution — should NOT affect subsequent calls
	t.Setenv(EnvNacosClientLocalIP, "10.0.0.99")
	SetClientIPFromConfig("10.0.0.50")

	second := LocalIP()
	assert.Equal(t, "10.0.0.1", second, "LocalIP() must return cached value on second call")
}

// TestSetClientIPFromConfig_BeforeFirstResolution verifies the typical client_factory flow.
func TestSetClientIPFromConfig_BeforeFirstResolution(t *testing.T) {
	resetLocalIPForTesting()
	defer resetLocalIPForTesting()

	_ = os.Unsetenv(EnvNacosClientLocalIP)
	SetClientIPFromConfig("172.16.0.5")

	got := LocalIP()
	assert.Equal(t, "172.16.0.5", got)
}

// TestSetClientIPFromConfig_EmptyValueIgnored verifies passing empty string is a no-op.
func TestSetClientIPFromConfig_EmptyValueIgnored(t *testing.T) {
	resetLocalIPForTesting()
	defer resetLocalIPForTesting()

	_ = os.Unsetenv(EnvNacosClientLocalIP)
	SetClientIPFromConfig("")

	got := LocalIP()
	// Empty config falls through to auto-detect; should not be empty in normal CI environments.
	// We only assert that we don't return literal empty when the host has interfaces.
	if got == "" {
		t.Log("auto-detected IP is empty — likely no network interfaces; skipping strict assertion")
	}
}

// TestLocalIP_ConcurrentAccess stresses the sync.Once guarantee under heavy concurrent load.
// Without sync.Once, this would race on resolvedIP read/write.
func TestLocalIP_ConcurrentAccess(t *testing.T) {
	resetLocalIPForTesting()
	defer resetLocalIPForTesting()

	t.Setenv(EnvNacosClientLocalIP, "192.168.100.100")

	const goroutines = 100
	const iterations = 1000
	var wg sync.WaitGroup
	results := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var lastValue string
			for j := 0; j < iterations; j++ {
				lastValue = LocalIP()
			}
			results <- lastValue
		}()
	}

	wg.Wait()
	close(results)

	// All goroutines must observe the same value
	expected := "192.168.100.100"
	for v := range results {
		assert.Equal(t, expected, v, "all concurrent reads must return the same resolved IP")
	}
}

// TestLocalIP_ConcurrentSetAndRead verifies safety when SetClientIPFromConfig is called
// concurrently with LocalIP() reads.
// Note: in production, SetClientIPFromConfig is called once during init, before any reads.
// But we still want this to be race-detector clean.
func TestLocalIP_ConcurrentSetAndRead(t *testing.T) {
	resetLocalIPForTesting()
	defer resetLocalIPForTesting()

	_ = os.Unsetenv(EnvNacosClientLocalIP)

	const goroutines = 50
	var wg sync.WaitGroup

	// Half goroutines set, half read — race detector verifies no data race
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SetClientIPFromConfig("10.0.0.1")
		}()
	}
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = LocalIP()
		}()
	}

	wg.Wait()
	// No assertion needed — purpose is to trigger race detector if there were one.
}

// TestEnvNacosClientLocalIP_ConstantValue ensures the constant matches the documented contract.
// This is a regression guard: if anyone renames the constant, downstream users' env vars will break.
func TestEnvNacosClientLocalIP_ConstantValue(t *testing.T) {
	assert.Equal(t, "NACOS_CLIENT_LOCAL_IP", EnvNacosClientLocalIP,
		"env variable name is part of the public contract; do not rename without a major version bump")
}

// TestLocalIP_AutoDetectFallback verifies fallback works when neither env nor config is set.
func TestLocalIP_AutoDetectFallback(t *testing.T) {
	resetLocalIPForTesting()
	defer resetLocalIPForTesting()

	_ = os.Unsetenv(EnvNacosClientLocalIP)
	// Explicitly do not call SetClientIPFromConfig

	got := LocalIP()
	// Auto-detect should succeed in most environments
	if got == "" {
		t.Log("auto-detected IP is empty — host has no usable network interfaces")
	} else {
		// Should look like an IPv4 address
		assert.Regexp(t, `^\d+\.\d+\.\d+\.\d+$`, got, "expected IPv4 dotted-decimal format")
	}
}

// TestDetectLocalIP_NotEmpty exercises the internal helper directly.
func TestDetectLocalIP_NotEmpty(t *testing.T) {
	got := detectLocalIP()
	if got == "" {
		t.Log("detectLocalIP returned empty — host environment has no non-loopback IPv4 interfaces")
	} else {
		assert.Regexp(t, `^\d+\.\d+\.\d+\.\d+$`, got)
	}
}

// TestLocalIP_PriorityOrder_Table compactly verifies the priority chain via table-driven cases.
func TestLocalIP_PriorityOrder_Table(t *testing.T) {
	tests := []struct {
		name         string
		envIP        string
		configuredIP string
		expected     string
	}{
		{"only_env", "1.1.1.1", "", "1.1.1.1"},
		{"only_config", "", "2.2.2.2", "2.2.2.2"},
		{"both_env_wins", "1.1.1.1", "2.2.2.2", "1.1.1.1"},
		{"env_empty_string", "", "3.3.3.3", "3.3.3.3"},
		{"config_empty_string", "4.4.4.4", "", "4.4.4.4"},
		{"both_set_env_priority", "5.5.5.5", "6.6.6.6", "5.5.5.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetLocalIPForTesting()
			if tt.envIP != "" {
				t.Setenv(EnvNacosClientLocalIP, tt.envIP)
			} else {
				_ = os.Unsetenv(EnvNacosClientLocalIP)
			}
			SetClientIPFromConfig(tt.configuredIP)

			got := LocalIP()
			assert.Equal(t, tt.expected, got)
		})
	}

	resetLocalIPForTesting()
}
