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
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_cache"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_proxy"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

type ServiceInfoUpdater struct {
	ctx               context.Context
	serviceInfoHolder *naming_cache.ServiceInfoHolder
	updateThreadNum   int
	namingProxy       naming_proxy.INamingProxy
}

func NewServiceInfoUpdater(ctx context.Context, serviceInfoHolder *naming_cache.ServiceInfoHolder, updateThreadNum int,
	namingProxy naming_proxy.INamingProxy) *ServiceInfoUpdater {

	return &ServiceInfoUpdater{
		ctx:               ctx,
		serviceInfoHolder: serviceInfoHolder,
		updateThreadNum:   updateThreadNum,
		namingProxy:       namingProxy,
	}
}

func (s *ServiceInfoUpdater) asyncUpdateService() {
	sema := util.NewSemaphore(s.updateThreadNum)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			s.serviceInfoHolder.ServiceInfoMap.Range(func(key, value interface{}) bool {
				service := value.(model.Service)
				lastRefTime, ok := s.serviceInfoHolder.UpdateTimeMap.Load(util.GetServiceCacheKey(util.GetGroupName(service.Name, service.GroupName),
					service.Clusters))
				if !ok {
					lastRefTime = uint64(0)
				}
				if uint64(util.CurrentMillis())-lastRefTime.(uint64) > service.CacheMillis {
					sema.Acquire()
					go func() {
						defer sema.Release()
						s.updateServiceNow(service.Name, service.GroupName, service.Clusters)
					}()
				}
				return true
			})
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *ServiceInfoUpdater) updateServiceNow(serviceName, groupName, clusters string) {
	result, err := s.namingProxy.QueryInstancesOfService(serviceName, groupName, clusters, 0, false)

	if err != nil {
		logger.Errorf("QueryInstances error, serviceName:%s, cluster:%s, err:%v", serviceName, clusters, err)
		return
	}
	s.serviceInfoHolder.ProcessService(result)
}
