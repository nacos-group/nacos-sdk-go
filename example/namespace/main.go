package main

import (
	"encoding/json"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

var (
	sc = []constant.ServerConfig{
		*constant.NewServerConfig("console.nacos.io", 80, constant.WithContextPath("/nacos")),
	}

	cc = *constant.NewClientConfig(
		constant.WithTimeoutMs(10*1000),
		constant.WithBeatInterval(5*1000),
		constant.WithNotLoadCacheAtStart(true),
	)
)

func main() {

	// a more graceful way to create naming client
	client, err := clients.NewNamespaceClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)

	if err != nil {
		panic(err)
	}

	namespacesInfo, err := client.GetAllNamespacesInfo()
	if err != nil {
		return
	}
	jsonStr, err := json.Marshal(namespacesInfo)
	if err != nil {
		return
	}
	fmt.Println(string(jsonStr))

}
