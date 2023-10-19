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

package config_client

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/assert"
	"testing"
)

var localServerConfigWithOptions = constant.NewServerConfig(
	"mse-d12e6112-p.nacos-ans.mse.aliyuncs.com",
	8848,
)

var localClientConfigWithOptions = constant.NewClientConfig(
	constant.WithTimeoutMs(10*1000),
	constant.WithBeatInterval(2*1000),
	constant.WithNotLoadCacheAtStart(true),
	constant.WithAccessKey("LTAxxxgQL"),
	constant.WithSecretKey("iG4xxxV6C"),
	constant.WithNamespaceId("c7b5f400-ca71-4791-83bc-09f55e645808"),
	constant.WithOpenKMS(true),
	constant.WithRegionId("cn-beijing"),
)

var localConfigList = []vo.ConfigParam{
	{
		DataId:  "common-config",
		Group:   "default",
		Content: "普通配置",
	},
	{
		DataId:  "cipher-crypt",
		Group:   "default",
		Content: "加密",
	},
	{
		DataId:  "cipher-kms-aes-128-crypt",
		Group:   "default",
		Content: "加密aes-128",
	},
	{
		DataId:  "cipher-kms-aes-256-crypt",
		Group:   "default",
		Content: "加密aes-256",
	},
}

func TestConfigEncryptionFilter(t *testing.T) {

	localAssert := assert.New(t)

	client, err := createConfigClient()
	if err != nil {
		panic(err)
	}

	for _, localConfig := range localConfigList {
		// to enable encrypt/decrypt, DataId should be start with "cipher-"
		t.Run(localConfig.DataId, func(t *testing.T) {
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
			localAssert.Nil(err)

			published, err := client.PublishConfig(configParam)
			localAssert.True(published)
			localAssert.Nil(err)

			//wait for config change callback to execute
			//time.Sleep(2 * time.Second)

			//get config
			content, err := client.GetConfig(configParam)
			localAssert.True(content == localConfig.Content)
			localAssert.Nil(err)

			//wait for config change callback to execute
			//time.Sleep(2 * time.Second)
		})
	}

}

func createConfigClient() (*ConfigClient, error) {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*localServerConfigWithOptions})
	_ = nc.SetClientConfig(*localClientConfigWithOptions)
	fmt.Println("ak: " + localClientConfigWithOptions.AccessKey)
	fmt.Println("sk: " + localClientConfigWithOptions.SecretKey)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, err := NewConfigClient(&nc)
	if err != nil {
		return nil, err
	}
	return client, nil
}
