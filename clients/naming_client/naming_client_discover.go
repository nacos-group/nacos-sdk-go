package naming_client

import (
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// Discover 用于获取服务实例列表。
// options: 可选配置项函数的集合，用于定制发现行为。
// 返回值: 返回符合筛选条件的服务实例列表和可能发生的错误。
func (sc *NamingClient) Discover(options ...DiscoverOptionFunc) ([]model.Instance, error) {
	opts := &discoverOptions{}
	for _, opt := range options {
		opt(opts)
	}
	return sc.discover(opts)
}

// DiscoverOne 用于获取单个服务实例。
// options: 可选配置项函数的集合，用于定制发现行为。
// 返回值: 返回符合条件的单个服务实例和可能发生的错误。如果没有找到符合条件的实例，将返回一个空的实例和错误信息。
func (sc *NamingClient) DiscoverOne(options ...DiscoverOptionFunc) (model.Instance, error) {
	opts := &discoverOptions{}
	for _, opt := range options {
		opt(opts)
	}
	instances, err := sc.discover(opts)
	if err != nil {
		return model.Instance{}, err
	}
	instance := opts.Choose(instances...)
	return instance, nil
}

// discover 是Discover和DiscoverOne方法的内部实现。
// opts: 包含所有经过DiscoverOptionFunc处理的配置选项。
// 返回值: 返回过滤后的服务实例列表和可能发生的错误。
func (sc *NamingClient) discover(opts *discoverOptions) ([]model.Instance, error) {
	// 从Nacos服务注册中心选择符合条件的实例
	instances, err := sc.SelectInstances(vo.SelectInstancesParam{
		GroupName:   opts.group,
		ServiceName: opts.service,
		Clusters:    opts.clusters,
		HealthyOnly: true,
	})
	if err != nil {
		return nil, err
	}

	// 筛选出满足Metadata条件的实例
	newInstances := []model.Instance{}
	for _, instance := range instances {
		if !opts.CheckMeta(instance.Metadata) {
			continue
		}
		newInstances = append(newInstances, instance)
	}
	return newInstances, nil
}
