package vo

import "github.com/fanghongbo/nacos-sdk-go/common/constant"

type NacosClientParam struct {
	ClientConfig  *constant.ClientConfig  // optional
	ServerConfigs []constant.ServerConfig // optional
}
