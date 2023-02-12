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
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
)

var ClientResponseMapping map[string]func() IResponse

func init() {
	ClientResponseMapping = make(map[string]func() IResponse)
	registerClientResponses()
}

type IResponse interface {
	GetResponseType() string
	SetRequestId(requestId string)
	GetBody() string
	GetErrorCode() int
	IsSuccess() bool
	GetResultCode() int
	GetMessage() string
}

type Response struct {
	ResultCode int    `json:"resultCode"`
	ErrorCode  int    `json:"errorCode"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	RequestId  string `json:"requestId"`
}

func (r *Response) SetRequestId(requestId string) {
	r.RequestId = requestId
}

func (r *Response) GetBody() string {
	return util.ToJsonString(r)
}

func (r *Response) IsSuccess() bool {
	return r.Success
}

func (r *Response) GetErrorCode() int {
	return r.ErrorCode
}

func (r *Response) GetResultCode() int {
	return r.ResultCode
}

func (r *Response) GetMessage() string {
	return r.Message
}

func registerClientResponse(response func() IResponse) {
	responseType := response().GetResponseType()
	if responseType == "" {
		logger.Errorf("Register client response error: responseType is nil")
		return
	}
	ClientResponseMapping[responseType] = response
}

func registerClientResponses() {
	// register InstanceResponse.
	registerClientResponse(func() IResponse {
		return &InstanceResponse{Response: &Response{}}
	})

	// register BatchInstanceResponse.
	registerClientResponse(func() IResponse {
		return &BatchInstanceResponse{Response: &Response{}}
	})

	// register QueryServiceResponse.
	registerClientResponse(func() IResponse {
		return &QueryServiceResponse{Response: &Response{}}
	})

	// register SubscribeServiceResponse.
	registerClientResponse(func() IResponse {
		return &SubscribeServiceResponse{Response: &Response{}}
	})

	// register ServiceListResponse.
	registerClientResponse(func() IResponse {
		return &ServiceListResponse{Response: &Response{}}
	})

	// register NotifySubscriberResponse.
	registerClientResponse(func() IResponse {
		return &NotifySubscriberResponse{Response: &Response{}}
	})

	// register HealthCheckResponse.
	registerClientResponse(func() IResponse {
		return &HealthCheckResponse{Response: &Response{}}
	})

	// register ErrorResponse.
	registerClientResponse(func() IResponse {
		return &ErrorResponse{Response: &Response{}}
	})

	//register ConfigChangeBatchListenResponse
	registerClientResponse(func() IResponse {
		return &ConfigChangeBatchListenResponse{Response: &Response{}}
	})

	//register ConfigQueryResponse
	registerClientResponse(func() IResponse {
		return &ConfigQueryResponse{Response: &Response{}}
	})

	//register ConfigPublishResponse
	registerClientResponse(func() IResponse {
		return &ConfigPublishResponse{Response: &Response{}}
	})

	//register ConfigRemoveResponse
	registerClientResponse(func() IResponse {
		return &ConfigRemoveResponse{Response: &Response{}}
	})
}

// get grpc response status code with NA default.
func GetGrpcResponseStatusCode(response IResponse) string {
	if response != nil {
		return strconv.Itoa(response.GetResultCode())
	} else {
		return "NA"
	}
}
