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
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/nacos-group/nacos-sdk-go/v2/common/monitor"

	"github.com/buger/jsonparser"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"golang.org/x/sync/semaphore"
)

type BeatReactor struct {
	beatMap             cache.ConcurrentMap
	nacosServer         *nacos_server.NacosServer
	beatThreadCount     int
	beatThreadSemaphore *semaphore.Weighted
	beatRecordMap       cache.ConcurrentMap
	clientCfg           constant.ClientConfig
	mux                 *sync.Mutex
	cancelBeating       func()
	beatingCtx          context.Context
}

const DefaultBeatThreadNum = 20

func NewBeatReactor(clientCfg constant.ClientConfig, nacosServer *nacos_server.NacosServer) BeatReactor {
	br := BeatReactor{}
	br.beatMap = cache.NewConcurrentMap()
	br.nacosServer = nacosServer
	br.clientCfg = clientCfg
	br.beatThreadCount = DefaultBeatThreadNum
	br.beatRecordMap = cache.NewConcurrentMap()
	br.beatThreadSemaphore = semaphore.NewWeighted(int64(br.beatThreadCount))
	br.mux = new(sync.Mutex)
	bctx, cancelBeating := context.WithCancel(context.Background())
	br.beatingCtx, br.cancelBeating = bctx, cancelBeating
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
		beatInfo = data.(*model.BeatInfo)
		atomic.StoreInt32(&beatInfo.State, int32(model.StateShutdown))
		br.beatMap.Remove(k)
	}
	br.beatMap.Set(k, beatInfo)
	beatInfo.Metadata = util.DeepCopyMap(beatInfo.Metadata)
	monitor.GetDom2BeatSizeMonitor().Set(float64(br.beatMap.Count()))
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
	monitor.GetDom2BeatSizeMonitor().Set(float64(br.beatMap.Count()))
	br.beatMap.Remove(k)

}

func (br *BeatReactor) sendInstanceBeat(k string, beatInfo *model.BeatInfo) {
	defer func() {
		logger.Infof("beating of instance: %s@%s:%d has been cancelled",
			beatInfo.ServiceName, beatInfo.Ip, beatInfo.Port)
	}()

	beatTimer := time.NewTimer(beatInfo.Period)

	for {
		select {
		case <-beatTimer.C:
		case <-br.beatingCtx.Done():
			beatTimer.Stop()
			return
		}

		if err := br.beatThreadSemaphore.Acquire(br.beatingCtx, 1); err != nil {
			return
		}

		//如果当前实例注销，则进行停止心跳
		if atomic.LoadInt32(&beatInfo.State) == int32(model.StateShutdown) {
			logger.Infof("instance[%s] stop heartBeating", k)
			br.beatThreadSemaphore.Release(1)
			return
		}

		//进行心跳通信
		beatInterval, err := br.SendBeat(beatInfo)
		if err != nil {
			logger.Errorf("beat to server return error:%+v", err)
		} else {
			if beatInterval > 0 {
				beatInfo.Period = time.Duration(time.Millisecond.Nanoseconds() * beatInterval)
			}
			br.beatRecordMap.Set(k, util.CurrentMillis())
		}
		br.beatThreadSemaphore.Release(1)
		beatTimer.Reset(beatInfo.Period)
	}
}

func (br *BeatReactor) SendBeat(info *model.BeatInfo) (int64, error) {
	logger.Infof("namespaceId:<%s> sending beat to server:<%s>",
		br.clientCfg.NamespaceId, util.ToJsonString(info))
	params := map[string]string{}
	params["namespaceId"] = br.clientCfg.NamespaceId
	params["serviceName"] = info.ServiceName
	params["beat"] = util.ToJsonString(info)
	api := constant.SERVICE_BASE_PATH + "/instance/beat"
	result, err := br.nacosServer.ReqApi(api, params, http.MethodPut)
	if err != nil {
		return 0, err
	}
	if result != "" {
		interVal, err := jsonparser.GetInt([]byte(result), "clientBeatInterval")
		if err != nil {
			return 0, errors.New(fmt.Sprintf("namespaceId:<%s> sending beat to server:<%s> get 'clientBeatInterval' from <%s> error:<%+v>", br.clientCfg.NamespaceId, util.ToJsonString(info), result, err))
		} else {
			return interVal, nil
		}
	}
	return 0, nil
}

func (br *BeatReactor) Close() {
	br.cancelBeating()
}
