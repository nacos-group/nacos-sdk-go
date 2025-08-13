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
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/assert"
)

func TestEventDispatcher_AddCallbackFuncs(t *testing.T) {
	service := model.Service{
		Clusters:    strings.Join([]string{"default"}, ","),
		CacheMillis: 10000,
		Checksum:    "abcd",
		LastRefTime: uint64(time.Now().Unix()),
	}
	var hosts []model.Instance
	host := model.Instance{
		Enable:      true,
		InstanceId:  "123",
		Port:        8080,
		Ip:          "127.0.0.1",
		Weight:      10,
		ServiceName: "public@@Test",
		ClusterName: strings.Join([]string{"default"}, ","),
	}
	hosts = append(hosts, host)
	service.Hosts = hosts

	ed := NewSubscribeCallback()
	param := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.Instance, err error) {
			fmt.Println(util.ToJsonString(ed.callbackFuncMap))
		},
	}
	clusterSelector := NewClusterSelector(param.Clusters)
	callbackWrapper := NewSubscribeCallbackFuncWrapper(clusterSelector, &param.SubscribeCallback)
	ed.AddCallbackFunc(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), callbackWrapper)
	key := util.GetServiceCacheKey(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	for k, v := range ed.callbackFuncMap.Items() {
		assert.Equal(t, key, k, "key should be equal!")
		funcs := v.([]*SubscribeCallbackFuncWrapper)
		assert.Equal(t, len(funcs), 1)
		assert.Equal(t, funcs[0].CallbackFunc, &param.SubscribeCallback, "callback function must be equal!")

	}
}

func TestEventDispatcher_RemoveCallbackFuncs(t *testing.T) {
	service := model.Service{
		Clusters:    strings.Join([]string{"default"}, ","),
		CacheMillis: 10000,
		Checksum:    "abcd",
		LastRefTime: uint64(time.Now().Unix()),
	}
	var hosts []model.Instance
	host := model.Instance{
		Enable:      true,
		InstanceId:  "123",
		Port:        8080,
		Ip:          "127.0.0.1",
		Weight:      10,
		ServiceName: "public@@Test",
		ClusterName: strings.Join([]string{"default"}, ","),
	}
	hosts = append(hosts, host)
	service.Hosts = hosts

	ed := NewSubscribeCallback()
	param := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.Instance, err error) {
			fmt.Printf("func1:%s \n", util.ToJsonString(services))
		},
	}
	clusterSelector := NewClusterSelector(param.Clusters)
	callbackWrapper := NewSubscribeCallbackFuncWrapper(clusterSelector, &param.SubscribeCallback)
	ed.AddCallbackFunc(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), callbackWrapper)
	assert.Equal(t, len(ed.callbackFuncMap.Items()), 1, "callback funcs map length should be 1")

	param2 := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.Instance, err error) {
			fmt.Printf("func2:%s \n", util.ToJsonString(services))
		},
	}
	clusterSelector2 := NewClusterSelector(param2.Clusters)
	callbackWrapper2 := NewSubscribeCallbackFuncWrapper(clusterSelector2, &param2.SubscribeCallback)
	ed.AddCallbackFunc(util.GetGroupName(param2.ServiceName, param2.GroupName), strings.Join(param2.Clusters, ","), callbackWrapper2)
	assert.Equal(t, len(ed.callbackFuncMap.Items()), 1, "callback funcs map length should be 2")

	for k, v := range ed.callbackFuncMap.Items() {
		log.Printf("key:%s,%d", k, len(v.([]*SubscribeCallbackFuncWrapper)))
	}

	ed.RemoveCallbackFunc(util.GetGroupName(param2.ServiceName, param2.GroupName), strings.Join(param2.Clusters, ","), callbackWrapper2)

	key := util.GetServiceCacheKey(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	for k, v := range ed.callbackFuncMap.Items() {
		assert.Equal(t, key, k, "key should be equal!")
		funcs := v.([]*SubscribeCallbackFuncWrapper)
		assert.Equal(t, len(funcs), 1)
		assert.Equal(t, funcs[0].CallbackFunc, &param.SubscribeCallback, "callback function must be equal!")

	}
}

func TestSubscribeCallback_ServiceChanged(t *testing.T) {
	service := model.Service{
		Name:        "public@@Test",
		Clusters:    strings.Join([]string{"default"}, ","),
		CacheMillis: 10000,
		Checksum:    "abcd",
		LastRefTime: uint64(time.Now().Unix()),
	}
	var hosts []model.Instance
	host := model.Instance{
		Enable:      true,
		InstanceId:  "123",
		Port:        8080,
		Ip:          "127.0.0.1",
		Weight:      10,
		ServiceName: "public@@Test",
		ClusterName: strings.Join([]string{"default"}, ","),
	}
	hosts = append(hosts, host)
	service.Hosts = hosts

	ed := NewSubscribeCallback()
	param := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.Instance, err error) {
			log.Printf("func1:%s \n", util.ToJsonString(services))
		},
	}
	clusterSelector := NewClusterSelector(param.Clusters)
	callbackWrapper := NewSubscribeCallbackFuncWrapper(clusterSelector, &param.SubscribeCallback)
	ed.AddCallbackFunc(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), callbackWrapper)

	param2 := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.Instance, err error) {
			log.Printf("func2:%s \n", util.ToJsonString(services))

		},
	}
	clusterSelector2 := NewClusterSelector(param2.Clusters)
	callbackWrapper2 := NewSubscribeCallbackFuncWrapper(clusterSelector2, &param2.SubscribeCallback)
	ed.AddCallbackFunc(util.GetGroupName(param2.ServiceName, param2.GroupName), strings.Join(param2.Clusters, ","), callbackWrapper2)
	cacheKey := util.GetServiceCacheKey(util.GetGroupName(service.Name, service.GroupName), service.Clusters)
	ed.ServiceChanged(cacheKey, &service)
}

func TestSubscribeCallback_RemoveCallbackFunc(t *testing.T) {
	ed := NewSubscribeCallback()
	serviceName := "Test"
	clusters := "default"
	groupName := "public"

	callback1 := func(services []model.Instance, err error) {
		log.Printf("callback1:%s \n", util.ToJsonString(services))
	}
	clusterSelector1 := NewClusterSelector([]string{clusters})
	callbackWrapper1 := NewSubscribeCallbackFuncWrapper(clusterSelector1, &callback1)

	callback2 := func(services []model.Instance, err error) {
		log.Printf("callback2:%s \n", util.ToJsonString(services))
	}
	clusterSelector2 := NewClusterSelector([]string{clusters})
	callbackWrapper2 := NewSubscribeCallbackFuncWrapper(clusterSelector2, &callback2)

	// Add both callbacks
	ed.AddCallbackFunc(util.GetGroupName(serviceName, groupName), clusters, callbackWrapper1)
	ed.AddCallbackFunc(util.GetGroupName(serviceName, groupName), clusters, callbackWrapper2)

	assert.True(t, ed.IsSubscribed(util.GetGroupName(serviceName, groupName), clusters))
	// Remove the first callback
	ed.RemoveCallbackFunc(util.GetGroupName(serviceName, groupName), clusters, callbackWrapper1)

	// Check if only the second callback remains
	cacheKey := util.GetServiceCacheKey(util.GetGroupName(serviceName, groupName), clusters)
	funcs, ok := ed.callbackFuncMap.Get(cacheKey)
	if !ok || len(funcs.([]*SubscribeCallbackFuncWrapper)) != 1 {
		t.Errorf("Expected 1 callback function, got %d", len(funcs.([]*SubscribeCallbackFuncWrapper)))
	}

	assert.True(t, ed.IsSubscribed(util.GetGroupName(serviceName, groupName), clusters))
	// Remove the second callback
	ed.RemoveCallbackFunc(util.GetGroupName(serviceName, groupName), clusters, callbackWrapper2)

	// Check if no callbacks remain
	funcs, ok = ed.callbackFuncMap.Get(cacheKey)
	if ok && len(funcs.([]*SubscribeCallbackFuncWrapper)) != 0 {
		t.Errorf("Expected 0 callback functions, got %d", len(funcs.([]*func(services []model.Instance, err error))))
	}
	assert.False(t, ed.IsSubscribed(util.GetGroupName(serviceName, groupName), clusters))
}
