package example

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func ExampleServiceClient_RegisterServiceInstance(client naming_client.INamingClient, param vo.RegisterServiceInstanceParam) {
	success, _ := client.RegisterServiceInstance(param)
	fmt.Println(success)
}

func ExampleServiceClient_DeRegisterServiceInstance(client naming_client.INamingClient, param vo.LogoutServiceInstanceParam) {
	success, _ := client.LogoutServiceInstance(param)
	fmt.Println(success)
}

func ExampleServiceClient_GetService(client naming_client.INamingClient) {
	service, _ := client.GetService(vo.GetServiceParam{
		ServiceName: "demo",
		Clusters:    []string{"a"},
	})
	fmt.Println(utils.ToJsonString(service))
}

func ExampleServiceClient_Subscribe(client naming_client.INamingClient, param *vo.SubscribeParam) {
	client.Subscribe(param)
}

func ExampleServiceClient_UnSubscribe(client naming_client.INamingClient, param *vo.SubscribeParam) {
	client.Unsubscribe(param)
}
