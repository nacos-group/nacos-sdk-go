package naming_client

import "github.com/nacos-group/nacos-sdk-go/v2/model"

// 定义DiscoverOptionFunc类型，它是一个函数类型，接收一个discoverOptions指针并对其进行修改
type DiscoverOptionFunc func(*discoverOptions)

// WithGroup 返回一个DiscoverOptionFunc，用来设置发现选项中的group字段
// 参数group: 要设置的组名
func WithGroup(group string) DiscoverOptionFunc {
	return func(o *discoverOptions) {
		o.group = group
	}
}

// WithService 返回一个DiscoverOptionFunc，用来设置发现选项中的service字段
// 参数service: 要设置的服务名
func WithService(service string) DiscoverOptionFunc {
	return func(o *discoverOptions) {
		o.service = service
	}
}

// WithVersion 返回一个DiscoverOptionFunc，用来添加一个版本到发现选项的versions列表中
// 参数version: 要添加的版本号
func WithVersion(version string) DiscoverOptionFunc {
	return func(o *discoverOptions) {
		if o.versions == nil {
			o.versions = []string{}
		}
		o.versions = append(o.versions, version)
	}
}

// WithCluster 返回一个DiscoverOptionFunc，用来添加一个集群名到发现选项的clusters列表中
// 参数cluster: 要添加的集群名
func WithCluster(cluster string) DiscoverOptionFunc {
	return func(o *discoverOptions) {
		if o.clusters == nil {
			o.clusters = []string{}
		}
		o.clusters = append(o.clusters, cluster)
	}
}

// WithChoose 是一个用于设置发现选项中选择函数的 DiscoverOptionFunc 构造函数。
// 参数: choose - 一个实现了 IChooseFunc 接口的选择函数，用于在发现过程中进行选择逻辑的定制。
// 返回值: 返回一个 DiscoverOptionFunc，它是一个函数类型，接受 discoverOptions 指针作为参数，用于配置发现选项。
func WithChoose(choose IChooseFunc) DiscoverOptionFunc {
	return func(o *discoverOptions) {
		o.choose = choose
	}
}

// discoverOptions 定义了发现选项的结构体，包括组名、服务名、集群名和版本号列表
type discoverOptions struct {
	group    string
	service  string
	clusters []string
	versions []string
	choose   IChooseFunc
}

// CheckMeta 用于检查给定的元数据中的版本信息是否符合预期。
// 参数meta: 一个包含键值对的映射，预期包含一个名为"version"的键。
// 返回值: 返回一个布尔值，表示元数据中的版本是否通过了检查。
func (o *discoverOptions) CheckMeta(meta map[string]string) bool {
	// 根据传入的元数据中的版本进行检查
	return o.VersionCheck(meta["version"])
}

// VersionCheck 检查给定的版本是否在版本列表中
// 参数ver: 要检查的版本号
// 返回值: 如果给定的版本存在于列表中，则返回true；否则返回false
func (o discoverOptions) VersionCheck(ver string) bool {
	if len(o.versions) == 0 {
		return true
	}
	if ver == "" {
		return false
	}
	for _, v := range o.versions {
		if v == ver {
			return true
		}
	}
	return false
}

// Choose 方法根据给定的实例列表选择一个实例。
// 参数: instances ...model.Instance - 一个或多个 model.Instance 类型的实例。
// 返回值: model.Instance - 选择出的单个实例。
func (o discoverOptions) Choose(instances ...model.Instance) model.Instance {
	if o.choose == nil {
		return defaultChoose(instances...)
	}
	return o.choose(instances...)
}

// IChooseFunc 是一个函数类型，用于从多个model.Instance中选择一个。
// 参数instances是可变长度的model.Instance类型切片。
// 返回值是一个model.Instance类型，表示选择的结果。
type IChooseFunc func(instances ...model.Instance) model.Instance

// defaultChoose 是一个默认的选择函数，基于提供的model.Instance实例列表进行选择。
// 参数is是可变长度的model.Instance类型切片。
// 返回值是一个model.Instance类型，表示选择的结果。
func defaultChoose(is ...model.Instance) model.Instance {
	return newChooser(is).pick()
}
