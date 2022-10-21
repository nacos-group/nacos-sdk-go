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
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/nacos-group/nacos-sdk-go/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/logger"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/util"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

type NamingClient struct {
	nacos_client.INacosClient
	hostReactor  HostReactor
	serviceProxy NamingProxy
	subCallback  SubscribeCallback
	beatReactor  BeatReactor
	indexMap     cache.ConcurrentMap
	NamespaceId  string
}

type Chooser struct {
	data   []model.Instance
	totals []int
	max    int
}

func NewNamingClient(nc nacos_client.INacosClient) (NamingClient, error) {
	rand.Seed(time.Now().UnixNano())
	naming := NamingClient{INacosClient: nc}
	clientConfig, err := nc.GetClientConfig()
	if err != nil {
		return naming, err
	}
	naming.NamespaceId = clientConfig.NamespaceId
	serverConfig, err := nc.GetServerConfig()
	if err != nil {
		return naming, err
	}
	httpAgent, err := nc.GetHttpAgent()
	if err != nil {
		return naming, err
	}
	loggerConfig := logger.Config{
		LogFileName:      constant.LOG_FILE_NAME,
		Level:            clientConfig.LogLevel,
		Sampling:         clientConfig.LogSampling,
		LogRollingConfig: clientConfig.LogRollingConfig,
		LogDir:           clientConfig.LogDir,
		CustomLogger:     clientConfig.CustomLogger,
		LogStdout:        clientConfig.AppendToStdout,
	}
	err = logger.InitLogger(loggerConfig)
	if err != nil {
		return naming, err
	}
	logger.GetLogger().Infof("logDir:<%s>   cacheDir:<%s>", clientConfig.LogDir, clientConfig.CacheDir)
	naming.subCallback = NewSubscribeCallback()
	naming.serviceProxy, err = NewNamingProxy(clientConfig, serverConfig, httpAgent)
	if err != nil {
		return naming, err
	}
	naming.hostReactor = NewHostReactor(naming.serviceProxy, clientConfig.CacheDir+string(os.PathSeparator)+"naming",
		clientConfig.UpdateThreadNum, clientConfig.NotLoadCacheAtStart, naming.subCallback, clientConfig.UpdateCacheWhenEmpty)
	naming.beatReactor = NewBeatReactor(naming.serviceProxy, clientConfig.BeatInterval)
	naming.indexMap = cache.NewConcurrentMap()
	return naming, nil
}

// RegisterInstance register instance
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
	beatInfo := &model.BeatInfo{
		Ip:          param.Ip,
		Port:        param.Port,
		Metadata:    param.Metadata,
		ServiceName: util.GetGroupName(param.ServiceName, param.GroupName),
		Cluster:     param.ClusterName,
		Weight:      param.Weight,
		Period:      util.GetDurationWithDefault(param.Metadata, constant.HEART_BEAT_INTERVAL, time.Second*5),
		State:       model.StateRunning,
	}
	_, err := sc.serviceProxy.RegisterInstance(util.GetGroupName(param.ServiceName, param.GroupName), param.GroupName, instance)
	if err != nil {
		return false, err
	}
	if instance.Ephemeral {
		sc.beatReactor.AddBeatInfo(util.GetGroupName(param.ServiceName, param.GroupName), beatInfo)
	}
	return true, nil

}

// DeregisterInstance deregister instance
func (sc *NamingClient) DeregisterInstance(param vo.DeregisterInstanceParam) (bool, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	sc.beatReactor.RemoveBeatInfo(util.GetGroupName(param.ServiceName, param.GroupName), param.Ip, param.Port)

	_, err := sc.serviceProxy.DeregisterInstance(util.GetGroupName(param.ServiceName, param.GroupName), param.Ip, param.Port, param.Cluster, param.Ephemeral)
	if err != nil {
		return false, err
	}
	return true, nil
}

// UpdateInstance update information for exist instance.
func (sc *NamingClient) UpdateInstance(param vo.UpdateInstanceParam) (bool, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}

	if param.Ephemeral {
		// Update the heartbeat information first to prevent the information
		// from being flushed back to the original information after reconnecting
		sc.beatReactor.RemoveBeatInfo(util.GetGroupName(param.ServiceName, param.GroupName), param.Ip, param.Port)
		beatInfo := &model.BeatInfo{
			Ip:          param.Ip,
			Port:        param.Port,
			Metadata:    param.Metadata,
			ServiceName: util.GetGroupName(param.ServiceName, param.GroupName),
			Cluster:     param.ClusterName,
			Weight:      param.Weight,
			Period:      util.GetDurationWithDefault(param.Metadata, constant.HEART_BEAT_INTERVAL, time.Second*5),
			State:       model.StateRunning,
		}
		sc.beatReactor.AddBeatInfo(util.GetGroupName(param.ServiceName, param.GroupName), beatInfo)
	}

	// Do update instance
	_, err := sc.serviceProxy.UpdateInstance(
		util.GetGroupName(param.ServiceName, param.GroupName), param.Ip, param.Port, param.ClusterName, param.Ephemeral,
		param.Weight, param.Enable, param.Metadata)

	if err != nil {
		return false, err
	}
	return true, nil
}

// GetService get service info
func (sc *NamingClient) GetService(param vo.GetServiceParam) (model.Service, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	service, err := sc.hostReactor.GetServiceInfo(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	return service, err
}

// GetAllServicesInfo get all services info
func (sc *NamingClient) GetAllServicesInfo(param vo.GetAllServiceInfoParam) (model.ServiceList, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	if len(param.NameSpace) == 0 {
		if len(sc.NamespaceId) == 0 {
			param.NameSpace = constant.DEFAULT_NAMESPACE_ID
		} else {
			param.NameSpace = sc.NamespaceId
		}
	}
	if param.PageNo == 0 {
		param.PageNo = 1
	}
	if param.PageSize == 0 {
		param.PageSize = 10
	}
	return sc.hostReactor.GetAllServiceInfo(param.NameSpace, param.GroupName, param.PageNo, param.PageSize)
}

// SelectAllInstances select all instances
func (sc *NamingClient) SelectAllInstances(param vo.SelectAllInstancesParam) ([]model.Instance, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	service, err := sc.hostReactor.GetServiceInfo(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	if err != nil || service.Hosts == nil || len(service.Hosts) == 0 {
		return []model.Instance{}, err
	}
	return service.Hosts, err
}

// SelectInstances select instances
func (sc *NamingClient) SelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	service, err := sc.hostReactor.GetServiceInfo(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	if err != nil {
		return nil, err
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

// SelectOneHealthyInstance select one healthy instance
func (sc *NamingClient) SelectOneHealthyInstance(param vo.SelectOneHealthInstanceParam) (*model.Instance, error) {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	service, err := sc.hostReactor.GetServiceInfo(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	if err != nil {
		return nil, err
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

type instances []model.Instance

func (a instances) Len() int {
	return len(a)
}

func (a instances) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a instances) Less(i, j int) bool {
	return a[i].Weight < a[j].Weight
}

// NewChooser initializes a new Chooser for picking from the provided Choices.
func newChooser(instances instances) Chooser {
	sort.Sort(instances)
	totals := make([]int, len(instances))
	runningTotal := 0
	for i, c := range instances {
		runningTotal += int(c.Weight)
		totals[i] = runningTotal
	}
	return Chooser{data: instances, totals: totals, max: runningTotal}
}

func (chs Chooser) pick() model.Instance {
	r := rand.Intn(chs.max) + 1
	i := sort.SearchInts(chs.totals, r)
	return chs.data[i]
}

// Subscribe subscribe service
func (sc *NamingClient) Subscribe(param *vo.SubscribeParam) error {
	if len(param.GroupName) == 0 {
		param.GroupName = constant.DEFAULT_GROUP
	}
	serviceParam := vo.GetServiceParam{
		ServiceName: param.ServiceName,
		GroupName:   param.GroupName,
		Clusters:    param.Clusters,
	}

	sc.subCallback.AddCallbackFuncs(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), &param.SubscribeCallback)
	svc, err := sc.GetService(serviceParam)
	if err != nil {
		return err
	}
	if sc.hostReactor.serviceProxy.clientConfig.NotLoadCacheAtStart {
		sc.subCallback.ServiceChanged(&svc)
	}
	return nil
}

// Unsubscribe unsubscribe service
func (sc *NamingClient) Unsubscribe(param *vo.SubscribeParam) error {
	sc.subCallback.RemoveCallbackFuncs(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), &param.SubscribeCallback)
	return nil
}
