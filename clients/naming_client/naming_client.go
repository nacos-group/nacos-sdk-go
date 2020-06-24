package naming_client

import (
	"github.com/nacos-group/nacos-sdk-go/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/logger"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/pkg/errors"
	"github.com/valyala/fastrand"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type NamingClient struct {
	nacos_client.INacosClient
	hostReactor  HostReactor
	serviceProxy NamingProxy
	subCallback  SubscribeCallback
	beatReactor  BeatReactor
	indexMap     cache.ConcurrentMap
}

type Chooser struct {
	data   []model.Instance
	totals []int
	max    int
}

func NewNamingClient(nc nacos_client.INacosClient) (NamingClient, error) {
	naming := NamingClient{}
	clientConfig, err :=
		nc.GetClientConfig()
	if err != nil {
		return naming, err
	}
	serverConfig, err := nc.GetServerConfig()
	if err != nil {
		return naming, err
	}
	httpAgent, err := nc.GetHttpAgent()
	if err != nil {
		return naming, err
	}
	err = logger.InitLog(clientConfig.LogDir)
	if err != nil {
		return naming, err
	}
	naming.subCallback = NewSubscribeCallback()
	naming.serviceProxy, err = NewNamingProxy(clientConfig, serverConfig, httpAgent)
	if err != nil {
		return naming, err
	}
	naming.hostReactor = NewHostReactor(naming.serviceProxy, clientConfig.CacheDir+string(os.PathSeparator)+"naming",
		clientConfig.UpdateThreadNum, clientConfig.NotLoadCacheAtStart, naming.subCallback, clientConfig.UpdateCacheWhenEmpty)
	naming.beatReactor = NewBeatReactor(naming.serviceProxy, clientConfig.BeatInterval)
	naming.indexMap = cache.NewConcurrentMap()

	return naming, nil
}

// 注册服务实例
func (sc *NamingClient) RegisterInstance(param vo.RegisterInstanceParam) (bool, error) {
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	instance := model.Instance{
		Ip:          param.Ip,
		Port:        param.Port,
		Metadata:    param.Metadata,
		ClusterName: param.ClusterName,
		Healthy:     param.Healthy,
		Enable:      param.Enable,
		Weight:      param.Weight,
		Ephemeral:   param.Ephemeral,
	}
	beatInfo := model.BeatInfo{
		Ip:          param.Ip,
		Port:        param.Port,
		Metadata:    param.Metadata,
		ServiceName: utils.GetGroupName(param.ServiceName, param.GroupName),
		Cluster:     param.ClusterName,
		Weight:      param.Weight,
		Period:      utils.GetDurationWithDefault(param.Metadata, constant.HEART_BEAT_INTERVAL, time.Second*5),
	}
	_, err := sc.serviceProxy.RegisterInstance(utils.GetGroupName(param.ServiceName, param.GroupName), param.GroupName, instance)
	if err != nil {
		return false, err
	}
	if instance.Ephemeral {
		sc.beatReactor.AddBeatInfo(utils.GetGroupName(param.ServiceName, param.GroupName), beatInfo)
	}
	return true, nil

}

// 注销服务实例
func (sc *NamingClient) DeregisterInstance(param vo.DeregisterInstanceParam) (bool, error) {
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	sc.beatReactor.RemoveBeatInfo(utils.GetGroupName(param.ServiceName, param.GroupName), param.Ip, param.Port)

	_, err := sc.serviceProxy.DeregisterInstance(utils.GetGroupName(param.ServiceName, param.GroupName), param.Ip, param.Port, param.Cluster, param.Ephemeral)
	if err != nil {
		return false, err
	}
	return true, nil
}

// 获取服务列表
func (sc *NamingClient) GetService(param vo.GetServiceParam) (model.Service, error) {
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	service := sc.hostReactor.GetServiceInfo(utils.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	return service, nil
}

func (sc *NamingClient) GetAllServicesInfo(param vo.GetAllServiceInfoParam) ([]model.Service, error) {
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	if param.NameSpace == "" {
		param.NameSpace = constant.DEFAULT_NAMESPACE_ID
	}
	service := sc.hostReactor.GetAllServiceInfo(param.NameSpace, param.GroupName, strings.Join(param.Clusters, ","))
	return service, nil
}

func (sc *NamingClient) SelectAllInstances(param vo.SelectAllInstancesParam) ([]model.Instance, error) {
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	service := sc.hostReactor.GetServiceInfo(utils.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	if service.Hosts == nil || len(service.Hosts) == 0 {
		return []model.Instance{}, errors.New("instance list is empty!")
	}
	return service.Hosts, nil
}

func (sc *NamingClient) SelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error) {
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	service := sc.hostReactor.GetServiceInfo(utils.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	return sc.selectInstances(service, param.HealthyOnly)
}

func (sc *NamingClient) selectInstances(service model.Service, healthy bool) ([]model.Instance, error) {
	if service.Hosts == nil || len(service.Hosts) == 0 {
		return []model.Instance{}, errors.New("instance list is empty!")
	}
	hosts := service.Hosts
	var result []model.Instance
	for _, host := range hosts {
		if host.Healthy == healthy && host.Enable && host.Weight > 0 {
			result = append(result, host)
		}
	}
	return result, nil
}

func (sc *NamingClient) SelectOneHealthyInstance(param vo.SelectOneHealthInstanceParam) (*model.Instance, error) {
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	service := sc.hostReactor.GetServiceInfo(utils.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","))
	return sc.selectOneHealthyInstances(service)
}

func (sc *NamingClient) selectOneHealthyInstances(service model.Service) (*model.Instance, error) {
	if service.Hosts == nil || len(service.Hosts) == 0 {
		return nil, errors.New("instance list is empty!")
	}
	hosts := service.Hosts
	var result []model.Instance
	mw := 0
	for _, host := range hosts {
		if host.Healthy && host.Enable && host.Weight > 0 {
			cw := int(math.Ceil(host.Weight))
			if cw > mw {
				mw = cw
			}
			result = append(result, host)
		}
	}
	if len(result) == 0 {
		return nil, errors.New("healthy instance list is empty!")
	}

	chooser := newChooser(result)
	namingClient := chooser.pick()
	return &namingClient, nil
}

func random(instances []model.Instance, mw int) []model.Instance {
	if len(instances) <= 1 || mw <= 1 {
		return instances
	}
	//实例交叉插入列表，避免列表中是连续的实例
	var result = make([]model.Instance, 0)
	for i := 1; i <= mw; i++ {
		for _, host := range instances {
			if int(math.Ceil(host.Weight)) >= i {
				result = append(result, host)
			}
		}
	}
	return result
}
type instance []model.Instance

func (a instance) Len() int {
	return len(a)
}

func (a instance) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a instance) Less(i, j int) bool {
	return a[i].Weight < a[j].Weight
}
// NewChooser initializes a new Chooser for picking from the provided Choices.
func newChooser(instances []model.Instance) Chooser {
	sort.Sort(instance(instances))
	totals := make([]int, len(instances))
	runningTotal := 0
	for i, c := range instances {
		runningTotal += int(c.Weight)
		totals[i] = runningTotal
	}
	return Chooser{data: instances, totals: totals, max: runningTotal}
}

func (chs Chooser) pick() model.Instance {
	r := fastrand.Uint32n(uint32(chs.max))+1
	i := sort.SearchInts(chs.totals, int(r))
	return chs.data[i]
}

// 服务监听
func (sc *NamingClient) Subscribe(param *vo.SubscribeParam) error {
	if param.GroupName == "" {
		param.GroupName = constant.DEFAULT_GROUP
	}
	serviceParam := vo.GetServiceParam{
		ServiceName: param.ServiceName,
		GroupName:   param.GroupName,
		Clusters:    param.Clusters,
	}

	sc.subCallback.AddCallbackFuncs(utils.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), &param.SubscribeCallback)
	_, err := sc.GetService(serviceParam)
	if err != nil {
		return err
	}
	return nil
}

//取消服务监听
func (sc *NamingClient) Unsubscribe(param *vo.SubscribeParam) error {
	sc.subCallback.RemoveCallbackFuncs(utils.GetGroupName(param.ServiceName, param.GroupName), strings.Join(param.Clusters, ","), &param.SubscribeCallback)
	return nil
}
