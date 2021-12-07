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
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_proxy"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/pkg/errors"
)

type NamingClient struct {
	nacos_client.INacosClient
	serviceProxy      naming_proxy.INamingProxy
	serviceInfoHolder *naming_cache.ServiceInfoHolder
}

func NewNamingClient(nc nacos_client.INacosClient) (*NamingClient, error) {
	rand.Seed(time.Now().UnixNano())
	naming := &NamingClient{INacosClient: nc}
	clientConfig, err := nc.GetClientConfig()
	if err != nil {
		return naming, err
	}

	serverConfig, err := nc.GetServerConfig()
	if err != nil {
		return naming, err
	}

	httpAgent, err := nc.GetHttpAgent()
	if err != nil {
		return naming, err
	}

	if err = initLogger(clientConfig); err != nil {
		return naming, err
	}
	if clientConfig.NamespaceId == "" {
		clientConfig.NamespaceId = constant.DEFAULT_NAMESPACE_ID
	}
	naming.serviceInfoHolder = naming_cache.NewServiceInfoHolder(clientConfig.NamespaceId, clientConfig.CacheDir,
		clientConfig.UpdateCacheWhenEmpty, clientConfig.NotLoadCacheAtStart)

	naming.serviceProxy, err = NewNamingProxyDelegate(clientConfig, serverConfig, httpAgent, naming.serviceInfoHolder)

	go NewServiceInfoUpdater(naming.serviceInfoHolder, clientConfig.UpdateThreadNum, naming.serviceProxy).asyncUpdateService()
	if err != nil {
		return naming, err
	}

	return naming, nil
}

func initLogger(clientConfig constant.ClientConfig) error {
	return logger.InitLogger(logger.Config{
		Level:        clientConfig.LogLevel,
		OutputPath:   clientConfig.LogDir,
		RotationTime: clientConfig.RotateTime,
		MaxAge:       clientConfig.MaxAge,
	})
}

// RegisterInstance 注册服务实例
func (sc *NamingClient) RegisterInstance(param vo.RegisterInstanceParam) (bool, error) {
	if param.ServiceName == "" {
		return false, errors.New("serviceName cannot be empty!")
	}
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	if param.Metadata == nil {
		param.Metadata = make(map[string]string)
	}
	instance := model.Instance{
		Ip:          param.Ip,
		Port:        param.Port,
		Metadata:    param.Metadata,
		ClusterName: param.ClusterName,
		Healthy:     param.Healthy,
		Enable:      param.Enable,
		Weight:      param.Weight,
		Ephemeral:   param.Ephemeral,
	}

	return sc.serviceProxy.RegisterInstance(param.ServiceName, param.GroupName, instance)

}

// DeregisterInstance 注销服务实例
func (sc *NamingClient) DeregisterInstance(param vo.DeregisterInstanceParam) (bool, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	instance := model.Instance{
		Ip:          param.Ip,
		Port:        param.Port,
		ClusterName: param.Cluster,
		Ephemeral:   param.Ephemeral,
	}
	return sc.serviceProxy.DeregisterInstance(param.ServiceName, param.GroupName, instance)
}

// GetService 获取服务列表
func (sc *NamingClient) GetService(param vo.GetServiceParam) (service model.Service, err error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	var ok bool
	clusters := strings.Join(param.Clusters, ",")
	service, ok = sc.serviceInfoHolder.GetServiceInfo(param.ServiceName, param.GroupName, clusters)
	if !ok {
		service, err = sc.serviceProxy.Subscribe(param.ServiceName, param.GroupName, clusters)
	}
	return service, err
}

func (sc *NamingClient) GetAllServicesInfo(param vo.GetAllServiceInfoParam) (model.ServiceList, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	clientConfig, _ := sc.GetClientConfig()
	if len(param.NameSpace) == 0 {
		if len(clientConfig.NamespaceId) == 0 {
			param.NameSpace = constant.DEFAULT_NAMESPACE_ID
		} else {
			param.NameSpace = clientConfig.NamespaceId
		}
	}
	services, err := sc.serviceProxy.GetServiceList(param.PageNo, param.PageSize, param.GroupName, &model.ExpressionSelector{})
	return services, err
}

func (sc *NamingClient) SelectAllInstances(param vo.SelectAllInstancesParam) ([]model.Instance, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	clusters := strings.Join(param.Clusters, ",")
	var (
		service model.Service
		ok      bool
		err     error
	)

	service, ok = sc.serviceInfoHolder.GetServiceInfo(param.ServiceName, param.GroupName, clusters)
	if !ok {
		service, err = sc.serviceProxy.Subscribe(param.ServiceName, param.GroupName, clusters)
	}
	if err != nil || service.Hosts == nil || len(service.Hosts) == 0 {
		return []model.Instance{}, err
	}
	return service.Hosts, err
}

func (sc *NamingClient) SelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	var (
		service model.Service
		ok      bool
		err     error
	)
	clusters := strings.Join(param.Clusters, ",")
	service, ok = sc.serviceInfoHolder.GetServiceInfo(param.ServiceName, param.GroupName, clusters)
	if !ok {
		service, err = sc.serviceProxy.Subscribe(param.ServiceName, param.GroupName, clusters)
		if err != nil {
			return nil, err
		}
	}
	return sc.selectInstances(service, param.HealthyOnly)
}

func (sc *NamingClient) selectInstances(service model.Service, healthy bool) ([]model.Instance, error) {
	if service.Hosts == nil || len(service.Hosts) == 0 {
		return []model.Instance{}, errors.New("instance list is empty!")
	}
	hosts := service.Hosts
	var result []model.Instance
	for _, host := range hosts {
		if host.Healthy == healthy && host.Enable && host.Weight > 0 {
			result = append(result, host)
		}
	}
	return result, nil
}

func (sc *NamingClient) SelectOneHealthyInstance(param vo.SelectOneHealthInstanceParam) (*model.Instance, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	var (
		service model.Service
		ok      bool
		err     error
	)
	clusters := strings.Join(param.Clusters, ",")
	service, ok = sc.serviceInfoHolder.GetServiceInfo(param.ServiceName, param.GroupName, clusters)
	if !ok {
		service, err = sc.serviceProxy.Subscribe(param.ServiceName, param.GroupName, clusters)
		if err != nil {
			return nil, err
		}
	}

	return sc.selectOneHealthyInstances(service)
}

func (sc *NamingClient) selectOneHealthyInstances(service model.Service) (*model.Instance, error) {
	if service.Hosts == nil || len(service.Hosts) == 0 {
		return nil, errors.New("instance list is empty!")
	}
	hosts := service.Hosts
	var result []model.Instance
	mw := 0
	for _, host := range hosts {
		if host.Healthy && host.Enable && host.Weight > 0 {
			cw := int(math.Ceil(host.Weight))
			if cw > mw {
				mw = cw
			}
			result = append(result, host)
		}
	}
	if len(result) == 0 {
		return nil, errors.New("healthy instance list is empty!")
	}

	instance := newChooser(result).pick()
	return &instance, nil
}

// Subscribe 服务监听
func (sc *NamingClient) Subscribe(param *vo.SubscribeParam) error {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	clusters := strings.Join(param.Clusters, ",")
	sc.serviceInfoHolder.RegisterCallback(util.GetGroupName(param.ServiceName, param.GroupName), clusters, &param.SubscribeCallback)
	_, _ = sc.serviceProxy.Subscribe(param.ServiceName, param.GroupName, clusters)
	return nil
}

// Unsubscribe 取消服务监听
func (sc *NamingClient) Unsubscribe(param *vo.SubscribeParam) error {
	clusters := strings.Join(param.Clusters, ",")
	serviceFullName := util.GetGroupName(param.ServiceName, param.GroupName)
	sc.serviceInfoHolder.DeregisterCallback(serviceFullName, clusters, &param.SubscribeCallback)
	if sc.serviceInfoHolder.IsSubscribed(serviceFullName, clusters) {
		sc.serviceProxy.Unsubscribe(param.ServiceName, param.GroupName, clusters)
	}

	return nil
}
