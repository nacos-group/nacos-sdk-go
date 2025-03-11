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
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigConnectionEventListener(t *testing.T) {
	client := &ConfigClient{}
	taskId := "123"

	listener := NewConfigConnectionEventListener(client, taskId)

	assert.Equal(t, client, listener.client)
	assert.Equal(t, taskId, listener.taskId)
}

func TestOnDisConnectWithMock(t *testing.T) {
	client := &ConfigClient{
		cacheMap: cache.NewConcurrentMap(),
	}

	data1 := cacheData{
		dataId:           "dataId1",
		group:            "group1",
		tenant:           "",
		taskId:           1,
		isSyncWithServer: true,
	}

	data2 := cacheData{
		dataId:           "dataId2",
		group:            "group1",
		tenant:           "",
		taskId:           1,
		isSyncWithServer: true,
	}

	data3 := cacheData{
		dataId:           "dataId3",
		group:            "group2",
		tenant:           "",
		taskId:           2,
		isSyncWithServer: true,
	}

	key1 := util.GetConfigCacheKey(data1.dataId, data1.group, data1.tenant)
	key2 := util.GetConfigCacheKey(data2.dataId, data2.group, data2.tenant)
	key3 := util.GetConfigCacheKey(data3.dataId, data3.group, data3.tenant)

	client.cacheMap.Set(key1, data1)
	client.cacheMap.Set(key2, data2)
	client.cacheMap.Set(key3, data3)

	listener := NewConfigConnectionEventListener(client, "1")

	listener.OnDisConnect()

	item1, _ := client.cacheMap.Get(key1)
	item2, _ := client.cacheMap.Get(key2)
	item3, _ := client.cacheMap.Get(key3)

	updatedData1 := item1.(cacheData)
	updatedData2 := item2.(cacheData)
	updatedData3 := item3.(cacheData)

	assert.False(t, updatedData1.isSyncWithServer, "dataId1 should be marked as not sync")
	assert.False(t, updatedData2.isSyncWithServer, "dataId2 should be marked as not sync")
	assert.True(t, updatedData3.isSyncWithServer, "dataId3 should be marked as sync")
}

func TestOnConnectedWithMock(t *testing.T) {
	listenChan := make(chan struct{}, 1)

	client := &ConfigClient{
		listenExecute: listenChan,
	}

	listener := NewConfigConnectionEventListener(client, "1")

	listener.OnConnected()

	time.Sleep(100 * time.Millisecond)

	select {
	case <-listenChan:
		assert.True(t, true, "asyncNotifyListenConfig should be called")
	default:
		t.Fatalf("asyncNotifyListenConfig should be called but not")
	}
}

type MockRpcClientForListener struct {
	requestCalled rpc_request.IRequest
}

func (m *MockRpcClientForListener) Request(request rpc_request.IRequest) (rpc_response.IResponse, error) {
	m.requestCalled = request
	return &rpc_response.ConfigChangeBatchListenResponse{
		Response: &rpc_response.Response{
			ResultCode: 200,
		},
		ChangedConfigs: []model.ConfigContext{},
	}, nil
}

func TestReconnectionFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockRpc := &MockRpcClientForListener{}

	listenChan := make(chan struct{}, 1)

	client := &ConfigClient{
		ctx:           ctx,
		configProxy:   &MockConfigProxy{},
		cacheMap:      cache.NewConcurrentMap(),
		listenExecute: listenChan,
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case <-listenChan:
				mockRpc.Request(&rpc_request.ConfigBatchListenRequest{})
				done <- true
			case <-ctx.Done():
				return
			}
		}
	}()

	data1 := cacheData{
		dataId:           "dataId1",
		group:            "group1",
		tenant:           "",
		taskId:           1,
		isSyncWithServer: true,
	}

	key1 := util.GetConfigCacheKey(data1.dataId, data1.group, data1.tenant)
	client.cacheMap.Set(key1, data1)

	listener := NewConfigConnectionEventListener(client, "1")

	initialData, _ := client.cacheMap.Get(key1)
	assert.True(t, initialData.(cacheData).isSyncWithServer, "initial data should be sync with server")

	listener.OnDisConnect()

	afterDisconnectData, _ := client.cacheMap.Get(key1)
	assert.False(t, afterDisconnectData.(cacheData).isSyncWithServer, "disconnect should set isSyncWithServer to false")

	listener.OnConnected()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatalf("wait for done timeout")
	}

	assert.NotNil(t, mockRpc.requestCalled, "should call request")

	_, ok := mockRpc.requestCalled.(*rpc_request.ConfigBatchListenRequest)
	assert.True(t, ok, "should be a ConfigBatchListenRequest")
}
