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
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/util"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
)

func TestEventDispatcher_AddCallbackFuncs(t *testing.T) {
	service := model.Service{
		Dom:         "public@@Test",
		Clusters:    strings.Join([]string{"default"}, ","),
		CacheMillis: 10000,
		Checksum:    "abcd",
		LastRefTime: uint64(time.Now().Unix()),
	}
	var hosts []model.Instance
	host := model.Instance{
		Valid:       true,
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
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			fmt.Println(util.ToJsonString(ed.callbackFuncsMap))
		},
	}
	ed.AddCallbackFuncs(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), &param.SubscribeCallback)
	key := util.GetServiceCacheKey(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	for k, v := range ed.callbackFuncsMap.Items() {
		assert.Equal(t, key, k, "key should be equal!")
		funcs := v.([]*func(services []model.SubscribeService, err error))
		assert.Equal(t, len(funcs), 1)
		assert.Equal(t, funcs[0], &param.SubscribeCallback, "callback function must be equal!")

	}
}

func TestEventDispatcher_RemoveCallbackFuncs(t *testing.T) {
	service := model.Service{
		Dom:         "public@@Test",
		Clusters:    strings.Join([]string{"default"}, ","),
		CacheMillis: 10000,
		Checksum:    "abcd",
		LastRefTime: uint64(time.Now().Unix()),
	}
	var hosts []model.Instance
	host := model.Instance{
		Valid:       true,
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
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			fmt.Printf("func1:%s \n", util.ToJsonString(services))
		},
	}
	ed.AddCallbackFuncs(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), &param.SubscribeCallback)
	assert.Equal(t, len(ed.callbackFuncsMap.Items()), 1, "callback funcs map length should be 1")

	param2 := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			fmt.Printf("func2:%s \n", util.ToJsonString(services))
		},
	}
	ed.AddCallbackFuncs(util.GetGroupName(param2.ServiceName, param2.GroupName), strings.Join(param2.Clusters, ","), &param2.SubscribeCallback)
	assert.Equal(t, len(ed.callbackFuncsMap.Items()), 1, "callback funcs map length should be 2")

	for k, v := range ed.callbackFuncsMap.Items() {
		log.Printf("key:%s,%d", k, len(v.([]*func(services []model.SubscribeService, err error))))
	}

	ed.RemoveCallbackFuncs(util.GetGroupName(param2.ServiceName, param2.GroupName), strings.Join(param2.Clusters, ","), &param2.SubscribeCallback)

	key := util.GetServiceCacheKey(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	for k, v := range ed.callbackFuncsMap.Items() {
		assert.Equal(t, key, k, "key should be equal!")
		funcs := v.([]*func(services []model.SubscribeService, err error))
		assert.Equal(t, len(funcs), 1)
		assert.Equal(t, funcs[0], &param.SubscribeCallback, "callback function must be equal!")

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
		Valid:       true,
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
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			log.Printf("func1:%s \n", util.ToJsonString(services))
		},
	}
	ed.AddCallbackFuncs(util.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), &param.SubscribeCallback)

	param2 := vo.SubscribeParam{
		ServiceName: "Test",
		Clusters:    []string{"default"},
		GroupName:   "public",
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			log.Printf("func2:%s \n", util.ToJsonString(services))

		},
	}
	ed.AddCallbackFuncs(util.GetGroupName(param2.ServiceName, param2.GroupName), strings.Join(param2.Clusters, ","), &param2.SubscribeCallback)

	ed.ServiceChanged(&service)
}
