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
	"reflect"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

type ConnectionEventListener struct {
	clientProxy              *NamingGrpcProxy
	registeredInstanceCached cache.ConcurrentMap
	subscribes               cache.ConcurrentMap
}

func NewConnectionEventListener(clientProxy *NamingGrpcProxy) *ConnectionEventListener {
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
		if len(info) > 2 {
			_, err = c.clientProxy.Subscribe(info[0], info[1], info[2])
		} else {
			_, err = c.clientProxy.Subscribe(info[0], info[1], "")
		}

		if err != nil {
			logger.Warnf("redo subscribe service:%s faild:%+v", info[0], err)
		}
	}
}

func (c *ConnectionEventListener) redoRegisterEachService() {
	for k, v := range c.registeredInstanceCached.Items() {
		info := strings.Split(k, constant.SERVICE_INFO_SPLITER)
		serviceName := info[0]
		groupName := info[1]
		instances, ok := v.([]model.Instance)
		if !ok {
			logger.Warnf("redo register service:%s faild,instances type not is []model.instance", info[0])
		}
		for _, instance := range instances {
			_, err := c.clientProxy.RegisterInstance(serviceName, groupName, instance)
			if err != nil {
				logger.Warnf("redo register service:%s groupName:%s faild:%s", info[0], info[1], err.Error())
			}
		}
	}
}

func (c *ConnectionEventListener) CacheInstanceForRedo(serviceName, groupName string, instance model.Instance) {
	var flag bool
	key := util.GetGroupName(serviceName, groupName)
	instances, ok := c.registeredInstanceCached.Get(key)
	if !ok {
		instances = []model.Instance{instance}
		c.registeredInstanceCached.Set(key, instances)
	} else {
		instances, ok := instances.([]model.Instance)
		if !ok {
			c.registeredInstanceCached.Remove(key)
			logger.Warn("cacheInstanceForRedo faild,instances type not is []model.instance")
			return
		}
		for _, v := range instances {
			if reflect.DeepEqual(v, instance) {
				flag = false
			}
		}
		if flag {
			instances = append(instances, instance)
			c.registeredInstanceCached.Set(key, instances)
		}
	}
}

func (c *ConnectionEventListener) RemoveInstanceForRedo(serviceName, groupName string, instance model.Instance) {
	key := util.GetGroupName(serviceName, groupName)
	instances, ok := c.registeredInstanceCached.Get(key)
	if !ok {
		return
	}
	instanceSlice := instances.([]model.Instance)
	instancesNew := make([]model.Instance, 0, len(instanceSlice))
	for _, v := range instanceSlice {
		if !reflect.DeepEqual(v, instance) {
			instancesNew = append(instancesNew, v)
		}
	}
	c.registeredInstanceCached.Set(key, instancesNew)
}

func (c *ConnectionEventListener) CacheSubscriberForRedo(fullServiceName, clusters string) {
	key := util.GetServiceCacheKey(fullServiceName, clusters)
	if _, ok := c.subscribes.Get(key); !ok {
		c.subscribes.Set(key, struct{}{})
	}
	return
}

func (c *ConnectionEventListener) RemoveSubscriberForRedo(fullServiceName, clusters string) {
	c.subscribes.Remove(util.GetServiceCacheKey(fullServiceName, clusters))
}
