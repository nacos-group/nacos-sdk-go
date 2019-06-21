package naming_client

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
* @create : 2019-01-09 09:56
**/

//go:generate mockgen -destination ../../mock/mock_service_client_interface.go -package mock -source=./service_client_interface.go

type INamingClient interface {
	// 注册服务实例
	RegisterServiceInstance(param vo.RegisterServiceInstanceParam) (bool, error)
	// 注销服务实例
	LogoutServiceInstance(param vo.LogoutServiceInstanceParam) (bool, error)
	// 获取服务列表
	GetService(param vo.GetServiceParam) (model.Service, error)
	// 获取服务某个实例
	GetServiceInstance(param vo.GetServiceInstanceParam) (model.ServiceInstance, error)
	// 获取service的基本信息
	GetServiceDetail(param vo.GetServiceDetailParam) (model.ServiceDetail, error)
	// 服务监听
	Subscribe(param *vo.SubscribeParam) error
	//取消监听
	Unsubscribe(param *vo.SubscribeParam) error
}
