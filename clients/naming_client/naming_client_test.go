package naming_client

import (
	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/mock"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

var clientConfigTest = constant.ClientConfig{
	TimeoutMs:           20 * 1000,
	BeatInterval:        5 * 1000,
	ListenInterval:      10 * 1000,
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
		gomock.Eq(uint64(20*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "public",
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
	success, err := client.RegisterServiceInstance(vo.RegisterServiceInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
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
	success, err := client.RegisterServiceInstance(vo.RegisterServiceInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
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
		gomock.Eq(uint64(20*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "public",
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
	success, err := client.RegisterServiceInstance(vo.RegisterServiceInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
		ClusterName: "test",
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
		})).Times(3).
		Return(http_agent.FakeHttpResponse(401, `no auth`), nil)

	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	result, err := client.RegisterServiceInstance(vo.RegisterServiceInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
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
		gomock.Eq(uint64(20*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "public",
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
	client.LogoutServiceInstance(vo.LogoutServiceInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
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
		gomock.Eq(uint64(20*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "public",
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
	client.LogoutServiceInstance(vo.LogoutServiceInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
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
		gomock.Eq(uint64(20*1000)),
		gomock.Eq(map[string]string{
			"namespaceId": "public",
			"serviceName": "test_group@@DEMO",
			"clusterName": "",
			"ip":          "10.0.0.10",
			"port":        "80",
			"ephemeral":   "true",
		})).Times(3).
		Return(http_agent.FakeHttpResponse(401, `no auth`), nil)
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockIHttpAgent)
	client, _ := NewNamingClient(&nc)
	client.LogoutServiceInstance(vo.LogoutServiceInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		GroupName:   "test_group",
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
			}],
			"checksum": "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
			"lastRefTime": 1528787794594,
			"env": "",
			"clusters": "a"
		}`

var serviceTest = model.Service(model.Service{Name: "DEFAULT_GROUP@@DEMO",
	CacheMillis: 1000, UseSpecifiedURL: false,
	Hosts: []model.Host{
		model.Host{Valid: true, Marked: false, InstanceId: "10.10.10.10-8888-a-DEMO", Port: 0x22b8,
			Ip:     "10.10.10.10",
			Weight: 1, Metadata: map[string]string{}, ClusterName: "a",
			ServiceName: "DEMO", Enable: true}}, Checksum: "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
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
		gomock.Eq(uint64(20*1000)),
		gomock.Any()).Times(1).
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
