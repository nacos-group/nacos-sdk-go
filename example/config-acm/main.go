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
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"time"
)

var localServerConfigWithOptions = constant.NewServerConfig(
	"mse-d12e6112-p.nacos-ans.mse.aliyuncs.com",
	8848,
)

var localClientConfigWithOptions = constant.NewClientConfig(
	constant.WithTimeoutMs(10*1000),
	constant.WithBeatInterval(2*1000),
	constant.WithNotLoadCacheAtStart(true),
	//constant.WithAccessKey(getFileContent(path.Join(getWDR(), "ak"))),
	//constant.WithSecretKey(getFileContent(path.Join(getWDR(), "sk"))),
	constant.WithAccessKey("LTAxxxgQL"),
	constant.WithSecretKey("iG4xxxV6C"),
	constant.WithNamespaceId("791fd262-3735-40df-a605-e3236f8ff495"),
	constant.WithOpenKMS(true),
	constant.WithKMSVersion(constant.KMSv1),
	constant.WithRegionId("cn-beijing"),
)

var localConfigList = []vo.ConfigParam{
	{
		DataId:  "common-config",
		Group:   "default",
		Content: "common",
	},
	{
		DataId:  "cipher-crypt",
		Group:   "default",
		Content: "cipher",
	},
	{
		DataId:  "cipher-kms-aes-128-crypt",
		Group:   "default",
		Content: "cipher-aes-128",
	},
	{
		DataId:  "cipher-kms-aes-256-crypt",
		Group:   "default",
		Content: "cipher-aes-256",
	},
}

func main() {

	client, err := createConfigClient()
	if err != nil {
		panic(err)
	}

	for _, localConfig := range localConfigList {
		// to enable encrypt/decrypt, DataId should be start with "cipher-"
		configParam := vo.ConfigParam{
			DataId:  localConfig.DataId,
			Group:   localConfig.Group,
			Content: localConfig.Content,
			OnChange: func(namespace, group, dataId, data string) {
				fmt.Printf("successfully receive changed config: \n"+
					"group[%s], dataId[%s], data[%s]\n", group, dataId, data)
			},
		}

		err := client.ListenConfig(configParam)
		if err != nil {
			fmt.Printf("failed to listen: group[%s], dataId[%s] with error: %s\n",
				configParam.Group, configParam.DataId, err)
		} else {
			fmt.Printf("successfully ListenConfig: group[%s], dataId[%s]\n", configParam.Group, configParam.DataId)
		}

		published, err := client.PublishConfig(configParam)
		if published && err == nil {
			fmt.Printf("successfully publish: group[%s], dataId[%s], data[%s]\n", configParam.Group, configParam.DataId, configParam.Content)
		} else {
			fmt.Printf("failed to publish: group[%s], dataId[%s], data[%s]\n with error: %s\n",
				configParam.Group, configParam.DataId, configParam.Content, err)
		}

		//wait for config change callback to execute
		time.Sleep(2 * time.Second)

		//get config
		content, err := client.GetConfig(configParam)
		if err == nil {
			fmt.Printf("successfully get config: group[%s], dataId[%s], data[%s]\n", configParam.Group, configParam.DataId, configParam.Content)
		} else {
			fmt.Printf("failed to get config: group[%s], dataId[%s], data[%s]\n with error: %s\n",
				configParam.Group, configParam.DataId, configParam.Content, err)
		}

		if content != localConfig.Content {
			panic("publish/get encrypted config failed.")
		} else {
			fmt.Println("publish/get encrypted config success.")
		}
		//wait for config change callback to execute
		//time.Sleep(2 * time.Second)
	}

}

func createConfigClient() (*config_client.ConfigClient, error) {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*localServerConfigWithOptions})
	_ = nc.SetClientConfig(*localClientConfigWithOptions)
	fmt.Println("ak: " + localClientConfigWithOptions.AccessKey)
	fmt.Println("sk: " + localClientConfigWithOptions.SecretKey)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, err := config_client.NewConfigClient(&nc)
	if err != nil {
		return nil, err
	}
	return client, nil
}
