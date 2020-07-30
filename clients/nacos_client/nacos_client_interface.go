package nacos_client

import (
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
)

//go:generate mockgen -destination mock_nacos_client_interface.go -package nacos_client -source=./nacos_client_interface.go

type INacosClient interface {

	//SetClientConfig is use to set nacos client Config
	SetClientConfig(constant.ClientConfig) error
	//SetServerConfig is use to set nacos server config
	SetServerConfig([]constant.ServerConfig) error
	//GetClientConfig use to get client config
	GetClientConfig() (constant.ClientConfig, error)
	//GetServerConfig use to get server config
	GetServerConfig() ([]constant.ServerConfig, error)
	//SetHttpAgent use to set http agent
	SetHttpAgent(http_agent.IHttpAgent) error
	//GetHttpAgent use to get http agent
	GetHttpAgent() (http_agent.IHttpAgent, error)
}
