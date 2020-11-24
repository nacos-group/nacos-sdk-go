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

package nacos_server

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/stretchr/testify/assert"
)

func Test_getAddressWithScheme(t *testing.T) {
	var serverConfigTest = constant.ServerConfig{
		ContextPath: "/nacos",
		Port:        80,
		IpAddr:      "console.nacos.io",
		Scheme:      "https",
	}
	address := getAddress(serverConfigTest)
	assert.Equal(t, "https://console.nacos.io:80", address)
}

func Test_getAddressWithoutScheme(t *testing.T) {
	serverConfigTest := constant.ServerConfig{
		ContextPath: "/nacos",
		Port:        80,
		IpAddr:      "http://console.nacos.io",
	}
	assert.Equal(t, "http://console.nacos.io:80", getAddress(serverConfigTest))

	serverConfigTest.IpAddr = "https://console.nacos.io"
	assert.Equal(t, "https://console.nacos.io:80", getAddress(serverConfigTest))

}
