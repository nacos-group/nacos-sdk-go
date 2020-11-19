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

package config_client

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/mock"
	"github.com/nacos-group/nacos-sdk-go/util"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
)

var clientConfigTest = constant.ClientConfig{
	TimeoutMs:      10000,
	ListenInterval: 20000,
	BeatInterval:   10000,
}

var clientConfigTestWithTenant = constant.ClientConfig{
	TimeoutMs:      10000,
	ListenInterval: 20000,
	BeatInterval:   10000,
	NamespaceId:    "tenant",
}

var serverConfigTest = constant.ServerConfig{
	ContextPath: "/nacos",
	Port:        80,
	IpAddr:      "console.nacos.io",
}

var configParamMapTest = map[string]string{
	"dataId": "dataId",
	"group":  "group",
}

var configParamTest = vo.ConfigParam{
	DataId: "dataId",
	Group:  "group",
}

var localConfigTest = vo.ConfigParam{
	DataId:  "dataId",
	Group:   "group",
	Content: "content",
}

var localConfigMapTest = map[string]string{
	"dataId":  "dataId",
	"group":   "group",
	"content": "content",
}

var headerTest = map[string][]string{
	"Content-Type": {"application/x-www-form-urlencoded"},
}
var headerListenerTest = map[string][]string{
	"Content-Type":      {"application/x-www-form-urlencoded"},
	"Listening-Configs": {"30000"},
}

var serverConfigsTest = []constant.ServerConfig{serverConfigTest}

var httpAgentTest = mock.MockIHttpAgent{}

func createConfigClientTest() ConfigClient {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewConfigClient(&nc)
	return client
}

func createConfigClientTestWithTenant() ConfigClient {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTestWithTenant)
	nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewConfigClient(&nc)
	return client
}

func createConfigClientHttpTest(mockHttpAgent http_agent.IHttpAgent) ConfigClient {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockHttpAgent)
	client, _ := NewConfigClient(&nc)
	return client
}

func createConfigClientHttpTestWithTenant(mockHttpAgent http_agent.IHttpAgent) ConfigClient {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTestWithTenant)
	nc.SetHttpAgent(mockHttpAgent)
	client, _ := NewConfigClient(&nc)
	return client
}

func Test_GetConfig(t *testing.T) {

	client := createConfigClientTest()
	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "hello world!222222"})

	assert.Nil(t, err)
	assert.True(t, success)

	content, err := client.GetConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "group"})

	assert.Nil(t, err)
	assert.Equal(t, "hello world!222222", content)
}

func Test_SearchConfig(t *testing.T) {
	client := createConfigClientTest()
	configPage, err := client.SearchConfig(vo.SearchConfigParm{
		Search:   "accurate",
		DataId:   "",
		Group:    "DEFAULT_GROUP",
		PageNo:   1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, configPage)
	assert.NotEmpty(t, configPage.PageItems)
}

func Test_GetConfigWithErrorResponse_401(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := createConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no security"), nil)
	result, err := client.GetConfig(configParamTest)
	assert.Nil(t, err)
	fmt.Printf("result:%s \n", result)
}

func Test_GetConfigWithErrorResponse_404(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := createConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(404, ""), nil)
	reslut, err := client.GetConfig(configParamTest)
	assert.NotNil(t, err)
	assert.Equal(t, "", reslut)
	fmt.Println(err.Error())
}

func Test_GetConfigWithErrorResponse_403(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := createConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(403, ""), nil)
	reslut, err := client.GetConfig(configParamTest)
	assert.NotNil(t, err)
	assert.Equal(t, "", reslut)
	fmt.Println(err.Error())
}

func Test_GetConfigWithCache(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := createConfigClientHttpTest(mockHttpAgent)

	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "content"), nil)
	content, err := client.GetConfig(configParamTest)
	assert.Nil(t, err)
	assert.Equal(t, "content", content)

	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no security"), nil)
	content, err = client.GetConfig(configParamTest)
	assert.Nil(t, err)
	assert.Equal(t, "content", content)
}

// PublishConfig

func Test_PublishConfigWithoutDataId(t *testing.T) {
	client := createConfigClientTest()
	_, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "",
		Group:   "group",
		Content: "content",
	})
	assert.NotNil(t, err)
}

func Test_PublishConfigWithoutGroup(t *testing.T) {
	client := createConfigClientTest()
	_, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "",
		Content: "content",
	})
	assert.NotNil(t, err)
}

func Test_PublishConfigWithoutContent(t *testing.T) {
	client := createConfigClientTest()
	_, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "",
	})
	assert.NotNil(t, err)
}

func Test_PublishConfig(t *testing.T) {

	client := createConfigClientTest()

	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "hello world2!"})

	assert.Nil(t, err)
	assert.True(t, success)
}

func Test_PublishConfigWithErrorResponse(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := createConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodPost),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(localConfigMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no security"), nil)
	success, err := client.PublishConfig(localConfigTest)
	assert.NotNil(t, err)
	assert.True(t, !success)
}

func Test_PublishConfigWithErrorResponse_200(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := createConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodPost),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(localConfigMapTest),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "false"), nil)
	success, err := client.PublishConfig(localConfigTest)
	assert.NotNil(t, err)
	assert.True(t, !success)
}

// DeleteConfig

func Test_DeleteConfig(t *testing.T) {

	client := createConfigClientTest()

	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "hello world!"})

	assert.Nil(t, err)
	assert.True(t, success)

	success, err = client.DeleteConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "group"})

	assert.Nil(t, err)
	assert.True(t, success)
}

func Test_DeleteConfigWithErrorResponse_200(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := createConfigClientHttpTest(mockHttpAgent)

	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodDelete),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "false"), nil)
	success, err := client.DeleteConfig(configParamTest)
	assert.NotNil(t, err)
	assert.Equal(t, false, success)
}

func Test_DeleteConfigWithErrorResponse_401(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := createConfigClientHttpTest(mockHttpAgent)

	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodDelete),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no security"), nil)
	success, err := client.DeleteConfig(configParamTest)
	assert.NotNil(t, err)
	assert.Equal(t, false, success)
}

func Test_DeleteConfigWithoutDataId(t *testing.T) {
	client := createConfigClientTest()
	success, err := client.DeleteConfig(vo.ConfigParam{
		DataId: "",
		Group:  "group",
	})
	assert.NotNil(t, err)
	assert.Equal(t, false, success)
}

func Test_DeleteConfigWithoutGroup(t *testing.T) {
	client := createConfigClientTest()
	success, err := client.DeleteConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "",
	})
	assert.NotNil(t, err)
	assert.Equal(t, false, success)
}

// ListenConfig
func TestListen(t *testing.T) {
	// ListenConfig
	t.Run("TestListenConfig", func(t *testing.T) {
		client := createConfigClientTest()
		key := util.GetConfigCacheKey(localConfigTest.DataId, localConfigTest.Group, clientConfigTest.NamespaceId)
		cache.WriteConfigToFile(key, client.configCacheDir, "")
		var err error
		var success bool
		ch := make(chan string)
		go func() {
			err = client.ListenConfig(vo.ConfigParam{
				DataId: localConfigTest.DataId,
				Group:  localConfigTest.Group,
				OnChange: func(namespace, group, dataId, data string) {
					fmt.Println("group:" + group + ", dataId:" + dataId + ", data:" + data)
					ch <- data
				},
			})
			assert.Nil(t, err)
		}()

		time.Sleep(2 * time.Second)

		success, err = client.PublishConfig(vo.ConfigParam{
			DataId:  localConfigTest.DataId,
			Group:   localConfigTest.Group,
			Content: localConfigTest.Content})

		assert.Nil(t, err)
		assert.Equal(t, true, success)
		select {
		case c := <-ch:
			assert.Equal(t, c, localConfigTest.Content)
		case <-time.After(10 * time.Second):
			fmt.Println("timeout")
			assert.Errorf(t, errors.New("timeout"), "timeout")
		}
	})
	// ListenConfig no dataId
	t.Run("TestListenConfigNoDataId", func(t *testing.T) {
		listenConfigParam := vo.ConfigParam{
			Group: "gateway",
			OnChange: func(namespace, group, dataId, data string) {
			},
		}
		client := createConfigClientTest()
		err := client.ListenConfig(listenConfigParam)
		assert.Error(t, err)
	})
	// ListenConfig no change
	t.Run("TestListenConfigNoChange", func(t *testing.T) {
		client := createConfigClientTest()
		key := util.GetConfigCacheKey("ConfigNoChange", localConfigTest.Group, clientConfigTest.NamespaceId)
		cache.WriteConfigToFile(key, client.configCacheDir, localConfigTest.Content)
		var err error
		var success bool
		var content string

		go func() {
			err = client.ListenConfig(vo.ConfigParam{
				DataId: "ConfigNoChange",
				Group:  localConfigTest.Group,
				OnChange: func(namespace, group, dataId, data string) {
					content = "data"
				},
			})
			assert.Nil(t, err)
		}()

		time.Sleep(2 * time.Second)

		success, err = client.PublishConfig(vo.ConfigParam{
			DataId:  "ConfigNoChange",
			Group:   localConfigTest.Group,
			Content: localConfigTest.Content})

		assert.Nil(t, err)
		assert.Equal(t, true, success)
		assert.Equal(t, content, "")
	})
	// Multiple clients listen to the same configuration file
	t.Run("TestListenConfigWithMultipleClients", func(t *testing.T) {
		ch := make(chan string)
		listenConfigParam := vo.ConfigParam{
			DataId: "MultipleClients",
			Group:  localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
				fmt.Println("group:" + group + ", dataId:" + dataId + ", data:" + data)
				ch <- data
			},
		}
		client := createConfigClientTest()
		key := util.GetConfigCacheKey(listenConfigParam.DataId, listenConfigParam.Group, clientConfigTest.NamespaceId)
		cache.WriteConfigToFile(key, client.configCacheDir, "")
		client.ListenConfig(listenConfigParam)

		client1 := createConfigClientTest()
		client1.ListenConfig(listenConfigParam)

		success, err := client.PublishConfig(vo.ConfigParam{
			DataId:  "MultipleClients",
			Group:   localConfigTest.Group,
			Content: localConfigTest.Content})

		assert.Nil(t, err)
		assert.Equal(t, true, success)
		select {
		case c := <-ch:
			assert.Equal(t, localConfigTest.Content, c)
		case <-time.After(10 * time.Second):
			fmt.Println("timeout")
			assert.Errorf(t, errors.New("timeout"), "timeout")
		}

	})
	// Multiple clients listen to multiple configuration files
	t.Run("TestListenConfigWithMultipleClientsMultipleConfig", func(t *testing.T) {
		ch := make(chan string)
		listenConfigParam := vo.ConfigParam{
			DataId: "MultipleClientsMultipleConfig",
			Group:  localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
				fmt.Println("group:" + group + ", dataId:" + dataId + ", data:" + data)
				ch <- data
			},
		}
		client := createConfigClientTest()
		key := util.GetConfigCacheKey(listenConfigParam.DataId, listenConfigParam.Group, clientConfigTest.NamespaceId)
		cache.WriteConfigToFile(key, client.configCacheDir, "")
		client.ListenConfig(listenConfigParam)

		client1 := createConfigClientTest()
		client1.ListenConfig(listenConfigParam)

		success, err := client.PublishConfig(vo.ConfigParam{
			DataId:  "MultipleClientsMultipleConfig",
			Group:   localConfigTest.Group,
			Content: localConfigTest.Content})

		assert.Nil(t, err)
		assert.Equal(t, true, success)
		select {
		case c := <-ch:
			assert.Equal(t, localConfigTest.Content, c)
		case <-time.After(10 * time.Second):
			fmt.Println("timeout")
			assert.Errorf(t, errors.New("timeout"), "timeout")
		}

	})
}

// CancelListenConfig
func TestCancelListenConfig(t *testing.T) {
	//Multiple listeners listen for different configurations, cancel one
	t.Run("TestMultipleListenersCancelOne", func(t *testing.T) {
		client := createConfigClientTest()
		var err error
		var success bool
		var context string
		listenConfigParam := vo.ConfigParam{
			DataId: "CancelOne",
			Group:  "group",
			OnChange: func(namespace, group, dataId, data string) {
				fmt.Println("group:" + group + ", dataId:" + dataId + ", data:" + data)
			},
		}

		listenConfigParam1 := vo.ConfigParam{
			DataId: "CancelOne1",
			Group:  "group1",
			OnChange: func(namespace, group, dataId, data string) {
				fmt.Println("group1:" + group + ", dataId1:" + dataId + ", data:" + data)
				context = data
			},
		}
		go func() {
			client.ListenConfig(listenConfigParam)
		}()

		go func() {
			client.ListenConfig(listenConfigParam1)
		}()

		fmt.Println("Start listening")
		for i := 1; i <= 5; i++ {
			go func() {
				success, err = client.PublishConfig(vo.ConfigParam{
					DataId:  "CancelOne",
					Group:   "group",
					Content: "abcd" + strconv.Itoa(i)})
			}()

			go func() {
				success, err = client.PublishConfig(vo.ConfigParam{
					DataId:  "CancelOne1",
					Group:   "group1",
					Content: "abcd" + strconv.Itoa(i)})
			}()

			if i == 3 {
				client.CancelListenConfig(listenConfigParam)
				fmt.Println("Cancel listen config")
			}
			time.Sleep(2 * time.Second)
			assert.Nil(t, err)
			assert.Equal(t, true, success)
		}
		assert.Equal(t, "abcd5", context)
	})
	t.Run("TestCancelListenConfig", func(t *testing.T) {
		var context string
		var err error
		ch := make(chan string)
		client := createConfigClientTest()
		//
		key := util.GetConfigCacheKey(localConfigTest.DataId, localConfigTest.Group, clientConfigTest.NamespaceId)
		cache.WriteConfigToFile(key, client.configCacheDir, "")
		listenConfigParam := vo.ConfigParam{
			DataId: "cancel_listen_config",
			Group:  localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
				fmt.Println("group:" + group + ", dataId:" + dataId + ", data:" + data)
				context = data
				ch <- data
			},
		}
		go func() {
			err = client.ListenConfig(listenConfigParam)
			assert.Nil(t, err)
		}()
		success, err := client.PublishConfig(vo.ConfigParam{
			DataId:  "cancel_listen_config",
			Group:   localConfigTest.Group,
			Content: localConfigTest.Content})
		assert.Nil(t, err)
		assert.Equal(t, true, success)

		select {
		case c := <-ch:
			assert.Equal(t, c, localConfigTest.Content)
		}
		//Cancel listen config
		client.CancelListenConfig(listenConfigParam)

		success, err = client.PublishConfig(vo.ConfigParam{
			DataId:  "cancel_listen_config",
			Group:   localConfigTest.Group,
			Content: "abcd"})
		assert.Nil(t, err)
		assert.Equal(t, true, success)

		assert.Equal(t, localConfigTest.Content, context)
	})
}

func TestGetConfigWithSpecialSymbol(t *testing.T) {
	contentStr := "hello world!!@#$%^&&*()"

	client := createConfigClientTest()
	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "special_symbol",
		Group:   localConfigTest.Group,
		Content: contentStr})

	assert.Nil(t, err)
	assert.True(t, success)

	content, err := client.GetConfig(vo.ConfigParam{
		DataId: "special_symbol",
		Group:  localConfigTest.Group})

	assert.Nil(t, err)
	assert.Equal(t, contentStr, content)
}
