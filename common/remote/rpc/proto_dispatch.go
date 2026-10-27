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
	"encoding/json"

	nacos_grpc_service "github.com/nacos-group/nacos-sdk-proto/go"
	"github.com/nacos-group/nacos-sdk-proto/go/common"
	"github.com/nacos-group/nacos-sdk-proto/go/config"
	"github.com/nacos-group/nacos-sdk-proto/go/naming"
	"github.com/pkg/errors"

	"github.com/nacos-group/nacos-sdk-go/v3/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_response"
)

// decodeProtoResponse decodes responses whose types are migrated to
// sdk-proto messages and adapts them back into the legacy rpc_response
// structs, so downstream type assertions keep working. The bool result
// reports whether the type is migrated; unmigrated types must fall
// through to ClientResponseMapping. The dispatch is intentionally NOT
// keyed off the codec registry: the registry knows all 112 types while
// only the types below are migrated in this PR. Unlike
// decodeProtoServerRequest, a migrated type that fails to decode returns
// the error instead of falling back to legacy JSON: the caller is a
// synchronous unary RPC waiting on this exact response, so silently
// degrading would surface as a confusing downstream failure rather than
// the real decode error, and fail-fast lets the caller propagate a clear
// cause.
func decodeProtoResponse(payload *nacos_grpc_service.Payload) (rpc_response.IResponse, bool, error) {
	switch payload.GetMetadata().GetType() {
	case "HealthCheckResponse", "ServerCheckResponse", "ErrorResponse",
		"InstanceResponse", "BatchInstanceResponse", "QueryServiceResponse",
		"SubscribeServiceResponse", "ServiceListResponse", "NamingFuzzyWatchResponse",
		"ConfigQueryResponse", "ConfigPublishResponse", "ConfigRemoveResponse", "ConfigChangeBatchListenResponse":
	default:
		return nil, false, nil
	}
	msg, err := payloadCodec.Decode(payload)
	if err != nil {
		return nil, true, err
	}
	body := payload.GetBody().GetValue()
	switch m := msg.(type) {
	case *common.HealthCheckResponse:
		return &rpc_response.HealthCheckResponse{
			Response: adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
		}, true, nil
	case *common.ServerCheckResponse:
		return &rpc_response.ServerCheckResponse{
			Response:     adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
			ConnectionId: m.ConnectionId,
		}, true, nil
	case *common.ErrorResponse:
		return &rpc_response.ErrorResponse{
			Response: adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
		}, true, nil
	case *naming.InstanceResponse:
		return &rpc_response.InstanceResponse{Response: adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body)}, true, nil
	case *naming.BatchInstanceResponse:
		return &rpc_response.BatchInstanceResponse{Response: adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body)}, true, nil
	case *naming.QueryServiceResponse:
		return &rpc_response.QueryServiceResponse{
			Response:    adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
			ServiceInfo: fromProtoServiceInfo(m.ServiceInfo),
		}, true, nil
	case *naming.SubscribeServiceResponse:
		return &rpc_response.SubscribeServiceResponse{
			Response:    adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
			ServiceInfo: fromProtoServiceInfo(m.ServiceInfo),
		}, true, nil
	case *naming.ServiceListResponse:
		return &rpc_response.ServiceListResponse{
			Response:     adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
			Count:        int(m.Count),
			ServiceNames: m.ServiceNames,
		}, true, nil
	case *naming.NamingFuzzyWatchResponse:
		return &rpc_response.NamingFuzzyWatchResponse{
			Response: adaptBaseResponse(m.ResultCode, m.ErrorCode, m.Message, m.RequestId, body),
		}, true, nil
	case *config.ConfigQueryResponse:
		return adaptConfigQueryResponse(m, body), true, nil
	case *config.ConfigPublishResponse:
		return adaptConfigPublishResponse(m, body), true, nil
	case *config.ConfigRemoveResponse:
		return adaptConfigRemoveResponse(m, body), true, nil
	case *config.ConfigChangeBatchListenResponse:
		return adaptConfigBatchListenResponse(m, body), true, nil
	}
	return nil, true, errors.Errorf("no proto adapter for migrated type %s", payload.GetMetadata().GetType())
}

// adaptBaseResponse mirrors InnerResponseJsonUnmarshal's success rule:
// honor an explicit wire "success" field, otherwise derive it from
// resultCode (the proto definitions have no success field because it is
// a Java getter-derived property).
func adaptBaseResponse(resultCode, errorCode int32, message, requestId string, body []byte) *rpc_response.Response {
	resp := &rpc_response.Response{
		ResultCode: int(resultCode),
		ErrorCode:  int(errorCode),
		Message:    message,
		RequestId:  requestId,
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err == nil {
		// A non-boolean wire "success" (never produced by real servers) falls
		// through to resultCode derivation; the legacy path would hard-error.
		if s, ok := raw[rpc_response.ResponseSuccessField].(bool); ok {
			resp.Success = s
			return resp
		}
	}
	resp.Success = resp.ResultCode == int(rpc_response.ResponseSuccessCode)
	return resp
}

// decodeProtoServerRequest decodes migrated server-push requests into the
// legacy rpc_request structs. Unlike decodeProtoResponse it falls back to
// the legacy JSON path on decode failure: dropping a push (ConnectReset /
// ClientDetection) degrades connection health silently, so availability
// wins over fail-fast here.
func decodeProtoServerRequest(payload *nacos_grpc_service.Payload) (rpc_request.IRequest, bool) {
	switch payload.GetMetadata().GetType() {
	case "ConnectResetRequest", "ClientDetectionRequest", "NotifySubscriberRequest",
		"NamingFuzzyWatchSyncRequest", "NamingFuzzyWatchChangeNotifyRequest", "SetupAckRequest",
		"ConfigChangeNotifyRequest":
	default:
		return nil, false
	}
	msg, err := payloadCodec.Decode(payload)
	if err != nil {
		logger.Warnf("proto decode server request %s failed, falling back to legacy json: %v",
			payload.GetMetadata().GetType(), err)
		return nil, false
	}
	switch m := msg.(type) {
	case *common.ConnectResetRequest:
		req := &rpc_request.ConnectResetRequest{
			InternalRequest: rpc_request.NewInternalRequest(),
			ServerIp:        m.ServerIp,
			ServerPort:      m.ServerPort,
		}
		req.RequestId = m.RequestId
		return req, true
	case *common.ClientDetectionRequest:
		req := &rpc_request.ClientDetectionRequest{InternalRequest: rpc_request.NewInternalRequest()}
		req.RequestId = m.RequestId
		return req, true
	case *naming.NotifySubscriberRequest:
		req := &rpc_request.NotifySubscriberRequest{
			NamingRequest: rpc_request.NewNamingRequest(m.Namespace, m.ServiceName, m.GroupName),
			ServiceInfo:   fromProtoServiceInfo(m.ServiceInfo),
		}
		req.RequestId = m.RequestId
		return req, true
	case *naming.NamingFuzzyWatchSyncRequest:
		contexts := make([]rpc_request.NamingFuzzyWatchSyncContext, 0, len(m.Contexts))
		for _, c := range m.Contexts {
			contexts = append(contexts, rpc_request.NamingFuzzyWatchSyncContext{
				ServiceKey:  c.GetServiceKey(),
				ChangedType: c.GetChangedType(),
			})
		}
		req := &rpc_request.NamingFuzzyWatchSyncRequest{
			Request:         &rpc_request.Request{RequestId: m.RequestId},
			SyncType:        m.SyncType,
			GroupKeyPattern: m.GroupKeyPattern,
			Contexts:        contexts,
			TotalBatch:      int(m.TotalBatch),
			CurrentBatch:    int(m.CurrentBatch),
		}
		return req, true
	case *naming.NamingFuzzyWatchChangeNotifyRequest:
		req := &rpc_request.NamingFuzzyWatchChangeNotifyRequest{
			Request:     &rpc_request.Request{RequestId: m.RequestId},
			SyncType:    m.SyncType,
			ServiceKey:  m.ServiceKey,
			ChangedType: m.ChangedType,
		}
		return req, true
	case *common.SetupAckRequest:
		req := &rpc_request.SetupAckRequest{
			InternalRequest: rpc_request.NewInternalRequest(),
			AbilityTable:    m.AbilityTable,
		}
		req.RequestId = m.RequestId
		return req, true
	case *config.ConfigChangeNotifyRequest:
		return adaptConfigChangeNotifyRequest(m), true
	}
	return nil, false
}
