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

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func registerServiceInstance(client naming_client.INamingClient, param vo.RegisterInstanceParam) {
	success, err := client.RegisterInstance(param)
	if !success || err != nil {
		panic("RegisterServiceInstance failed!" + err.Error())
	}
	fmt.Printf("RegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)
}

func batchRegisterServiceInstance(client naming_client.INamingClient, param vo.BatchRegisterInstanceParam) {
	success, err := client.BatchRegisterInstance(param)
	if !success || err != nil {
		panic("BatchRegisterServiceInstance failed!" + err.Error())
	}
	fmt.Printf("BatchRegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)
}

func deRegisterServiceInstance(client naming_client.INamingClient, param vo.DeregisterInstanceParam) {
	success, err := client.DeregisterInstance(param)
	if !success || err != nil {
		panic("DeRegisterServiceInstance failed!" + err.Error())
	}
	fmt.Printf("DeRegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)
}

func updateServiceInstance(client naming_client.INamingClient, param vo.UpdateInstanceParam) {
	success, err := client.UpdateInstance(param)
	if !success || err != nil {
		panic("UpdateInstance failed!" + err.Error())
	}
	fmt.Printf("UpdateServiceInstance,param:%+v,result:%+v \n\n", param, success)
}

func getService(client naming_client.INamingClient, param vo.GetServiceParam) {
	service, err := client.GetService(param)
	if err != nil {
		panic("GetService failed!" + err.Error())
	}
	fmt.Printf("GetService,param:%+v, result:%+v \n\n", param, service)
}

func selectAllInstances(client naming_client.INamingClient, param vo.SelectAllInstancesParam) {
	instances, err := client.SelectAllInstances(param)
	if err != nil {
		panic("SelectAllInstances failed!" + err.Error())
	}
	fmt.Printf("SelectAllInstance,param:%+v, result:%+v \n\n", param, instances)
}

func selectInstances(client naming_client.INamingClient, param vo.SelectInstancesParam) {
	instances, err := client.SelectInstances(param)
	if err != nil {
		panic("SelectInstances failed!" + err.Error())
	}
	fmt.Printf("SelectInstances,param:%+v, result:%+v \n\n", param, instances)
}

func selectOneHealthyInstance(client naming_client.INamingClient, param vo.SelectOneHealthInstanceParam) {
	instances, err := client.SelectOneHealthyInstance(param)
	if err != nil {
		panic("SelectOneHealthyInstance failed!")
	}
	fmt.Printf("SelectOneHealthyInstance,param:%+v, result:%+v \n\n", param, instances)
}

func subscribe(client naming_client.INamingClient, param *vo.SubscribeParam) {
	client.Subscribe(param)
}

func unSubscribe(client naming_client.INamingClient, param *vo.SubscribeParam) {
	client.Unsubscribe(param)
}

func getAllService(client naming_client.INamingClient, param vo.GetAllServiceInfoParam) {
	service, err := client.GetAllServicesInfo(param)
	if err != nil {
		panic("GetAllService failed!")
	}
	fmt.Printf("GetAllService,param:%+v, result:%+v \n\n", param, service)
}
