package main

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/logger"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func Init() {
	// 可以没有，采用默认值
	clientConfig := constant.ClientConfig{
		TimeoutMs:      10 * 1000,
		ListenInterval: 30 * 1000,
		BeatInterval:   5 * 1000,
		LogDir:         "nacos/logs",
		CacheDir:       "nacos/cache",
		//SecretKey:      "",
		Username: "nacos",
		Password: "nacos",
	}

	// 至少一个
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      "127.0.0.1",
			ContextPath: "/nacos",
			Port:        8848,
		},
	}

	namingClient, err := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	if err != nil {
		fmt.Println("namingClient:", err)
	}

	configClient, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})

	if err != nil {
		fmt.Println(err)
	}

	success, _ := namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          "127.0.0.1",
		Port:        8089,
		ServiceName: "nacos-demo",
		Weight:      10,
		ClusterName: "DEFAULT_GROUP",
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	})
	logger.Info.Println("namingClient:", success)

	//instance, err := namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
	//	ServiceName: "nacos-demo",
	//	GroupName: "DEFAULT_GROUP",
	//	//Clusters:    []string{"DEFAULT_GROUP"},
	//})

	//logger.Info.Println("instance:", instance)

	success, err = configClient.PublishConfig(vo.ConfigParam{
		DataId:  "nacos-demo",
		Group:   "DEFAULT_GROUP",
		Content: "msg=hello world!222222"})

	logger.Info.Println("configClient:", success, err)

	content, err := configClient.GetConfig(vo.ConfigParam{
		DataId: "nacos-demo",
		Group:  "DEFAULT_GROUP"})
	logger.Info.Println("GetContent:", content)

	configClient.ListenConfig(vo.ConfigParam{
		DataId: "nacos-demo",
		Group:  "DEFAULT_GROUP",
		OnChange: func(namespace, group, dataId, data string) {
			logger.Info.Println("namespace, group, dataId, data:", namespace, group, dataId, data)
		},
	})
	logger.Info.Println("configClient.ListenConfig")
}

func main() {
	Init()

	select {}
}
