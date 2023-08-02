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
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func main() {
	//create ServerConfig
	sc := []constant.ServerConfig{
		*constant.NewServerConfig("127.0.0.1", 8848, constant.WithContextPath("/nacos")),
	}

	//create ClientConfig
	cc := *constant.NewClientConfig(
		constant.WithNamespaceId(""),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
	)

	// create config client
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)

	if err != nil {
		panic(err)
	}

	//publish config
	//config key=dataId+group+namespaceId
	_, err = client.PublishConfig(vo.ConfigParam{
		DataId:  "test-data",
		Group:   "test-group",
		Content: "hello world!",
	})
	_, err = client.PublishConfig(vo.ConfigParam{
		DataId:  "test-data-2",
		Group:   "test-group",
		Content: "hello world!",
	})
	if err != nil {
		fmt.Printf("PublishConfig err:%+v \n", err)
	}
	time.Sleep(1 * time.Second)
	//get config
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: "test-data",
		Group:  "test-group",
	})
	fmt.Println("GetConfig,config :" + content)

	//Listen config change,key=dataId+group+namespaceId.
	err = client.ListenConfig(vo.ConfigParam{
		DataId: "test-data",
		Group:  "test-group",
		OnChange: func(namespace, group, dataId, data string) {
			fmt.Println("config changed group:" + group + ", dataId:" + dataId + ", content:" + data)
		},
	})

	err = client.ListenConfig(vo.ConfigParam{
		DataId: "test-data-2",
		Group:  "test-group",
		OnChange: func(namespace, group, dataId, data string) {
			fmt.Println("config changed group:" + group + ", dataId:" + dataId + ", content:" + data)
		},
	})

	time.Sleep(1 * time.Second)

	_, err = client.PublishConfig(vo.ConfigParam{
		DataId:  "test-data",
		Group:   "test-group",
		Content: "test-listen",
	})

	time.Sleep(1 * time.Second)

	_, err = client.PublishConfig(vo.ConfigParam{
		DataId:  "test-data-2",
		Group:   "test-group",
		Content: "test-listen",
	})

	time.Sleep(2 * time.Second)

	time.Sleep(1 * time.Second)
	_, err = client.DeleteConfig(vo.ConfigParam{
		DataId: "test-data",
		Group:  "test-group",
	})
	time.Sleep(1 * time.Second)

	//cancel config change
	err = client.CancelListenConfig(vo.ConfigParam{
		DataId: "test-data",
		Group:  "test-group",
	})

	searchPage, _ := client.SearchConfig(vo.SearchConfigParam{
		Search:   "blur",
		DataId:   "",
		Group:    "",
		PageNo:   1,
		PageSize: 10,
	})
	fmt.Printf("Search config:%+v \n", searchPage)
}
