package naming_client

import (
	"github.com/nacos-group/nacos-sdk-go/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	nsema "github.com/toolkits/concurrent/semaphore"
	"log"
	"reflect"
	"time"
)

type HostReactor struct {
	serviceInfoMap  cache.ConcurrentMap
	cacheDir        string
	updateThreadNum int
	serviceProxy    NamingProxy
	pushReceiver    PushReceiver
	subCallback     SubscribeCallback
}

const Default_Update_Thread_Num = 20

func NewHostReactor(serviceProxy NamingProxy, cacheDir string, updateThreadNum int, notLoadCacheAtStart bool, subCallback SubscribeCallback) HostReactor {
	if updateThreadNum <= 0 {
		updateThreadNum = Default_Update_Thread_Num
	}
	hr := HostReactor{
		serviceProxy:    serviceProxy,
		cacheDir:        cacheDir,
		updateThreadNum: updateThreadNum,
		serviceInfoMap:  cache.NewConcurrentMap(),
		subCallback:     subCallback,
	}
	pr := NewPushRecevier(&hr)
	hr.pushReceiver = *pr
	if !notLoadCacheAtStart {
		hr.loadCacheFromDisk()
	}
	go hr.asyncUpdateService()
	return hr
}

func (hr *HostReactor) loadCacheFromDisk() {
	serviceMap := cache.ReadFromFile(hr.cacheDir)
	if serviceMap == nil || len(serviceMap) == 0 {
		return
	}
	for k, v := range serviceMap {
		hr.serviceInfoMap.Set(k, v)
	}
}

func (hr *HostReactor) ProcessServiceJson(result string) {
	service := utils.JsonToService(result)
	if service == nil {
		return
	}
	cacheKey := utils.GetServiceCacheKey(service.Name, service.Clusters)

	oldDomain, ok := hr.serviceInfoMap.Get(cacheKey)
	if ok {
		//如果无可用的实例，不更新当前缓存
		hosts := service.Hosts
		var result []model.Host
		for _, host := range hosts {
			if host.Enable && host.Valid && host.Weight > 0 {
				result = append(result, host)
			}
		}

		if len(result) == 0 {
			log.Printf("[ERROR]:do not have useful host, ignore it, name:%s \n", service.Name)
			return
		}
	}
	if !ok || ok && !reflect.DeepEqual(service.Hosts, oldDomain.(model.Service).Hosts) {
		if !ok {
			log.Println("service not found in cache " + cacheKey)
		} else {
			log.Printf("service key:%s was updated to:%s \n", cacheKey, utils.ToJsonString(service))
		}
		cache.WriteToFile(*service, hr.cacheDir)
		hr.subCallback.ServiceChanged(service)
	}
	//service.LastRefTime = uint64(utils.CurrentMillis())
	hr.serviceInfoMap.Set(cacheKey, *service)
}

func (hr *HostReactor) GetServiceInfo(serviceName string, clusters string) model.Service {
	key := utils.GetServiceCacheKey(serviceName, clusters)
	cacheService, ok := hr.serviceInfoMap.Get(key)
	if !ok {
		cacheService = model.Service{Name: serviceName, Clusters: clusters}
		hr.serviceInfoMap.Set(key, cacheService)
		hr.updateServiceNow(serviceName, clusters)
	}
	newService, _ := hr.serviceInfoMap.Get(key)

	return newService.(model.Service)
}

func (hr *HostReactor) updateServiceNow(serviceName string, clusters string) {
	result, err := hr.serviceProxy.QueryList(serviceName, clusters, hr.pushReceiver.port, false)
	if err != nil {
		log.Printf("[ERROR]:query list return error!servieName:%s cluster:%s  err:%s \n", serviceName, clusters, err.Error())
		return
	}
	if result == "" {
		log.Printf("[ERROR]:query list is empty!servieName:%s cluster:%s \n", serviceName, clusters)
		return
	}
	hr.ProcessServiceJson(result)
}

func (hr *HostReactor) asyncUpdateService() {
	sema := nsema.NewSemaphore(hr.updateThreadNum)
	for {
		for _, v := range hr.serviceInfoMap.Items() {
			service := v.(model.Service)
			if uint64(utils.CurrentMillis())-service.LastRefTime > service.CacheMillis {
				sema.Acquire()
				go func() {
					hr.updateServiceNow(service.Name, service.Clusters)
					sema.Release()
				}()
			}
		}
		time.Sleep(1 * time.Second)
	}

}
