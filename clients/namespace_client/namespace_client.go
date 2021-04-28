package namespace_client

import (
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/logger"
	"github.com/nacos-group/nacos-sdk-go/model"
)

type NamespaceClient struct {
	nc           nacos_client.INacosClient
	serviceProxy NamespaceProxy
}

func NewNamespaceClient(nc nacos_client.INacosClient) (NamespaceClient, error) {
	namespace := NamespaceClient{}
	namespace.nc = nc
	clientConfig, err := nc.GetClientConfig()
	if err != nil {
		return namespace, err
	}
	serverConfig, err := nc.GetServerConfig()
	if err != nil {
		return namespace, err
	}
	httpAgent, err := nc.GetHttpAgent()
	if err != nil {
		return namespace, err
	}
	err = logger.InitLogger(logger.Config{
		Level:        clientConfig.LogLevel,
		OutputPath:   clientConfig.LogDir,
		RotationTime: clientConfig.RotateTime,
		MaxAge:       clientConfig.MaxAge,
	})
	if err != nil {
		return namespace, err
	}
	namespace.serviceProxy, err = NewNamespaceProxy(clientConfig, serverConfig, httpAgent)
	if err != nil {
		return namespace, err
	}
	return namespace, nil
}

func (n NamespaceClient) GetAllNamespacesInfo() ([]model.NamespaceItem, error) {
	return n.serviceProxy.GetAllNamespacesInfo()
}
