package naming_cache

import (
	"sort"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

type Selector interface {
	SelectInstance(service *model.Service) []model.Instance
	Equals(o Selector) bool
}

type ClusterSelector struct {
	ClusterNames string
	Clusters     []string
}

func NewClusterSelector(clusters []string) *ClusterSelector {
	if len(clusters) == 0 {
		return &ClusterSelector{
			ClusterNames: "",
			Clusters:     []string{},
		}
	}

	// 创建副本避免外部修改
	clustersCopy := make([]string, len(clusters))
	copy(clustersCopy, clusters)

	return &ClusterSelector{
		ClusterNames: joinCluster(clusters),
		Clusters:     clustersCopy,
	}
}

func NewSubscribeCallbackFuncWrapper(selector Selector, callback *func(services []model.Instance, err error)) *SubscribeCallbackFuncWrapper {
	if selector == nil {
		panic("selector cannot be nil")
	}

	if callback == nil {
		panic("callback cannot be nil")
	}

	return &SubscribeCallbackFuncWrapper{
		Selector:     selector,
		CallbackFunc: callback,
	}
}

type SubscribeCallbackFuncWrapper struct {
	Selector     Selector
	CallbackFunc *func(services []model.Instance, err error)
}

func (ed *SubscribeCallbackFuncWrapper) notifyListener(service *model.Service) {
	instances := ed.Selector.SelectInstance(service)
	if ed.CallbackFunc != nil {
		(*ed.CallbackFunc)(instances, nil)
	}
}

func (cs *ClusterSelector) SelectInstance(service *model.Service) []model.Instance {
	var instances []model.Instance
	if cs.ClusterNames == "" {
		return service.Hosts
	}
	for _, instance := range service.Hosts {
		if util.Contains(cs.Clusters, instance.ClusterName) {
			instances = append(instances, instance)
		}
	}
	return instances
}

func (cs *ClusterSelector) Equals(o Selector) bool {
	if o == nil {
		return false
	}
	if o, ok := o.(*ClusterSelector); ok {
		return cs.ClusterNames == o.ClusterNames
	}
	return false
}

func joinCluster(cluster []string) string {
	// 使用map实现去重
	uniqueSet := make(map[string]struct{})
	for _, item := range cluster {
		if item != "" { // 过滤空字符串，类似Java中的isNotEmpty
			uniqueSet[item] = struct{}{}
		}
	}

	uniqueSlice := make([]string, 0, len(uniqueSet))
	for item := range uniqueSet {
		uniqueSlice = append(uniqueSlice, item)
	}
	sort.Strings(uniqueSlice)

	// 使用逗号连接
	return strings.Join(uniqueSlice, ",")
}
