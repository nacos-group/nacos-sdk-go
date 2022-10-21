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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/common/logger"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/util"
)

type HostReactor struct {
	serviceInfoMap       cache.ConcurrentMap
	cacheDir             string
	updateThreadNum      int
	serviceProxy         NamingProxy
	pushReceiver         PushReceiver
	subCallback          SubscribeCallback
	updateTimeMap        cache.ConcurrentMap
	updateCacheWhenEmpty bool
}

const Default_Update_Thread_Num = 20

func NewHostReactor(serviceProxy NamingProxy, cacheDir string, updateThreadNum int, notLoadCacheAtStart bool, subCallback SubscribeCallback, updateCacheWhenEmpty bool) HostReactor {
	if updateThreadNum <= 0 {
		updateThreadNum = Default_Update_Thread_Num
	}
	hr := HostReactor{
		serviceProxy:         serviceProxy,
		cacheDir:             cacheDir,
		updateThreadNum:      updateThreadNum,
		serviceInfoMap:       cache.NewConcurrentMap(),
		subCallback:          subCallback,
		updateTimeMap:        cache.NewConcurrentMap(),
		updateCacheWhenEmpty: updateCacheWhenEmpty,
	}
	pr := NewPushReceiver(&hr)
	hr.pushReceiver = *pr
	if !notLoadCacheAtStart {
		hr.loadCacheFromDisk()
	}
	go hr.asyncUpdateService()
	return hr
}

func (hr *HostReactor) loadCacheFromDisk() {
	serviceMap := cache.ReadServicesFromFile(hr.cacheDir)
	if len(serviceMap) == 0 {
		return
	}
	for k, v := range serviceMap {
		hr.serviceInfoMap.Set(k, v)
	}
}

func (hr *HostReactor) ProcessServiceJson(result string) {
	service := util.JsonToService(result)
	if service == nil {
		return
	}
	cacheKey := util.GetServiceCacheKey(service.Name, service.Clusters)

	oldDomain, ok := hr.serviceInfoMap.Get(cacheKey)
	if ok && !hr.updateCacheWhenEmpty {
		//if instance list is empty,not to update cache
		if service.Hosts == nil || len(service.Hosts) == 0 {
			logger.Errorf("do not have useful host, ignore it, name:%s", service.Name)
			return
		}
	}
	hr.updateTimeMap.Set(cacheKey, uint64(util.CurrentMillis()))
	hr.serviceInfoMap.Set(cacheKey, *service)
	oldService, serviceOk := oldDomain.(model.Service)
	if !ok || ok && serviceOk && isServiceInstanceChanged(&oldService, service) {
		if !ok {
			logger.Info("service not found in cache " + cacheKey)
		} else {
			logger.Info("service key:%s was updated to:%s", cacheKey, util.ToJsonString(service))
		}
		cache.WriteServicesToFile(*service, hr.cacheDir)
		hr.subCallback.ServiceChanged(service)
	}
}

func (hr *HostReactor) GetServiceInfo(serviceName string, clusters string) (model.Service, error) {
	key := util.GetServiceCacheKey(serviceName, clusters)
	cacheService, ok := hr.serviceInfoMap.Get(key)
	if !ok {
		hr.updateServiceNow(serviceName, clusters)
		if cacheService, ok = hr.serviceInfoMap.Get(key); !ok {
			return model.Service{}, errors.New("get service info failed")
		}
	}

	return cacheService.(model.Service), nil
}

func (hr *HostReactor) GetAllServiceInfo(nameSpace, groupName string, pageNo, pageSize uint32) (model.ServiceList, error) {
	data := model.ServiceList{}
	result, err := hr.serviceProxy.GetAllServiceInfoList(nameSpace, groupName, pageNo, pageSize)
	if err != nil {
		logger.Errorf("GetAllServiceInfoList return error!nameSpace:%s groupName:%s pageNo:%d, pageSize:%d err:%+v",
			nameSpace, groupName, pageNo, pageSize, err)
		return data, err
	}
	if result == "" {
		logger.Warnf("GetAllServiceInfoList result is empty!nameSpace:%s  groupName:%s pageNo:%d, pageSize:%d",
			nameSpace, groupName, pageNo, pageSize)
		return data, nil
	}

	err = json.Unmarshal([]byte(result), &data)
	if err != nil {
		logger.Errorf("GetAllServiceInfoList result json.Unmarshal error!nameSpace:%s groupName:%s pageNo:%d, pageSize:%d",
			nameSpace, groupName, pageNo, pageSize)
		return data, err
	}
	return data, nil
}

func (hr *HostReactor) updateServiceNow(serviceName, clusters string) {
	result, err := hr.serviceProxy.QueryList(serviceName, clusters, hr.pushReceiver.port, false)

	if err != nil {
		logger.Errorf("QueryList return error!serviceName:%s cluster:%s err:%+v", serviceName, clusters, err)
		return
	}
	if result == "" {
		logger.Errorf("QueryList result is empty!serviceName:%s cluster:%s", serviceName, clusters)
		return
	}
	hr.ProcessServiceJson(result)
}

func (hr *HostReactor) asyncUpdateService() {
	sema := util.NewSemaphore(hr.updateThreadNum)
	for {
		for _, v := range hr.serviceInfoMap.Items() {
			service := v.(model.Service)
			lastRefTime, ok := hr.updateTimeMap.Get(util.GetServiceCacheKey(service.Name, service.Clusters))
			if !ok {
				lastRefTime = uint64(0)
			}
			if uint64(util.CurrentMillis())-lastRefTime.(uint64) > service.CacheMillis {
				sema.Acquire()
				go func() {
					hr.updateServiceNow(service.Name, service.Clusters)
					sema.Release()
				}()
			}
		}
		time.Sleep(1 * time.Second)
	}
}

// return true when service instance changed ,otherwise return false.
func isServiceInstanceChanged(oldService, newService *model.Service) bool {
	oldHostsLen := len(oldService.Hosts)
	newHostsLen := len(newService.Hosts)
	if oldHostsLen != newHostsLen {
		return true
	}
	// compare refTime
	oldRefTime := oldService.LastRefTime
	newRefTime := newService.LastRefTime
	if oldRefTime > newRefTime {
		logger.Warn(fmt.Sprintf("out of date data received, old-t: %v , new-t:  %v", oldRefTime, newRefTime))
		return false
	}
	// sort instance list
	oldInstance := oldService.Hosts
	newInstance := make([]model.Instance, len(newService.Hosts))
	copy(newInstance, newService.Hosts)
	sortInstance(oldInstance)
	sortInstance(newInstance)
	return !reflect.DeepEqual(oldInstance, newInstance)
}

type instanceSorter []model.Instance

func (s instanceSorter) Len() int {
	return len(s)
}
func (s instanceSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s instanceSorter) Less(i, j int) bool {
	insI, insJ := s[i], s[j]
	// using ip and port to sort
	ipNum1, _ := strconv.Atoi(strings.ReplaceAll(insI.Ip, ".", ""))
	ipNum2, _ := strconv.Atoi(strings.ReplaceAll(insJ.Ip, ".", ""))
	if ipNum1 < ipNum2 {
		return true
	}
	if insI.Port < insJ.Port {
		return true
	}
	return false
}

// sort instances
func sortInstance(instances []model.Instance) {
	sort.Sort(instanceSorter(instances))
}
