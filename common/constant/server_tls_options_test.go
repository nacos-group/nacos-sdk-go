package constant

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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTLSConfigWithOptions(t *testing.T) {
	t.Run("TestNoOption", func(t *testing.T) {
		cfg := SkipVerifyConfig
		assert.Equal(t, "", cfg.CaFile)
		assert.Equal(t, "", cfg.CertFile)
		assert.Equal(t, "", cfg.KeyFile)
		assert.Equal(t, "", cfg.ServerNameOverride)
	})

	t.Run("TestCAOption", func(t *testing.T) {
		cfg := NewTLSConfig(
			WithCA("ca", "host"),
		)
		assert.Equal(t, "ca", cfg.CaFile)
		assert.Equal(t, "", cfg.CertFile)
		assert.Equal(t, "", cfg.KeyFile)
		assert.Equal(t, "host", cfg.ServerNameOverride)
	})

	t.Run("TestCertOption", func(t *testing.T) {
		cfg := NewTLSConfig(
			WithCA("ca", "host"),
			WithCertificate("cert", "key"),
		)
		assert.Equal(t, "ca", cfg.CaFile)
		assert.Equal(t, "cert", cfg.CertFile)
		assert.Equal(t, "key", cfg.KeyFile)
		assert.Equal(t, "host", cfg.ServerNameOverride)
	})
}
