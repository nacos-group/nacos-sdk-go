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

	//查询所有命名空间
	namespacesInfo, err := client.GetAllNamespacesInfo()
	if err != nil {
		return
	}
	jsonStr, err := json.Marshal(namespacesInfo)
	if err != nil {
		return
	}
	fmt.Println(string(jsonStr))

	for _, ns := range namespacesInfo {
		if ns.NamespaceShowName == "mynamespace" {
			delSuccess, err := client.DeleteNamespace(vo.DeleteNamespaceParam{NamespaceId: ns.Namespace})
			fmt.Printf("delete namespace: %s, %s %v\n", ns.Namespace, delSuccess, err)
		}
	}

	hasNamespace, err := client.CreateNamespace(vo.CreateNamespaceParam{
		CustomNamespaceId: "9d6afb07-b039-4b34-a941-1a65af7c6ecb",
		NamespaceName:     "mynamespace",
		NamespaceDesc:     "my namespace",
	})
	fmt.Printf("create namespace : %s %v\n", hasNamespace, err)

	modifySuccess, err := client.ModifyNamespace(vo.ModifyNamespaceParam{
		NamespaceId:   "9d6afb07-b039-4b34-a941-1a65af7c6ecb",
		NamespaceName: "开发环境2",
		NamespaceDesc: "modify my namespace",
	})

	fmt.Printf("modify namespace : %s %v\n", modifySuccess, err)

}
