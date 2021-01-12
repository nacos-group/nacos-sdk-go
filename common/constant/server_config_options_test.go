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

package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServerConfig(t *testing.T) {
	config := NewServerConfig("console.nacos.io", 80)

	assert.Equal(t, "console.nacos.io", config.IpAddr)
	assert.Equal(t, uint64(80), config.Port)
	assert.Equal(t, "/nacos", config.ContextPath)
	assert.Equal(t, "http", config.Scheme)
	assert.True(t, config.Port > 0 && config.Port < 65535)
}

func TestNewServerConfigWithOptions(t *testing.T) {
	config := NewServerConfig(
		"console.nacos.io",
		80,
		WithContextPath("/ns"),
		WithScheme("https"),
	)

	assert.Equal(t, "console.nacos.io", config.IpAddr)
	assert.Equal(t, uint64(80), config.Port)
	assert.Equal(t, "/ns", config.ContextPath)
	assert.Equal(t, "https", config.Scheme)
	assert.True(t, config.Port > 0 && config.Port < 65535)
}
