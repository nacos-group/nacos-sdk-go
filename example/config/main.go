package main

import (
	"fmt"
	"time"

	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

var clientConfigTest = constant.ClientConfig{
	TimeoutMs:           10 * 1000,
	BeatInterval:        5 * 1000,
	ListenInterval:      300 * 1000,
	NotLoadCacheAtStart: true,
	//Username:            "nacos",
	//Password:            "nacos",
}

var serverConfigTest = constant.ServerConfig{
	IpAddr:      "console.nacos.io",
	Port:        80,
	ContextPath: "/nacos",
}

func cretateConfigClientTest() config_client.ConfigClient {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := config_client.NewConfigClient(&nc)
	return client
}

func main() {
	client := cretateConfigClientTest()
	content, _ := client.GetConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "group",
	})
	fmt.Println("config :" + content)
	_, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "hello world!"})
	if err != nil {
		fmt.Printf("success err:%s", err.Error())
	}
	content = ""

	client.ListenConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "group",
		OnChange: func(namespace, group, dataId, data string) {
			fmt.Println("config changed group:" + group + ", dataId:" + dataId + ", data:" + data)
			content = data
		},
	})

	client.ListenConfig(vo.ConfigParam{
		DataId: "abc",
		Group:  "DEFAULT_GROUP",
		OnChange: func(namespace, group, dataId, data string) {
			fmt.Println("config changed group:" + group + ", dataId:" + dataId + ", data:" + data)
		},
	})

	time.Sleep(5 * time.Second)
	_, err = client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "abc"})

	time.Sleep(2 * time.Second)
	err = client.CancelListenConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "group"})
	if err == nil {
		fmt.Println("cancel listen config")
	}

	time.Sleep(2 * time.Second)
	ok, err := client.DeleteConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "group"})
	if ok && err == nil {
		fmt.Printf("delete config dataId:[%s] group:[%s]", "dataId", "group")
	}

	select {}

}
