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

import (
	"encoding/json"
	"testing"

	"github.com/nacos-group/nacos-sdk-proto/go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/codec"
)

// Java 3.2.3 verification (Step 0), sources under
// https://raw.githubusercontent.com/alibaba/nacos/3.2.3/ :
//  1. api/.../common/Constants.java: ConfigChangedType.ADD_CONFIG="ADD_CONFIG",
//     DELETE_CONFIG="DELETE_CONFIG", CONFIG_CHANGED="CONFIG_CHANGED". The
//     sync/watch type strings (FUZZY_WATCH_INIT_NOTIFY, FINISH_FUZZY_WATCH_INIT_NOTIFY,
//     FUZZY_WATCH_DIFF_SYNC_NOTIFY, FUZZY_WATCH_RESOURCE_CHANGED, WATCH, CANCEL_WATCH)
//     are shared with naming and already live in common/constant/fuzzy_watch.go.
//  2. ConfigFuzzyWatchChangeNotifyRequest has no syncType on the wire (only
//     groupKey/changeType). client/.../config/impl/ConfigFuzzyWatchGroupKeyHolder
//     .handlerFuzzyWatchChangeNotifyRequest fills the event syncType with the
//     constant FUZZY_WATCH_RESOURCE_CHANGED. For ChangeNotify, changeType ADD_CONFIG
//     or CONFIG_CHANGED both emit an ADD_CONFIG event (only if the group key was
//     not already received); DELETE_CONFIG emits DELETE_CONFIG (only if it was
//     received). For SyncRequest the event syncType is request.getSyncType().
//     ConfigFuzzyWatchRequest's boolean initializing is sent as "initializing"
//     (Jackson bean name), which sdk-proto beta.9 maps via json_name.

func TestConfigFuzzyWatchRequestProtoMessage(t *testing.T) {
	r := NewConfigFuzzyWatchRequest("public>>g*>>d*", constant.FUZZY_WATCH_TYPE_WATCH,
		[]string{"d1+g+ns", "d2+g+ns"}, true)
	r.RequestId = "1"

	assert.Equal(t, constant.CONFIG_FUZZY_WATCH_REQUEST_NAME, r.GetRequestType())
	assert.Equal(t, "ConfigFuzzyWatchRequest", r.GetRequestType())

	msg, ok := r.ProtoMessage().(*config.ConfigFuzzyWatchRequest)
	require.True(t, ok)
	assert.Equal(t, "1", msg.RequestId)
	assert.Equal(t, "public>>g*>>d*", msg.GroupKeyPattern)
	assert.Equal(t, []string{"d1+g+ns", "d2+g+ns"}, msg.ReceivedGroupKeys)
	assert.Equal(t, "WATCH", msg.WatchType)
	assert.True(t, msg.IsInitializing)
}

func TestConfigFuzzyWatchRequestProtoMessage_Cancel(t *testing.T) {
	r := NewConfigFuzzyWatchRequest("p", constant.FUZZY_WATCH_TYPE_CANCEL_WATCH, nil, false)
	msg := r.ProtoMessage().(*config.ConfigFuzzyWatchRequest)
	assert.Equal(t, "CANCEL_WATCH", msg.WatchType)
	assert.False(t, msg.IsInitializing)
}

func TestConfigFuzzyWatchRequestWireParity(t *testing.T) {
	r := NewConfigFuzzyWatchRequest("public>>g*>>d*", constant.FUZZY_WATCH_TYPE_WATCH, []string{"d1+g+ns"}, true)
	r.RequestId = "1"

	var req IRequest = r
	pc := req.(codec.ProtoConvertible) // compile-time interface verification
	wire, err := protojson.MarshalOptions{EmitDefaultValues: true}.Marshal(pc.ProtoMessage())
	require.NoError(t, err)

	// isInitializing must travel as "initializing" (json_name), never the
	// discarded default name: the server's Jackson binding drops unknown keys.
	var wireMap map[string]interface{}
	require.NoError(t, json.Unmarshal(wire, &wireMap))
	assert.Equal(t, true, wireMap["initializing"])
	_, hasIs := wireMap["isInitializing"]
	assert.False(t, hasIs)

	assertSameJSONKeys(t, []byte(req.GetBody(req)), wire, nil, nil)

	var legacy map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(req.GetBody(req)), &legacy))
	assert.Equal(t, true, legacy["initializing"])
	_, hasIs = legacy["isInitializing"]
	assert.False(t, hasIs)
	assert.Equal(t, "public>>g*>>d*", legacy["groupKeyPattern"])
	assert.Equal(t, "WATCH", legacy["watchType"])
}

func TestConfigFuzzyWatchPushRequestTypes(t *testing.T) {
	assert.Equal(t, "ConfigFuzzyWatchChangeNotifyRequest",
		(&ConfigFuzzyWatchChangeNotifyRequest{Request: &Request{}}).GetRequestType())
	assert.Equal(t, "ConfigFuzzyWatchSyncRequest",
		(&ConfigFuzzyWatchSyncRequest{Request: &Request{}}).GetRequestType())
	assert.Equal(t, constant.CONFIG_FUZZY_WATCH_CHANGE_NOTIFY_REQUEST_NAME, "ConfigFuzzyWatchChangeNotifyRequest")
	assert.Equal(t, constant.CONFIG_FUZZY_WATCH_SYNC_REQUEST_NAME, "ConfigFuzzyWatchSyncRequest")
	assert.Equal(t, "ADD_CONFIG", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	assert.Equal(t, "DELETE_CONFIG", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG)
	assert.Equal(t, "CONFIG_CHANGED", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_CONFIG_CHANGED)
}
