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

import "github.com/nacos-group/nacos-sdk-go/v2/model"

type ConfigRequest struct {
	*Request
	Group  string `json:"group"`
	DataId string `json:"dataId"`
	Tenant string `json:"tenant"`
	Module string `json:"module"`
}

func NewConfigRequest(group, dataId, tenant string) *ConfigRequest {
	request := Request{
		Headers: make(map[string]string, 8),
	}
	return &ConfigRequest{
		Request: &request,
		Group:   group,
		DataId:  dataId,
		Tenant:  tenant,
		Module:  "config",
	}
}

func (r *ConfigRequest) GetDataId() string {
	return r.DataId
}

func (r *ConfigRequest) GetGroup() string {
	return r.Group
}

func (r *ConfigRequest) GetTenant() string {
	return r.Tenant
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
		ConfigRequest:        NewConfigRequest("", "", ""),
	}
}

func (r *ConfigBatchListenRequest) GetRequestType() string {
	return "ConfigBatchListenRequest"
}

type ConfigChangeNotifyRequest struct {
	*ConfigRequest
}

func NewConfigChangeNotifyRequest(group, dataId, tenant string) *ConfigChangeNotifyRequest {
	return &ConfigChangeNotifyRequest{ConfigRequest: NewConfigRequest(group, dataId, tenant)}
}

func (r *ConfigChangeNotifyRequest) GetRequestType() string {
	return "ConfigChangeNotifyRequest"
}

type ConfigQueryRequest struct {
	*ConfigRequest
	Tag string `json:"tag"`
}

func NewConfigQueryRequest(group, dataId, tenant string) *ConfigQueryRequest {
	return &ConfigQueryRequest{ConfigRequest: NewConfigRequest(group, dataId, tenant)}
}

func (r *ConfigQueryRequest) GetRequestType() string {
	return "ConfigQueryRequest"
}

type ConfigPublishRequest struct {
	*ConfigRequest
	Content     string            `json:"content"`
	CasMd5      string            `json:"casMd5"`
	AdditionMap map[string]string `json:"additionMap"`
}

func NewConfigPublishRequest(group, dataId, tenant, content, casMd5 string) *ConfigPublishRequest {
	return &ConfigPublishRequest{ConfigRequest: NewConfigRequest(group, dataId, tenant),
		Content: content, CasMd5: casMd5, AdditionMap: make(map[string]string)}
}

func (r *ConfigPublishRequest) GetRequestType() string {
	return "ConfigPublishRequest"
}

type ConfigRemoveRequest struct {
	*ConfigRequest
}

func NewConfigRemoveRequest(group, dataId, tenant string) *ConfigRemoveRequest {
	return &ConfigRemoveRequest{ConfigRequest: NewConfigRequest(group, dataId, tenant)}
}

func (r *ConfigRemoveRequest) GetRequestType() string {
	return "ConfigRemoveRequest"
}
