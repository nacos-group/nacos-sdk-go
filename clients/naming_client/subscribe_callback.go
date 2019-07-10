package naming_client

import (
	"errors"
	"github.com/nacos-group/nacos-sdk-go/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"log"
	"strings"
)

type SubscribeCallback struct {
	callbackFuncsMap cache.ConcurrentMap
}

func NewSubscribeCallback() SubscribeCallback {
	ed := SubscribeCallback{}
	ed.callbackFuncsMap = cache.NewConcurrentMap()
	return ed
}

func (ed *SubscribeCallback) AddCallbackFuncs(clusters string, serviceName string, callbackFuncId string, callbackFunc *func(services []model.SubscribeService, err error)) {
	log.Printf("[INFO] adding " + serviceName + " with " + clusters + "'s callbackFunc[" + callbackFuncId + "] to listener map")
	key := utils.GetServiceCacheKey(serviceName, clusters)
	old, ok := ed.callbackFuncsMap.Get(key)
	if ok && old != nil {
		funcs := old.(map[string]*func(services []model.SubscribeService, err error))
		funcs[callbackFuncId] = callbackFunc
	} else {
		var funcs = make(map[string]*func(services []model.SubscribeService, err error), 0)
		funcs[callbackFuncId] = callbackFunc
		ed.callbackFuncsMap.Set(key, funcs)
	}
}

func (ed *SubscribeCallback) RemoveCallbackFuncs(clusters string, serviceName string, callbackFuncId string, callbackFunc *func(services []model.SubscribeService, err error)) {
	log.Printf("[INFO] removing " + serviceName + " with " + clusters + "'s callbackFunc[" + callbackFuncId + "] to listener map")
	key := utils.GetServiceCacheKey(serviceName, clusters)
	funcs, ok := ed.callbackFuncsMap.Get(key)
	if ok && funcs != nil {
		delete(funcs.(map[string]*func(services []model.SubscribeService, err error)), callbackFuncId)
	}
}

func (ed *SubscribeCallback) ServiceChanged(service *model.Service) {
	if service == nil || service.Name == "" {
		return
	}

	clusters := strings.Split(service.Clusters, ",")
	if clusters == nil || len(clusters) == 0 {
		clusters = []string{constant.STRING_EMPTY}
	}

	for index := range clusters {
		clusterName := clusters[index]
		key := utils.GetServiceCacheKey(service.Name, clusterName)
		funcs, ok := ed.callbackFuncsMap.Get(key)
		if ok && funcs != nil {
			for _, funcItem := range funcs.(map[string]*func(services []model.SubscribeService, err error)) {
				var subscribeServices []model.SubscribeService
				if len(service.Hosts) == 0 {
					(*funcItem)(subscribeServices, errors.New("[client.Subscribe] subscribe failed,hosts is empty"))
					return
				}
				for _, host := range service.Hosts {
					var subscribeService model.SubscribeService
					if clusterName == constant.STRING_EMPTY || clusterName == host.ClusterName {
						subscribeService.Valid = host.Valid
						subscribeService.Port = host.Port
						subscribeService.Ip = host.Ip
						subscribeService.Metadata = service.Metadata
						subscribeService.ServiceName = host.ServiceName
						subscribeService.ClusterName = host.ClusterName
						subscribeService.Weight = host.Weight
						subscribeService.InstanceId = host.InstanceId
						subscribeService.Enable = host.Enable
						subscribeServices = append(subscribeServices, subscribeService)
					}
				}
				if subscribeServices != nil && len(subscribeServices) > 0 {
					(*funcItem)(subscribeServices, nil)
				}
			}
		}
	}
}
