package config_client

import (
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

/**
*
* @description :
*
* @author : codezhang
*
* @create : 2019-01-08 10:03
**/

//go:generate mockgen -destination ../../mock/mock_config_client_interface.go -package mock -source=./config_client_interface.go

type IConfigClient interface {
	// 获取配置
	// dataId  require
	// group   require
	// tenant ==>nacos.namespace optional
	GetConfig(param vo.ConfigParam) (string, error)

	// 发布配置
	// dataId  require
	// group   require
	// content require
	// tenant ==>nacos.namespace optional
	PublishConfig(param vo.ConfigParam) (bool, error)

	// 删除配置
	// dataId  require
	// group   require
	// tenant ==>nacos.namespace optional
	DeleteConfig(param vo.ConfigParam) (bool, error)

	// 监听配置
	// dataId  require
	// group   require
	// tenant ==>nacos.namespace optional
	ListenConfig(params vo.ConfigParam) (err error)

	// 搜索配置
	// search  require search=accurate--精确搜索  search=blur--模糊搜索
	// group   option
	// dataId  option
	// tenant ==>nacos.namespace optional
	// pageNo  option
	// pageSize option
	SearchConfig(param vo.SearchConfigParm) (*model.ConfigPage, error)
}
