package main

import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/example"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"log"
	"time"
)

func main() {
	client, err := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": []constant.ServerConfig{
			{
				IpAddr: "console.nacos.io",
				Port:   80,
			},
		},
		"clientConfig": constant.ClientConfig{
			TimeoutMs:           5000,
			ListenInterval:      10000,
			NotLoadCacheAtStart: true,
			LogDir:              "data/nacos/log",
			Username:			 "nacos",
			Password:			 "nacos",
		},
	})

	if err != nil {
		panic(err)
	}

	example.ExampleServiceClient_RegisterServiceInstance(client, vo.RegisterInstanceParam{
		Ip:          "10.0.0.11",
		Port:        8848,
		ServiceName: "demo.go",
		Weight:      10,
		ClusterName: "a",
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	})

	example.ExampleServiceClient_GetService(client)
	param := &vo.SubscribeParam{
		ServiceName: "demo.go",
		Clusters:    []string{"a"},
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			log.Printf("\n\n callback return services:%s \n\n", utils.ToJsonString(services))
		},
	}
	example.ExampleServiceClient_Subscribe(client, param)
	time.Sleep(20 * time.Second)
	example.ExampleServiceClient_RegisterServiceInstance(client, vo.RegisterInstanceParam{
		Ip:          "10.0.0.12",
		Port:        8848,
		ServiceName: "demo.go",
		Weight:      10,
		ClusterName: "a",
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	})
	time.Sleep(20 * time.Second)
	example.ExampleServiceClient_UnSubscribe(client, param)
	example.ExampleServiceClient_DeRegisterServiceInstance(client, vo.DeregisterInstanceParam{
		Ip:          "10.0.0.11",
		Ephemeral:   true,
		Port:        8848,
		ServiceName: "demo.go",
		Cluster:     "a",
	})

}
