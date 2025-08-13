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
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

type SubscribeCallback struct {
	callbackFuncMap cache.ConcurrentMap
	mux             *sync.Mutex
}

func NewSubscribeCallback() *SubscribeCallback {
	return &SubscribeCallback{callbackFuncMap: cache.NewConcurrentMap(), mux: new(sync.Mutex)}
}

func (ed *SubscribeCallback) IsSubscribed(serviceName, clusters string) bool {
	key := util.GetServiceCacheKey(serviceName, clusters)
	funcs, ok := ed.callbackFuncMap.Get(key)
	if ok {
		return len(funcs.([]*SubscribeCallbackFuncWrapper)) > 0
	}
	return false
}

func (ed *SubscribeCallback) AddCallbackFunc(serviceName string, clusters string, callbackWrapper *SubscribeCallbackFuncWrapper) {
	key := util.GetServiceCacheKey(serviceName, clusters)
	ed.mux.Lock()
	defer ed.mux.Unlock()
	var funcSlice []*SubscribeCallbackFuncWrapper
	old, ok := ed.callbackFuncMap.Get(key)
	if ok {
		funcSlice = append(funcSlice, old.([]*SubscribeCallbackFuncWrapper)...)
	}
	funcSlice = append(funcSlice, callbackWrapper)
	ed.callbackFuncMap.Set(key, funcSlice)
}

func (ed *SubscribeCallback) RemoveCallbackFunc(serviceName string, clusters string, callbackWrapper *SubscribeCallbackFuncWrapper) {
	logger.Info("removing " + serviceName + " with " + clusters + " to listener map")
	key := util.GetServiceCacheKey(serviceName, clusters)
	funcs, ok := ed.callbackFuncMap.Get(key)
	if ok && funcs != nil {
		var newFuncs []*SubscribeCallbackFuncWrapper
		for _, funcItem := range funcs.([]*SubscribeCallbackFuncWrapper) {
			if funcItem.CallbackFunc != callbackWrapper.CallbackFunc || !funcItem.Selector.Equals(callbackWrapper.Selector) {
				newFuncs = append(newFuncs, funcItem)
			}
		}
		ed.callbackFuncMap.Set(key, newFuncs)
	}

}

func (ed *SubscribeCallback) ServiceChanged(cacheKey string, service *model.Service) {
	funcs, ok := ed.callbackFuncMap.Get(cacheKey)
	if ok {
		for _, funcItem := range funcs.([]*SubscribeCallbackFuncWrapper) {
			funcItem.notifyListener(service)
		}
	}
}
