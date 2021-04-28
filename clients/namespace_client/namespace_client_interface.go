package namespace_client

import (
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

type INamespaceClient interface {
	GetAllNamespacesInfo() ([]model.NamespaceItem, error)
	CreateNamespace(param vo.CreateNamespaceParam) (bool, error)
	ModifyNamespace(param vo.ModifyNamespaceParam) (bool, error)
	DeleteNamespace(param vo.DeleteNamespaceParam) (bool, error)
}
