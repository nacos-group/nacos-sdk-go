package naming_client

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/utils"

	"github.com/stretchr/testify/assert"

	"github.com/nacos-group/nacos-sdk-go/vo"

	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
)

func TestHostReactor_GetServiceInfo(t *testing.T) {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewNamingClient(&nc)
	param := vo.RegisterInstanceParam{
		Ip:          "10.0.0.11",
		Port:        8848,
		ServiceName: "test",
		Weight:      10,
		ClusterName: "test",
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	}
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	param.ServiceName = utils.GetGroupName(param.ServiceName, param.GroupName)
	client.RegisterInstance(param)
	_, err := client.hostReactor.GetServiceInfo(param.ServiceName, param.ClusterName)
	assert.Nil(t, err)
}

func TestHostReactor_GetServiceInfoErr(t *testing.T) {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewNamingClient(&nc)
	param := vo.RegisterInstanceParam{
		Ip:          "10.0.0.11",
		Port:        8848,
		ServiceName: "test",
		Weight:      10,
		ClusterName: "test",
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	}
	client.RegisterInstance(param)
	_, err := client.hostReactor.GetServiceInfo(param.ServiceName, param.ClusterName)
	assert.NotNil(t, err)
}

func TestHostReactor_GetServiceInfoConcurrent(t *testing.T) {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewNamingClient(&nc)
	param := vo.RegisterInstanceParam{
		Ip:          "10.0.0.11",
		Port:        8848,
		ServiceName: "test",
		Weight:      10,
		ClusterName: "test",
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	}
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	param.ServiceName = utils.GetGroupName(param.ServiceName, param.GroupName)
	client.RegisterInstance(param)
	for i := 0; i < 10000; i++ {
		go func() {
			_, err := client.hostReactor.GetServiceInfo(param.ServiceName, param.ClusterName)
			assert.Nil(t, err)
		}()

	}
}

func BenchmarkHostReactor_GetServiceInfoConcurrent(b *testing.B) {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewNamingClient(&nc)
	param := vo.RegisterInstanceParam{
		Ip:          "10.0.0.11",
		Port:        8848,
		ServiceName: "test",
		Weight:      10,
		ClusterName: "test",
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	}
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	param.ServiceName = utils.GetGroupName(param.ServiceName, param.GroupName)
	client.RegisterInstance(param)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.hostReactor.GetServiceInfo(param.ServiceName, param.ClusterName)
	}
}
