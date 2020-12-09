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

package factoryv2

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
)

func TestCreateNamingClient(t *testing.T) {
	serverConfigs := &[]constant.ServerConfig{
		{
			IpAddr: "console.nacos.io",
			Port:   80,
		},
	}

	clientConfig := &constant.ClientConfig{
		NamespaceId:         "e525eafa-f7d7-4029-83d9-008937f9d468", //namespace id
		TimeoutMs:           5000,
		ListenInterval:      10000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		RotateTime:          "1h",
		MaxAge:              3,
		LogLevel:            "debug",
	}

	client, err := CreateNamingClient(&constant.Config{
		ServerConfigs: serverConfigs,
		ClientConfig:  clientConfig},
	)
	assert.Nil(t, err)

	success, err := client.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		Ephemeral:   false,
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, true, success)

	servers, err := client.GetAllServicesInfo(vo.GetAllServiceInfoParam{
		PageNo:   1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, servers)
}

func TestCreateConfigClient(t *testing.T) {
	serverConfigs := &[]constant.ServerConfig{
		{
			IpAddr: "console.nacos.io",
			Port:   80,
		},
	}

	clientConfig := &constant.ClientConfig{
		NamespaceId:         "e525eafa-f7d7-4029-83d9-008937f9d468", //namespace id
		TimeoutMs:           5000,
		ListenInterval:      10000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		RotateTime:          "1h",
		MaxAge:              3,
		LogLevel:            "debug",
	}

	client, err := CreateConfigClient(&constant.Config{
		ServerConfigs: serverConfigs,
		ClientConfig:  clientConfig},
	)

	assert.Nil(t, err)

	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "hello world"})

	assert.Nil(t, err)
	assert.True(t, success)

	content, err := client.GetConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "group"})

	assert.Nil(t, err)
	assert.Equal(t, "hello world", content)
}
