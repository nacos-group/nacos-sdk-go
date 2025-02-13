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
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
)

type RequestResource struct {
	requestType string
	namespace   string
	group       string
	resource    string
}

func BuildConfigResourceByRequest(request rpc_request.IRequest) RequestResource {
	if request.GetRequestType() == "ConfigQueryRequest" {
		configQueryRequest := request.(*rpc_request.ConfigQueryRequest)
		return RequestResource{
			requestType: "config",
			namespace:   configQueryRequest.Tenant,
			group:       configQueryRequest.Group,
			resource:    configQueryRequest.DataId,
		}
	}
	if request.GetRequestType() == "ConfigPublishRequest" {
		configPublishRequest := request.(*rpc_request.ConfigPublishRequest)
		return RequestResource{
			requestType: "config",
			namespace:   configPublishRequest.Tenant,
			group:       configPublishRequest.Group,
			resource:    configPublishRequest.DataId,
		}
	}
	if request.GetRequestType() == "ConfigRemoveRequest" {
		configRemoveRequest := request.(*rpc_request.ConfigRemoveRequest)
		return RequestResource{
			requestType: "config",
			namespace:   configRemoveRequest.Tenant,
			group:       configRemoveRequest.Group,
			resource:    configRemoveRequest.DataId,
		}
	}
	return RequestResource{
		requestType: "config",
	}
}

func BuildNamingResourceByRequest(request rpc_request.IRequest) RequestResource {
	if request.GetRequestType() == "InstanceRequest" {
		instanceRequest := request.(*rpc_request.InstanceRequest)
		return RequestResource{
			requestType: "naming",
			namespace:   instanceRequest.Namespace,
			group:       instanceRequest.GroupName,
			resource:    instanceRequest.ServiceName,
		}
	}
	if request.GetRequestType() == "BatchInstanceRequest" {
		batchInstanceRequest := request.(*rpc_request.BatchInstanceRequest)
		return RequestResource{
			requestType: "naming",
			namespace:   batchInstanceRequest.Namespace,
			group:       batchInstanceRequest.GroupName,
			resource:    batchInstanceRequest.ServiceName,
		}
	}
	if request.GetRequestType() == "ServiceListRequest" {
		serviceListRequest := request.(*rpc_request.ServiceListRequest)
		return RequestResource{
			requestType: "naming",
			namespace:   serviceListRequest.Namespace,
			group:       serviceListRequest.GroupName,
			resource:    serviceListRequest.ServiceName,
		}
	}
	if request.GetRequestType() == "ServiceQueryRequest" {
		serviceQueryRequest := request.(*rpc_request.ServiceQueryRequest)
		return RequestResource{
			requestType: "naming",
			namespace:   serviceQueryRequest.Namespace,
			group:       serviceQueryRequest.GroupName,
			resource:    serviceQueryRequest.ServiceName,
		}
	}
	if request.GetRequestType() == "SubscribeServiceRequest" {
		subscribeServiceRequest := request.(*rpc_request.SubscribeServiceRequest)
		return RequestResource{
			requestType: "naming",
			namespace:   subscribeServiceRequest.Namespace,
			group:       subscribeServiceRequest.GroupName,
			resource:    subscribeServiceRequest.ServiceName,
		}
	}
	return RequestResource{
		requestType: "naming",
	}
}

func BuildConfigResource(tenant, group, dataId string) RequestResource {
	return RequestResource{
		requestType: "config",
		namespace:   tenant,
		group:       group,
		resource:    dataId,
	}
}

func BuildNamingResource(namespace, group, serviceName string) RequestResource {
	return RequestResource{
		requestType: "naming",
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
	var securityInfo = make(map[string]string)
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
		var timer *time.Timer
		timer = time.NewTimer(time.Second * time.Duration(5))
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
	securityProxy.Clients = make([]AuthClient, 0)
	securityProxy.Clients = append(securityProxy.Clients, NewNacosAuthClient(clientCfg, serverCfgs, agent))
	securityProxy.Clients = append(securityProxy.Clients, NewRamAuthClient(clientCfg))
	return securityProxy
}
