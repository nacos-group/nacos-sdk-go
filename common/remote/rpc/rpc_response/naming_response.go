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

package rpc_response

import (
	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

type ConnectResetResponse struct {
	*Response
}

func (c *ConnectResetResponse) GetResponseType() string {
	return "ConnectResetResponse"
}

type ClientDetectionResponse struct {
	*Response
}

func (c *ClientDetectionResponse) GetResponseType() string {
	return "ClientDetectionResponse"
}

type ServerCheckResponse struct {
	*Response
	ConnectionId string `json:"connectionId"`
}

func (c *ServerCheckResponse) GetResponseType() string {
	return "ServerCheckResponse"
}

type InstanceResponse struct {
	*Response
}

func (c *InstanceResponse) GetResponseType() string {
	return "InstanceResponse"
}

type BatchInstanceResponse struct {
	*Response
}

func (c *BatchInstanceResponse) GetResponseType() string {
	return "BatchInstanceResponse"
}

type QueryServiceResponse struct {
	*Response
	ServiceInfo model.Service `json:"serviceInfo"`
}

func (c *QueryServiceResponse) GetResponseType() string {
	return "QueryServiceResponse"
}

type SubscribeServiceResponse struct {
	*Response
	ServiceInfo model.Service `json:"serviceInfo"`
}

func (c *SubscribeServiceResponse) GetResponseType() string {
	return "SubscribeServiceResponse"
}

type ServiceListResponse struct {
	*Response
	Count        int      `json:"count"`
	ServiceNames []string `json:"serviceNames"`
}

func (c *ServiceListResponse) GetResponseType() string {
	return "ServiceListResponse"
}

type NotifySubscriberResponse struct {
	*Response
}

func (c *NotifySubscriberResponse) GetResponseType() string {
	return "NotifySubscriberResponse"
}

type HealthCheckResponse struct {
	*Response
}

func (c *HealthCheckResponse) GetResponseType() string {
	return "HealthCheckResponse"
}

type ErrorResponse struct {
	*Response
}

func (c *ErrorResponse) GetResponseType() string {
	return "ErrorResponse"
}

type MockResponse struct {
	*Response
}

func (c *MockResponse) GetResponseType() string {
	return "MockResponse"
}
