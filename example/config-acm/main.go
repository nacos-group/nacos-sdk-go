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

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func main() {
	cc := constant.ClientConfig{
		Endpoint:    "acm.aliyun.com:8080",
		NamespaceId: "e525eafa-f7d7-4029-83d9-008937f9d468",
		RegionId:    "cn-shanghai",
		AccessKey:   "LTAI4G8KxxxxxxxxxxxxxbwZLBr",
		SecretKey:   "n5jTL9YxxxxxxxxxxxxaxmPLZV9",
		OpenKMS:     true,
		TimeoutMs:   5000,
	}

	// a more graceful way to create config client
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig: &cc,
		},
	)

	if err != nil {
		panic(err)
	}

	// to enable encrypt/decrypt, DataId should be start with "cipher-"
	_, err = client.PublishConfig(vo.ConfigParam{
		DataId:  "cipher-dataId-1",
		Group:   "test-group",
		Content: "hello world!",
	})

	if err != nil {
		fmt.Printf("PublishConfig err: %v\n", err)
	}

	//get config
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: "cipher-dataId-3",
		Group:  "test-group",
	})
	fmt.Printf("GetConfig, config: %s, error: %v\n", content, err)

	// DataId is not start with "cipher-", content will not be encrypted.
	_, err = client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId-1",
		Group:   "test-group",
		Content: "hello world!",
	})

	if err != nil {
		fmt.Printf("PublishConfig err: %v\n", err)
	}

	//get config
	content, err = client.GetConfig(vo.ConfigParam{
		DataId: "dataId-1",
		Group:  "test-group",
	})
	fmt.Printf("GetConfig, config: %s, error: %v\n", content, err)
}
