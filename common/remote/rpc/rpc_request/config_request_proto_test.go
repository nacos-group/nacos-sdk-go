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

	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/codec"
	"github.com/nacos-group/nacos-sdk-go/v3/model"
)

func TestConfigPublishRequestProtoMessage(t *testing.T) {
	r := NewConfigPublishRequest("g", "d", "ns", "content-1", "abc123")
	r.AdditionMap["appName"] = "app"
	m := r.ProtoMessage()
	pm, ok := m.(*config.ConfigPublishRequest)
	require.True(t, ok)
	assert.Equal(t, "d", pm.DataId)
	assert.Equal(t, "g", pm.Group)
	assert.Equal(t, "ns", pm.Tenant)
	assert.Equal(t, "content-1", pm.Content)
	assert.Equal(t, "abc123", pm.CasMd5)
	assert.Equal(t, "app", pm.AdditionMap["appName"])
}

func TestConfigQueryRequestProtoMessage(t *testing.T) {
	r := NewConfigQueryRequest("g", "d", "ns")
	r.Tag = "tag-1"
	m := r.ProtoMessage()
	pm, ok := m.(*config.ConfigQueryRequest)
	require.True(t, ok)
	assert.Equal(t, "d", pm.DataId)
	assert.Equal(t, "g", pm.Group)
	assert.Equal(t, "ns", pm.Tenant)
	assert.Equal(t, "tag-1", pm.Tag)
}

func TestConfigRemoveRequestProtoMessage(t *testing.T) {
	r := NewConfigRemoveRequest("g", "d", "ns")
	m := r.ProtoMessage()
	pm, ok := m.(*config.ConfigRemoveRequest)
	require.True(t, ok)
	assert.Equal(t, "d", pm.DataId)
	assert.Equal(t, "g", pm.Group)
	assert.Equal(t, "ns", pm.Tenant)
}

func TestConfigBatchListenRequestProtoMessage(t *testing.T) {
	r := NewConfigBatchListenRequest(1)
	r.Listen = false
	r.ConfigListenContexts = append(r.ConfigListenContexts,
		model.ConfigListenContext{Group: "g", Md5: "m", DataId: "d", Tenant: "ns"})
	pm := r.ProtoMessage().(*config.ConfigBatchListenRequest)
	assert.False(t, pm.Listen)
	require.Len(t, pm.ConfigListenContexts, 1)
	assert.Equal(t, "d", pm.ConfigListenContexts[0].DataId)
	assert.Equal(t, "m", pm.ConfigListenContexts[0].Md5)
	assert.Equal(t, "g", pm.ConfigListenContexts[0].Group)
	assert.Equal(t, "ns", pm.ConfigListenContexts[0].Tenant)
}

// assertSameJSONKeys compares the key sets of two JSON byte slices, excluding known differences.
// Both are unmarshaled to maps and key sets are compared after removing excluded keys.
func assertSameJSONKeys(t *testing.T, legacy, wire []byte, legacyExclude, protoExclude []string) {
	t.Helper()
	var legacyMap, wireMap map[string]interface{}
	require.NoError(t, json.Unmarshal(legacy, &legacyMap))
	require.NoError(t, json.Unmarshal(wire, &wireMap))

	// Remove known server-derived/transport-level fields
	commonExclude := []string{"module", "headers", "requestId"}
	for _, k := range commonExclude {
		delete(legacyMap, k)
		delete(wireMap, k)
	}

	// Remove request-specific excludes
	for _, k := range legacyExclude {
		delete(legacyMap, k)
	}
	for _, k := range protoExclude {
		delete(wireMap, k)
	}

	// Check key sets are identical
	for k := range legacyMap {
		_, ok := wireMap[k]
		assert.True(t, ok, "expected key %q in proto JSON", k)
	}
	for k := range wireMap {
		_, ok := legacyMap[k]
		assert.True(t, ok, "expected key %q in legacy JSON", k)
	}
}

func TestConfigRequestProtoWireParity(t *testing.T) {
	testCases := []struct {
		req           IRequest
		legacyExclude []string
		protoExclude  []string
	}{
		{
			req:           NewConfigQueryRequest("g", "d", "ns"),
			legacyExclude: nil,
			protoExclude:  nil,
		},
		{
			req:           NewConfigPublishRequest("g", "d", "ns", "content", "md5"),
			legacyExclude: nil,
			protoExclude:  nil,
		},
		{
			req:           NewConfigRemoveRequest("g", "d", "ns"),
			legacyExclude: nil,
			protoExclude:  []string{"tag"}, // ConfigRemoveRequest proto has tag field, legacy doesn't
		},
		{
			req:           NewConfigBatchListenRequest(1),
			legacyExclude: nil,
			protoExclude:  nil,
		},
	}

	for _, tc := range testCases {
		pc := tc.req.(codec.ProtoConvertible) // compile-time interface verification
		wire, err := protojson.MarshalOptions{EmitDefaultValues: true}.Marshal(pc.ProtoMessage())
		require.NoError(t, err)
		assertSameJSONKeys(t, []byte(tc.req.GetBody(tc.req)), wire, tc.legacyExclude, tc.protoExclude)
	}
}
