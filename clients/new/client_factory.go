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

package new

import (
	"errors"

	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
)

// CreateConfigClient use to create config client
func CreateConfigClient(configs *constant.Config) (iClient config_client.IConfigClient, err error) {
	nacosClient, err := setConfig(configs)
	if err != nil {
		return
	}
	nacosClient.SetHttpAgent(&http_agent.HttpAgent{})
	config, err := config_client.NewConfigClient(nacosClient)
	if err != nil {
		return
	}
	iClient = &config
	return
}

//CreateNamingClient use to create a nacos naming client
func CreateNamingClient(configs *constant.Config) (iClient naming_client.INamingClient, err error) {
	nacosClient, err := setConfig(configs)
	if err != nil {
		return
	}
	nacosClient.SetHttpAgent(&http_agent.HttpAgent{})
	naming, err := naming_client.NewNamingClient(nacosClient)
	if err != nil {
		return
	}
	iClient = &naming
	return
}

func setConfig(configs *constant.Config) (iClient nacos_client.INacosClient, err error) {
	client := nacos_client.NacosClient{}

	if configs.ClientConfig == nil {
		err = client.SetClientConfig(*configs.ClientConfig)
	} else {
		_ = client.SetClientConfig(constant.ClientConfig{
			TimeoutMs:    10 * 1000,
			BeatInterval: 5 * 1000,
		})
	}

	if configs.ServerConfigs != nil {
		err = client.SetServerConfig(*configs.ServerConfigs)
	} else {
		clientConfig, _ := client.GetClientConfig()
		if len(clientConfig.Endpoint) <= 0 {
			err = errors.New("server configs not found in properties")
			return
		}
		_ = client.SetServerConfig([]constant.ServerConfig{})
	}

	iClient = &client

	return
}
