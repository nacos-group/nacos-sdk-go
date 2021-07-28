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

package naming_client

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/buger/jsonparser"

	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/common/logger"
	"github.com/nacos-group/nacos-sdk-go/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/util"
)

type NamingProxy struct {
	clientConfig constant.ClientConfig
	nacosServer  *nacos_server.NacosServer
}

func NewNamingProxy(clientCfg constant.ClientConfig, serverCfgs []constant.ServerConfig, httpAgent http_agent.IHttpAgent) (NamingProxy, error) {
	srvProxy := NamingProxy{}
	srvProxy.clientConfig = clientCfg

	var err error
	srvProxy.nacosServer, err = nacos_server.NewNacosServer(serverCfgs, clientCfg, httpAgent, clientCfg.TimeoutMs, clientCfg.Endpoint)
	if err != nil {
		return srvProxy, err
	}

	return srvProxy, nil
}

func (proxy *NamingProxy) RegisterInstance(serviceName string, groupName string, instance model.Instance) (string, error) {
	logger.Infof("register instance namespaceId:<%s>,serviceName:<%s> with instance:<%s>",
		proxy.clientConfig.NamespaceId, serviceName, util.ToJsonString(instance))
	params := map[string]string{}
	params["namespaceId"] = proxy.clientConfig.NamespaceId
	params["serviceName"] = serviceName
	params["groupName"] = groupName
	params["app"] = proxy.clientConfig.AppName
	params["clusterName"] = instance.ClusterName
	params["ip"] = instance.Ip
	params["port"] = strconv.Itoa(int(instance.Port))
	params["weight"] = strconv.FormatFloat(instance.Weight, 'f', -1, 64)
	params["enable"] = strconv.FormatBool(instance.Enable)
	params["healthy"] = strconv.FormatBool(instance.Healthy)
	params["metadata"] = util.ToJsonString(instance.Metadata)
	params["ephemeral"] = strconv.FormatBool(instance.Ephemeral)
	return proxy.nacosServer.ReqApi(constant.SERVICE_PATH, params, http.MethodPost)
}

func (proxy *NamingProxy) DeregisterInstance(serviceName string, ip string, port uint64, clusterName string, ephemeral bool) (string, error) {
	logger.Infof("deregister instance namespaceId:<%s>,serviceName:<%s> with instance:<%s:%d@%s>",
		proxy.clientConfig.NamespaceId, serviceName, ip, port, clusterName)
	params := map[string]string{}
	params["namespaceId"] = proxy.clientConfig.NamespaceId
	params["serviceName"] = serviceName
	params["clusterName"] = clusterName
	params["ip"] = ip
	params["port"] = strconv.Itoa(int(port))
	params["ephemeral"] = strconv.FormatBool(ephemeral)
	return proxy.nacosServer.ReqApi(constant.SERVICE_PATH, params, http.MethodDelete)
}

func (proxy *NamingProxy) UpdateInstance(serviceName string, ip string, port uint64, clusterName string, ephemeral bool, weight float64, enable bool, metadata map[string]string) (string, error) {
	logger.Infof("modify instance namespaceId:<%s>,serviceName:<%s> with instance:<%s:%d@%s>",
		proxy.clientConfig.NamespaceId, serviceName, ip, port, clusterName)
	params := map[string]string{}
	params["namespaceId"] = proxy.clientConfig.NamespaceId
	params["serviceName"] = serviceName
	params["clusterName"] = clusterName
	params["ip"] = ip
	params["port"] = strconv.Itoa(int(port))
	params["ephemeral"] = strconv.FormatBool(ephemeral)
	params["weight"] = strconv.FormatFloat(weight, 'f', -1, 64)
	params["enable"] = strconv.FormatBool(enable)
	params["metadata"] = util.ToJsonString(metadata)
	return proxy.nacosServer.ReqApi(constant.SERVICE_PATH, params, http.MethodPut)
}

func (proxy *NamingProxy) SendBeat(info model.BeatInfo) (int64, error) {
	logger.Infof("namespaceId:<%s> sending beat to server:<%s>",
		proxy.clientConfig.NamespaceId, util.ToJsonString(info))
	params := map[string]string{}
	params["namespaceId"] = proxy.clientConfig.NamespaceId
	params["serviceName"] = info.ServiceName
	params["beat"] = util.ToJsonString(info)
	api := constant.SERVICE_BASE_PATH + "/instance/beat"
	result, err := proxy.nacosServer.ReqApi(api, params, http.MethodPut)
	if err != nil {
		return 0, err
	}
	if result != "" {
		interVal, err := jsonparser.GetInt([]byte(result), "clientBeatInterval")
		if err != nil {
			return 0, errors.New(fmt.Sprintf("namespaceId:<%s> sending beat to server:<%s> get 'clientBeatInterval' from <%s> error:<%+v>", proxy.clientConfig.NamespaceId, util.ToJsonString(info), result, err))
		} else {
			return interVal, nil
		}
	}
	return 0, nil

}

func (proxy *NamingProxy) GetServiceList(pageNo int, pageSize int, groupName string, selector *model.ExpressionSelector) (*model.ServiceList, error) {
	params := map[string]string{}
	params["namespaceId"] = proxy.clientConfig.NamespaceId
	params["groupName"] = groupName
	params["pageNo"] = strconv.Itoa(pageNo)
	params["pageSize"] = strconv.Itoa(pageSize)

	if selector != nil {
		switch selector.Type {
		case "label":
			params["selector"] = util.ToJsonString(selector)
			break
		default:
			break

		}
	}

	api := constant.SERVICE_BASE_PATH + "/service/list"
	result, err := proxy.nacosServer.ReqApi(api, params, http.MethodGet)
	if err != nil {
		return nil, err
	}
	if result == "" {
		return nil, errors.New("request server return empty")
	}

	serviceList := model.ServiceList{}
	count, err := jsonparser.GetInt([]byte(result), "count")
	if err != nil {
		return nil, errors.New(fmt.Sprintf("namespaceId:<%s> get service list pageNo:<%d> pageSize:<%d> selector:<%s> from <%s> get 'count' from <%s> error:<%+v>", proxy.clientConfig.NamespaceId, pageNo, pageSize, util.ToJsonString(selector), groupName, result, err))
	}
	var doms []string
	_, err = jsonparser.ArrayEach([]byte(result), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		doms = append(doms, string(value))
	}, "doms")
	if err != nil {
		return nil, errors.New(fmt.Sprintf("namespaceId:<%s> get service list pageNo:<%d> pageSize:<%d> selector:<%s> from <%s> get 'doms' from <%s> error:<%+v> ", proxy.clientConfig.NamespaceId, pageNo, pageSize, util.ToJsonString(selector), groupName, result, err))
	}
	serviceList.Count = count
	serviceList.Doms = doms
	return &serviceList, nil
}

func (proxy *NamingProxy) ServerHealthy() bool {
	api := constant.SERVICE_BASE_PATH + "/operator/metrics"
	result, err := proxy.nacosServer.ReqApi(api, map[string]string{}, http.MethodGet)
	if err != nil {
		logger.Errorf("namespaceId:[%s] sending server healthy failed!,result:%s error:%+v", proxy.clientConfig.NamespaceId, result, err)
		return false
	}
	if result != "" {
		status, err := jsonparser.GetString([]byte(result), "status")
		if err != nil {
			logger.Errorf("namespaceId:[%s] sending server healthy failed!,result:%s error:%+v", proxy.clientConfig.NamespaceId, result, err)
		} else {
			return status == "UP"
		}
	}
	return false
}

func (proxy *NamingProxy) QueryList(serviceName string, clusters string, udpPort int, healthyOnly bool) (string, error) {
	param := make(map[string]string)
	param["namespaceId"] = proxy.clientConfig.NamespaceId
	param["serviceName"] = serviceName
	param["app"] = proxy.clientConfig.AppName
	param["clusters"] = clusters
	param["udpPort"] = strconv.Itoa(udpPort)
	param["healthyOnly"] = strconv.FormatBool(healthyOnly)
	param["clientIP"] = util.LocalIP()
	api := constant.SERVICE_PATH + "/list"
	return proxy.nacosServer.ReqApi(api, param, http.MethodGet)
}

func (proxy *NamingProxy) GetAllServiceInfoList(namespace, groupName string, pageNo, pageSize uint32) (string, error) {
	param := make(map[string]string)
	param["namespaceId"] = namespace
	param["groupName"] = groupName
	param["pageNo"] = strconv.Itoa(int(pageNo))
	param["pageSize"] = strconv.Itoa(int(pageSize))
	api := constant.SERVICE_INFO_PATH + "/list"
	return proxy.nacosServer.ReqApi(api, param, http.MethodGet)
}
