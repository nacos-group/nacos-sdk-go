package namespace_client

import (
	"encoding/json"
	"github.com/buger/jsonparser"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/model"
	"net/http"
)

type NamespaceProxy struct {
	nacosServer *nacos_server.NacosServer
}

func NewNamespaceProxy(clientCfg constant.ClientConfig, serverCfgs []constant.ServerConfig, httpAgent http_agent.IHttpAgent) (NamespaceProxy, error) {
	srvProxy := NamespaceProxy{}
	var err error
	srvProxy.nacosServer, err = nacos_server.NewNacosServer(serverCfgs, clientCfg, httpAgent, clientCfg.TimeoutMs, clientCfg.Endpoint)
	if err != nil {
		return srvProxy, err
	}
	return srvProxy, nil
}

func (proxy *NamespaceProxy) GetAllNamespacesInfo() ([]model.NamespaceItem, error) {
	params := map[string]string{}
	result, err := proxy.nacosServer.ReqApi(constant.NAMESPACE_PATH, params, http.MethodGet)
	if err != nil {
		return nil, err
	}
	var namespaces []model.NamespaceItem
	_, err = jsonparser.ArrayEach([]byte(result), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		var namespaceItem model.NamespaceItem
		jserr := json.Unmarshal(value, &namespaceItem)
		if jserr == nil {
			namespaces = append(namespaces, namespaceItem)
		}

	}, "data")

	return namespaces, err
}
