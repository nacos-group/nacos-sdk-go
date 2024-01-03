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
	"context"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_proxy"
	"time"

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
	redoService       IRedoService
	serviceInfoHolder *naming_cache.ServiceInfoHolder
}

var _ naming_proxy.INamingProxy = new(NamingGrpcProxy)

// NewNamingGrpcProxy create naming grpc proxy
func NewNamingGrpcProxy(ctx context.Context, clientCfg constant.ClientConfig, nacosServer *nacos_server.NacosServer,
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

	iRpcClient, err := rpc.CreateClient(ctx, uid.String(), rpc.GRPC, labels, srvProxy.nacosServer)
	if err != nil {
		return nil, err
	}

	srvProxy.rpcClient = iRpcClient

	rpcClient := srvProxy.rpcClient.GetRpcClient()
	rpcClient.Start()

	rpcClient.RegisterServerRequestHandler(func() rpc_request.IRequest {
		return &rpc_request.NotifySubscriberRequest{NamingRequest: &rpc_request.NamingRequest{}}
	}, &rpc.NamingPushRequestHandler{ServiceInfoHolder: serviceInfoHolder})

	srvProxy.redoService = NewRedoService(ctx, &srvProxy)
	rpcClient.RegisterConnectionListener(srvProxy.redoService)

	return &srvProxy, nil
}

func (proxy *NamingGrpcProxy) requestToServer(request rpc_request.IRequest) (rpc_response.IResponse, error) {
	start := time.Now()
	proxy.nacosServer.InjectSign(request, request.GetHeaders(), proxy.clientConfig)
	proxy.nacosServer.InjectSecurityInfo(request.GetHeaders())
	response, err := proxy.rpcClient.GetRpcClient().Request(request, int64(proxy.clientConfig.TimeoutMs))
	if err == nil && response != nil && !response.IsSuccess() {
		return nil, fmt.Errorf("nacos server response type [%s] error code=%d, message=[%s]", response.GetResponseType(), response.GetErrorCode(), response.GetMessage())
	}
	monitor.GetConfigRequestMonitor(constant.GRPC, request.GetRequestType(), rpc_response.GetGrpcResponseStatusCode(response)).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	return response, err
}

// RegisterInstance ...
func (proxy *NamingGrpcProxy) RegisterInstance(serviceName string, groupName string, instance model.Instance) error {
	logger.Infof("register instance namespaceId:<%s>,serviceName:<%s>, groupName:<%s> with instance:<%s>",
		proxy.clientConfig.NamespaceId, serviceName, util.ToJsonString(instance))
	proxy.redoService.CacheInstanceForRedo(serviceName, groupName, instance)
	return proxy.DoRegisterInstance(serviceName, groupName, instance)
}

func (proxy *NamingGrpcProxy) DoRegisterInstance(serviceName string, groupName string, instance model.Instance) error {
	instanceRequest := rpc_request.NewInstanceRequest(proxy.clientConfig.NamespaceId, serviceName, groupName, "registerInstance", instance)
	_, err := proxy.requestToServer(instanceRequest)
	if err != nil {
		return err
	}

	proxy.redoService.InstanceRegistered(serviceName, groupName)
	return nil
}

// BatchRegisterInstance ...
func (proxy *NamingGrpcProxy) BatchRegisterInstance(serviceName string, groupName string, instances []model.Instance) error {
	logger.Infof("batch register instance namespaceId:<%s>,serviceName:<%s>, groupName:<%s> with instance:<%s>",
		proxy.clientConfig.NamespaceId, serviceName, groupName, util.ToJsonString(instances))
	proxy.redoService.CacheInstancesForRedo(serviceName, groupName, instances)
	return proxy.DoBatchRegisterInstance(serviceName, groupName, instances)
}

func (proxy *NamingGrpcProxy) DoBatchRegisterInstance(serviceName, groupName string, instances []model.Instance) error {
	batchInstanceRequest := rpc_request.NewBatchInstanceRequest(proxy.clientConfig.NamespaceId, serviceName, groupName, "batchRegisterInstance", instances)
	_, err := proxy.requestToServer(batchInstanceRequest)
	if err != nil {
		return err
	}
	proxy.redoService.InstanceRegistered(serviceName, groupName)
	return nil
}

func (proxy *NamingGrpcProxy) DoDeRegisterInstance(serviceName, groupName string, instance model.Instance) error {
	instanceRequest := rpc_request.NewInstanceRequest(proxy.clientConfig.NamespaceId, serviceName, groupName, "deregisterInstance", instance)
	_, err := proxy.requestToServer(instanceRequest)
	if err != nil {
		return err
	}
	proxy.redoService.InstanceDeRegistered(serviceName, groupName)
	return nil
}

// DeregisterInstance ...
func (proxy *NamingGrpcProxy) DeregisterInstance(serviceName, groupName string, instance model.Instance) error {
	logger.Infof("deregister instance namespaceId:<%s>,serviceName:<%s>, groupName:<%s> with instance:<%s:%d@%s>",
		proxy.clientConfig.NamespaceId, serviceName, instance.Ip, instance.Port, instance.ClusterName)
	proxy.redoService.InstanceDeRegister(serviceName, groupName)
	return proxy.DoDeRegisterInstance(serviceName, groupName, instance)

}

// GetServiceList ...
func (proxy *NamingGrpcProxy) GetServiceList(pageNo uint32, pageSize uint32, groupName, namespaceId string, selector *model.ExpressionSelector) (model.ServiceList, error) {
	var selectorStr string
	if selector != nil {
		switch selector.Type {
		case "label":
			selectorStr = util.ToJsonString(selector)
		default:
			break
		}
	}
	response, err := proxy.requestToServer(rpc_request.NewServiceListRequest(namespaceId, "",
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
func (proxy *NamingGrpcProxy) QueryInstancesOfService(serviceName, groupName, cluster string, udpPort int, healthyOnly bool) (*model.Service, error) {
	response, err := proxy.requestToServer(rpc_request.NewServiceQueryRequest(proxy.clientConfig.NamespaceId, serviceName, groupName, cluster,
		healthyOnly, udpPort))
	if err != nil {
		return nil, err
	}
	queryServiceResponse := response.(*rpc_response.QueryServiceResponse)
	return &queryServiceResponse.ServiceInfo, nil
}

func (proxy *NamingGrpcProxy) IsSubscribed(serviceName, groupName string, clusters string) bool {
	return proxy.redoService.IsSubscriberCached(serviceName, groupName, clusters)
}

// Subscribe ...
func (proxy *NamingGrpcProxy) Subscribe(serviceName, groupName string, clusters string) (model.Service, error) {
	logger.Infof("Subscribe Service namespaceId:<%s>, serviceName:<%s>, groupName:<%s>, clusters:<%s>",
		proxy.clientConfig.NamespaceId, serviceName, groupName, clusters)
	proxy.redoService.CacheSubscriberForRedo(serviceName, groupName, clusters)
	return proxy.DoSubscribe(serviceName, groupName, clusters)
}

func (proxy *NamingGrpcProxy) DoSubscribe(serviceName, groupName string, clusters string) (model.Service, error) {
	request := rpc_request.NewSubscribeServiceRequest(proxy.clientConfig.NamespaceId, serviceName,
		groupName, clusters, true)
	request.Headers["app"] = proxy.clientConfig.AppName
	response, err := proxy.requestToServer(request)
	if err != nil {
		return model.Service{}, err
	}

	proxy.redoService.SubscribeRegistered(serviceName, groupName, clusters)
	subscribeServiceResponse := response.(*rpc_response.SubscribeServiceResponse)
	return subscribeServiceResponse.ServiceInfo, nil
}

// Unsubscribe ...
func (proxy *NamingGrpcProxy) Unsubscribe(serviceName, groupName, clusters string) error {
	logger.Infof("Unsubscribe Service namespaceId:<%s>, serviceName:<%s>, groupName:<%s>, clusters:<%s>",
		proxy.clientConfig.NamespaceId, serviceName, groupName, clusters)
	proxy.redoService.SubscribeDeRegister(serviceName, groupName, clusters)
	return proxy.DoUnSubscribe(serviceName, groupName, clusters)
}

// DoUnSubscribe ...
func (proxy *NamingGrpcProxy) DoUnSubscribe(serviceName, groupName, clusters string) error {
	_, err := proxy.requestToServer(rpc_request.NewSubscribeServiceRequest(proxy.clientConfig.NamespaceId, serviceName, groupName,
		clusters, false))
	if err == nil {
		proxy.redoService.SubscribeDeRegistered(serviceName, groupName, clusters)
	}
	return err
}

func (proxy *NamingGrpcProxy) IsSubscribe(serviceName, groupName, clusters string) bool {
	return proxy.redoService.IsSubscriberCached(serviceName, groupName, clusters)
}

func (proxy *NamingGrpcProxy) CloseClient() {
	logger.Info("Close Nacos Go SDK Client...")
	proxy.rpcClient.GetRpcClient().Shutdown()
}
