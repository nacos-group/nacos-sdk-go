package namespace_client

import "github.com/nacos-group/nacos-sdk-go/model"

type INamespaceClient interface {
	GetAllNamespacesInfo() ([]model.NamespaceItem, error)
}
