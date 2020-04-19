package config_client

import (
	"encoding/json"
	"errors"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/common/util"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type ConfigProxy struct {
	nacosServer  nacos_server.NacosServer
	clientConfig constant.ClientConfig
}

func NewConfigProxy(serverConfig []constant.ServerConfig, clientConfig constant.ClientConfig, httpAgent http_agent.IHttpAgent) (ConfigProxy, error) {
	proxy := ConfigProxy{}
	var err error
	proxy.nacosServer, err = nacos_server.NewNacosServer(serverConfig, clientConfig, httpAgent, clientConfig.TimeoutMs, clientConfig.Endpoint)
	proxy.clientConfig = clientConfig
	return proxy, err

}

func (cp *ConfigProxy) GetServerList() []constant.ServerConfig {
	return cp.nacosServer.GetServerList()
}

func (cp *ConfigProxy) GetConfigProxy(param vo.ConfigParam, tenant, accessKey, secretKey string) (string, error) {
	params := util.TransformObject2Param(param)
	if len(tenant) > 0 {
		params["tenant"] = tenant
	}

	var headers = map[string]string{}
	headers["accessKey"] = accessKey
	headers["secretKey"] = secretKey

	result, err := cp.nacosServer.ReqConfigApi(constant.CONFIG_PATH, params, headers, http.MethodGet, cp.clientConfig.TimeoutMs)
	return result, err
}

func (cp *ConfigProxy) SearchConfigProxy(param vo.SearchConfigParm, tenant, accessKey, secretKey string) (*model.ConfigPage, error) {
	params := util.TransformObject2Param(param)
	if len(tenant) > 0 {
		params["tenant"] = tenant
	}
	if _, ok := params["group"]; !ok {
		params["group"] = ""
	}
	if _, ok := params["dataId"]; !ok {
		params["dataId"] = ""
	}
	var headers = map[string]string{}
	headers["accessKey"] = accessKey
	headers["secretKey"] = secretKey
	result, err := cp.nacosServer.ReqConfigApi(constant.CONFIG_PATH, params, headers, http.MethodGet, cp.clientConfig.TimeoutMs)
	if err != nil {
		return nil, err
	}
	var configPage model.ConfigPage
	err = json.Unmarshal([]byte(result), &configPage)
	if err != nil {
		return nil, err
	}
	return &configPage, nil
}
func (cp *ConfigProxy) PublishConfigProxy(param vo.ConfigParam, tenant, accessKey, secretKey string) (bool, error) {
	params := util.TransformObject2Param(param)
	if len(tenant) > 0 {
		params["tenant"] = tenant
	}

	var headers = map[string]string{}
	headers["accessKey"] = accessKey
	headers["secretKey"] = secretKey
	result, err := cp.nacosServer.ReqConfigApi(constant.CONFIG_PATH, params, headers, http.MethodPost, cp.clientConfig.TimeoutMs)
	if err != nil {
		return false, errors.New("[client.PublishConfig] publish config failed:" + err.Error())
	}
	if strings.ToLower(strings.Trim(result, " ")) == "true" {
		return true, nil
	} else {
		return false, errors.New("[client.PublishConfig] publish config failed:" + string(result))
	}
}

func (cp *ConfigProxy) DeleteConfigProxy(param vo.ConfigParam, tenant, accessKey, secretKey string) (bool, error) {
	params := util.TransformObject2Param(param)
	if len(tenant) > 0 {
		params["tenant"] = tenant
	}
	var headers = map[string]string{}
	headers["accessKey"] = accessKey
	headers["secretKey"] = secretKey
	result, err := cp.nacosServer.ReqConfigApi(constant.CONFIG_PATH, params, headers, http.MethodDelete, cp.clientConfig.TimeoutMs)
	if err != nil {
		return false, errors.New("[client.DeleteConfig] deleted config failed:" + err.Error())
	}
	if strings.ToLower(strings.Trim(result, " ")) == "true" {
		return true, nil
	} else {
		return false, errors.New("[client.DeleteConfig] deleted config failed: " + string(result))
	}
}

func (cp *ConfigProxy) ListenConfig(params map[string]string, tenant, accessKey, secretKey string) (string, error) {
	headers := map[string]string{
		"Content-Type":         "application/x-www-form-urlencoded;charset=utf-8",
		"Long-Pulling-Timeout": strconv.FormatUint(cp.clientConfig.ListenInterval, 10),
	}
	headers["accessKey"] = accessKey
	headers["secretKey"] = secretKey
	log.Printf("[client.ListenConfig] request params:%+v header:%+v \n", params, headers)
	result, err := cp.nacosServer.ReqConfigApi(constant.CONFIG_LISTEN_PATH, params, headers, http.MethodPost, cp.clientConfig.ListenInterval)
	return result, err
}
