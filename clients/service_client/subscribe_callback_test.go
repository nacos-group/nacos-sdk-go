package service_client

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"log"
	"strings"
	"testing"
	"time"
)

func TestEventDispatcher_AddCallbackFuncs(t *testing.T) {
	service := model.Service{
		Dom:         "public@@Test",
		Clusters:    strings.Join([]string{"default"}, ","),
		CacheMillis: 10000,
		Checksum:    "abcd",
		LastRefTime: uint64(time.Now().Unix()),
	}
	var hosts []model.Host
	host := model.Host{
		Valid:       true,
		Enable:      true,
		InstanceId:  "123",
		Port:        8080,
		Ip:          "127.0.0.1",
		Weight:      10,
		ServiceName: "public@@Test",
		ClusterName: strings.Join([]string{"default"}, ","),
	}
	hosts = append(hosts, host)
	service.Hosts = hosts

	ed := NewSubscribeCallback()
	param := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			fmt.Println(utils.ToJsonString(ed.callbackFuncsMap))
		},
	}
	ed.AddCallbackFuncs(utils.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), &param.SubscribeCallback)
	for k, v := range ed.callbackFuncsMap.Items() {
		log.Printf("key:%s,%v", k, v)
	}
	if len(ed.callbackFuncsMap.Items()) == 1 {
		log.Println("add callback funcs success")
	} else {
		log.Panicln("add callback funcs failed")
	}
}

func TestEventDispatcher_RemoveCallbackFuncs(t *testing.T) {
	service := model.Service{
		Dom:         "public@@Test",
		Clusters:    strings.Join([]string{"default"}, ","),
		CacheMillis: 10000,
		Checksum:    "abcd",
		LastRefTime: uint64(time.Now().Unix()),
	}
	var hosts []model.Host
	host := model.Host{
		Valid:       true,
		Enable:      true,
		InstanceId:  "123",
		Port:        8080,
		Ip:          "127.0.0.1",
		Weight:      10,
		ServiceName: "public@@Test",
		ClusterName: strings.Join([]string{"default"}, ","),
	}
	hosts = append(hosts, host)
	service.Hosts = hosts

	ed := NewSubscribeCallback()
	param := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			fmt.Printf("func1:%s \n", utils.ToJsonString(services))
		},
	}
	ed.AddCallbackFuncs(utils.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), &param.SubscribeCallback)

	param2 := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			fmt.Printf("func2:%s \n", utils.ToJsonString(services))
		},
	}
	ed.AddCallbackFuncs(utils.GetGroupName(param2.ServiceName, param2.GroupName), strings.Join(param2.Clusters, ","), &param2.SubscribeCallback)

	for k, v := range ed.callbackFuncsMap.Items() {
		log.Printf("key:%s,%d", k, len(v.([]*func(services []model.SubscribeService, err error))))
	}

	ed.RemoveCallbackFuncs(utils.GetGroupName(param2.ServiceName, param2.GroupName), strings.Join(param2.Clusters, ","), &param2.SubscribeCallback)
	for k, v := range ed.callbackFuncsMap.Items() {
		log.Printf("key:%s,%d", k, len(v.([]*func(services []model.SubscribeService, err error))))
	}
}

func TestSubscribeCallback_ServiceChanged(t *testing.T) {
	service := model.Service{
		Dom:         "public@@Test",
		Clusters:    strings.Join([]string{"default"}, ","),
		CacheMillis: 10000,
		Checksum:    "abcd",
		LastRefTime: uint64(time.Now().Unix()),
	}
	var hosts []model.Host
	host := model.Host{
		Valid:       true,
		Enable:      true,
		InstanceId:  "123",
		Port:        8080,
		Ip:          "127.0.0.1",
		Weight:      10,
		ServiceName: "public@@Test",
		ClusterName: strings.Join([]string{"default"}, ","),
	}
	hosts = append(hosts, host)
	service.Hosts = hosts

	ed := NewSubscribeCallback()
	param := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			log.Printf("func1:%s \n", utils.ToJsonString(services))
		},
	}
	ed.AddCallbackFuncs(utils.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), &param.SubscribeCallback)

	param2 := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			log.Printf("func2:%s \n", utils.ToJsonString(services))

		},
	}
	ed.AddCallbackFuncs(utils.GetGroupName(param2.ServiceName, param2.GroupName), strings.Join(param2.Clusters, ","), &param2.SubscribeCallback)

	ed.ServiceChanged(&service)
}
