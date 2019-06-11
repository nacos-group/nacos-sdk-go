package example

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients/service_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func ExampleServiceClient_RegisterServiceInstance(client service_client.IServiceClient, param vo.RegisterServiceInstanceParam) {
	success, _ := client.RegisterServiceInstance(param)
	fmt.Println(success)
}

func ExampleServiceClient_DeRegisterServiceInstance(client service_client.IServiceClient, param vo.LogoutServiceInstanceParam) {
	success, _ := client.LogoutServiceInstance(param)
	fmt.Println(success)
}

func ExampleServiceClient_GetService(client service_client.IServiceClient) {
	service, _ := client.GetService(vo.GetServiceParam{
		ServiceName: "demo",
		Clusters:    []string{"a"},
	})
	fmt.Println(service)
}

func ExampleServiceClient_Subscribe(client service_client.IServiceClient, param *vo.SubscribeParam) {
	client.Subscribe(param)
}

func ExampleServiceClient_UnSubscribe(client service_client.IServiceClient, param *vo.SubscribeParam) {
	client.Unsubscribe(param)
}
