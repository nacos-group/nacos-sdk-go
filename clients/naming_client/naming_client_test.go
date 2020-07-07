package naming_client

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/mock"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
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
	"Client-Version":  []string{constant.CLIENT_VERSION},
	"User-Agent":      []string{constant.CLIENT_VERSION},
	"Accept-Encoding": []string{"gzip,deflate,sdch"},
	"Connection":      []string{"Keep-Alive"},
	"Request-Module":  []string{"Naming"},
	"Content-Type":    []string{"application/x-www-form-urlencoded"},
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
			"metadata":    "null",
			"ephemeral":   "true",
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
		Ephemeral:   true,
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
		gomock.Eq(uint64(20*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "public",
			"serviceName": "test_group@@DEMO",
			"groupName":   "test_group",
			"clusterName": "",
			"ip":          "10.0.0.10",
			"port":        "80",
			"weight":      "0",
			"enable":      "false",
			"healthy":     "false",
			"metadata":    "null",
			"ephemeral":   "true",
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
		GroupName:   "test_group",
		Ephemeral:   true,
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
			"serviceName": "test_group@@DEMO",
			"groupName":   "test_group",
			"clusterName": "test",
			"ip":          "10.0.0.10",
			"port":        "80",
			"weight":      "0",
			"enable":      "false",
			"healthy":     "false",
			"metadata":    "null",
			"ephemeral":   "true",
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
		GroupName:   "test_group",
		ClusterName: "test",
		Ephemeral:   true,
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
			"serviceName": "test_group@@DEMO",
			"groupName":   "test_group",
			"clusterName": "",
			"ip":          "10.0.0.10",
			"port":        "80",
			"weight":      "0",
			"enable":      "false",
			"healthy":     "false",
			"metadata":    "null",
			"ephemeral":   "true",
		})).Times(3).
		Return(http_agent.FakeHttpResponse(401, `no security`), nil)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	result, err := client.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
		Ephemeral:   true,
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
			"serviceName": "DEFAULT_GROUP@@DEMO",
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
		ServiceName: "DEMO",
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
			"serviceName": "test_group@@DEMO",
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
		ServiceName: "DEMO",
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
			"serviceName": "test_group@@DEMO",
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
		ServiceName: "DEMO",
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

func TestNamingProxy_GetService_WithoutGroupName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	mockIHttpAgent.EXPECT().Request(gomock.Eq("GET"),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance/list"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(uint64(10*1000)),
		gomock.Any()).Times(2).
		Return(http_agent.FakeHttpResponse(200, serviceJsonTest), nil)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	result, err := client.GetService(vo.GetServiceParam{
		ServiceName: "DEMO",
		Clusters:    []string{"a"},
	})
	assert.Nil(t, err)
	assert.Equal(t, serviceTest, result)

}

func TestNamingClient_SelectAllInstancs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()
	mockIHttpAgent := mock.NewMockIHttpAgent(ctrl)

	mockIHttpAgent.EXPECT().Request(gomock.Eq("GET"),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/ns/instance/list"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(uint64(10*1000)),
		gomock.Any()).Times(2).
		Return(http_agent.FakeHttpResponse(200, serviceJsonTest), nil)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	instances, err := client.SelectAllInstances(vo.SelectAllInstancesParam{
		ServiceName: "DEMO",
		Clusters:    []string{"a"},
	})
	fmt.Println(utils.ToJsonString(instances))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(instances))
}

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
	instance1, err := client.selectOneHealthyInstances(services)
	fmt.Println(utils.ToJsonString(instance1))
	assert.Nil(t, err)
	assert.NotNil(t, instance1)
	instance2, err := client.selectOneHealthyInstances(services)
	fmt.Println(utils.ToJsonString(instance2))
	assert.Nil(t, err)
	assert.NotNil(t, instance2)
	assert.NotEqual(t, instance1, instance2)
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
	fmt.Println(utils.ToJsonString(instance))
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
	fmt.Println(utils.ToJsonString(instances))
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
	fmt.Println(utils.ToJsonString(instances))
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
	fmt.Println(utils.ToJsonString(instances))
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
	})
	fmt.Println(len(reslut))
	assert.NotNil(t, err)
}
