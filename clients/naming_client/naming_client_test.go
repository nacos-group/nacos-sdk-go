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

package naming_client

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/mock"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/util"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
)

var clientConfigTest = constant.ClientConfig{
	TimeoutMs:           10 * 1000,
	BeatInterval:        5 * 1000,
	ListenInterval:      30 * 1000,
	NotLoadCacheAtStart: true,
}

var serverConfigTest = constant.ServerConfig{
	IpAddr:      "console.nacos.io",
	Port:        80,
	ContextPath: "/nacos",
}

var headers = map[string][]string{
	"Client-Version":  {constant.CLIENT_VERSION},
	"User-Agent":      {constant.CLIENT_VERSION},
	"Accept-Encoding": {"gzip,deflate,sdch"},
	"Connection":      {"Keep-Alive"},
	"Request-Module":  {"Naming"},
	"Content-Type":    {"application/x-www-form-urlencoded"},
}

func Test_RegisterServiceInstance_withoutGroupeName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)
	mockIHttpAgent.EXPECT().Request(gomock.Eq("POST"),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(uint64(10*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "",
			"serviceName": "DEFAULT_GROUP@@DEMO",
			"groupName":   "DEFAULT_GROUP",
			"clusterName": "",
			"ip":          "10.0.0.10",
			"port":        "80",
			"weight":      "0",
			"enable":      "false",
			"healthy":     "false",
			"metadata":    "{}",
			"ephemeral":   "false",
		})).Times(1).
		Return(http_agent.FakeHttpResponse(200, `ok`), nil)
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	success, err := client.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		Ephemeral:   false,
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, true, success)
}

func Test_RegisterServiceInstance_withGroupeName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	mockIHttpAgent.EXPECT().Request(gomock.Eq("POST"),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(uint64(10*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "",
			"serviceName": "test_group@@DEMO2",
			"groupName":   "test_group",
			"clusterName": "",
			"ip":          "10.0.0.10",
			"port":        "80",
			"weight":      "0",
			"enable":      "false",
			"healthy":     "false",
			"metadata":    "{}",
			"ephemeral":   "false",
		})).Times(1).
		Return(http_agent.FakeHttpResponse(200, `ok`), nil)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	success, err := client.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: "DEMO2",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
		Ephemeral:   false,
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, true, success)
}

func Test_RegisterServiceInstance_withCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	mockIHttpAgent.EXPECT().Request(gomock.Eq("POST"),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(uint64(10*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "",
			"serviceName": "test_group@@DEMO3",
			"groupName":   "test_group",
			"clusterName": "test",
			"ip":          "10.0.0.10",
			"port":        "80",
			"weight":      "0",
			"enable":      "false",
			"healthy":     "false",
			"metadata":    "{}",
			"ephemeral":   "false",
		})).Times(1).
		Return(http_agent.FakeHttpResponse(200, `ok`), nil)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	success, err := client.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: "DEMO3",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
		ClusterName: "test",
		Ephemeral:   false,
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, true, success)
}

func Test_RegisterServiceInstance_401(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	mockIHttpAgent.EXPECT().Request(gomock.Eq("POST"),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(uint64(10*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "",
			"serviceName": "test_group@@DEMO4",
			"groupName":   "test_group",
			"clusterName": "",
			"ip":          "10.0.0.10",
			"port":        "80",
			"weight":      "0",
			"enable":      "false",
			"healthy":     "false",
			"metadata":    "{}",
			"ephemeral":   "false",
		})).Times(3).
		Return(http_agent.FakeHttpResponse(401, `no security`), nil)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	result, err := client.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: "DEMO4",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
		Ephemeral:   false,
	})
	assert.Equal(t, false, result)
	assert.NotNil(t, err)
}

func TestNamingProxy_DeristerService_WithoutGroupName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	mockIHttpAgent.EXPECT().Request(gomock.Eq("DELETE"),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(uint64(10*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "",
			"serviceName": "DEFAULT_GROUP@@DEMO5",
			"clusterName": "",
			"ip":          "10.0.0.10",
			"port":        "80",
			"ephemeral":   "true",
		})).Times(1).
		Return(http_agent.FakeHttpResponse(200, `ok`), nil)
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	client.DeregisterInstance(vo.DeregisterInstanceParam{
		ServiceName: "DEMO5",
		Ip:          "10.0.0.10",
		Port:        80,
		Ephemeral:   true,
	})
}

func TestNamingProxy_DeristerService_WithGroupName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	mockIHttpAgent.EXPECT().Request(gomock.Eq("DELETE"),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(uint64(10*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "",
			"serviceName": "test_group@@DEMO6",
			"clusterName": "",
			"ip":          "10.0.0.10",
			"port":        "80",
			"ephemeral":   "true",
		})).Times(1).
		Return(http_agent.FakeHttpResponse(200, `ok`), nil)
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	client.DeregisterInstance(vo.DeregisterInstanceParam{
		ServiceName: "DEMO6",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
		Ephemeral:   true,
	})
}

func TestNamingProxy_DeristerService_401(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	mockIHttpAgent.EXPECT().Request(gomock.Eq("DELETE"),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(uint64(10*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "",
			"serviceName": "test_group@@DEMO7",
			"clusterName": "",
			"ip":          "10.0.0.10",
			"port":        "80",
			"ephemeral":   "true",
		})).Times(3).
		Return(http_agent.FakeHttpResponse(401, `no security`), nil)
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	client.DeregisterInstance(vo.DeregisterInstanceParam{
		ServiceName: "DEMO7",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
		Ephemeral:   true,
	})
}

var serviceJsonTest = `{
			"name": "DEFAULT_GROUP@@DEMO",
			"cacheMillis": 1000,
			"useSpecifiedURL": false,
			"hosts": [{
				"valid": true,
				"marked": false,
				"instanceId": "10.10.10.10-8888-a-DEMO",
				"port": 8888,
				"ip": "10.10.10.10",
				"weight": 1.0,
				"metadata": {},
				"serviceName":"DEMO",
				"enabled":true,
				"clusterName":"a"
			},{
				"valid": true,
				"marked": false,
				"instanceId": "10.10.10.11-8888-a-DEMO",
				"port": 8888,
				"ip": "10.10.10.11",
				"weight": 1.0,
				"metadata": {},
				"serviceName":"DEMO",
				"enabled":true,
				"clusterName":"a"
			}
			],
			"checksum": "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
			"lastRefTime": 1528787794594,
			"env": "",
			"clusters": "a"
		}`

var serviceTest = model.Service(model.Service{Name: "DEFAULT_GROUP@@DEMO",
	CacheMillis: 1000, UseSpecifiedURL: false,
	Hosts: []model.Instance{
		{
			Valid:       true,
			Marked:      false,
			InstanceId:  "10.10.10.10-8888-a-DEMO",
			Port:        0x22b8,
			Ip:          "10.10.10.10",
			Weight:      1,
			Metadata:    map[string]string{},
			ClusterName: "a",
			ServiceName: "DEMO",
			Enable:      true,
		},
		{
			Valid:       true,
			Marked:      false,
			InstanceId:  "10.10.10.11-8888-a-DEMO",
			Port:        0x22b8,
			Ip:          "10.10.10.11",
			Weight:      1,
			Metadata:    map[string]string{},
			ClusterName: "a",
			ServiceName: "DEMO",
			Enable:      true,
		},
	},
	Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
	LastRefTime: 1528787794594, Env: "", Clusters: "a",
	Metadata: map[string]string(nil)})

//func TestNamingProxy_GetService_WithoutGroupName(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer func() {
//		ctrl.Finish()
//	}()
//	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)
//
//	mockIHttpAgent.EXPECT().Request(gomock.Eq("GET"),
//		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance/list"),
//		gomock.AssignableToTypeOf(http.Header{}),
//		gomock.Eq(uint64(10*1000)),
//		gomock.Any()).Times(2).
//		Return(http_agent.FakeHttpResponse(200, serviceJsonTest), nil)
//
//	nc := nacos_client.NacosClient{}
//	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
//	nc.SetClientConfig(clientConfigTest)
//	nc.SetHttpAgent(mockIHttpAgent)
//	client, _ := NewNamingClient(&nc)
//	result, err := client.GetService(vo.GetServiceParam{
//		ServiceName: "DEMO",
//		Clusters:    []string{"a"},
//	})
//	assert.Nil(t, err)
//	assert.Equal(t, serviceTest, result)
//
//}

//func TestNamingClient_SelectAllInstancs(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer func() {
//		ctrl.Finish()
//	}()
//	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)
//
//	mockIHttpAgent.EXPECT().Request(gomock.Eq("GET"),
//		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance/list"),
//		gomock.AssignableToTypeOf(http.Header{}),
//		gomock.Eq(uint64(10*1000)),
//		gomock.Any()).Times(2).
//		Return(http_agent.FakeHttpResponse(200, serviceJsonTest), nil)
//
//	nc := nacos_client.NacosClient{}
//	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
//	nc.SetClientConfig(clientConfigTest)
//	nc.SetHttpAgent(mockIHttpAgent)
//	client, _ := NewNamingClient(&nc)
//	instances, err := client.SelectAllInstances(vo.SelectAllInstancesParam{
//		ServiceName: "DEMO",
//		Clusters:    []string{"a"},
//	})
//	fmt.Println(utils.ToJsonString(instances))
//	assert.Nil(t, err)
//	assert.Equal(t, 2, len(instances))
//}

func TestNamingClient_SelectOneHealthyInstance_SameWeight(t *testing.T) {
	services := model.Service(model.Service{
		Name:            "DEFAULT_GROUP@@DEMO",
		CacheMillis:     1000,
		UseSpecifiedURL: false,
		Hosts: []model.Instance{
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.10-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.10",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO1",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.11-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.11",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.12-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.12",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     false,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.13-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.13",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      false,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.14-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.14",
				Weight:      0,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
		},
		Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime: 1528787794594, Env: "", Clusters: "a",
		Metadata: map[string]string(nil)})
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	instance1, err := client.selectOneHealthyInstances(services)
	fmt.Println(util.ToJsonString(instance1))
	assert.Nil(t, err)
	assert.NotNil(t, instance1)
	instance2, err := client.selectOneHealthyInstances(services)
	fmt.Println(util.ToJsonString(instance2))
	assert.Nil(t, err)
	assert.NotNil(t, instance2)
	//assert.NotEqual(t, instance1, instance2)
}

func TestNamingClient_SelectOneHealthyInstance_Empty(t *testing.T) {
	services := model.Service(model.Service{
		Name:            "DEFAULT_GROUP@@DEMO",
		CacheMillis:     1000,
		UseSpecifiedURL: false,
		Hosts:           []model.Instance{},
		Checksum:        "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime:     1528787794594, Env: "", Clusters: "a",
		Metadata: map[string]string(nil)})
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	instance, err := client.selectOneHealthyInstances(services)
	fmt.Println(util.ToJsonString(instance))
	assert.NotNil(t, err)
	assert.Nil(t, instance)
}

func TestNamingClient_SelectInstances_Healthy(t *testing.T) {
	services := model.Service(model.Service{
		Name:            "DEFAULT_GROUP@@DEMO",
		CacheMillis:     1000,
		UseSpecifiedURL: false,
		Hosts: []model.Instance{
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.10-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.10",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.11-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.11",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.12-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.12",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     false,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.13-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.13",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      false,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.14-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.14",
				Weight:      0,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
		},
		Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime: 1528787794594, Env: "", Clusters: "a",
		Metadata: map[string]string(nil)})
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	instances, err := client.selectInstances(services, true)
	fmt.Println(util.ToJsonString(instances))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(instances))
}

func TestNamingClient_SelectInstances_Unhealthy(t *testing.T) {
	services := model.Service(model.Service{
		Name:            "DEFAULT_GROUP@@DEMO",
		CacheMillis:     1000,
		UseSpecifiedURL: false,
		Hosts: []model.Instance{
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.10-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.10",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.11-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.11",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.12-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.12",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     false,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.13-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.13",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      false,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.14-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.14",
				Weight:      0,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
		},
		Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime: 1528787794594, Env: "", Clusters: "a",
		Metadata: map[string]string(nil)})
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	instances, err := client.selectInstances(services, false)
	fmt.Println(util.ToJsonString(instances))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(instances))
}

func TestNamingClient_SelectInstances_Empty(t *testing.T) {
	services := model.Service(model.Service{
		Name:            "DEFAULT_GROUP@@DEMO",
		CacheMillis:     1000,
		UseSpecifiedURL: false,
		Hosts:           []model.Instance{},
		Checksum:        "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime:     1528787794594, Env: "", Clusters: "a",
		Metadata: map[string]string(nil)})
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	instances, err := client.selectInstances(services, false)
	fmt.Println(util.ToJsonString(instances))
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(instances))
}

func TestNamingClient_GetAllServicesInfo(t *testing.T) {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewNamingClient(&nc)
	reslut, err := client.GetAllServicesInfo(vo.GetAllServiceInfoParam{
		GroupName: "DEFAULT_GROUP",
		PageNo:    1,
		PageSize:  20,
	})

	assert.NotNil(t, reslut.Doms)
	assert.Nil(t, err)
}

func TestNamingClient_selectOneHealthyInstanceResult(t *testing.T) {
	services := model.Service(model.Service{
		Name: "DEFAULT_GROUP@@DEMO",
		Hosts: []model.Instance{
			{
				Ip:      "127.0.0.1",
				Weight:  1,
				Enable:  true,
				Healthy: true,
			},
			{
				Ip:      "127.0.0.2",
				Weight:  9,
				Enable:  true,
				Healthy: true,
			},
		}})
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	client, _ := NewNamingClient(&nc)
	for i := 0; i < 10; i++ {
		i, _ := client.selectOneHealthyInstances(services)
		fmt.Println(i.Ip)
	}
}

func BenchmarkNamingClient_SelectOneHealthyInstances(b *testing.B) {
	services := model.Service(model.Service{
		Name:            "DEFAULT_GROUP@@DEMO",
		CacheMillis:     1000,
		UseSpecifiedURL: false,
		Hosts: []model.Instance{
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.10-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.10",
				Weight:      10,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO1",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.11-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.11",
				Weight:      10,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO2",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.12-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.12",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO3",
				Enable:      true,
				Healthy:     false,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.13-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.13",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO4",
				Enable:      false,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.14-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.14",
				Weight:      0,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO5",
				Enable:      true,
				Healthy:     true,
			},
		},
		Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime: 1528787794594, Env: "", Clusters: "a",
		Metadata: map[string]string(nil)})
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	client, _ := NewNamingClient(&nc)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.selectOneHealthyInstances(services)
	}

}

func BenchmarkNamingClient_Random(b *testing.B) {
	services := model.Service(model.Service{
		Name:            "DEFAULT_GROUP@@DEMO",
		CacheMillis:     1000,
		UseSpecifiedURL: false,
		Hosts: []model.Instance{
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.10-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.10",
				Weight:      10,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO1",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.11-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.11",
				Weight:      9,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.12-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.12",
				Weight:      8,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     false,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.13-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.13",
				Weight:      8,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      false,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.14-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.14",
				Weight:      7,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
		},
		Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime: 1528787794594, Env: "", Clusters: "a",
		Metadata: map[string]string(nil)})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		random(services.Hosts, 10)
	}
}

func BenchmarkNamingClient_ChooserPick(b *testing.B) {
	services := model.Service(model.Service{
		Name:            "DEFAULT_GROUP@@DEMO",
		CacheMillis:     1000,
		UseSpecifiedURL: false,
		Hosts: []model.Instance{
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.10-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.10",
				Weight:      10,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO1",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.11-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.11",
				Weight:      9,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.12-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.12",
				Weight:      8,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     false,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.13-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.13",
				Weight:      7,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      false,
				Healthy:     true,
			},
			{
				Valid:       true,
				Marked:      false,
				InstanceId:  "10.10.10.14-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.14",
				Weight:      6,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEMO",
				Enable:      true,
				Healthy:     true,
			},
		},
		Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime: 1528787794594, Env: "", Clusters: "a",
		Metadata: map[string]string(nil)})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chooser := newChooser(services.Hosts)
		chooser.pick()
	}
}
