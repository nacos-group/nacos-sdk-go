package vo

import "github.com/nacos-group/nacos-sdk-go/common/constant"

type NacosClientParam struct {
	ClientConfig  *constant.ClientConfig  // optional
	ServerConfigs []constant.ServerConfig // optional
}
