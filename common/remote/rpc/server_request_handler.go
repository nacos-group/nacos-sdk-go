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

package rpc

import (
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
)

// IServerRequestHandler to process the request from server side.
type IServerRequestHandler interface {
	Name() string
	//RequestReply Handle request from server.
	RequestReply(request rpc_request.IRequest, rpcClient *RpcClient) rpc_response.IResponse
}

type ConnectResetRequestHandler struct {
}

func (c *ConnectResetRequestHandler) Name() string {
	return "ConnectResetRequestHandler"
}

func (c *ConnectResetRequestHandler) RequestReply(request rpc_request.IRequest, rpcClient *RpcClient) rpc_response.IResponse {
	connectResetRequest, ok := request.(*rpc_request.ConnectResetRequest)
	if ok {
		rpcClient.mux.Lock()
		defer rpcClient.mux.Unlock()
		if rpcClient.IsRunning() {
			if connectResetRequest.ServerIp != "" {
				serverPortNum, err := strconv.Atoi(connectResetRequest.ServerPort)
				if err != nil {
					logger.Errorf("ConnectResetRequest ServerPort type conversion error:%+v", err)
					return nil
				}
				rpcClient.switchServerAsync(ServerInfo{serverIp: connectResetRequest.ServerIp, serverPort: uint64(serverPortNum)}, false)
			} else {
				rpcClient.switchServerAsync(ServerInfo{}, true)
			}
		}
		return &rpc_response.ConnectResetResponse{Response: &rpc_response.Response{ResultCode: constant.RESPONSE_CODE_SUCCESS}}
	}
	return nil
}

type ClientDetectionRequestHandler struct {
}

func (c *ClientDetectionRequestHandler) Name() string {
	return "ClientDetectionRequestHandler"
}

func (c *ClientDetectionRequestHandler) RequestReply(request rpc_request.IRequest, _ *RpcClient) rpc_response.IResponse {
	_, ok := request.(*rpc_request.ClientDetectionRequest)
	if ok {
		return &rpc_response.ClientDetectionResponse{
			Response: &rpc_response.Response{ResultCode: constant.RESPONSE_CODE_SUCCESS},
		}
	}
	return nil
}

type NamingPushRequestHandler struct {
	ServiceInfoHolder *naming_cache.ServiceInfoHolder
}

func (*NamingPushRequestHandler) Name() string {
	return "NamingPushRequestHandler"
}

func (c *NamingPushRequestHandler) RequestReply(request rpc_request.IRequest, _ *RpcClient) rpc_response.IResponse {
	notifySubscriberRequest, ok := request.(*rpc_request.NotifySubscriberRequest)
	if ok {
		c.ServiceInfoHolder.ProcessService(&notifySubscriberRequest.ServiceInfo)
		return &rpc_response.NotifySubscriberResponse{
			Response: &rpc_response.Response{ResultCode: constant.RESPONSE_CODE_SUCCESS, Success: true},
		}
	}
	return nil
}
