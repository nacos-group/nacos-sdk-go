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

package naming_grpc

import (
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/common/monitor"
	"github.com/nacos-group/nacos-sdk-go/v2/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v2/inner/uuid"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

// NamingGrpcProxy ...
type NamingGrpcProxy struct {
	clientConfig      constant.ClientConfig
	nacosServer       *nacos_server.NacosServer
	rpcClient         rpc.IRpcClient
	eventListener     *ConnectionEventListener
	serviceInfoHolder *naming_cache.ServiceInfoHolder
}

// NewNamingGrpcProxy create naming grpc proxy
func NewNamingGrpcProxy(clientCfg constant.ClientConfig, nacosServer *nacos_server.NacosServer,
	serviceInfoHolder *naming_cache.ServiceInfoHolder) (*NamingGrpcProxy, error) {
	srvProxy := NamingGrpcProxy{
		clientConfig:      clientCfg,
		nacosServer:       nacosServer,
		serviceInfoHolder: serviceInfoHolder,
	}

	uid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	labels := map[string]string{
		constant.LABEL_SOURCE: constant.LABEL_SOURCE_SDK,
		constant.LABEL_MODULE: constant.LABEL_MODULE_NAMING,
	}

	iRpcClient, err := rpc.CreateClient(uid.String(), rpc.GRPC, labels, srvProxy.nacosServer)
	if err != nil {
		return nil, err
	}

	srvProxy.rpcClient = iRpcClient

	rpcClient := srvProxy.rpcClient.GetRpcClient()
	rpcClient.Start()

	rpcClient.RegisterServerRequestHandler(func() rpc_request.IRequest {
		return &rpc_request.NotifySubscriberRequest{NamingRequest: &rpc_request.NamingRequest{}}
	}, &rpc.NamingPushRequestHandler{ServiceInfoHolder: serviceInfoHolder})

	srvProxy.eventListener = NewConnectionEventListener(&srvProxy)
	rpcClient.RegisterConnectionListener(srvProxy.eventListener)

	return &srvProxy, nil
}

func (proxy *NamingGrpcProxy) requestToServer(request rpc_request.IRequest) (rpc_response.IResponse, error) {
	proxy.nacosServer.InjectSecurityInfo(request.GetHeaders())
	// todo ak/sk
	response, err := proxy.rpcClient.GetRpcClient().Request(request, int64(proxy.clientConfig.TimeoutMs))
	if response != nil {
		monitor.GetConfigRequestMonitor(monitor.GRPC, request.GetRequestType(), strconv.Itoa(response.GetResultCode()))
	} else {
		monitor.GetConfigRequestMonitor(monitor.GRPC, request.GetRequestType(), "NA")
	}
	return response, err
}

// RegisterInstance ...
func (proxy *NamingGrpcProxy) RegisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error) {
	logger.Infof("instance namespaceId:<%s>,serviceName:<%s> with instance:<%s>",
		proxy.clientConfig.NamespaceId, serviceName, util.ToJsonString(instance))
	instanceRequest := rpc_request.NewInstanceRequest(proxy.clientConfig.NamespaceId, serviceName, groupName, "registerInstance", instance)
	response, err := proxy.requestToServer(instanceRequest)
	proxy.eventListener.CacheInstanceForRedo(serviceName, groupName, instance)
	if err != nil {
		return false, err
	}
	return response.IsSuccess(), err
}

// DeregisterInstance ...
func (proxy *NamingGrpcProxy) DeregisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error) {
	logger.Infof("deregister instance namespaceId:<%s>,serviceName:<%s> with instance:<%s:%d@%s>",
		proxy.clientConfig.NamespaceId, serviceName, instance.Ip, instance.Port, instance.ClusterName)
	instanceRequest := rpc_request.NewInstanceRequest(proxy.clientConfig.NamespaceId, serviceName, groupName, "deregisterInstance", instance)
	response, err := proxy.requestToServer(instanceRequest)
	proxy.eventListener.RemoveInstanceForRedo(serviceName, groupName, instance)
	if err != nil {
		return false, err
	}
	return response.IsSuccess(), err
}

// GetServiceList ...
func (proxy *NamingGrpcProxy) GetServiceList(pageNo uint32, pageSize uint32, groupName string, selector *model.ExpressionSelector) (model.ServiceList, error) {
	var selectorStr string
	if selector != nil {
		switch selector.Type {
		case "label":
			selectorStr = util.ToJsonString(selector)
		default:
			break
		}
	}
	response, err := proxy.requestToServer(rpc_request.NewServiceListRequest(proxy.clientConfig.NamespaceId, "",
		groupName, int(pageNo), int(pageSize), selectorStr))
	if err != nil {
		return model.ServiceList{}, err
	}
	serviceListResponse := response.(*rpc_response.ServiceListResponse)
	return model.ServiceList{
		Count: int64(serviceListResponse.Count),
		Doms:  serviceListResponse.ServiceNames,
	}, nil
}

// ServerHealthy ...
func (proxy *NamingGrpcProxy) ServerHealthy() bool {
	return proxy.rpcClient.GetRpcClient().IsRunning()
}

// QueryInstancesOfService ...
func (proxy *NamingGrpcProxy) QueryInstancesOfService(serviceName, groupName, clusters string, udpPort int, healthyOnly bool) (*model.Service, error) {
	response, err := proxy.requestToServer(rpc_request.NewServiceQueryRequest(proxy.clientConfig.NamespaceId, serviceName, groupName, clusters,
		healthyOnly, udpPort))
	if err != nil {
		return nil, err
	}
	queryServiceResponse := response.(*rpc_response.QueryServiceResponse)
	return &queryServiceResponse.ServiceInfo, nil
}

// Subscribe ...
func (proxy *NamingGrpcProxy) Subscribe(serviceName, groupName string, clusters string) (model.Service, error) {
	proxy.eventListener.CacheSubscriberForRedo(util.GetGroupName(serviceName, groupName), clusters)
	request := rpc_request.NewSubscribeServiceRequest(proxy.clientConfig.NamespaceId, serviceName,
		groupName, clusters, true)
	request.Headers["app"] = proxy.clientConfig.AppName
	response, err := proxy.requestToServer(request)
	if err != nil {
		return model.Service{}, err
	}
	subscribeServiceResponse := response.(*rpc_response.SubscribeServiceResponse)
	return subscribeServiceResponse.ServiceInfo, nil
}

// Unsubscribe ...
func (proxy *NamingGrpcProxy) Unsubscribe(serviceName, groupName, clusters string) error {
	proxy.eventListener.RemoveSubscriberForRedo(util.GetGroupName(serviceName, groupName), clusters)
	_, err := proxy.requestToServer(rpc_request.NewSubscribeServiceRequest(proxy.clientConfig.NamespaceId, serviceName, groupName,
		clusters, false))
	return err
}

func (proxy *NamingGrpcProxy) CloseClient() {
	proxy.rpcClient.GetRpcClient().Shutdown()
}
