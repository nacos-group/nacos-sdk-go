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
	"io/ioutil"
	"os"
)

var localServerConfigWithOptions = constant.NewServerConfig(
	"mse-b3c2b0a2-p.nacos-ans.mse.aliyuncs.com",
	8848,
)

var localClientConfigWithOptions = constant.NewClientConfig(
	constant.WithTimeoutMs(10*1000),
	constant.WithBeatInterval(2*1000),
	constant.WithNotLoadCacheAtStart(true),
	//constant.WithAccessKey(getFileContent(path.Join(getWDR(), "ak"))),
	//constant.WithSecretKey(getFileContent(path.Join(getWDR(), "sk"))),
	constant.WithAccessKey("LTAI5xxxxxxxx21E6"),
	constant.WithSecretKey("kr6xxxxxxxxxsD6"),
	constant.WithNamespaceId("791fd262-3735-40df-a605-e3236f8ff495"),
	constant.WithOpenKMS(true),
	constant.WithRegionId("cn-beijing"),
)

var localConfig = vo.ConfigParam{
	DataId:  "cipher-crypt-1",
	Group:   "default",
	Content: "crypt",
}

func main() {

	client, err := createConfigClient()
	if err != nil {
		panic(err)
	}

	// to enable encrypt/decrypt, DataId should be start with "cipher-"
	_, err = client.PublishConfig(vo.ConfigParam{
		DataId:  localConfig.DataId,
		Group:   localConfig.Group,
		Content: localConfig.Content,
	})

	if err != nil {
		fmt.Printf("PublishConfig err: %v\n", err)
	} else {
		fmt.Printf("successfully PublishConfig: %s\n", localConfig.Content)
	}

	//get config
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: localConfig.DataId,
		Group:  localConfig.Group,
	})
	if err != nil {
		fmt.Printf("GetConfig err: %v\n", err)
	} else {
		fmt.Printf("Successfully GetConfig : %v\n", content)
	}

	if content != localConfig.Content {
		panic("publish/get encrypted config failed.")
	} else {
		fmt.Println("publish/get encrypted config success.")
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

func getWDR() string {
	getwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return getwd
}

func getFileContent(filePath string) string {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return string(file)
}
