/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config_client

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/nacos-group/nacos-sdk-go/v2/common/monitor"

	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type ConfigProxy struct {
	nacosServer  *nacos_server.NacosServer
	clientConfig constant.ClientConfig
}

func NewConfigProxy(serverConfig []constant.ServerConfig, clientConfig constant.ClientConfig, httpAgent http_agent.IHttpAgent) (IConfigProxy, error) {
	proxy := ConfigProxy{}
	var err error
	proxy.nacosServer, err = nacos_server.NewNacosServer(serverConfig, clientConfig, httpAgent, clientConfig.TimeoutMs, clientConfig.Endpoint)
	proxy.clientConfig = clientConfig
	return &proxy, err
}

func (cp *ConfigProxy) requestProxy(rpcClient *rpc.RpcClient, request rpc_request.IRequest, timeoutMills uint64) (rpc_response.IResponse, error) {
	start := time.Now()
	cp.nacosServer.InjectSecurityInfo(request.GetHeaders())
	cp.injectCommHeader(request.GetHeaders())
	cp.nacosServer.InjectSkAk(request.GetHeaders(), cp.clientConfig)
	signHeaders := nacos_server.GetSignHeadersFromRequest(request.(rpc_request.IConfigRequest), cp.clientConfig.SecretKey)
	request.PutAllHeaders(signHeaders)
	//todo Config Limiter
	response, err := rpcClient.Request(request, int64(timeoutMills))
	monitor.GetConfigRequestMonitor(constant.GRPC, request.GetRequestType(), rpc_response.GetGrpcResponseStatusCode(response)).Observe(float64(time.Now().Nanosecond() - start.Nanosecond()))
	return response, err
}

func (cp *ConfigProxy) injectCommHeader(param map[string]string) {
	now := strconv.FormatInt(util.CurrentMillis(), 10)
	param[constant.CLIENT_APPNAME_HEADER] = cp.clientConfig.AppName
	param[constant.CLIENT_REQUEST_TS_HEADER] = now
	param[constant.CLIENT_REQUEST_TOKEN_HEADER] = util.Md5(now + cp.clientConfig.AppKey)
	param[constant.EX_CONFIG_INFO] = "true"
	param[constant.CHARSET_KEY] = "utf-8"
}

func (cp *ConfigProxy) searchConfigProxy(param vo.SearchConfigParm, tenant, accessKey, secretKey string) (*model.ConfigPage, error) {
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

func (cp *ConfigProxy) queryConfig(dataId, group, tenant string, timeout uint64, notify bool, client *ConfigClient) (*rpc_response.ConfigQueryResponse, error) {
	if group == "" {
		group = constant.DEFAULT_GROUP
	}
	configQueryRequest := rpc_request.NewConfigQueryRequest(group, dataId, tenant)
	configQueryRequest.Headers["notify"] = strconv.FormatBool(notify)
	iResponse, err := cp.requestProxy(cp.getRpcClient(client), configQueryRequest, timeout)
	if err != nil {
		return nil, err
	}
	response, ok := iResponse.(*rpc_response.ConfigQueryResponse)
	if !ok {
		return nil, errors.New("ConfigQueryRequest returns type error")
	}
	if response.IsSuccess() {
		//todo LocalConfigInfoProcessor.saveSnapshot
		cacheKey := util.GetConfigCacheKey(dataId, group, tenant)
		cache.WriteConfigToFile(cacheKey, cp.clientConfig.CacheDir, response.Content)
		//todo LocalConfigInfoProcessor.saveEncryptDataKeySnapshot
		if response.ContentType == "" {
			response.ContentType = "text"
		}
		return response, nil
	}

	if response.GetErrorCode() == 300 {
		//todo LocalConfigInfoProcessor.saveSnapshot
		cacheKey := util.GetConfigCacheKey(dataId, group, tenant)
		cache.WriteConfigToFile(cacheKey, cp.clientConfig.CacheDir, "")
		//todo LocalConfigInfoProcessor.saveEncryptDataKeySnapshot
		return response, nil
	}

	if response.GetErrorCode() == 400 {
		logger.Errorf(
			"[config_rpc_client] [sub-server-error] get server config being modified concurrently, dataId=%s, group=%s, "+
				"tenant=%s", dataId, group, tenant)
		return nil, errors.New("data being modified, dataId=" + dataId + ",group=" + group + ",tenant=" + tenant)
	}

	if response.GetErrorCode() > 0 {
		logger.Errorf("[config_rpc_client] [sub-server-error]  dataId=%s, group=%s, tenant=%s, code=%+v", dataId, group,
			tenant, response)
	}
	return response, nil
}

func (cp *ConfigProxy) createRpcClient(taskId string, client *ConfigClient) *rpc.RpcClient {
	labels := map[string]string{
		constant.LABEL_SOURCE: constant.LABEL_SOURCE_SDK,
		constant.LABEL_MODULE: constant.LABEL_MODULE_CONFIG,
		"taskId":              taskId,
	}

	iRpcClient, _ := rpc.CreateClient("config-"+taskId+"-"+client.uid, rpc.GRPC, labels, cp.nacosServer)
	rpcClient := iRpcClient.GetRpcClient()
	if rpcClient.IsInitialized() {
		rpcClient.RegisterServerRequestHandler(func() rpc_request.IRequest {
			// TODO fix the group/dataId empty problem
			return rpc_request.NewConfigChangeNotifyRequest("", "", "")
		}, &ConfigChangeNotifyRequestHandler{client: client})
		rpcClient.Tenant = cp.clientConfig.NamespaceId
		rpcClient.Start()
	}
	return rpcClient
}

func (cp *ConfigProxy) getRpcClient(client *ConfigClient) *rpc.RpcClient {
	return cp.createRpcClient("0", client)
}

type ConfigChangeNotifyRequestHandler struct {
	client *ConfigClient
}

func (c *ConfigChangeNotifyRequestHandler) Name() string {
	return "ConfigChangeNotifyRequestHandler"
}

func (c *ConfigChangeNotifyRequestHandler) RequestReply(request rpc_request.IRequest, rpcClient *rpc.RpcClient) rpc_response.IResponse {
	configChangeNotifyRequest, ok := request.(*rpc_request.ConfigChangeNotifyRequest)
	if ok {
		logger.Infof("%s [server-push] config changed. dataId=%s, group=%s,tenant=%s", rpcClient.Name,
			configChangeNotifyRequest.DataId, configChangeNotifyRequest.Group, configChangeNotifyRequest.Tenant)

		cacheKey := util.GetConfigCacheKey(configChangeNotifyRequest.DataId, configChangeNotifyRequest.Group,
			configChangeNotifyRequest.Tenant)
		data, ok := c.client.cacheMap.Get(cacheKey)
		if !ok {
			return nil
		}
		cData := data.(*cacheData)
		cData.isSyncWithServer = false
		c.client.notifyListenConfig()
		return &rpc_response.NotifySubscriberResponse{
			Response: &rpc_response.Response{ResultCode: constant.RESPONSE_CODE_SUCCESS},
		}
	}
	return nil
}
