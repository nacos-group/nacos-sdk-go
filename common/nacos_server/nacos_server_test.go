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
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/common/security"
	"github.com/nacos-group/nacos-sdk-go/v2/mock"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
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

func TestEndpointMode_LoginAndInjectToken(t *testing.T) {
	endpoint := "endpoint.example.com:8080"
	clientConfig := constant.ClientConfig{
		Username:            "nacos",
		Password:            "nacos",
		NamespaceId:         "public",
		Endpoint:            endpoint,
		EndpointContextPath: "nacos",
		ClusterName:         "serverlist",
		TimeoutMs:           3000,
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgent := mock.NewMockIHttpAgent(ctrl)

	// 1) Endpoint discovery returns a server list
	mockAgent.
		EXPECT().
		RequestOnlyResult(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(method, path string, header http.Header, timeoutMs uint64, params map[string]string) string {
			if method != http.MethodGet {
				t.Fatalf("expected GET for endpoint discovery, got %s", method)
			}
			if !strings.Contains(path, endpoint) {
				t.Fatalf("unexpected endpoint discovery URL: %s", path)
			}
			// Return one server
			return "127.0.0.1:8848"
		})

	// 2) Login against discovered server
	mockAgent.
		EXPECT().
		Post(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(path string, header http.Header, timeoutMs uint64, params map[string]string) (*http.Response, error) {
			if !strings.Contains(path, "127.0.0.1:8848") || !strings.Contains(path, "/v1/auth/users/login") {
				t.Fatalf("unexpected login URL: %s", path)
			}
			if params["username"] != "nacos" || params["password"] != "nacos" {
				t.Fatalf("unexpected login params: %+v", params)
			}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"accessToken":"token-abc","tokenTtl":1000}`)),
			}
			return resp, nil
		})

	// 3) Subsequent API call should include accessToken injected in params
	mockAgent.
		EXPECT().
		Request(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(method, url string, header http.Header, timeoutMs uint64, params map[string]string) (*http.Response, error) {
			if params[constant.KEY_ACCESS_TOKEN] != "token-abc" {
				t.Fatalf("expected access token to be injected, got: %s", params[constant.KEY_ACCESS_TOKEN])
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("ok")),
			}, nil
		})

	server, err := NewNacosServer(context.Background(),
		nil,
		clientConfig,
		mockAgent,
		1000,
		endpoint,
		nil)
	if err != nil {
		t.FailNow()
	}

	params := map[string]string{
		"namespaceId": clientConfig.NamespaceId,
		"serviceName": "svc",
		"groupName":   constant.DEFAULT_GROUP,
	}
	_, err = server.ReqApi(constant.SERVICE_PATH, params, http.MethodPost, clientConfig)
	assert.NoError(t, err)
}
