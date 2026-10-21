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
	"github.com/nacos-group/nacos-sdk-proto/go/config"

	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v3/model"
)

func adaptConfigQueryResponse(m *config.ConfigQueryResponse, body []byte) *rpc_response.ConfigQueryResponse {
	return &rpc_response.ConfigQueryResponse{
		Response:         adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
		Content:          m.Content,
		EncryptedDataKey: m.EncryptedDataKey,
		ContentType:      m.ContentType,
		Md5:              m.Md5,
		LastModified:     m.LastModified,
		IsBeta:           m.IsBeta,
		// legacy Tag field is bool type dead field (no consumer), proto is string, not mapped; see PR body
	}
}

func adaptConfigPublishResponse(m *config.ConfigPublishResponse, body []byte) *rpc_response.ConfigPublishResponse {
	return &rpc_response.ConfigPublishResponse{
		Response: adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
	}
}

func adaptConfigRemoveResponse(m *config.ConfigRemoveResponse, body []byte) *rpc_response.ConfigRemoveResponse {
	return &rpc_response.ConfigRemoveResponse{
		Response: adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
	}
}

func adaptConfigBatchListenResponse(m *config.ConfigChangeBatchListenResponse, body []byte) *rpc_response.ConfigChangeBatchListenResponse {
	changed := make([]model.ConfigContext, 0, len(m.ChangedConfigs))
	for _, c := range m.ChangedConfigs {
		changed = append(changed, model.ConfigContext{Group: c.Group, DataId: c.DataId, Tenant: c.Tenant})
	}
	return &rpc_response.ConfigChangeBatchListenResponse{
		Response:       adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
		ChangedConfigs: changed,
	}
}

func adaptConfigChangeNotifyRequest(m *config.ConfigChangeNotifyRequest) *rpc_request.ConfigChangeNotifyRequest {
	req := &rpc_request.ConfigChangeNotifyRequest{ConfigRequest: rpc_request.NewConfigRequest(m.Group, m.DataId, m.Tenant)}
	req.RequestId = m.RequestId
	return req
}
