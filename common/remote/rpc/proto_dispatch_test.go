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
	"testing"

	nacos_grpc_service "github.com/nacos-group/nacos-sdk-proto/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_response"
)

func protoPayload(typeName, body string) *nacos_grpc_service.Payload {
	return &nacos_grpc_service.Payload{
		Metadata: &nacos_grpc_service.Metadata{Type: typeName},
		Body:     &anypb.Any{Value: []byte(body)},
	}
}

func TestDecodeProtoResponseServerCheck(t *testing.T) {
	// 服务端真实形态：含 proto 中不存在的 success 派生字段
	resp, migrated, err := decodeProtoResponse(protoPayload("ServerCheckResponse",
		`{"resultCode":200,"connectionId":"c-1","success":true,"requestId":"r-1"}`))
	require.NoError(t, err)
	require.True(t, migrated)
	sc, ok := resp.(*rpc_response.ServerCheckResponse)
	require.True(t, ok, "downstream type assertions must keep working")
	assert.Equal(t, "c-1", sc.ConnectionId)
	assert.True(t, sc.IsSuccess())
	assert.Equal(t, "r-1", sc.RequestId)
}

func TestDecodeProtoResponseSuccessDerivedFromResultCode(t *testing.T) {
	// wire 无显式 success 时按 resultCode==200 推导（对齐 InnerResponseJsonUnmarshal）
	resp, migrated, err := decodeProtoResponse(protoPayload("HealthCheckResponse", `{"resultCode":200}`))
	require.NoError(t, err)
	require.True(t, migrated)
	assert.True(t, resp.IsSuccess())
}

func TestDecodeProtoResponseExplicitFailure(t *testing.T) {
	resp, migrated, err := decodeProtoResponse(protoPayload("ErrorResponse",
		`{"resultCode":500,"errorCode":403,"message":"forbidden","success":false}`))
	require.NoError(t, err)
	require.True(t, migrated)
	assert.False(t, resp.IsSuccess())
	assert.Equal(t, 403, resp.GetErrorCode())
}

func TestDecodeProtoResponseUnmigratedFallsThrough(t *testing.T) {
	_, migrated, err := decodeProtoResponse(protoPayload("UnknownType", `{"resultCode":200}`))
	require.NoError(t, err)
	assert.False(t, migrated, "unmigrated types fall through to legacy path")
}

func TestDecodeProtoServerRequestConnectReset(t *testing.T) {
	req, ok := decodeProtoServerRequest(protoPayload("ConnectResetRequest",
		`{"requestId":"r-9","serverIp":"10.0.0.2","serverPort":"8848"}`))
	require.True(t, ok)
	reset, isReset := req.(*rpc_request.ConnectResetRequest)
	require.True(t, isReset, "handler type assertions must keep working")
	assert.Equal(t, "10.0.0.2", reset.ServerIp)
	assert.Equal(t, "8848", reset.ServerPort)
	assert.Equal(t, "r-9", reset.GetRequestId())
}

func TestDecodeProtoServerRequestClientDetection(t *testing.T) {
	req, ok := decodeProtoServerRequest(protoPayload("ClientDetectionRequest", `{"requestId":"r-2"}`))
	require.True(t, ok)
	_, isDetection := req.(*rpc_request.ClientDetectionRequest)
	assert.True(t, isDetection)
	assert.Equal(t, "r-2", req.GetRequestId())
}

func TestDecodeProtoServerRequestUnknownFallsThrough(t *testing.T) {
	_, ok := decodeProtoServerRequest(protoPayload("UnknownRequest", `{}`))
	assert.False(t, ok, "unmigrated requests fall through to legacy path")
}

func TestDecodeProtoServerRequestBadJsonFallsBack(t *testing.T) {
	_, ok := decodeProtoServerRequest(protoPayload("ConnectResetRequest", `{not-json`))
	assert.False(t, ok, "decode failure falls back to the legacy path for resilience")
}
