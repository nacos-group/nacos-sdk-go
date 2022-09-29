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

package naming_http

import (
	"context"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestBeatReactor_AddBeatInfo(t *testing.T) {
	br := NewBeatReactor(context.Background(), constant.ClientConfig{}, &nacos_server.NacosServer{})
	serviceName := "Test"
	groupName := "public"
	beatInfo := &model.BeatInfo{
		Ip:          "127.0.0.1",
		Port:        8080,
		Metadata:    map[string]string{},
		ServiceName: util.GetGroupName(serviceName, groupName),
		Cluster:     "default",
		Weight:      1,
	}
	br.AddBeatInfo(util.GetGroupName(serviceName, groupName), beatInfo)
	key := buildKey(util.GetGroupName(serviceName, groupName), beatInfo.Ip, beatInfo.Port)
	result, ok := br.beatMap.Get(key)
	assert.Equal(t, ok, true, "key should exists!")
	assert.ObjectsAreEqual(result.(*model.BeatInfo), beatInfo)
}

func TestBeatReactor_RemoveBeatInfo(t *testing.T) {
	br := NewBeatReactor(context.Background(), constant.ClientConfig{}, &nacos_server.NacosServer{})
	serviceName := "Test"
	groupName := "public"
	beatInfo1 := &model.BeatInfo{
		Ip:          "127.0.0.1",
		Port:        8080,
		Metadata:    map[string]string{},
		ServiceName: util.GetGroupName(serviceName, groupName),
		Cluster:     "default",
		Weight:      1,
	}
	br.AddBeatInfo(util.GetGroupName(serviceName, groupName), beatInfo1)
	beatInfo2 := &model.BeatInfo{
		Ip:          "127.0.0.2",
		Port:        8080,
		Metadata:    map[string]string{},
		ServiceName: util.GetGroupName(serviceName, groupName),
		Cluster:     "default",
		Weight:      1,
	}
	br.AddBeatInfo(util.GetGroupName(serviceName, groupName), beatInfo2)
	br.RemoveBeatInfo(util.GetGroupName(serviceName, groupName), "127.0.0.1", 8080)
	key := buildKey(util.GetGroupName(serviceName, groupName), beatInfo2.Ip, beatInfo2.Port)
	result, ok := br.beatMap.Get(key)
	assert.Equal(t, br.beatMap.Count(), 1, "beatinfo map length should be 1")
	assert.Equal(t, ok, true, "key should exists!")
	assert.ObjectsAreEqual(result.(*model.BeatInfo), beatInfo2)

}
