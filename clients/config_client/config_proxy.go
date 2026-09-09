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
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/common/monitor"
	"github.com/nacos-group/nacos-sdk-go/v2/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v2/common/security"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/pkg/errors"
)

type ConfigProxy struct {
	nacosServer  *nacos_server.NacosServer
	clientConfig constant.ClientConfig
}

func NewConfigProxy(ctx context.Context, serverConfig []constant.ServerConfig, clientConfig constant.ClientConfig, httpAgent http_agent.IHttpAgent) (IConfigProxy, error) {
	return NewConfigProxyWithRamCredentialProvider(ctx, serverConfig, clientConfig, httpAgent, nil)
}

func NewConfigProxyWithRamCredentialProvider(ctx context.Context, serverConfig []constant.ServerConfig, clientConfig constant.ClientConfig, httpAgent http_agent.IHttpAgent, provider security.RamCredentialProvider) (IConfigProxy, error) {
	proxy := ConfigProxy{}
	var err error
	proxy.nacosServer, err = nacos_server.NewNacosServerWithRamCredentialProvider(ctx, serverConfig, clientConfig, httpAgent, clientConfig.TimeoutMs, clientConfig.Endpoint, nil, provider)
	proxy.clientConfig = clientConfig
	return &proxy, err
}

func (cp *ConfigProxy) requestProxy(rpcClient *rpc.RpcClient, request rpc_request.IRequest, timeoutMills uint64) (rpc_response.IResponse, error) {
	start := time.Now()
	cp.nacosServer.InjectSecurityInfo(request.GetHeaders(), security.BuildConfigResourceByRequest(request))
	cp.injectCommHeader(request.GetHeaders())
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

func (cp *ConfigProxy) searchConfigProxy(param vo.SearchConfigParam, tenant, accessKey, secretKey string) (*model.ConfigPage, error) {
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
	var version = "v2"
	result, err := cp.nacosServer.ReqConfigApi(constant.CONFIG_PATH, params, headers, http.MethodGet, cp.clientConfig.TimeoutMs)
	if err != nil {
		if len(tenant) > 0 {
			params["namespaceId"] = params["tenant"]
		}
		params["groupName"] = params["group"]
		result, err = cp.nacosServer.ReqConfigApi("/v3/admin/cs/config/list", params, headers, http.MethodGet, cp.clientConfig.TimeoutMs)
		if err != nil {
			return nil, err
		}
		version = "v3"
	}
	var configPage model.ConfigPage
	if version == "v2" {
		err = json.Unmarshal([]byte(result), &configPage)
	} else {
		configPage, err = parseV3ConfigListResult([]byte(result))
	}
	if err != nil {
		return nil, err
	}
	return &configPage, nil
}

// v3AdminConfigItem mirrors a page item of the v3 admin config list API, whose
// response model uses groupName/namespaceId instead of group/tenant and carries
// no content field.
type v3AdminConfigItem struct {
	Id          json.Number `json:"id"`
	DataId      string      `json:"dataId"`
	GroupName   string      `json:"groupName"`
	NamespaceId string      `json:"namespaceId"`
	Md5         string      `json:"md5"`
	AppName     string      `json:"appName"`
}

type v3AdminConfigPage struct {
	TotalCount     int                 `json:"totalCount"`
	PageNumber     int                 `json:"pageNumber"`
	PagesAvailable int                 `json:"pagesAvailable"`
	PageItems      []v3AdminConfigItem `json:"pageItems"`
}

type v3AdminConfigPageResult struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Data    v3AdminConfigPage `json:"data"`
}

// parseV3ConfigListResult parses a v3 admin config list response and normalizes
// the v3 field names (groupName/namespaceId) into the v1-style ConfigItem, so
// SearchConfig callers observe consistent fields no matter which server API
// version answered the request. The v3 list API does not return content by
// design; callers needing the content should query each config individually.
func parseV3ConfigListResult(body []byte) (model.ConfigPage, error) {
	var result v3AdminConfigPageResult
	if err := json.Unmarshal(body, &result); err != nil {
		return model.ConfigPage{}, err
	}
	page := model.ConfigPage{
		TotalCount:     result.Data.TotalCount,
		PageNumber:     result.Data.PageNumber,
		PagesAvailable: result.Data.PagesAvailable,
		PageItems:      make([]model.ConfigItem, 0, len(result.Data.PageItems)),
	}
	for _, item := range result.Data.PageItems {
		page.PageItems = append(page.PageItems, model.ConfigItem{
			Id:      item.Id,
			DataId:  item.DataId,
			Group:   item.GroupName,
			Tenant:  item.NamespaceId,
			Md5:     item.Md5,
			Appname: item.AppName,
		})
	}
	return page, nil
}

func (cp *ConfigProxy) queryConfig(dataId, group, tenant string, timeout uint64, notify bool, client *ConfigClient) (*rpc_response.ConfigQueryResponse, error) {
	if group == "" {
		group = constant.DEFAULT_GROUP
	}
	configQueryRequest := rpc_request.NewConfigQueryRequest(group, dataId, tenant)
	configQueryRequest.Headers["notify"] = strconv.FormatBool(notify)
	cacheKey := util.GetConfigCacheKey(dataId, group, tenant)
	// use the same key of config file as the limit checker's key
	if IsLimited(cacheKey) {
		// return error when check limited
		return nil, errors.New("ConfigQueryRequest is limited")
	}
	iResponse, err := cp.requestProxy(cp.getRpcClient(client), configQueryRequest, timeout)
	if err != nil {
		return nil, err
	}
	response, ok := iResponse.(*rpc_response.ConfigQueryResponse)
	if !ok {
		return nil, errors.New("ConfigQueryRequest returns type error")
	}
	if response.IsSuccess() {
		cache.WriteConfigToFile(cacheKey, cp.clientConfig.CacheDir, response.Content)
		cache.WriteEncryptedDataKeyToFile(cacheKey, cp.clientConfig.CacheDir, response.EncryptedDataKey)
		if response.ContentType == "" {
			response.ContentType = "text"
		}
		return response, nil
	}

	if response.GetErrorCode() == 300 {
		cache.WriteConfigToFile(cacheKey, cp.clientConfig.CacheDir, "")
		cache.WriteEncryptedDataKeyToFile(cacheKey, cp.clientConfig.CacheDir, "")
		response.SetSuccess(true)
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

func appName(client *ConfigClient) string {
	if clientConfig, err := client.GetClientConfig(); err == nil {
		appName := clientConfig.AppName
		return appName
	}
	return "unknown"
}

func (cp *ConfigProxy) createRpcClient(ctx context.Context, taskId string, client *ConfigClient) *rpc.RpcClient {
	labels := map[string]string{
		constant.LABEL_SOURCE:   constant.LABEL_SOURCE_SDK,
		constant.LABEL_MODULE:   constant.LABEL_MODULE_CONFIG,
		constant.APPNAME_HEADER: appName(client),
		"taskId":                taskId,
	}

	iRpcClient, _ := rpc.CreateClient(ctx, "config-"+taskId+"-"+client.uid, rpc.GRPC, labels, cp.nacosServer, &cp.clientConfig.TLSCfg, cp.clientConfig.AppConnLabels)
	rpcClient := iRpcClient.GetRpcClient()
	if rpcClient.IsInitialized() {
		rpcClient.RegisterServerRequestHandler(func() rpc_request.IRequest {
			// TODO fix the group/dataId empty problem
			return rpc_request.NewConfigChangeNotifyRequest("", "", "")
		}, &ConfigChangeNotifyRequestHandler{client: client})

		configListener := NewConfigConnectionEventListener(client, taskId)
		rpcClient.RegisterConnectionListener(configListener)

		rpcClient.Tenant = cp.clientConfig.NamespaceId
		rpcClient.Start()
	}
	return rpcClient
}

func (cp *ConfigProxy) getRpcClient(client *ConfigClient) *rpc.RpcClient {
	return cp.createRpcClient(client.ctx, "0", client)
}

type ConfigChangeNotifyRequestHandler struct {
	client *ConfigClient
}

func (c *ConfigChangeNotifyRequestHandler) Name() string {
	return "ConfigChangeNotifyRequestHandler"
}

func (c *ConfigChangeNotifyRequestHandler) RequestReply(request rpc_request.IRequest, rpcClient *rpc.RpcClient) rpc_response.IResponse {
	configChangeNotifyRequest, ok := request.(*rpc_request.ConfigChangeNotifyRequest)
	if !ok {
		return nil
	}
	logger.Infof("%s [server-push] config changed. dataId=%s, group=%s,tenant=%s", rpcClient.Name(),
		configChangeNotifyRequest.DataId, configChangeNotifyRequest.Group, configChangeNotifyRequest.Tenant)

	cacheKey := util.GetConfigCacheKey(configChangeNotifyRequest.DataId, configChangeNotifyRequest.Group,
		configChangeNotifyRequest.Tenant)
	data, ok := c.client.cacheMap.Get(cacheKey)
	if !ok {
		return nil
	}
	cData := data.(cacheData)
	cData.isSyncWithServer = false
	c.client.cacheMap.Set(cacheKey, cData)
	c.client.asyncNotifyListenConfig()
	return &rpc_response.NotifySubscriberResponse{
		Response: &rpc_response.Response{ResultCode: constant.RESPONSE_CODE_SUCCESS},
	}
}
