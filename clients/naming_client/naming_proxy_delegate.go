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

package naming_client

import (
	"context"
	"github.com/nacos-group/nacos-sdk-go/v2/inner/uuid"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_cache"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_grpc"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_http"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_proxy"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

// NamingProxyDelegate ...
type NamingProxyDelegate struct {
	httpClientProxy   *naming_http.NamingHttpProxy
	grpcClientProxy   *naming_grpc.NamingGrpcProxy
	serviceInfoHolder *naming_cache.ServiceInfoHolder
}

func NewNamingProxyDelegate(ctx context.Context, clientCfg constant.ClientConfig, serverCfgs []constant.ServerConfig,
	httpAgent http_agent.IHttpAgent, serviceInfoHolder *naming_cache.ServiceInfoHolder) (naming_proxy.INamingProxy, error) {

	uid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	namingHeader := map[string][]string{
		"Client-Version": {constant.CLIENT_VERSION},
		"User-Agent":     {constant.CLIENT_VERSION},
		"RequestId":      {uid.String()},
		"Request-Module": {"Naming"},
	}
	nacosServer, err := nacos_server.NewNacosServer(ctx, serverCfgs, clientCfg, httpAgent, clientCfg.TimeoutMs, clientCfg.Endpoint, namingHeader)
	if err != nil {
		return nil, err
	}

	httpClientProxy, err := naming_http.NewNamingHttpProxy(ctx, clientCfg, nacosServer, serviceInfoHolder)
	if err != nil {
		return nil, err
	}

	grpcClientProxy, err := naming_grpc.NewNamingGrpcProxy(ctx, clientCfg, nacosServer, serviceInfoHolder)
	if err != nil {
		return nil, err
	}

	return &NamingProxyDelegate{
		httpClientProxy:   httpClientProxy,
		grpcClientProxy:   grpcClientProxy,
		serviceInfoHolder: serviceInfoHolder,
	}, nil
}

func (proxy *NamingProxyDelegate) getExecuteClientProxy(instance model.Instance) (namingProxy naming_proxy.INamingProxy) {
	if instance.Ephemeral {
		namingProxy = proxy.grpcClientProxy
	} else {
		namingProxy = proxy.httpClientProxy
	}
	return namingProxy
}

func (proxy *NamingProxyDelegate) RegisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error) {
	return proxy.getExecuteClientProxy(instance).RegisterInstance(serviceName, groupName, instance)
}

func (proxy *NamingProxyDelegate) BatchRegisterInstance(serviceName string, groupName string, instances []model.Instance) (bool, error) {
	return proxy.grpcClientProxy.BatchRegisterInstance(serviceName, groupName, instances)
}

func (proxy *NamingProxyDelegate) DeregisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error) {
	return proxy.getExecuteClientProxy(instance).DeregisterInstance(serviceName, groupName, instance)
}

func (proxy *NamingProxyDelegate) GetServiceList(pageNo uint32, pageSize uint32, groupName, namespaceId string, selector *model.ExpressionSelector) (model.ServiceList, error) {
	return proxy.grpcClientProxy.GetServiceList(pageNo, pageSize, groupName, namespaceId, selector)
}

func (proxy *NamingProxyDelegate) ServerHealthy() bool {
	return proxy.grpcClientProxy.ServerHealthy() || proxy.httpClientProxy.ServerHealthy()
}

func (proxy *NamingProxyDelegate) QueryInstancesOfService(serviceName, groupName, clusters string, udpPort int, healthyOnly bool) (*model.Service, error) {
	return proxy.grpcClientProxy.QueryInstancesOfService(serviceName, groupName, clusters, udpPort, healthyOnly)
}

func (proxy *NamingProxyDelegate) Subscribe(serviceName, groupName string, clusters string) (model.Service, error) {
	var err error
	isSubscribed := proxy.grpcClientProxy.IsSubscribed(serviceName, groupName, clusters)
	serviceNameWithGroup := util.GetServiceCacheKey(util.GetGroupName(serviceName, groupName), clusters)
	serviceInfo, ok := proxy.serviceInfoHolder.ServiceInfoMap.Load(serviceNameWithGroup)
	if !isSubscribed || !ok {
		serviceInfo, err = proxy.grpcClientProxy.Subscribe(serviceName, groupName, clusters)
		if err != nil {
			return model.Service{}, err
		}
	}

	service := serviceInfo.(model.Service)
	proxy.serviceInfoHolder.ProcessService(&service)
	return service, nil
}

func (proxy *NamingProxyDelegate) Unsubscribe(serviceName, groupName, clusters string) error {
	proxy.serviceInfoHolder.StopUpdateIfContain(util.GetGroupName(serviceName, groupName), clusters)
	return proxy.grpcClientProxy.Unsubscribe(serviceName, groupName, clusters)
}

func (proxy *NamingProxyDelegate) CloseClient() {
	proxy.grpcClientProxy.CloseClient()
}
