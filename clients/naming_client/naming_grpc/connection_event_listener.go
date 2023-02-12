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
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_proxy"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

type ConnectionEventListener struct {
	clientProxy              naming_proxy.INamingProxy
	registeredInstanceCached cache.ConcurrentMap
	subscribes               cache.ConcurrentMap
}

func NewConnectionEventListener(clientProxy naming_proxy.INamingProxy) *ConnectionEventListener {
	return &ConnectionEventListener{
		clientProxy:              clientProxy,
		registeredInstanceCached: cache.NewConcurrentMap(),
		subscribes:               cache.NewConcurrentMap(),
	}
}

func (c *ConnectionEventListener) OnConnected() {
	c.redoSubscribe()
	c.redoRegisterEachService()
}

func (c *ConnectionEventListener) OnDisConnect() {

}

func (c *ConnectionEventListener) redoSubscribe() {
	for _, key := range c.subscribes.Keys() {
		info := strings.Split(key, constant.SERVICE_INFO_SPLITER)
		var err error
		var service model.Service
		if len(info) > 2 {
			service, err = c.clientProxy.Subscribe(info[1], info[0], info[2])
		} else {
			service, err = c.clientProxy.Subscribe(info[1], info[0], "")
		}

		if err != nil {
			logger.Warnf("redo subscribe service:%s faild:%+v", info[1], err)
			return
		}

		grpcProxy, ok := c.clientProxy.(*NamingGrpcProxy)
		if !ok {
			return
		}
		grpcProxy.serviceInfoHolder.ProcessService(&service)
	}
}

func (c *ConnectionEventListener) redoRegisterEachService() {
	for k, v := range c.registeredInstanceCached.Items() {
		info := strings.Split(k, constant.SERVICE_INFO_SPLITER)
		serviceName := info[1]
		groupName := info[0]
		if instance, ok := v.(model.Instance); ok {
			if _, err := c.clientProxy.RegisterInstance(serviceName, groupName, instance); err != nil {
				logger.Warnf("redo register service:%s groupName:%s faild:%s", info[1], info[0], err.Error())
				continue
			}
		}
		if instances, ok := v.([]model.Instance); ok {
			if _, err := c.clientProxy.BatchRegisterInstance(serviceName, groupName, instances); err != nil {
				logger.Warnf("redo batch register service:%s groupName:%s faild:%s", info[1], info[0], err.Error())
				continue
			}
		}
	}
}

func (c *ConnectionEventListener) CacheInstanceForRedo(serviceName, groupName string, instance model.Instance) {
	key := util.GetGroupName(serviceName, groupName)
	c.registeredInstanceCached.Set(key, instance)
}

func (c *ConnectionEventListener) CacheInstancesForRedo(serviceName, groupName string, instances []model.Instance) {
	key := util.GetGroupName(serviceName, groupName)
	c.registeredInstanceCached.Set(key, instances)
}

func (c *ConnectionEventListener) RemoveInstanceForRedo(serviceName, groupName string, instance model.Instance) {
	key := util.GetGroupName(serviceName, groupName)
	_, ok := c.registeredInstanceCached.Get(key)
	if !ok {
		return
	}
	c.registeredInstanceCached.Remove(key)
}

func (c *ConnectionEventListener) CacheSubscriberForRedo(fullServiceName, clusters string) {
	key := util.GetServiceCacheKey(fullServiceName, clusters)
	if !c.IsSubscriberCached(key) {
		c.subscribes.Set(key, struct{}{})
	}
}

func (c *ConnectionEventListener) IsSubscriberCached(key string) bool {
	_, ok := c.subscribes.Get(key)
	return ok
}

func (c *ConnectionEventListener) RemoveSubscriberForRedo(fullServiceName, clusters string) {
	c.subscribes.Remove(util.GetServiceCacheKey(fullServiceName, clusters))
}
