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
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nacos-group/nacos-sdk-go/v3/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v3/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/model"
	"github.com/nacos-group/nacos-sdk-go/v3/vo"
)

func newFuzzyTestClient(t *testing.T) *ConfigClient {
	t.Helper()
	client := createConfigClientTest()
	client.fuzzyWatchHolder.SetRequester(&fakeConfigFuzzyRequester{})
	t.Cleanup(client.CloseClient)
	return client
}

func fuzzyParam(dataId, group string) *vo.ConfigFuzzyWatchParam {
	return &vo.ConfigFuzzyWatchParam{
		DataIdPattern: dataId,
		Group:         group,
		OnChange:      func(model.ConfigFuzzyWatchChangeEvent) {},
	}
}

func TestFuzzyWatchConfigValidation(t *testing.T) {
	client := newFuzzyTestClient(t)

	_, err := client.FuzzyWatchConfig(nil)
	assert.Error(t, err)
	for _, bad := range []string{"", "   ", "\t\n"} {
		_, err = client.FuzzyWatchConfig(fuzzyParam(bad, "g"))
		assert.Error(t, err, "dataIdPattern %q must be rejected", bad)
	}
	p := fuzzyParam("app*", "g")
	p.OnChange = nil
	_, err = client.FuzzyWatchConfig(p)
	assert.Error(t, err)
	assert.Empty(t, client.fuzzyWatchHolder.Patterns(), "rejected params must not register anything")
}

func TestFuzzyWatchConfigPatternDefaultsAndTrim(t *testing.T) {
	client := newFuzzyTestClient(t)

	h, err := client.FuzzyWatchConfig(fuzzyParam("  app*  ", ""))
	require.NoError(t, err)
	assert.Equal(t, "public>>DEFAULT_GROUP>>app*", h.Pattern())
	assert.Contains(t, client.fuzzyWatchHolder.Patterns(), h.Pattern())

	h2, err := client.FuzzyWatchConfig(fuzzyParam("*", "g1"))
	require.NoError(t, err)
	assert.Equal(t, BuildConfigFuzzyWatchPattern(client.fuzzyWatchHolder.namespace, "*", "g1"), h2.Pattern())
}

func TestFuzzyWatchConfigOnClosedClientReturnsSentinel(t *testing.T) {
	client := createConfigClientTest()
	client.fuzzyWatchHolder.SetRequester(&fakeConfigFuzzyRequester{})
	client.CloseClient()
	h, err := client.FuzzyWatchConfig(fuzzyParam("app*", "g"))
	assert.Nil(t, h)
	assert.ErrorIs(t, err, ErrConfigFuzzyWatchClientClosed)
	assert.NotPanics(t, client.CloseClient)
}

func TestFuzzyWatchConfigNotSupportedPropagates(t *testing.T) {
	client := createConfigClientTest()
	t.Cleanup(client.CloseClient)
	client.fuzzyWatchHolder.SetRequester(&fakeConfigFuzzyRequester{supportErr: ErrConfigFuzzyWatchNotSupported})
	_, err := client.FuzzyWatchConfig(fuzzyParam("app*", "g"))
	assert.ErrorIs(t, err, ErrConfigFuzzyWatchNotSupported)
}

func TestConfigFuzzyWatchHandleCancelIsIdempotentAndScoped(t *testing.T) {
	client := newFuzzyTestClient(t)
	h1, err := client.FuzzyWatchConfig(fuzzyParam("app*", "g"))
	require.NoError(t, err)
	h2, err := client.FuzzyWatchConfig(fuzzyParam("app*", "g"))
	require.NoError(t, err)
	require.Equal(t, h1.Pattern(), h2.Pattern())

	h1.Cancel()
	h1.Cancel() // idempotent
	ctx, ok := client.fuzzyWatchHolder.get(h1.Pattern())
	require.True(t, ok)
	ctx.mu.Lock()
	remaining := len(ctx.entries)
	discard := ctx.discard
	ctx.mu.Unlock()
	assert.Equal(t, 1, remaining, "Cancel must remove only its own registration")
	assert.False(t, discard)

	h2.Cancel()
	ctx.mu.Lock()
	discard = ctx.discard
	ctx.mu.Unlock()
	assert.True(t, discard, "last Cancel flips the pattern to pending-cancel")
}

func TestConfigFuzzyWatchHandleMatchedGroupKeysBlocksUntilFinish(t *testing.T) {
	client := newFuzzyTestClient(t)
	h, err := client.FuzzyWatchConfig(fuzzyParam("app*", "g"))
	require.NoError(t, err)

	// ctx cancel bounds the wait before the initial sync finished.
	waitCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = h.MatchedGroupKeys(waitCtx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	var wg sync.WaitGroup
	var keys []string
	var gotErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		keys, gotErr = h.MatchedGroupKeys(context.Background())
	}()
	client.fuzzyWatchHolder.HandleSync(h.Pattern(), constant.FINISH_FUZZY_WATCH_INIT_NOTIFY,
		[]rpc_request.ConfigFuzzyWatchSyncContext{}, 1, 1)
	wg.Wait()
	assert.NoError(t, gotErr)
	assert.Empty(t, keys)
}

// The push handlers register inside createRpcClient only for the first
// getRpcClient, so the holder must already be wired when the constructor
// returns. Build a client with the REAL proxy and inspect the rpc client's
// server-request handler table.
func TestNewConfigClientRegistersFuzzyWatchPushHandlers(t *testing.T) {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*serverConfigWithOptions})
	_ = nc.SetClientConfig(*clientConfigWithOptions)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, err := NewConfigClient(&nc)
	require.NoError(t, err)
	t.Cleanup(client.CloseClient)

	require.NotNil(t, client.fuzzyWatchHolder, "holder must exist before the first rpc client")
	require.NotNil(t, client.fuzzyWatchHolder.getRequester(), "requester must be wired by the constructor")

	cp, ok := client.configProxy.(*ConfigProxy)
	require.True(t, ok)
	rpcClient := cp.getRpcClient(client)

	table := reflect.ValueOf(rpcClient).Elem().FieldByName("serverRequestHandlerMapping")
	require.True(t, table.IsValid())
	m := (*sync.Map)(unsafe.Pointer(table.UnsafeAddr()))
	for _, req := range []rpc_request.IRequest{
		&rpc_request.ConfigFuzzyWatchSyncRequest{Request: &rpc_request.Request{}},
		&rpc_request.ConfigFuzzyWatchChangeNotifyRequest{Request: &rpc_request.Request{}},
	} {
		_, found := m.Load(req.GetRequestType())
		assert.True(t, found, "handler for %s must be registered", req.GetRequestType())
	}
}
