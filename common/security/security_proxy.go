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

package security

import (
	"context"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
)

type RequestResource struct {
	requestType string
	namespace   string
	group       string
	resource    string
}

const (
	REQUEST_TYPE_CONFIG = "config"
	REQUEST_TYPE_NAMING = "naming"
)

func BuildConfigResourceByRequest(request rpc_request.IRequest) RequestResource {
	if request.GetRequestType() == constant.CONFIG_QUERY_REQUEST_NAME {
		configQueryRequest := request.(*rpc_request.ConfigQueryRequest)
		return BuildConfigResource(configQueryRequest.Tenant, configQueryRequest.Group, configQueryRequest.DataId)
	}
	if request.GetRequestType() == constant.CONFIG_PUBLISH_REQUEST_NAME {
		configPublishRequest := request.(*rpc_request.ConfigPublishRequest)
		return BuildConfigResource(configPublishRequest.Tenant, configPublishRequest.Group, configPublishRequest.DataId)
	}
	if request.GetRequestType() == "ConfigRemoveRequest" {
		configRemoveRequest := request.(*rpc_request.ConfigRemoveRequest)
		return BuildConfigResource(configRemoveRequest.Tenant, configRemoveRequest.Group, configRemoveRequest.DataId)
	}
	return RequestResource{
		requestType: REQUEST_TYPE_CONFIG,
	}
}

func BuildNamingResourceByRequest(request rpc_request.IRequest) RequestResource {
	if request.GetRequestType() == constant.INSTANCE_REQUEST_NAME {
		instanceRequest := request.(*rpc_request.InstanceRequest)
		return BuildNamingResource(instanceRequest.Namespace, instanceRequest.GroupName, instanceRequest.ServiceName)
	}
	if request.GetRequestType() == constant.BATCH_INSTANCE_REQUEST_NAME {
		batchInstanceRequest := request.(*rpc_request.BatchInstanceRequest)
		return BuildNamingResource(batchInstanceRequest.Namespace, batchInstanceRequest.GroupName, batchInstanceRequest.ServiceName)
	}
	if request.GetRequestType() == constant.SERVICE_LIST_REQUEST_NAME {
		serviceListRequest := request.(*rpc_request.ServiceListRequest)
		return BuildNamingResource(serviceListRequest.Namespace, serviceListRequest.GroupName, serviceListRequest.ServiceName)
	}
	if request.GetRequestType() == constant.SERVICE_QUERY_REQUEST_NAME {
		serviceQueryRequest := request.(*rpc_request.ServiceQueryRequest)
		return BuildNamingResource(serviceQueryRequest.Namespace, serviceQueryRequest.GroupName, serviceQueryRequest.ServiceName)
	}
	if request.GetRequestType() == constant.SUBSCRIBE_SERVICE_REQUEST_NAME {
		subscribeServiceRequest := request.(*rpc_request.SubscribeServiceRequest)
		return BuildNamingResource(subscribeServiceRequest.Namespace, subscribeServiceRequest.GroupName, subscribeServiceRequest.ServiceName)
	}
	return RequestResource{
		requestType: REQUEST_TYPE_NAMING,
	}
}

func BuildConfigResource(tenant, group, dataId string) RequestResource {
	return RequestResource{
		requestType: REQUEST_TYPE_CONFIG,
		namespace:   tenant,
		group:       group,
		resource:    dataId,
	}
}

func BuildNamingResource(namespace, group, serviceName string) RequestResource {
	return RequestResource{
		requestType: REQUEST_TYPE_NAMING,
		namespace:   namespace,
		group:       group,
		resource:    serviceName,
	}
}

type AuthClient interface {
	Login() (bool, error)
	GetSecurityInfo(resource RequestResource) map[string]string
	UpdateServerList(serverList []constant.ServerConfig)
}

type SecurityProxy struct {
	Clients []AuthClient
}

func (sp *SecurityProxy) Login() {
	for _, client := range sp.Clients {
		_, err := client.Login()
		if err != nil {
			logger.Errorf("login in err:%v", err)
		}
	}
}

func (sp *SecurityProxy) GetSecurityInfo(resource RequestResource) map[string]string {
	var securityInfo = make(map[string]string, 4)
	for _, client := range sp.Clients {
		info := client.GetSecurityInfo(resource)
		if info != nil {
			for k, v := range info {
				securityInfo[k] = v
			}
		}
	}
	return securityInfo
}

func (sp *SecurityProxy) UpdateServerList(serverList []constant.ServerConfig) {
	for _, client := range sp.Clients {
		client.UpdateServerList(serverList)
	}
}

func (sp *SecurityProxy) AutoRefresh(ctx context.Context) {
	go func() {
		var timer = time.NewTimer(time.Second * time.Duration(5))
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				sp.Login()
				timer.Reset(time.Second * time.Duration(5))
			case <-ctx.Done():
				return
			}
		}
	}()
}

func NewSecurityProxy(clientCfg constant.ClientConfig, serverCfgs []constant.ServerConfig, agent http_agent.IHttpAgent) SecurityProxy {
	var securityProxy = SecurityProxy{}
	securityProxy.Clients = make([]AuthClient, 2)
	securityProxy.Clients = append(securityProxy.Clients, NewNacosAuthClient(clientCfg, serverCfgs, agent))
	securityProxy.Clients = append(securityProxy.Clients, NewRamAuthClient(clientCfg))
	return securityProxy
}
