package config_client

import (
	"errors"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/common/util"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"net/http"
	"strings"
)

type ConfigProxy struct {
	nacosServer nacos_server.NacosServer
}

func NewConfigProxy(serverConfig []constant.ServerConfig, clientConfig constant.ClientConfig, httpAgent http_agent.IHttpAgent) (ConfigProxy, error) {
	proxy := ConfigProxy{}
	var err error
	proxy.nacosServer, err = nacos_server.NewNacosServer(serverConfig, httpAgent, clientConfig.TimeoutMs, clientConfig.Endpoint)
	return proxy, err

}

func (cp *ConfigProxy) GetConfigProxy(param vo.ConfigParam, tenant string) (string, error) {
	params := util.TransformObject2Param(param)
	if len(tenant) > 0 {
		params["tenant"] = tenant
	}
	result, err := cp.nacosServer.ReqApi(constant.CONFIG_PATH, params, http.MethodGet)
	return result, err
}

func (cp *ConfigProxy) PublishConfigProxy(param vo.ConfigParam, tenant string) (bool, error) {
	params := util.TransformObject2Param(param)
	if len(tenant) > 0 {
		params["tenant"] = tenant
	}
	result, err := cp.nacosServer.ReqApi(constant.CONFIG_PATH, params, http.MethodPost)
	if err != nil {
		return false, errors.New("[client.PublishConfig] publish config failed:" + err.Error())
	}
	if strings.ToLower(strings.Trim(result, " ")) == "true" {
		return true, nil
	} else {
		return false, errors.New("[client.PublishConfig] publish config failed:" + string(result))
	}
}

func (cp *ConfigProxy) DeleteConfigProxy(param vo.ConfigParam, tenant string) (bool, error) {
	params := util.TransformObject2Param(param)
	if len(tenant) > 0 {
		params["tenant"] = tenant
	}
	result, err := cp.nacosServer.ReqApi(constant.CONFIG_PATH, params, http.MethodDelete)
	if err != nil {
		return false, errors.New("[client.DeleteConfig] deleted config failed:" + err.Error())
	}
	if strings.ToLower(strings.Trim(result, " ")) == "true" {
		return true, nil
	} else {
		return false, errors.New("[client.DeleteConfig] deleted config failed: " + string(result))
	}
}
