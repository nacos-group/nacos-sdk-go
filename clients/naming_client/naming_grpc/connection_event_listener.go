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
	"sync"
	"sync/atomic"

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
	// subscribesMu makes the intent transition and the conditional removal of a
	// subscriber redo entry atomic, so a subscription started while an
	// unsubscribe response is in flight can never be removed by that response.
	subscribesMu sync.Mutex
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
	c.SubscriberOnDisconnect()
}

// SubscriberOnDisconnect resets the confirmed state of every cached subscriber
// redo entry, aligning with the Java SDK NamingGrpcRedoService onDisConnect:
// subscriptions confirmed on the lost connection become unconfirmed, so the
// reconnect redo retries them and IsSubscriberRegistered no longer reports
// them as subscribed.
func (c *ConnectionEventListener) SubscriberOnDisconnect() {
	for _, key := range c.subscribes.Keys() {
		if entry, ok := c.subscribes.Get(key); ok {
			if redoEntry, ok := entry.(*subscriberRedoEntry); ok {
				redoEntry.registered.Store(false)
			}
		}
	}
}

func (c *ConnectionEventListener) redoSubscribe() {
	grpcProxy, ok := c.clientProxy.(*NamingGrpcProxy)
	if !ok {
		logger.Error("redo subscribe clientProxy type error")
		return
	}
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
			continue
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

// subscriberRedoEntry tracks a cached subscription redo intent along with
// whether the subscription has been confirmed by the server. It aligns with
// the Java SDK SubscriberRedoData: the redo intent is cached before the
// subscribe request is sent, while the registered state is only confirmed
// after a successful subscribe response.
type subscriberRedoEntry struct {
	registered atomic.Bool
	// expectedRegistered records the current intent for this entry: true when
	// the latest request was a subscribe, false when an unsubscribe is in
	// flight. It aligns with the Java SDK SubscriberRedoData.expectedRegistered,
	// which tells a delayed unsubscribe response whether the entry still
	// belongs to that unsubscribe.
	expectedRegistered atomic.Bool
}

func (c *ConnectionEventListener) CacheSubscriberForRedo(fullServiceName, clusters string) {
	key := util.GetServiceCacheKey(fullServiceName, clusters)
	c.subscribesMu.Lock()
	defer c.subscribesMu.Unlock()
	// The Java SDK replaces the redo entry on every subscribe; the Go SDK reuses
	// it, so a new subscription must take the intent back explicitly, otherwise
	// a delayed unsubscribe response would remove its redo entry.
	if entry, ok := c.subscribes.Get(key); ok {
		if redoEntry, ok := entry.(*subscriberRedoEntry); ok {
			redoEntry.expectedRegistered.Store(true)
			return
		}
	}
	redoEntry := &subscriberRedoEntry{}
	redoEntry.expectedRegistered.Store(true)
	c.subscribes.Set(key, redoEntry)
}

func (c *ConnectionEventListener) IsSubscriberCached(key string) bool {
	_, ok := c.subscribes.Get(key)
	return ok
}

// SubscriberRegistered marks the cached redo entry as confirmed after a
// successful subscribe response.
func (c *ConnectionEventListener) SubscriberRegistered(fullServiceName, clusters string) {
	if entry, ok := c.subscribes.Get(util.GetServiceCacheKey(fullServiceName, clusters)); ok {
		if redoEntry, ok := entry.(*subscriberRedoEntry); ok {
			redoEntry.registered.Store(true)
		}
	}
}

// IsSubscriberRegistered reports whether the subscription has been confirmed
// by the server. A redo entry cached before a failed subscribe request is
// not considered registered.
func (c *ConnectionEventListener) IsSubscriberRegistered(key string) bool {
	entry, ok := c.subscribes.Get(key)
	if !ok {
		return false
	}
	redoEntry, ok := entry.(*subscriberRedoEntry)
	return ok && redoEntry.registered.Load()
}

// SubscriberDeregistering marks the cached redo entry as being unsubscribed.
// It is called before the unsubscribe request is sent, aligning with the Java
// SDK NamingGrpcRedoService.subscriberDeregister, so that a later successful
// unsubscribe response only removes the entry if no new subscription has taken
// it over in the meantime.
func (c *ConnectionEventListener) SubscriberDeregistering(fullServiceName, clusters string) {
	c.subscribesMu.Lock()
	defer c.subscribesMu.Unlock()
	if entry, ok := c.subscribes.Get(util.GetServiceCacheKey(fullServiceName, clusters)); ok {
		if redoEntry, ok := entry.(*subscriberRedoEntry); ok {
			redoEntry.expectedRegistered.Store(false)
		}
	}
}

func (c *ConnectionEventListener) RemoveSubscriberForRedo(fullServiceName, clusters string) {
	key := util.GetServiceCacheKey(fullServiceName, clusters)
	c.subscribesMu.Lock()
	defer c.subscribesMu.Unlock()
	entry, ok := c.subscribes.Get(key)
	if !ok {
		return
	}
	redoEntry, ok := entry.(*subscriberRedoEntry)
	// The entry belongs to a newer subscription, so this delayed unsubscribe
	// response must not remove it: the entry is what restores that subscription
	// when the connection is re-established.
	if !ok || redoEntry.expectedRegistered.Load() {
		return
	}
	c.subscribes.Remove(key)
}
