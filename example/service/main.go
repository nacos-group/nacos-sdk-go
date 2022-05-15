/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func main() {
	// create ServerConfig
	sc := []constant.ServerConfig{
		*constant.NewServerConfig(os.Getenv("nacos_server_address"), 8848, constant.WithContextPath("/nacos")),
	}

	username := os.Getenv("nacos_username")
	passwd := os.Getenv("nacos_password")
	// create ClientConfig
	cc := *constant.NewClientConfig(
		constant.WithNamespaceId(""),
		constant.WithUsername(username),
		constant.WithPassword(passwd),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
	)

	// create naming client
	client, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)

	if err != nil {
		panic(err)
	}

	clusterName := os.Getenv("nacos_cluster_name")
	// Register
	ExampleServiceClient_RegisterServiceInstance(client, vo.RegisterInstanceParam{
		Ip:          "10.0.0.10",
		Port:        8848,
		ServiceName: "demo.go",
		GroupName:   "group-a",
		ClusterName: clusterName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"idc": "shanghai"},
	})

	// DeRegister
	ExampleServiceClient_DeRegisterServiceInstance(client, vo.DeregisterInstanceParam{
		Ip:          "10.0.0.10",
		Port:        8848,
		ServiceName: "demo.go",
		GroupName:   "group-a",
		Cluster:     clusterName,
		Ephemeral:   true, // it must be true
	})

	// Register
	ExampleServiceClient_RegisterServiceInstance(client, vo.RegisterInstanceParam{
		Ip:          "10.0.0.10",
		Port:        8848,
		ServiceName: "demo.go",
		GroupName:   "group-a",
		ClusterName: clusterName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"idc": "shanghai"},
	})

	time.Sleep(1 * time.Second)

	// Get service with serviceName, groupName , clusters
	ExampleServiceClient_GetService(client, vo.GetServiceParam{
		ServiceName: "demo.go",
		GroupName:   "group-a",
		Clusters:    []string{clusterName},
	})

	// SelectAllInstance
	// GroupName=DEFAULT_GROUP
	ExampleServiceClient_SelectAllInstances(client, vo.SelectAllInstancesParam{
		ServiceName: "demo.go",
		GroupName:   "group-a",
		Clusters:    []string{clusterName},
	})

	// SelectInstances only return the instances of healthy=${HealthyOnly},enable=true and weight>0
	// ClusterName=DEFAULT,GroupName=DEFAULT_GROUP
	ExampleServiceClient_SelectInstances(client, vo.SelectInstancesParam{
		ServiceName: "demo.go",
		GroupName:   "group-a",
		Clusters:    []string{clusterName},
	})

	// SelectOneHealthyInstance return one instance by WRR strategy for load balance
	// And the instance should be health=true,enable=true and weight>0
	// ClusterName=DEFAULT,GroupName=DEFAULT_GROUP
	ExampleServiceClient_SelectOneHealthyInstance(client, vo.SelectOneHealthInstanceParam{
		ServiceName: "demo.go",
		GroupName:   "group-a",
		Clusters:    []string{clusterName},
	})

	// Subscribe key=serviceName+groupName+cluster
	// Note:We call add multiple SubscribeCallback with the same key.
	param1 := &vo.SubscribeParam{
		ServiceName: "demo.go",
		GroupName:   "group-a",
		SubscribeCallback: func(services []model.Instance, err error) {
			fmt.Printf("callback1 return services:%s \n\n", util.ToJsonString(services))
		},
	}
	ExampleServiceClient_Subscribe(client, param1)
	// Subscribe key=serviceName+groupName+cluster, this is second subscribe
	// it will append callback2 to the first subscribe callback slice, if the serviceName has some events change,
	// all callback will be range to call.
	// NOTE: if you want to unsubscribe, you can call UnSubscribe with the same param struct pointer, otherwise it will not be
	// removed the old callback in the slice. will cause a memory leak.
	param2 := &vo.SubscribeParam{
		ServiceName: "demo.go",
		GroupName:   "group-a",
		SubscribeCallback: func(services []model.Instance, err error) {
			fmt.Printf("callback2 return services:%s \n\n", util.ToJsonString(services))
		},
	}
	ExampleServiceClient_Subscribe(client, param2)

	ExampleServiceClient_RegisterServiceInstance(client, vo.RegisterInstanceParam{
		Ip:          "10.0.0.10",
		Port:        8848,
		ServiceName: "demo.go",
		GroupName:   "group-a",
		ClusterName: clusterName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"idc": "beijing"},
	})
	// wait for client pull change from server
	time.Sleep(3 * time.Second)

	ExampleServiceClient_UpdateServiceInstance(client, vo.UpdateInstanceParam{
		Ip:          "10.0.0.11", // update ip
		Port:        8848,
		ServiceName: "demo.go",
		GroupName:   "group-a",
		ClusterName: clusterName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"idc": "beijing1"}, // update metadata
	})

	// wait for client pull change from server
	time.Sleep(3 * time.Second)

	// Now we just unsubscribe callback1, will remove callback1 in callback slice,
	// and callback2 will not remove, also not receive change event
	// if you subscribe again, the callback2 will receive change event
	ExampleServiceClient_UnSubscribe(client, param1)
	// Now we unsubscribe callback2, will remove callback2 in callback slice.
	ExampleServiceClient_UnSubscribe(client, param2)

	ExampleServiceClient_DeRegisterServiceInstance(client, vo.DeregisterInstanceParam{
		Ip:          "10.0.0.112",
		Ephemeral:   true,
		Port:        8848,
		ServiceName: "demo.go",
		Cluster:     "cluster-b",
	})
	// wait for client pull change from server
	time.Sleep(3 * time.Second)

	// GeAllService will get the list of service name
	// NameSpace default value is public.If the client set the namespaceId, NameSpace will use it.
	// GroupName default value is DEFAULT_GROUP
	ExampleServiceClient_GetAllService(client, vo.GetAllServiceInfoParam{
		PageNo:   1,
		PageSize: 10,
	})
}
