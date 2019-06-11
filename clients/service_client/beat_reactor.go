package service_client

import (
	"github.com/nacos-group/nacos-sdk-go/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	nsema "github.com/toolkits/concurrent/semaphore"
	"log"
	"strconv"
	"time"
)

type BeatReactor struct {
	beatMap            cache.ConcurrentMap
	serviceProxy       ServiceProxy
	clientBeatInterval int64
	beatThreadCount    int
	beatRecordMap      cache.ConcurrentMap
}

const Default_Bead_Thread_Num = 20

func NewBeatReactor(serviceProxy ServiceProxy, clientBeatInterval int64) BeatReactor {
	br := BeatReactor{}
	if clientBeatInterval <= 0 {
		clientBeatInterval = 5 * 1000
	}
	br.beatMap = cache.NewConcurrentMap()
	br.serviceProxy = serviceProxy
	br.clientBeatInterval = clientBeatInterval
	br.beatThreadCount = Default_Bead_Thread_Num
	br.beatRecordMap = cache.NewConcurrentMap()
	go br.sendBeat()
	return br
}

func buildKey(serviceName string, ip string, port uint64) string {
	return serviceName + constant.NAMING_INSTANCE_ID_SPLITTER + ip + constant.NAMING_INSTANCE_ID_SPLITTER + strconv.Itoa(int(port))
}

func (br *BeatReactor) AddBeatInfo(serviceName string, beatInfo model.BeatInfo) {
	log.Printf("[BEAT] adding beat: {%v} to beat map.\n", beatInfo)
	br.beatMap.Set(buildKey(serviceName, beatInfo.Ip, beatInfo.Port), beatInfo)
}

func (br *BeatReactor) RemoveBeatInfo(serviceName string, ip string, port uint64) {
	log.Printf("[BEAT] remove beat: %s@%s:%d from beat map.\n", serviceName, ip, port)
	br.beatMap.Remove(buildKey(serviceName, ip, port))
}

func (br *BeatReactor) sendBeat() {
	sema := nsema.NewSemaphore(br.beatThreadCount)
	for {
		if br.beatMap.Count() > 0 {
			for k, item := range br.beatMap.Items() {
				var lastBeatTime int64
				beatInfo := item.(model.BeatInfo)
				store, ok := br.beatRecordMap.Get(k)
				if !ok {
					lastBeatTime = 0
				} else {
					lastBeatTime = store.(int64)
				}
				if utils.CurrentMillis()-lastBeatTime > br.clientBeatInterval && !beatInfo.Scheduled {
					sema.Acquire()
					beatInfo.Scheduled = true
					br.beatMap.Set(k, beatInfo)
					go func(k string, beatInfo model.BeatInfo) {
						defer sema.Release()
						beatInterval, err := br.serviceProxy.SendBeat(beatInfo)
						if err != nil {
							log.Printf("[ERROR]:beat to server return error:%s \n", err.Error())
							beatInfo.Scheduled = false
							br.beatMap.Set(k, beatInfo)
							return
						}
						if beatInterval > 0 {
							br.clientBeatInterval = beatInterval
						}
						beatInfo.Scheduled = false
						br.beatMap.Set(k, beatInfo)
						br.beatRecordMap.Set(k, utils.CurrentMillis())
					}(k, beatInfo)
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

}
