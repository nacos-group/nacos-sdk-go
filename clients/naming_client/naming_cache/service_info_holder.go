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

package naming_cache

import (
	"os"
	"reflect"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

type ServiceInfoHolder struct {
	ServiceInfoMap       cache.ConcurrentMap
	updateCacheWhenEmpty bool
	cacheDir             string
	notLoadCacheAtStart  bool
	subCallback          *SubscribeCallback
	UpdateTimeMap        cache.ConcurrentMap
}

func NewServiceInfoHolder(namespace, cacheDir string, updateCacheWhenEmpty, notLoadCacheAtStart bool) *ServiceInfoHolder {
	cacheDir = cacheDir + string(os.PathSeparator) + "naming" + string(os.PathSeparator) + namespace
	serviceInfoHolder := &ServiceInfoHolder{
		updateCacheWhenEmpty: updateCacheWhenEmpty,
		notLoadCacheAtStart:  notLoadCacheAtStart,
		cacheDir:             cacheDir,
		subCallback:          NewSubscribeCallback(),
		UpdateTimeMap:        cache.NewConcurrentMap(),
		ServiceInfoMap:       cache.NewConcurrentMap(),
	}

	if !notLoadCacheAtStart {
		serviceInfoHolder.loadCacheFromDisk()
	}
	return serviceInfoHolder
}

func (s *ServiceInfoHolder) loadCacheFromDisk() {
	serviceMap := cache.ReadServicesFromFile(s.cacheDir)
	if serviceMap == nil || len(serviceMap) == 0 {
		return
	}
	for k, v := range serviceMap {
		s.ServiceInfoMap.Set(k, v)
	}
}

func (s *ServiceInfoHolder) ProcessServiceJson(data string) {
	s.ProcessService(util.JsonToService(data))
}

func (s *ServiceInfoHolder) ProcessService(service *model.Service) {
	if service == nil {
		return
	}
	cacheKey := util.GetServiceCacheKey(util.GetGroupName(service.Name, service.GroupName), service.Clusters)

	oldDomain, ok := s.ServiceInfoMap.Get(cacheKey)
	if ok && !s.updateCacheWhenEmpty {
		//if instance list is empty,not to update cache
		if service.Hosts == nil || len(service.Hosts) == 0 {
			logger.Errorf("do not have useful host, ignore it, name:%s", service.Name)
			return
		}
	}
	if ok && oldDomain.(model.Service).LastRefTime >= service.LastRefTime {
		logger.Warnf("out of date data received, old-t: %d, new-t: %d", oldDomain.(model.Service).LastRefTime, service.LastRefTime)
		return
	}

	s.UpdateTimeMap.Set(cacheKey, uint64(util.CurrentMillis()))
	s.ServiceInfoMap.Set(cacheKey, *service)
	if !ok || ok && !reflect.DeepEqual(service.Hosts, oldDomain.(model.Service).Hosts) {
		if !ok {
			logger.Info("service not found in cache " + cacheKey)
		} else {
			logger.Info("service key:%s was updated to:%s", cacheKey, util.ToJsonString(service))
		}
		cache.WriteServicesToFile(*service, s.cacheDir)
		s.subCallback.ServiceChanged(service)
	}
}

func (s *ServiceInfoHolder) GetServiceInfo(serviceName, groupName, clusters string) (model.Service, bool) {
	cacheKey := util.GetServiceCacheKey(util.GetGroupName(serviceName, groupName), clusters)
	//todo FailoverReactor
	service, ok := s.ServiceInfoMap.Get(cacheKey)
	if ok {
		return service.(model.Service), ok
	}
	return model.Service{}, ok
}

func (s *ServiceInfoHolder) RegisterCallback(serviceName string, clusters string, callbackFunc *func(services []model.Instance, err error)) {
	s.subCallback.AddCallbackFunc(serviceName, clusters, callbackFunc)
}

func (s *ServiceInfoHolder) DeregisterCallback(serviceName string, clusters string, callbackFunc *func(services []model.Instance, err error)) {
	s.subCallback.RemoveCallbackFunc(serviceName, clusters, callbackFunc)
}

func (s *ServiceInfoHolder) StopUpdateIfContain(serviceName, clusters string) {
	cacheKey := util.GetServiceCacheKey(serviceName, clusters)
	s.ServiceInfoMap.Remove(cacheKey)
}

func (s *ServiceInfoHolder) IsSubscribed(serviceName, clusters string) bool {
	return s.subCallback.IsSubscribed(serviceName, clusters)
}
