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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClientConfig(t *testing.T) {
	config := NewClientConfig()

	jsonStr, _ := json.Marshal(config)
	fmt.Printf("client cofing: %s", jsonStr)
}

func TestNewClientConfigWithOptions(t *testing.T) {
	config := NewClientConfig(
		WithLogLevel("error"),
		WithEndpoint("http://console.nacos.io:80"),
	)
	assert.Equal(t, config.LogLevel, "error")
	assert.Equal(t, config.Endpoint, "http://console.nacos.io:80")

	jsonStr, _ := json.Marshal(config)
	fmt.Printf("client cofing: %s", jsonStr)
}
