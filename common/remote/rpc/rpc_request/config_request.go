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

package rpc_request

import "github.com/nacos-group/nacos-sdk-go/model"

type ConfigRequest struct {
	*Request
	Module string `json:"module"`
}

func NewConfigRequest() *ConfigRequest {
	request := Request{
		Headers: make(map[string]string, 8),
	}
	return &ConfigRequest{
		Request: &request,
		Module:  "config",
	}
}

//request of listening a batch of configs.
type ConfigBatchListenRequest struct {
	*ConfigRequest
	Listen               bool                        `json:"listen"`
	ConfigListenContexts []model.ConfigListenContext `json:"configListenContexts"`
}

func NewConfigBatchListenRequest(cacheLen int) *ConfigBatchListenRequest {
	return &ConfigBatchListenRequest{
		Listen:               true,
		ConfigListenContexts: make([]model.ConfigListenContext, 0, cacheLen),
		ConfigRequest:        NewConfigRequest(),
	}
}

func (r *ConfigBatchListenRequest) GetRequestType() string {
	return "ConfigBatchListenRequest"
}

type ConfigChangeNotifyRequest struct {
	*ConfigRequest
	Group  string `json:"group"`
	DataId string `json:"dataId"`
	Tenant string `json:"tenant"`
}

func NewConfigChangeNotifyRequest(group, dataId, tenant string) *ConfigChangeNotifyRequest {
	return &ConfigChangeNotifyRequest{ConfigRequest: NewConfigRequest(), Group: group, DataId: dataId, Tenant: tenant}
}

func (r *ConfigChangeNotifyRequest) GetRequestType() string {
	return "ConfigChangeNotifyRequest"
}

type ConfigQueryRequest struct {
	*ConfigRequest
	Group  string `json:"group"`
	DataId string `json:"dataId"`
	Tenant string `json:"tenant"`
	Tag    string `json:"tag"`
}

func NewConfigQueryRequest(group, dataId, tenant string) *ConfigQueryRequest {
	return &ConfigQueryRequest{ConfigRequest: NewConfigRequest(), Group: group, DataId: dataId, Tenant: tenant}
}

func (r *ConfigQueryRequest) GetRequestType() string {
	return "ConfigQueryRequest"
}

type ConfigPublishRequest struct {
	*ConfigRequest
	Group       string            `json:"group"`
	DataId      string            `json:"dataId"`
	Tenant      string            `json:"tenant"`
	Content     string            `json:"content"`
	CasMd5      string            `json:"casMd5"`
	AdditionMap map[string]string `json:"additionMap"`
}

func NewConfigPublishRequest(group, dataId, tenant, content, casMd5 string) *ConfigPublishRequest {
	return &ConfigPublishRequest{ConfigRequest: NewConfigRequest(),
		Group: group, DataId: dataId, Tenant: tenant, Content: content, CasMd5: casMd5, AdditionMap: make(map[string]string)}
}

func (r *ConfigPublishRequest) GetRequestType() string {
	return "ConfigPublishRequest"
}

type ConfigRemoveRequest struct {
	*ConfigRequest
	Group  string `json:"group"`
	DataId string `json:"dataId"`
	Tenant string `json:"tenant"`
}

func NewConfigRemoveRequest(group, dataId, tenant string) *ConfigRemoveRequest {
	return &ConfigRemoveRequest{ConfigRequest: NewConfigRequest(),
		Group: group, DataId: dataId, Tenant: tenant}
}

func (r *ConfigRemoveRequest) GetRequestType() string {
	return "ConfigRemoveRequest"
}
