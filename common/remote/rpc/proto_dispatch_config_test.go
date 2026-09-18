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

	"github.com/nacos-group/nacos-sdk-proto/go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_response"
)

func TestDecodeProtoConfigQueryResponse(t *testing.T) {
	msg := &config.ConfigQueryResponse{
		ResultCode:       200,
		Content:          "c1",
		Md5:              "m1",
		ContentType:      "text",
		EncryptedDataKey: "ek",
		LastModified:     123,
		IsBeta:           true,
	}
	payload, err := payloadCodec.Encode("ConfigQueryResponse", msg, nil, "127.0.0.1")
	require.NoError(t, err)

	resp, migrated, err := decodeProtoResponse(payload)
	require.True(t, migrated)
	require.NoError(t, err)
	qr := resp.(*rpc_response.ConfigQueryResponse)
	assert.True(t, qr.IsSuccess())
	assert.Equal(t, "c1", qr.Content)
	assert.Equal(t, "ek", qr.EncryptedDataKey)
	assert.Equal(t, int64(123), qr.LastModified)
	assert.True(t, qr.IsBeta)
}

func TestDecodeProtoConfigPublishResponseSuccess(t *testing.T) {
	msg := &config.ConfigPublishResponse{ResultCode: 200, RequestId: "1"}
	payload, err := payloadCodec.Encode("ConfigPublishResponse", msg, nil, "127.0.0.1")
	require.NoError(t, err)

	resp, migrated, err := decodeProtoResponse(payload)
	require.True(t, migrated)
	require.NoError(t, err)
	pr := resp.(*rpc_response.ConfigPublishResponse)
	assert.True(t, pr.IsSuccess())
	assert.Equal(t, "1", pr.RequestId)
}

func TestDecodeProtoConfigPublishResponseFailure(t *testing.T) {
	msg := &config.ConfigPublishResponse{
		ResultCode: 500,
		ErrorCode:  500,
		Message:    "internal error",
		RequestId:  "1",
	}
	payload, err := payloadCodec.Encode("ConfigPublishResponse", msg, nil, "127.0.0.1")
	require.NoError(t, err)

	resp, migrated, err := decodeProtoResponse(payload)
	require.True(t, migrated)
	require.NoError(t, err)
	pr := resp.(*rpc_response.ConfigPublishResponse)
	assert.False(t, pr.IsSuccess())
	assert.Equal(t, "internal error", pr.GetMessage())
}

func TestDecodeProtoConfigRemoveResponseSuccess(t *testing.T) {
	msg := &config.ConfigRemoveResponse{ResultCode: 200, RequestId: "2"}
	payload, err := payloadCodec.Encode("ConfigRemoveResponse", msg, nil, "127.0.0.1")
	require.NoError(t, err)

	resp, migrated, err := decodeProtoResponse(payload)
	require.True(t, migrated)
	require.NoError(t, err)
	rr := resp.(*rpc_response.ConfigRemoveResponse)
	assert.True(t, rr.IsSuccess())
	assert.Equal(t, "2", rr.RequestId)
}

func TestDecodeProtoConfigRemoveResponseFailure(t *testing.T) {
	msg := &config.ConfigRemoveResponse{
		ResultCode: 500,
		ErrorCode:  500,
		Message:    "config not found",
		RequestId:  "2",
	}
	payload, err := payloadCodec.Encode("ConfigRemoveResponse", msg, nil, "127.0.0.1")
	require.NoError(t, err)

	resp, migrated, err := decodeProtoResponse(payload)
	require.True(t, migrated)
	require.NoError(t, err)
	rr := resp.(*rpc_response.ConfigRemoveResponse)
	assert.False(t, rr.IsSuccess())
	assert.Equal(t, "config not found", rr.GetMessage())
}

func TestDecodeProtoConfigBatchListenResponse(t *testing.T) {
	msg := &config.ConfigChangeBatchListenResponse{
		ResultCode: 200,
		ChangedConfigs: []*config.ConfigChangeBatchListenResponseConfigContext{
			{Group: "g", DataId: "d", Tenant: "ns"},
		},
	}
	payload, err := payloadCodec.Encode("ConfigChangeBatchListenResponse", msg, nil, "127.0.0.1")
	require.NoError(t, err)

	resp, migrated, err := decodeProtoResponse(payload)
	require.True(t, migrated)
	require.NoError(t, err)
	br := resp.(*rpc_response.ConfigChangeBatchListenResponse)
	require.Len(t, br.ChangedConfigs, 1)
	assert.Equal(t, "d", br.ChangedConfigs[0].DataId)
	assert.Equal(t, "g", br.ChangedConfigs[0].Group)
	assert.Equal(t, "ns", br.ChangedConfigs[0].Tenant)
}

func TestDecodeProtoConfigChangeNotifyRequest(t *testing.T) {
	msg := &config.ConfigChangeNotifyRequest{
		RequestId: "1",
		DataId:    "d",
		Group:     "g",
		Tenant:    "ns",
	}
	payload, err := payloadCodec.Encode("ConfigChangeNotifyRequest", msg, nil, "127.0.0.1")
	require.NoError(t, err)

	req, migrated := decodeProtoServerRequest(payload)
	require.True(t, migrated)
	n := req.(*rpc_request.ConfigChangeNotifyRequest)
	assert.Equal(t, "d", n.DataId)
	assert.Equal(t, "g", n.Group)
	assert.Equal(t, "ns", n.Tenant)
	assert.Equal(t, "1", n.RequestId)
}
