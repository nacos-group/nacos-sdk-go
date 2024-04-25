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

package rpc_request

import "github.com/nacos-group/nacos-sdk-go/v2/util"

type Request struct {
	Headers   map[string]string `json:"-"`
	RequestId string            `json:"requestId"`
}

type IRequest interface {
	GetHeaders() map[string]string
	GetRequestType() string
	GetBody(request IRequest) string
	PutAllHeaders(headers map[string]string)
	GetRequestId() string
	GetStringToSign() string
}

type IConfigRequest interface {
	GetDataId() string
	GetGroup() string
	GetTenant() string
}

func (r *Request) PutAllHeaders(headers map[string]string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	for k, v := range headers {
		r.Headers[k] = v
	}
}

func (r *Request) ClearHeaders() {
	r.Headers = make(map[string]string)
}

func (r *Request) GetHeaders() map[string]string {
	return r.Headers
}

func (r *Request) GetBody(request IRequest) string {
	return util.ToJsonString(request)
}
func (r *Request) GetRequestId() string {
	return r.RequestId
}

func (r *Request) GetStringToSign() string {
	return ""
}
