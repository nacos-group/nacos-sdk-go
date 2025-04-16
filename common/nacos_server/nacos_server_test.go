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
	"context"
	"testing"

	"github.com/dbsyk/nacos-sdk-go/v2/common/http_agent"
	"github.com/dbsyk/nacos-sdk-go/v2/common/security"

	"github.com/dbsyk/nacos-sdk-go/v2/common/constant"
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

func buildNacosServer(clientConfig constant.ClientConfig) (*NacosServer, error) {
	return NewNacosServer(context.Background(),
		[]constant.ServerConfig{*constant.NewServerConfig("http://console.nacos.io", 80)},
		clientConfig,
		&http_agent.HttpAgent{},
		1000,
		"",
		nil)
}

func TestNacosServer_InjectSignForNamingHttp_NoAk(t *testing.T) {
	clientConfig := constant.ClientConfig{
		AccessKey: "",
		SecretKey: "",
	}
	server, err := buildNacosServer(clientConfig)
	if err != nil {
		t.FailNow()
	}

	param := make(map[string]string, 4)
	param["serviceName"] = "s-0"
	param["groupName"] = "g-0"
	server.InjectSecurityInfo(param, security.BuildNamingResource(param["namespaceId"], param["groupName"], param["serviceName"]))
	assert.Empty(t, param["ak"])
	assert.Empty(t, param["data"])
	assert.Empty(t, param["signature"])
}

func TestNacosServer_InjectSignForNamingHttp_WithGroup(t *testing.T) {
	clientConfig := constant.ClientConfig{
		AccessKey: "123",
		SecretKey: "321",
	}
	server, err := buildNacosServer(clientConfig)
	if err != nil {
		t.FailNow()
	}

	param := make(map[string]string, 4)
	param["serviceName"] = "s-0"
	param["groupName"] = "g-0"
	server.InjectSecurityInfo(param, security.BuildNamingResource(param["namespaceId"], param["groupName"], param["serviceName"]))
	assert.Equal(t, "123", param["ak"])
	assert.Contains(t, param["data"], "@@g-0@@s-0")
	_, has := param["signature"]
	assert.True(t, has)
}

func TestNacosServer_InjectSignForNamingHttp_WithoutGroup(t *testing.T) {
	clientConfig := constant.ClientConfig{
		AccessKey: "123",
		SecretKey: "321",
	}
	server, err := buildNacosServer(clientConfig)
	if err != nil {
		t.FailNow()
	}

	param := make(map[string]string, 4)
	param["serviceName"] = "s-0"
	server.InjectSecurityInfo(param, security.BuildNamingResource(param["namespaceId"], param["groupName"], param["serviceName"]))
	assert.Equal(t, "123", param["ak"])
	assert.NotContains(t, param["data"], "@@g-0@@s-0")
	assert.Contains(t, param["data"], "@@s-0")
	_, has := param["signature"]
	assert.True(t, has)
}

func TestNacosServer_InjectSignForNamingHttp_WithoutServiceName(t *testing.T) {
	clientConfig := constant.ClientConfig{
		AccessKey: "123",
		SecretKey: "321",
	}
	server, err := buildNacosServer(clientConfig)
	if err != nil {
		t.FailNow()
	}

	param := make(map[string]string, 4)
	param["groupName"] = "g-0"
	server.InjectSecurityInfo(param, security.BuildNamingResource(param["namespaceId"], param["groupName"], param["serviceName"]))
	assert.Equal(t, "123", param["ak"])
	assert.Contains(t, param["data"], "@@")
	assert.Regexp(t, "\\d+", param["data"])
	_, has := param["signature"]
	assert.True(t, has)
}

func TestNacosServer_InjectSignForNamingHttp_WithoutServiceNameAndGroup(t *testing.T) {
	clientConfig := constant.ClientConfig{
		AccessKey: "123",
		SecretKey: "321",
	}
	server, err := buildNacosServer(clientConfig)
	if err != nil {
		t.FailNow()
	}

	param := make(map[string]string, 4)
	server.InjectSecurityInfo(param, security.BuildNamingResource(param["namespaceId"], param["serviceName"], param["groupName"]))
	assert.Equal(t, "123", param["ak"])
	assert.NotContains(t, param["data"], "@@")
	assert.Regexp(t, "\\d+", param["data"])
	_, has := param["signature"]
	assert.True(t, has)
}

func TestNacosServer_UpdateServerListForSecurityLogin(t *testing.T) {
	endpoint := "console.nacos.io:80"
	clientConfig := constant.ClientConfig{
		Username:            "nacos",
		Password:            "nacos",
		NamespaceId:         "namespace_1",
		Endpoint:            endpoint,
		EndpointContextPath: "nacos",
		ClusterName:         "serverlist",
		AppendToStdout:      true,
	}
	server, err := NewNacosServer(context.Background(),
		nil,
		clientConfig,
		&http_agent.HttpAgent{},
		1000,
		endpoint,
		nil)
	if err != nil {
		t.FailNow()
	}
	nacosAuthClient := server.securityLogin.Clients[0]
	client, ok := nacosAuthClient.(*security.NacosAuthClient)
	assert.True(t, ok)
	assert.Equal(t, server.GetServerList(), client.GetServerList())
}
