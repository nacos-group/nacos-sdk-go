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
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nacos-group/nacos-sdk-go/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/logger"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/util"
	"golang.org/x/sync/semaphore"
)

type BeatReactor struct {
	beatMap             cache.ConcurrentMap
	serviceProxy        NamingProxy
	clientBeatInterval  int64
	beatThreadCount     int
	beatThreadSemaphore *semaphore.Weighted
	beatRecordMap       cache.ConcurrentMap
	mux                 *sync.Mutex
}

const DefaultBeatThreadNum = 20

var ctx = context.Background()

func NewBeatReactor(serviceProxy NamingProxy, clientBeatInterval int64) BeatReactor {
	br := BeatReactor{}
	if clientBeatInterval <= 0 {
		clientBeatInterval = 5 * 1000
	}
	br.beatMap = cache.NewConcurrentMap()
	br.serviceProxy = serviceProxy
	br.clientBeatInterval = clientBeatInterval
	br.beatThreadCount = DefaultBeatThreadNum
	br.beatRecordMap = cache.NewConcurrentMap()
	br.beatThreadSemaphore = semaphore.NewWeighted(int64(br.beatThreadCount))
	br.mux = new(sync.Mutex)
	return br
}

func buildKey(serviceName string, ip string, port uint64) string {
	return serviceName + constant.NAMING_INSTANCE_ID_SPLITTER + ip + constant.NAMING_INSTANCE_ID_SPLITTER + strconv.Itoa(int(port))
}

func (br *BeatReactor) AddBeatInfo(serviceName string, beatInfo *model.BeatInfo) {
	logger.Infof("adding beat: <%s> to beat map", util.ToJsonString(beatInfo))
	k := buildKey(serviceName, beatInfo.Ip, beatInfo.Port)
	defer br.mux.Unlock()
	br.mux.Lock()
	if data, ok := br.beatMap.Get(k); ok {
		oldBeatInfo := data.(*model.BeatInfo)
		atomic.StoreInt32(&oldBeatInfo.State, int32(model.StateShutdown))
		br.beatMap.Remove(k)
	}
	br.beatMap.Set(k, beatInfo)
	beatInfo.Metadata = util.DeepCopyMap(beatInfo.Metadata)
	go br.sendInstanceBeat(k, beatInfo)
}

func (br *BeatReactor) RemoveBeatInfo(serviceName string, ip string, port uint64) {
	logger.Infof("remove beat: %s@%s:%d from beat map", serviceName, ip, port)
	k := buildKey(serviceName, ip, port)
	defer br.mux.Unlock()
	br.mux.Lock()
	data, exist := br.beatMap.Get(k)
	if exist {
		beatInfo := data.(*model.BeatInfo)
		atomic.StoreInt32(&beatInfo.State, int32(model.StateShutdown))
	}
	br.beatMap.Remove(k)
}

func (br *BeatReactor) sendInstanceBeat(k string, beatInfo *model.BeatInfo) {
	for {
		err := br.beatThreadSemaphore.Acquire(ctx, 1)
		if err != nil {
			logger.Errorf("sendInstanceBeat failed to acquire semaphore: %v", err)
			return
		}
		//如果当前实例注销，则进行停止心跳
		if atomic.LoadInt32(&beatInfo.State) == int32(model.StateShutdown) {
			logger.Infof("instance[%s] stop heartBeating", k)
			br.beatThreadSemaphore.Release(1)
			return
		}

		//进行心跳通信
		beatInterval, err := br.serviceProxy.SendBeat(beatInfo)
		if err != nil {
			logger.Errorf("beat to server return error:%+v", err)
			br.beatThreadSemaphore.Release(1)
			time.Sleep(beatInfo.Period)
			continue
		}
		if beatInterval > 0 {
			beatInfo.Period = time.Duration(time.Millisecond.Nanoseconds() * beatInterval)
		}

		br.beatRecordMap.Set(k, util.CurrentMillis())
		br.beatThreadSemaphore.Release(1)

		time.Sleep(beatInfo.Period)
	}
}
