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
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

type SubscribeCallback struct {
	callbackFuncMap cache.ICache[string, *[]*vo.SubscribeCallbackFunc]
	mux             *sync.Mutex
}

func NewSubscribeCallback() *SubscribeCallback {
	return &SubscribeCallback{callbackFuncMap: cache.NewCache[string, *[]*vo.SubscribeCallbackFunc](), mux: new(sync.Mutex)}
}

func (ed *SubscribeCallback) IsSubscribed(serviceName, clusters string) bool {
	key := util.GetServiceCacheKey(serviceName, clusters)
	_, ok := ed.callbackFuncMap.Load(key)
	return ok
}

func (ed *SubscribeCallback) AddCallbackFunc(serviceName string, clusters string, callbackFunc *vo.SubscribeCallbackFunc) {
	key := util.GetServiceCacheKey(serviceName, clusters)
	defer ed.mux.Unlock()
	ed.mux.Lock()
	var funcSlice []*vo.SubscribeCallbackFunc
	old, ok := ed.callbackFuncMap.Load(key)
	if ok {
		funcSlice = append(funcSlice, *old...)
	}
	funcSlice = append(funcSlice, callbackFunc)
	ed.callbackFuncMap.Store(key, &funcSlice)
}

func (ed *SubscribeCallback) RemoveCallbackFunc(serviceName string, clusters string, callbackFunc *vo.SubscribeCallbackFunc) {
	logger.Info("removing " + serviceName + " with " + clusters + " to listener map")
	key := util.GetServiceCacheKey(serviceName, clusters)
	ed.mux.Lock()
	defer ed.mux.Unlock()
	funcs, ok := ed.callbackFuncMap.Load(key)
	if ok {
		var newFuncs []*vo.SubscribeCallbackFunc
		for _, funcItem := range *funcs {
			if funcItem != callbackFunc {
				newFuncs = append(newFuncs, funcItem)
			}
		}
		if len(newFuncs) == 0 {
			ed.callbackFuncMap.Delete(key)
			return
		}
		ed.callbackFuncMap.Store(key, &newFuncs)
	}

}

func (ed *SubscribeCallback) ServiceChanged(cacheKey string, service *model.Service) {
	funcs, ok := ed.callbackFuncMap.Load(cacheKey)
	if ok {
		for _, funcItem := range *funcs {
			(*funcItem)(service.Hosts, nil)
		}
	}
}
