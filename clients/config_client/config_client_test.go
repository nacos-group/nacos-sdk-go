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
	"context"
	"errors"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/util"

	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v2/model"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/assert"
)

var serverConfigWithOptions = constant.NewServerConfig("127.0.0.1", 80, constant.WithContextPath("/nacos"))

var clientConfigWithOptions = constant.NewClientConfig(
	constant.WithTimeoutMs(10*1000),
	constant.WithBeatInterval(2*1000),
	constant.WithNotLoadCacheAtStart(true),
)

var localConfigTest = vo.ConfigParam{
	DataId:  "dataId",
	Group:   "group",
	Content: "content",
}

func createConfigClientTest() *ConfigClient {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*serverConfigWithOptions})
	_ = nc.SetClientConfig(*clientConfigWithOptions)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewConfigClient(&nc)
	client.configProxy = &MockConfigProxy{}
	return client
}

type MockConfigProxy struct {
}

func (m *MockConfigProxy) queryConfig(dataId, group, tenant string, timeout uint64, notify bool, client *ConfigClient) (*rpc_response.ConfigQueryResponse, error) {
	cacheKey := util.GetConfigCacheKey(dataId, group, tenant)
	if IsLimited(cacheKey) {
		return nil, errors.New("request is limited")
	}
	return &rpc_response.ConfigQueryResponse{Content: "hello world"}, nil
}
func (m *MockConfigProxy) searchConfigProxy(param vo.SearchConfigParam, tenant, accessKey, secretKey string) (*model.ConfigPage, error) {
	return &model.ConfigPage{TotalCount: 1}, nil
}
func (m *MockConfigProxy) requestProxy(rpcClient *rpc.RpcClient, request rpc_request.IRequest, timeoutMills uint64) (rpc_response.IResponse, error) {
	return &rpc_response.MockResponse{Response: &rpc_response.Response{Success: true}}, nil
}
func (m *MockConfigProxy) createRpcClient(ctx context.Context, taskId string, client *ConfigClient) *rpc.RpcClient {
	return &rpc.RpcClient{}
}
func (m *MockConfigProxy) getRpcClient(client *ConfigClient) *rpc.RpcClient {
	return &rpc.RpcClient{}
}

func Test_GetConfig(t *testing.T) {
	client := createConfigClientTest()
	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   localConfigTest.Group,
		Content: "hello world"})

	assert.Nil(t, err)
	assert.True(t, success)

	content, err := client.GetConfig(vo.ConfigParam{
		DataId: localConfigTest.DataId,
		Group:  "group"})

	assert.Nil(t, err)
	assert.Equal(t, "hello world", content)
}

func Test_SearchConfig(t *testing.T) {
	client := createConfigClientTest()
	_, _ = client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "DEFAULT_GROUP",
		Content: "hello world"})
	configPage, err := client.SearchConfig(vo.SearchConfigParam{
		Search:   "accurate",
		DataId:   localConfigTest.DataId,
		Group:    "DEFAULT_GROUP",
		PageNo:   1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, configPage)
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

func Test_PublishConfigWithoutContent(t *testing.T) {
	client := createConfigClientTest()
	_, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "group",
		Content: "",
	})
	assert.NotNil(t, err)
}

func Test_PublishConfig(t *testing.T) {

	client := createConfigClientTest()

	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "group",
		SrcUser: "nacos-client-go",
		Content: "hello world"})

	assert.Nil(t, err)
	assert.True(t, success)
}

// DeleteConfig
func Test_DeleteConfig(t *testing.T) {

	client := createConfigClientTest()

	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "group",
		Content: "hello world!"})

	assert.Nil(t, err)
	assert.True(t, success)

	success, err = client.DeleteConfig(vo.ConfigParam{
		DataId: localConfigTest.DataId,
		Group:  "group"})

	assert.Nil(t, err)
	assert.True(t, success)
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

func TestListen(t *testing.T) {
	t.Run("TestListenConfig", func(t *testing.T) {
		client := createConfigClientTest()
		err := client.ListenConfig(vo.ConfigParam{
			DataId: localConfigTest.DataId,
			Group:  localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
			},
		})
		assert.Nil(t, err)
	})
	// ListenConfig no dataId
	t.Run("TestListenConfigNoDataId", func(t *testing.T) {
		listenConfigParam := vo.ConfigParam{
			Group: localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
			},
		}
		client := createConfigClientTest()
		err := client.ListenConfig(listenConfigParam)
		assert.Error(t, err)
	})
}

// CancelListenConfig
func TestCancelListenConfig(t *testing.T) {
	//Multiple listeners listen for different configurations, cancel one
	t.Run("TestMultipleListenersCancelOne", func(t *testing.T) {
		client := createConfigClientTest()
		var err error
		listenConfigParam := vo.ConfigParam{
			DataId: localConfigTest.DataId,
			Group:  localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
			},
		}

		listenConfigParam1 := vo.ConfigParam{
			DataId: localConfigTest.DataId + "1",
			Group:  localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
			},
		}
		_ = client.ListenConfig(listenConfigParam)

		_ = client.ListenConfig(listenConfigParam1)

		err = client.CancelListenConfig(listenConfigParam)
		assert.Nil(t, err)
	})
}
