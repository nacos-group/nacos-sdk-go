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
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc"
	"sync/atomic"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

type (
	IRedoService interface {
		rpc.IConnectionEventListener
		CacheInstanceForRedo(serviceName, groupName string, instance model.Instance)

		CacheInstancesForRedo(serviceName, groupName string, instances []model.Instance)

		CacheSubscriberForRedo(service, group, clusters string)

		InstanceDeRegister(service, group string)

		SubscribeDeRegister(service, group, clusters string)

		InstanceRegistered(service, group string)

		InstanceDeRegistered(service, group string)

		SubscribeRegistered(service, group, clusters string)

		SubscribeDeRegistered(service, group, clusters string)

		RemoveInstanceForRedo(serviceName, groupName string)

		RemoveSubscriberForRedo(service, group, clusters string)

		IsSubscriberCached(service, group, clusters string) bool

		IsConnected() bool

		FindNeedRedoData() []IRedoData
	}

	IWeRedoTask interface {
		DoRedo()
	}

	WeRedoService struct {
		registeredRedoInstanceCached cache.IComputeCache[string, IRedoData]
		ctx                          context.Context
		connected                    atomic.Bool
		task                         IWeRedoTask
	}
	WeRedoTask struct {
		clientProxy *NamingGrpcProxy
		redoService IRedoService
	}
)

func NewRedoService(ctx context.Context, clientProxy *NamingGrpcProxy) *WeRedoService {
	w := &WeRedoService{
		ctx:                          ctx,
		registeredRedoInstanceCached: cache.NewCache[string, IRedoData](),
	}
	w.task = &WeRedoTask{clientProxy: clientProxy, redoService: w}
	go w.scheduleRedo()
	return w
}

func (c *WeRedoService) scheduleRedo() {
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.task.DoRedo()
		}
	}
}

func getSubscribeCacheKey(service, clusters string) string {
	return service + constant.SERVICE_INFO_SPLITER + clusters
}

func (c *WeRedoService) OnConnected() {
	c.connected.Store(true)
	logger.Infof("redo notice connection connected")
}

func (c *WeRedoService) OnDisConnect() {
	c.connected.Store(false)
	logger.Infof("redo notice connection disconnected")
	c.registeredRedoInstanceCached.Range(func(key string, value IRedoData) bool {
		value.SetRegistered(false)
		return true
	})
}

func (c *WeRedoService) CacheInstanceForRedo(serviceName, groupName string, instance model.Instance) {
	key := util.GetGroupName(serviceName, groupName)
	c.registeredRedoInstanceCached.Store(key, NewInstanceRedoData(serviceName, groupName, instance))
}

func (c *WeRedoService) CacheInstancesForRedo(serviceName, groupName string, instances []model.Instance) {
	key := util.GetGroupName(serviceName, groupName)
	c.registeredRedoInstanceCached.Store(key, NewBatchInstancesRedoData(serviceName, groupName, instances))
}

func (c *WeRedoService) CacheSubscriberForRedo(service, group, clusters string) {
	c.registeredRedoInstanceCached.Store(getSubscribeCacheKey(util.GetGroupName(service, group), clusters), NewSubscribeRedoData(service, group, clusters))
}

func (c *WeRedoService) InstanceRegistered(service, group string) {
	c.registeredRedoInstanceCached.ComputeIfPresent(util.GetGroupName(service, group), func(value IRedoData) IRedoData {
		value.Registered()
		return value
	})
}

func (c *WeRedoService) SubscribeRegistered(service, group, clusters string) {
	c.registeredRedoInstanceCached.ComputeIfPresent(getSubscribeCacheKey(util.GetGroupName(service, group), clusters), func(value IRedoData) IRedoData {
		value.Registered()
		return value
	})
}

func (c *WeRedoService) RemoveInstanceForRedo(serviceName, groupName string) {
	c.registeredRedoInstanceCached.Delete(util.GetGroupName(serviceName, groupName))
}

func (c *WeRedoService) RemoveSubscriberForRedo(service, group, clusters string) {
	c.registeredRedoInstanceCached.Delete(getSubscribeCacheKey(util.GetGroupName(service, group), clusters))
}

func (c *WeRedoService) IsSubscriberCached(service, group, clusters string) bool {
	_, ok := c.registeredRedoInstanceCached.Load(getSubscribeCacheKey(util.GetGroupName(service, group), clusters))
	return ok
}

func (c *WeRedoService) InstanceDeRegister(service, group string) {
	c.registeredRedoInstanceCached.ComputeIfPresent(util.GetGroupName(service, group), func(value IRedoData) IRedoData {
		value.SetUnRegistering(true)
		value.SetExpectRegistered(false)
		return value
	})
}

func (c *WeRedoService) SubscribeDeRegister(service, group, clusters string) {
	c.registeredRedoInstanceCached.ComputeIfPresent(getSubscribeCacheKey(util.GetGroupName(service, group), clusters), func(value IRedoData) IRedoData {
		value.SetUnRegistering(true)
		value.SetExpectRegistered(false)
		return value
	})
}

func (c *WeRedoService) InstanceDeRegistered(service, group string) {
	c.registeredRedoInstanceCached.ComputeIfPresent(util.GetGroupName(service, group), func(value IRedoData) IRedoData {
		value.Unregistered()
		return value
	})
}

func (c *WeRedoService) SubscribeDeRegistered(service, group, clusters string) {
	c.registeredRedoInstanceCached.ComputeIfPresent(getSubscribeCacheKey(util.GetGroupName(service, group), clusters), func(value IRedoData) IRedoData {
		value.Unregistered()
		return value
	})
}

func (c *WeRedoService) IsConnected() bool {
	return c.connected.Load()
}

func (c *WeRedoService) FindNeedRedoData() []IRedoData {
	var result []IRedoData
	c.registeredRedoInstanceCached.Range(func(key string, value IRedoData) bool {
		if value.IsNeedRedo() {
			result = append(result, value)
		}
		return true
	})
	return result
}

func (w *WeRedoTask) DoRedo() {
	if !w.redoService.IsConnected() {
		logger.Debug("gRPC connection status is disconnected, skip current redo task")
		return
	}
	needRedoData := w.redoService.FindNeedRedoData()
	for _, value := range needRedoData {
		redoType := value.GetRedoType()
		logger.Infof("redo task will process type %T, target %v", value, redoType)
		switch t := value.(type) {
		case *InstanceRedoData:
			switch redoType {
			case register:
				if err := w.clientProxy.DoRegisterInstance(t.ServiceName, t.GroupName, t.Get()); err != nil {
					logger.Warnf("redo register service:%s groupName:%s failed:%s", t.ServiceName, t.GroupName, err.Error())
				}
			case unregister:
				if err := w.clientProxy.DoDeRegisterInstance(t.ServiceName, t.GroupName, model.Instance{}); err != nil {
					logger.Warnf("redo register service:%s groupName:%s failed:%s", t.ServiceName, t.GroupName, err.Error())
				}
			case remove:
				w.redoService.RemoveInstanceForRedo(t.ServiceName, t.GroupName)
			}
		case *BatchInstancesRedoData:
			switch redoType {
			case register:
				if err := w.clientProxy.DoBatchRegisterInstance(t.ServiceName, t.GroupName, t.Get()); err != nil {
					logger.Warnf("redo register service:%s groupName:%s failed:%s", t.ServiceName, t.GroupName, err.Error())
				}
			case unregister:
				if err := w.clientProxy.DoDeRegisterInstance(t.ServiceName, t.GroupName, model.Instance{}); err != nil {
					logger.Warnf("redo register service:%s groupName:%s failed:%s", t.ServiceName, t.GroupName, err.Error())
				}
			case remove:
				w.redoService.RemoveInstanceForRedo(t.ServiceName, t.GroupName)
			}
		case *SubscribeRedoData:
			switch redoType {
			case register:
				_, err := w.clientProxy.DoSubscribe(t.ServiceName, t.GroupName, t.Get())
				if err != nil {
					logger.Warnf("redo register service:%s groupName:%s failed:%s", t.ServiceName, t.GroupName, err.Error())
				}
			case unregister:
				if err := w.clientProxy.DoUnSubscribe(t.ServiceName, t.GroupName, t.Get()); err != nil {
					logger.Warnf("redo register service:%s groupName:%s failed:%s", t.ServiceName, t.GroupName, err.Error())
				}
			case remove:
				w.redoService.RemoveSubscriberForRedo(t.ServiceName, t.GroupName, t.cluster)
			}
		}
	}
}
