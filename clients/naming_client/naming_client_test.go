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
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/assert"
)

var clientConfigTest = *constant.NewClientConfig(
	constant.WithTimeoutMs(10*1000),
	constant.WithBeatInterval(5*1000),
	constant.WithNotLoadCacheAtStart(true),
)

var serverConfigTest = *constant.NewServerConfig("127.0.0.1", 80, constant.WithContextPath("/nacos"))

type MockNamingProxy struct {
}

func (m *MockNamingProxy) RegisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error) {
	return true, nil
}

func (m *MockNamingProxy) BatchRegisterInstance(serviceName string, groupName string, instances []model.Instance) (bool, error) {
	return true, nil
}

func (m *MockNamingProxy) DeregisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error) {
	return true, nil
}

func (m *MockNamingProxy) GetServiceList(pageNo uint32, pageSize uint32, groupName, namespaceId string, selector *model.ExpressionSelector) (model.ServiceList, error) {
	return model.ServiceList{Doms: []string{""}}, nil
}

func (m *MockNamingProxy) ServerHealthy() bool {
	return true
}

func (m *MockNamingProxy) QueryInstancesOfService(serviceName, groupName, clusters string, udpPort int, healthyOnly bool) (*model.Service, error) {
	return &model.Service{}, nil
}

func (m *MockNamingProxy) Subscribe(serviceName, groupName, clusters string) (model.Service, error) {
	return model.Service{}, nil
}

func (m *MockNamingProxy) Unsubscribe(serviceName, groupName, clusters string) error {
	return nil
}

func (m *MockNamingProxy) CloseClient() {}

func NewTestNamingClient() *NamingClient {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	_ = nc.SetClientConfig(clientConfigTest)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewNamingClient(&nc)
	client.serviceProxy = &MockNamingProxy{}
	return client
}
func Test_RegisterServiceInstance_withoutGroupName(t *testing.T) {
	success, err := NewTestNamingClient().RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		Ephemeral:   false,
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, true, success)
}

func Test_RegisterServiceInstance_withGroupName(t *testing.T) {
	success, err := NewTestNamingClient().RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
		Ephemeral:   false,
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, true, success)
}

func Test_RegisterServiceInstance_withCluster(t *testing.T) {
	success, err := NewTestNamingClient().RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
		ClusterName: "test",
		Ephemeral:   false,
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, true, success)
}
func TestNamingProxy_DeregisterService_WithoutGroupName(t *testing.T) {
	success, err := NewTestNamingClient().DeregisterInstance(vo.DeregisterInstanceParam{
		ServiceName: "DEMO5",
		Ip:          "10.0.0.10",
		Port:        80,
		Ephemeral:   true,
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, true, success)
}

func TestNamingProxy_DeregisterService_WithGroupName(t *testing.T) {
	success, err := NewTestNamingClient().DeregisterInstance(vo.DeregisterInstanceParam{
		ServiceName: "DEMO6",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
		Ephemeral:   true,
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, true, success)
}

func TestNamingClient_SelectOneHealthyInstance_SameWeight(t *testing.T) {
	services := model.Service{
		Name:        "DEFAULT_GROUP@@DEMO",
		CacheMillis: 1000,
		Hosts: []model.Instance{
			{
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
		LastRefTime: 1528787794594, Clusters: "a"}
	instance1, err := NewTestNamingClient().selectOneHealthyInstances(services)
	assert.Nil(t, err)
	assert.NotNil(t, instance1)
	instance2, err := NewTestNamingClient().selectOneHealthyInstances(services)
	assert.Nil(t, err)
	assert.NotNil(t, instance2)
}

func TestNamingClient_SelectOneHealthyInstance_Empty(t *testing.T) {
	services := model.Service{
		Name:        "DEFAULT_GROUP@@DEMO",
		CacheMillis: 1000,
		Hosts:       []model.Instance{},
		Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime: 1528787794594, Clusters: "a"}
	instance, err := NewTestNamingClient().selectOneHealthyInstances(services)
	assert.NotNil(t, err)
	assert.Nil(t, instance)
}

func TestNamingClient_SelectInstances_Healthy(t *testing.T) {
	services := model.Service{
		Name:        "DEFAULT_GROUP@@DEMO",
		CacheMillis: 1000,
		Hosts: []model.Instance{
			{
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
		LastRefTime: 1528787794594, Clusters: "a"}
	instances, err := NewTestNamingClient().selectInstances(services, true)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(instances))
}

func TestNamingClient_SelectInstances_Unhealthy(t *testing.T) {
	services := model.Service{
		Name:        "DEFAULT_GROUP@@DEMO",
		CacheMillis: 1000,
		Hosts: []model.Instance{
			{
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
		LastRefTime: 1528787794594, Clusters: "a"}
	instances, err := NewTestNamingClient().selectInstances(services, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(instances))
}

func TestNamingClient_SelectInstances_Empty(t *testing.T) {
	services := model.Service{
		Name:        "DEFAULT_GROUP@@DEMO",
		CacheMillis: 1000,
		Hosts:       []model.Instance{},
		Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime: 1528787794594, Clusters: "a"}
	instances, err := NewTestNamingClient().selectInstances(services, false)
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(instances))
}

func TestNamingClient_GetAllServicesInfo(t *testing.T) {
	result, err := NewTestNamingClient().GetAllServicesInfo(vo.GetAllServiceInfoParam{
		GroupName: "DEFAULT_GROUP",
		PageNo:    1,
		PageSize:  20,
	})

	assert.NotNil(t, result.Doms)
	assert.Nil(t, err)
}

func BenchmarkNamingClient_SelectOneHealthyInstances(b *testing.B) {
	services := model.Service{
		Name:        "DEFAULT_GROUP@@DEMO",
		CacheMillis: 1000,
		Hosts: []model.Instance{
			{
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
		LastRefTime: 1528787794594, Clusters: "a"}
	client := NewTestNamingClient()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.selectOneHealthyInstances(services)
	}

}
