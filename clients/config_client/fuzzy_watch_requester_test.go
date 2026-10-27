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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_response"
)

func TestRequesterCheckSupported(t *testing.T) {
	r := &configFuzzyWatchRequester{abilitySupported: func() bool { return true }}
	assert.NoError(t, r.CheckFuzzyWatchSupported())
	r = &configFuzzyWatchRequester{abilitySupported: func() bool { return false }}
	assert.True(t, errors.Is(r.CheckFuzzyWatchSupported(), ErrConfigFuzzyWatchNotSupported))
}

func TestRequesterSendBuildsRequestAndMapsResponses(t *testing.T) {
	var got *rpc_request.ConfigFuzzyWatchRequest
	var resp rpc_response.IResponse
	var sendErr error
	r := &configFuzzyWatchRequester{send: func(req rpc_request.IRequest) (rpc_response.IResponse, error) {
		got = req.(*rpc_request.ConfigFuzzyWatchRequest)
		return resp, sendErr
	}}

	resp = &rpc_response.ConfigFuzzyWatchResponse{Response: &rpc_response.Response{Success: true, ResultCode: constant.RESPONSE_CODE_SUCCESS}}
	require.NoError(t, r.SendFuzzyWatchRequest("public>>g>>a*", constant.FUZZY_WATCH_TYPE_WATCH, []string{"k1"}, true))
	assert.Equal(t, "public>>g>>a*", got.GroupKeyPattern)
	assert.Equal(t, constant.FUZZY_WATCH_TYPE_WATCH, got.WatchType)
	assert.Equal(t, []string{"k1"}, got.ReceivedGroupKeys)
	assert.True(t, got.IsInitializing)
	assert.NotNil(t, got.Headers, "headers must be non-nil so auth injection can write them")

	resp = &rpc_response.ConfigFuzzyWatchResponse{Response: &rpc_response.Response{Success: false, ErrorCode: 50311, Message: "over"}}
	err := r.SendFuzzyWatchRequest("p", constant.FUZZY_WATCH_TYPE_WATCH, nil, false)
	var se *ConfigFuzzyWatchServerError
	require.True(t, errors.As(err, &se))
	assert.Equal(t, 50311, se.ErrorCode)
	assert.True(t, isOverLimitError(err))
	assert.False(t, isOverLimitError(errors.New("x")))
	assert.False(t, isOverLimitError(&ConfigFuzzyWatchServerError{ErrorCode: 500}))
	assert.True(t, isOverLimitError(&ConfigFuzzyWatchServerError{ErrorCode: 50310}))

	resp = nil
	assert.Error(t, r.SendFuzzyWatchRequest("p", constant.FUZZY_WATCH_TYPE_WATCH, nil, false))

	sendErr = errors.New("transport")
	assert.EqualError(t, r.SendFuzzyWatchRequest("p", constant.FUZZY_WATCH_TYPE_WATCH, nil, false), "transport")
}

func TestPushHandlersAckWithLegacyTypesAndFeedHolder(t *testing.T) {
	h, _ := newReconcileHolder(t)
	sink := &eventSink{}
	registerDispatch(t, h, sink.cb)

	// Adapter-built push requests leave Request.Headers nil; handlers must cope.
	notify := &rpc_request.ConfigFuzzyWatchChangeNotifyRequest{Request: &rpc_request.Request{},
		GroupKey: dispatchKey("app1"), ChangeType: constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_CONFIG_CHANGED}
	ack := (&configFuzzyWatchChangeNotifyHandler{holder: h}).RequestReply(notify, (*rpc.RpcClient)(nil))
	_, ok := ack.(*rpc_response.ConfigFuzzyWatchChangeNotifyResponse)
	require.True(t, ok)
	assert.True(t, ack.IsSuccess())
	assert.Nil(t, notify.Headers)

	syncReq := &rpc_request.ConfigFuzzyWatchSyncRequest{Request: &rpc_request.Request{}, GroupKeyPattern: dispatchPattern,
		SyncType: constant.FUZZY_WATCH_INIT_NOTIFY, Contexts: syncContexts("app2"), TotalBatch: 1, CurrentBatch: 1}
	ack = (&configFuzzyWatchSyncHandler{holder: h}).RequestReply(syncReq, (*rpc.RpcClient)(nil))
	_, ok = ack.(*rpc_response.ConfigFuzzyWatchSyncResponse)
	require.True(t, ok)
	assert.True(t, ack.IsSuccess())
	assert.Nil(t, syncReq.Headers)
	waitFor(t, func() bool { return len(sink.snapshot()) == 2 })

	// Wrong request type: handlers return nil like the naming handlers.
	assert.Nil(t, (&configFuzzyWatchSyncHandler{holder: h}).RequestReply(notify, nil))
	assert.Nil(t, (&configFuzzyWatchChangeNotifyHandler{holder: h}).RequestReply(syncReq, nil))
}

func TestConnectedResetsFuzzyConsistency(t *testing.T) {
	h, _ := newReconcileHolder(t)
	sink := &eventSink{}
	ctx, _ := registerDispatch(t, h, sink.cb)
	h.executeSync()
	require.True(t, consistent(ctx))

	(&ConfigConnectionEventListener{client: &ConfigClient{fuzzyWatchHolder: h}, taskId: "0"}).OnConnected()
	assert.False(t, consistent(ctx), "reconnect invalidates server-side WATCH state")
}
