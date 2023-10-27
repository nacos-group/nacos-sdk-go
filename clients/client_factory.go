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

package clients

import (
	"github.com/pkg/errors"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// CreateConfigClient use to create config client
func CreateConfigClient(properties map[string]interface{}) (iClient config_client.IConfigClient, err error) {
	param := getConfigParam(properties)
	return NewConfigClient(param)
}

// CreateNamingClient use to create a nacos naming client
func CreateNamingClient(properties map[string]interface{}) (iClient naming_client.INamingClient, err error) {
	param := getConfigParam(properties)
	return NewNamingClient(param)
}

func NewConfigClient(param vo.NacosClientParam) (iClient config_client.IConfigClient, err error) {
	nacosClient, err := setConfig(param)
	if err != nil {
		return
	}
	config, err := config_client.NewConfigClient(nacosClient)
	if err != nil {
		return
	}
	iClient = config
	return
}

func NewNamingClient(param vo.NacosClientParam) (iClient naming_client.INamingClient, err error) {
	nacosClient, err := setConfig(param)
	if err != nil {
		return
	}
	naming, err := naming_client.NewNamingClient(nacosClient)
	if err != nil {
		return
	}
	iClient = naming
	return
}

func getConfigParam(properties map[string]interface{}) (param vo.NacosClientParam) {

	if clientConfigTmp, exist := properties[constant.KEY_CLIENT_CONFIG]; exist {
		if clientConfig, ok := clientConfigTmp.(constant.ClientConfig); ok {
			param.ClientConfig = &clientConfig
		}
	}
	if serverConfigTmp, exist := properties[constant.KEY_SERVER_CONFIGS]; exist {
		if serverConfigs, ok := serverConfigTmp.([]constant.ServerConfig); ok {
			param.ServerConfigs = serverConfigs
		}
	}
	return
}

func setConfig(param vo.NacosClientParam) (iClient nacos_client.INacosClient, err error) {
	client := &nacos_client.NacosClient{}
	if param.ClientConfig == nil {
		// default clientConfig
		_ = client.SetClientConfig(constant.ClientConfig{
			TimeoutMs:    10 * 1000,
			BeatInterval: 5 * 1000,
		})
	} else {
		err = client.SetClientConfig(*param.ClientConfig)
		if err != nil {
			return nil, err
		}
	}

	if len(param.ServerConfigs) == 0 {
		clientConfig, _ := client.GetClientConfig()
		if len(clientConfig.Endpoint) <= 0 {
			err = errors.New("server configs not found in properties")
			return nil, err
		}
		_ = client.SetServerConfig(nil)
	} else {
		for i := range param.ServerConfigs {
			if param.ServerConfigs[i].Port == 0 {
				param.ServerConfigs[i].Port = 8848
			}
			if param.ServerConfigs[i].GrpcPort == 0 {
				param.ServerConfigs[i].GrpcPort = param.ServerConfigs[i].Port + constant.RpcPortOffset
			}
		}
		err = client.SetServerConfig(param.ServerConfigs)
		if err != nil {
			return nil, err
		}
	}

	if _, _err := client.GetHttpAgent(); _err != nil {
		if clientCfg, err := client.GetClientConfig(); err == nil {
			_ = client.SetHttpAgent(&http_agent.HttpAgent{TlsConfig: clientCfg.TLSCfg})
		}
	}
	iClient = client
	return
}
