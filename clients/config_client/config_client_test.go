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
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v3/common/security"
	"github.com/nacos-group/nacos-sdk-go/v3/util"

	"github.com/nacos-group/nacos-sdk-go/v3/common/filter"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v3/model"

	"github.com/nacos-group/nacos-sdk-go/v3/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v3/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v3/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serverConfigWithOptions = constant.NewServerConfig("127.0.0.1", 8848)

var clientConfigWithOptions = constant.NewClientConfig(
	constant.WithTimeoutMs(10*1000),
	constant.WithBeatInterval(2*1000),
	constant.WithNotLoadCacheAtStart(true),
	constant.WithAccessKey("LTAxxx"),
	constant.WithSecretKey("EdPxxx"),
	constant.WithOpenKMS(true),
	constant.WithKMSVersion(constant.KMSv1),
	constant.WithRegionId("cn-hangzhou"),
)

var clientTLsConfigWithOptions = constant.NewClientConfig(
	constant.WithTimeoutMs(10*1000),
	constant.WithBeatInterval(2*1000),
	constant.WithNotLoadCacheAtStart(true),

	/*constant.WithTLS(constant.TLSConfig{
		Enable:   true,
		TrustAll: false,
		CaFile:   "mse-nacos-ca.cer",
	}),*/
)

var localConfigTest = vo.ConfigParam{
	DataId:  "dataId",
	Group:   "group",
	Content: "content",
}

func createConfigClientTest() *ConfigClient {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*serverConfigWithOptions})
	_ = nc.SetClientConfig(*clientConfigWithOptions)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewConfigClient(&nc)
	client.configProxy = &MockConfigProxy{}
	return client
}

func createConfigClientTestTls() *ConfigClient {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*serverConfigWithOptions})
	_ = nc.SetClientConfig(*clientTLsConfigWithOptions)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewConfigClient(&nc)
	client.configProxy = &MockConfigProxy{}
	return client
}

func createConfigClientCommon() *ConfigClient {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*serverConfigWithOptions})
	_ = nc.SetClientConfig(*clientConfigWithOptions)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewConfigClient(&nc)
	client.configProxy = &MockConfigProxy{}
	return client
}

func createConfigClientForKms() *ConfigClient {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*serverConfigWithOptions})
	_ = nc.SetClientConfig(*clientConfigWithOptions)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewConfigClient(&nc)
	client.configProxy = &MockConfigProxyForUsingLocalDiskCache{}
	return client
}

type MockConfigProxyForUsingLocalDiskCache struct {
	MockConfigProxy
}

func (m *MockConfigProxyForUsingLocalDiskCache) queryConfig(dataId, group, tenant string, timeout uint64, notify bool, client *ConfigClient) (*rpc_response.ConfigQueryResponse, error) {
	return nil, errors.New("mock err for using localCache")
}

type MockConfigProxy struct {
}

func (m *MockConfigProxy) queryConfig(dataId, group, tenant string, timeout uint64, notify bool, client *ConfigClient) (*rpc_response.ConfigQueryResponse, error) {
	cacheKey := util.GetConfigCacheKey(dataId, group, tenant)
	if IsLimited(cacheKey) {
		return nil, errors.New("request is limited")
	}
	return &rpc_response.ConfigQueryResponse{Content: "hello world", Response: &rpc_response.Response{Success: true}}, nil
}
func (m *MockConfigProxy) searchConfigProxy(param vo.SearchConfigParam, tenant, accessKey, secretKey string) (*model.ConfigPage, error) {
	return &model.ConfigPage{TotalCount: 1}, nil
}
func (m *MockConfigProxy) requestProxy(rpcClient *rpc.RpcClient, request rpc_request.IRequest, timeoutMills uint64) (rpc_response.IResponse, error) {
	return &rpc_response.MockResponse{Response: &rpc_response.Response{Success: true}}, nil
}
func (m *MockConfigProxy) createRpcClient(ctx context.Context, taskId string, client *ConfigClient) *rpc.RpcClient {
	return &rpc.RpcClient{}
}
func (m *MockConfigProxy) getRpcClient(client *ConfigClient) *rpc.RpcClient {
	return &rpc.RpcClient{}
}

func Test_GetConfig(t *testing.T) {
	client := createConfigClientTest()
	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   localConfigTest.Group,
		Content: "hello world"})

	assert.Nil(t, err)
	assert.True(t, success)

	content, err := client.GetConfig(vo.ConfigParam{
		DataId: localConfigTest.DataId,
		Group:  localConfigTest.Group})

	assert.Nil(t, err)
	assert.Equal(t, "hello world", content)
}

func Test_SearchConfig(t *testing.T) {
	client := createConfigClientTest()
	_, _ = client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "DEFAULT_GROUP",
		Content: "hello world"})
	configPage, err := client.SearchConfig(vo.SearchConfigParam{
		Search:   "accurate",
		DataId:   localConfigTest.DataId,
		Group:    "DEFAULT_GROUP",
		PageNo:   1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, configPage)
}

func Test_GetConfigTls(t *testing.T) {
	client := createConfigClientTestTls()
	_, _ = client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "DEFAULT_GROUP",
		Content: "hello world"})
	configPage, err := client.SearchConfig(vo.SearchConfigParam{
		Search:   "accurate",
		DataId:   localConfigTest.DataId,
		Group:    "DEFAULT_GROUP",
		PageNo:   1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, configPage)

}

// only using by ak sk for cipher config of aliyun kms
/*
func TestPublishAndGetConfigByUsingLocalCache(t *testing.T) {
	param := vo.ConfigParam{
		DataId:  "cipher-kms-aes-256-usingCache" + strconv.Itoa(rand.Int()),
		Group:   "DEFAULT",
		Content: "content加密&&" + strconv.Itoa(rand.Int()),
	}
	t.Run("PublishAndGetConfigByUsingLocalCache", func(t *testing.T) {
		commonClient := createConfigClientCommon()
		_, err := commonClient.PublishConfig(param)
		assert.Nil(t, err)

		time.Sleep(2 * time.Second)
		configQueryContent, err := commonClient.GetConfig(param)
		assert.Nil(t, err)
		assert.Equal(t, param.Content, configQueryContent)

		usingKmsCacheClient := createConfigClientForKms()
		configQueryContentByUsingCache, err := usingKmsCacheClient.GetConfig(param)
		assert.Nil(t, err)
		assert.Equal(t, param.Content, configQueryContentByUsingCache)

		newCipherContent := param.Content + "new"
		param.Content = newCipherContent
		err = commonClient.ListenConfig(vo.ConfigParam{
			DataId: param.DataId,
			Group:  param.Group,
			OnChange: func(namespace, group, dataId, data string) {
				t.Log("origin data: " + newCipherContent + "; new data: " + data)
				assert.Equal(t, newCipherContent, data)
			},
		})
		assert.Nil(t, err)

		result, err := commonClient.PublishConfig(param)
		assert.Nil(t, err)
		assert.True(t, result)

		time.Sleep(2 * time.Second)
		newContentCommon, err := commonClient.GetConfig(param)
		assert.Nil(t, err)
		assert.Equal(t, param.Content, newContentCommon)
		newContentKms, err := usingKmsCacheClient.GetConfig(param)
		assert.Nil(t, err)
		assert.Equal(t, param.Content, newContentKms)
	})
}
*/

// PublishConfig
func Test_PublishConfigWithoutDataId(t *testing.T) {
	client := createConfigClientTest()
	_, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "",
		Group:   "group",
		Content: "content",
	})
	assert.NotNil(t, err)
}

func Test_PublishConfigWithoutContent(t *testing.T) {
	client := createConfigClientTest()
	_, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "group",
		Content: "",
	})
	assert.NotNil(t, err)
}

func Test_PublishConfig(t *testing.T) {

	client := createConfigClientTest()

	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "group",
		SrcUser: "nacos-client-go",
		Content: "hello world"})

	assert.Nil(t, err)
	assert.True(t, success)
}

// DeleteConfig
func Test_DeleteConfig(t *testing.T) {

	client := createConfigClientTest()

	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "group",
		Content: "hello world!"})

	assert.Nil(t, err)
	assert.True(t, success)

	success, err = client.DeleteConfig(vo.ConfigParam{
		DataId: localConfigTest.DataId,
		Group:  "group"})

	assert.Nil(t, err)
	assert.True(t, success)
}

func Test_DeleteConfigWithoutDataId(t *testing.T) {
	client := createConfigClientTest()
	success, err := client.DeleteConfig(vo.ConfigParam{
		DataId: "",
		Group:  "group",
	})
	assert.NotNil(t, err)
	assert.Equal(t, false, success)
}

func TestListen(t *testing.T) {
	t.Run("TestListenConfig", func(t *testing.T) {
		client := createConfigClientTest()
		err := client.ListenConfig(vo.ConfigParam{
			DataId: localConfigTest.DataId,
			Group:  localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
			},
		})
		assert.Nil(t, err)
	})
	// ListenConfig no dataId
	t.Run("TestListenConfigNoDataId", func(t *testing.T) {
		listenConfigParam := vo.ConfigParam{
			Group: localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
			},
		}
		client := createConfigClientTest()
		err := client.ListenConfig(listenConfigParam)
		assert.Error(t, err)
	})
}

// CancelListenConfig
func TestCancelListenConfig(t *testing.T) {
	//Multiple listeners listen for different configurations, cancel one
	t.Run("TestMultipleListenersCancelOne", func(t *testing.T) {
		client := createConfigClientTest()
		var err error
		listenConfigParam := vo.ConfigParam{
			DataId: localConfigTest.DataId,
			Group:  localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
			},
		}

		listenConfigParam1 := vo.ConfigParam{
			DataId: localConfigTest.DataId + "1",
			Group:  localConfigTest.Group,
			OnChange: func(namespace, group, dataId, data string) {
			},
		}
		_ = client.ListenConfig(listenConfigParam)

		_ = client.ListenConfig(listenConfigParam1)

		err = client.CancelListenConfig(listenConfigParam)
		assert.Nil(t, err)
	})
}

// TestListenConfigAppendsSecondListener guards against the historical bug
// where a second ListenConfig call on the same key silently replaced (and
// therefore dropped) the first listener instead of appending to it.
func TestListenConfigAppendsSecondListener(t *testing.T) {
	client := createConfigClientTest()
	got := make(chan string, 2)
	for i := 0; i < 2; i++ {
		idx := strconv.Itoa(i)
		require.NoError(t, client.ListenConfig(vo.ConfigParam{DataId: "d", Group: "g",
			OnChange: func(ns, g, d, data string) { got <- idx + ":" + data }}))
	}
	clientConfig, err := client.GetClientConfig()
	require.NoError(t, err)
	cd, ok := client.holder.get(util.GetConfigCacheKey("d", "g", clientConfig.NamespaceId))
	require.True(t, ok)
	assert.Len(t, cd.listeners, 2)
}

// TestCancelListenConfigMarksDiscard verifies CancelListenConfig does not
// remove the cache entry outright (the executor/removeIfDiscarded reconcile
// it later); it must simply mark it discarded. Uses newExecutorTestClient
// (no background executeConfigListen loop) since, now that the executor
// really does reap discarded entries via removeIfDiscarded, a client with a
// live loop could race the assertion below and remove the entry before it
// runs.
func TestCancelListenConfigMarksDiscard(t *testing.T) {
	client := newExecutorTestClient(&MockConfigProxy{})
	p := vo.ConfigParam{DataId: "d", Group: "g", OnChange: func(ns, g, d, data string) {}}
	require.NoError(t, client.ListenConfig(p))
	require.NoError(t, client.CancelListenConfig(p))

	clientConfig, err := client.GetClientConfig()
	require.NoError(t, err)
	key := util.GetConfigCacheKey("d", "g", clientConfig.NamespaceId)
	cd, ok := client.holder.get(key)
	require.True(t, ok, "entry must still be present after cancel")
	assert.True(t, cd.discard)
}

// TestCancelListenConfigOnUnknownKeyIsNoop verifies cancelling a key that was
// never listened on is a no-op that returns nil, per the brief.
func TestCancelListenConfigOnUnknownKeyIsNoop(t *testing.T) {
	client := createConfigClientTest()
	err := client.CancelListenConfig(vo.ConfigParam{DataId: "never-listened", Group: "g"})
	assert.NoError(t, err)
}

// TestCancelThenListenRevives verifies that a Cancel followed by a Listen on
// the same key revives the existing entry (discard flips back to false)
// rather than leaking a second entry, and that the revived entry survives a
// subsequent removeIfDiscarded sweep. Uses newExecutorTestClient for the same
// race-avoidance reason as TestCancelListenConfigMarksDiscard above.
func TestCancelThenListenRevives(t *testing.T) {
	client := newExecutorTestClient(&MockConfigProxy{})
	p := vo.ConfigParam{DataId: "d", Group: "g", OnChange: func(ns, g, d, data string) {}}
	require.NoError(t, client.ListenConfig(p))
	require.NoError(t, client.CancelListenConfig(p))
	clientConfig, err := client.GetClientConfig()
	require.NoError(t, err)
	key := util.GetConfigCacheKey("d", "g", clientConfig.NamespaceId)
	cd, _ := client.holder.get(key)
	assert.True(t, cd.discard)
	require.NoError(t, client.ListenConfig(p))
	assert.False(t, cd.discard)
	client.holder.removeIfDiscarded(key)
	_, ok := client.holder.get(key)
	assert.True(t, ok, "revived entry must survive removeIfDiscarded")
}

// TestListenCancelConcurrentNeverLeavesDiscardedWithListeners is a
// regression test for a TOCTOU between getOrCreate's internal revive
// (discard=false) and a subsequently, separately-locked addListener call in
// ListenConfig: a concurrent CancelListenConfig (markDiscard) could
// interleave between the two and leave the entry with discard==true and a
// non-empty listeners slice. That combination must never be observable,
// since Task 4's reap wiring treats discard==true as "safe to remove once
// listeners is empty" and would otherwise build on an inconsistent
// intermediate state. An observer goroutine samples the entry continuously
// while two producer goroutines hammer ListenConfig/CancelListenConfig
// concurrently, so a transient (not just a final) violation would be
// caught.
func TestListenCancelConcurrentNeverLeavesDiscardedWithListeners(t *testing.T) {
	client := createConfigClientTest()
	p := vo.ConfigParam{DataId: "race-d", Group: "race-g", OnChange: func(ns, g, d, data string) {}}

	clientConfig, err := client.GetClientConfig()
	require.NoError(t, err)
	key := util.GetConfigCacheKey(p.DataId, p.Group, clientConfig.NamespaceId)

	const iterations = 500

	var producers sync.WaitGroup
	producers.Add(2)
	go func() {
		defer producers.Done()
		for i := 0; i < iterations; i++ {
			_ = client.ListenConfig(p)
		}
	}()
	go func() {
		defer producers.Done()
		for i := 0; i < iterations; i++ {
			_ = client.CancelListenConfig(p)
		}
	}()

	stop := make(chan struct{})
	var violation atomic.Bool
	var observer sync.WaitGroup
	observer.Add(1)
	go func() {
		defer observer.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if cData, ok := client.holder.get(key); ok {
				cData.mu.Lock()
				if cData.discard && len(cData.listeners) > 0 {
					violation.Store(true)
				}
				cData.mu.Unlock()
			}
		}
	}()

	producers.Wait()
	close(stop)
	observer.Wait()

	assert.False(t, violation.Load(), "entry must never be observed with discard==true and non-empty listeners")

	// Also assert the final resting state is consistent.
	if cData, ok := client.holder.get(key); ok {
		cData.mu.Lock()
		finalInconsistent := cData.discard && len(cData.listeners) > 0
		cData.mu.Unlock()
		assert.False(t, finalInconsistent, "final state must not be discard==true with non-empty listeners")
	}
}

type MockAccessKeyCredentialProvider struct {
	accessKey         string
	secretKey         string
	signatureRegionId string
}

func (provider *MockAccessKeyCredentialProvider) MatchProvider() bool {
	return true
}

func (provider *MockAccessKeyCredentialProvider) Init() error {
	return nil
}

func (provider *MockAccessKeyCredentialProvider) GetCredentialsForNacosClient() security.RamContext {
	ramContext := security.RamContext{
		AccessKey:         provider.accessKey,
		SecretKey:         provider.secretKey,
		SignatureRegionId: "",
	}
	return ramContext
}

func Test_ConfigClientWithProvider(t *testing.T) {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*serverConfigWithOptions})
	clientConfigWithOptions.AccessKey = ""
	clientConfigWithOptions.SecretKey = ""
	_ = nc.SetClientConfig(*clientConfigWithOptions)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	provider := &MockAccessKeyCredentialProvider{
		accessKey: "LTAxxx",
		secretKey: "EdPxxx",
	}
	client, _ := NewConfigClientWithRamCredentialProvider(&nc, provider)
	client.configProxy = &MockConfigProxy{}
	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   localConfigTest.Group,
		Content: "hello world"})

	assert.Nil(t, err)
	assert.True(t, success)

	content, err := client.GetConfig(vo.ConfigParam{
		DataId: localConfigTest.DataId,
		Group:  localConfigTest.Group})

	assert.Nil(t, err)
	assert.Equal(t, "hello world", content)
}

// ---------------------------------------------------------------------------
// executeConfigListen dual-batch executor tests (#629).
//
// scriptedProxy is a programmable IConfigProxy fake: it records every
// ConfigBatchListenRequest it sees and replies via the scripted reply func.
// It embeds the (nil) IConfigProxy interface so that any method the tests
// don't need and don't override panics on a nil-interface dispatch -- an
// unexpected call fails the test loudly instead of silently succeeding.
// ---------------------------------------------------------------------------

type scriptedProxy struct {
	IConfigProxy // embedded, intentionally nil: unoverridden methods panic if called

	mu   sync.Mutex
	sent []*rpc_request.ConfigBatchListenRequest

	reply         func(req *rpc_request.ConfigBatchListenRequest) (rpc_response.IResponse, error)
	queryConfigFn func(dataId, group, tenant string, timeout uint64, notify bool, client *ConfigClient) (*rpc_response.ConfigQueryResponse, error)
}

func (p *scriptedProxy) requestProxy(rpcClient *rpc.RpcClient, req rpc_request.IRequest, timeout uint64) (rpc_response.IResponse, error) {
	if blr, ok := req.(*rpc_request.ConfigBatchListenRequest); ok {
		p.mu.Lock()
		p.sent = append(p.sent, blr)
		p.mu.Unlock()
		return p.reply(blr)
	}
	return nil, errors.New("unexpected request in test")
}

func (p *scriptedProxy) createRpcClient(ctx context.Context, taskId string, c *ConfigClient) *rpc.RpcClient {
	return nil
}

func (p *scriptedProxy) getRpcClient(c *ConfigClient) *rpc.RpcClient {
	return nil
}

// queryConfig is only wired up for tests that exercise the changed-key
// refresh path; other tests never trigger it, and calling it without
// queryConfigFn set (a nil func) panics, preserving the "unexpected call
// fails the test" property.
func (p *scriptedProxy) queryConfig(dataId, group, tenant string, timeout uint64, notify bool, client *ConfigClient) (*rpc_response.ConfigQueryResponse, error) {
	return p.queryConfigFn(dataId, group, tenant, timeout, notify, client)
}

func (p *scriptedProxy) sentSnapshot() []*rpc_request.ConfigBatchListenRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]*rpc_request.ConfigBatchListenRequest, len(p.sent))
	copy(out, p.sent)
	return out
}

func batchListenSuccess(changed ...model.ConfigContext) *rpc_response.ConfigChangeBatchListenResponse {
	return &rpc_response.ConfigChangeBatchListenResponse{
		Response:       &rpc_response.Response{Success: true},
		ChangedConfigs: changed,
	}
}

// newExecutorTestClient builds a ConfigClient wired to proxy without starting
// the background executeConfigListen loop (startInternal is never called), so
// tests can drive client.executeConfigListen() directly and deterministically
// -- no concurrent background round can interleave with the call under test.
func newExecutorTestClient(proxy IConfigProxy) *ConfigClient {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*serverConfigWithOptions})
	_ = nc.SetClientConfig(*clientConfigWithOptions)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})

	client := &ConfigClient{}
	client.ctx, client.cancel = context.WithCancel(context.Background())
	client.INacosClient = &nc
	client.configFilterChainManager = filter.NewConfigFilterChainManager()
	client.configProxy = proxy
	client.holder = newConfigCacheHolder()
	client.listenExecute = make(chan struct{}, 1)
	return client
}

func testConfigKey(t *testing.T, client *ConfigClient, dataId, group string) string {
	t.Helper()
	clientConfig, err := client.GetClientConfig()
	require.NoError(t, err)
	return util.GetConfigCacheKey(dataId, group, clientConfig.NamespaceId)
}

func noopOnChange(string, string, string, string) {}

// TestExecuteConfigListen_CancelSendsListenFalseBatch is the canonical RED
// test for #629: cancelling the only listened key must make the executor
// send a Listen=false batch naming that key. The pre-fix single-batch
// executor never distinguishes discarded entries from active ones, so it
// either omits the request or sends it with Listen=true, never notifying the
// server that the key should stop being pushed.
func TestExecuteConfigListen_CancelSendsListenFalseBatch(t *testing.T) {
	p := &scriptedProxy{
		reply: func(req *rpc_request.ConfigBatchListenRequest) (rpc_response.IResponse, error) {
			return batchListenSuccess(), nil
		},
	}
	client := newExecutorTestClient(p)

	param := vo.ConfigParam{DataId: "d1", Group: "g1", OnChange: noopOnChange}
	require.NoError(t, client.ListenConfig(param))
	require.NoError(t, client.CancelListenConfig(param))

	client.executeConfigListen()

	sent := p.sentSnapshot()
	require.Len(t, sent, 1, "cancelling the only listened key must produce exactly one batch request")
	assert.False(t, sent[0].Listen, "cancel batch must be sent with Listen=false")

	key := testConfigKey(t, client, "d1", "g1")
	found := false
	for _, ctx := range sent[0].ConfigListenContexts {
		if util.GetConfigCacheKey(ctx.DataId, ctx.Group, ctx.Tenant) == key {
			found = true
		}
	}
	assert.True(t, found, "cancel batch must include the cancelled key")
}

// TestExecuteConfigListen_CancelBatchReconciliation covers reconciliation
// after the Listen=false batch response: success reaps the entry via
// removeIfDiscarded, while a transport error or a non-success response must
// leave the entry in place so it is retried on the next round.
func TestExecuteConfigListen_CancelBatchReconciliation(t *testing.T) {
	t.Run("success reaps entry", func(t *testing.T) {
		p := &scriptedProxy{
			reply: func(req *rpc_request.ConfigBatchListenRequest) (rpc_response.IResponse, error) {
				return batchListenSuccess(), nil
			},
		}
		client := newExecutorTestClient(p)
		param := vo.ConfigParam{DataId: "d2", Group: "g2", OnChange: noopOnChange}
		require.NoError(t, client.ListenConfig(param))
		require.NoError(t, client.CancelListenConfig(param))

		client.executeConfigListen()

		key := testConfigKey(t, client, "d2", "g2")
		_, ok := client.holder.get(key)
		assert.False(t, ok, "entry must be reaped after a successful cancel batch")
	})

	t.Run("transport error keeps entry for retry", func(t *testing.T) {
		p := &scriptedProxy{
			reply: func(req *rpc_request.ConfigBatchListenRequest) (rpc_response.IResponse, error) {
				return nil, errors.New("boom")
			},
		}
		client := newExecutorTestClient(p)
		param := vo.ConfigParam{DataId: "d3", Group: "g3", OnChange: noopOnChange}
		require.NoError(t, client.ListenConfig(param))
		require.NoError(t, client.CancelListenConfig(param))

		client.executeConfigListen()

		key := testConfigKey(t, client, "d3", "g3")
		cd, ok := client.holder.get(key)
		require.True(t, ok, "entry must be kept for retry when the cancel batch errors")
		cd.mu.Lock()
		defer cd.mu.Unlock()
		assert.True(t, cd.discard)
	})

	t.Run("non-success response keeps entry for retry", func(t *testing.T) {
		p := &scriptedProxy{
			reply: func(req *rpc_request.ConfigBatchListenRequest) (rpc_response.IResponse, error) {
				return &rpc_response.ConfigChangeBatchListenResponse{Response: &rpc_response.Response{Success: false}}, nil
			},
		}
		client := newExecutorTestClient(p)
		param := vo.ConfigParam{DataId: "d3b", Group: "g3b", OnChange: noopOnChange}
		require.NoError(t, client.ListenConfig(param))
		require.NoError(t, client.CancelListenConfig(param))

		client.executeConfigListen()

		key := testConfigKey(t, client, "d3b", "g3b")
		_, ok := client.holder.get(key)
		require.True(t, ok, "entry must be kept for retry when the cancel batch response is not success")
	})
}

// TestExecuteConfigListen_CancelAndListenBatchesDoNotCrossContaminate checks
// that when both a discarded key and an active key are pending, the executor
// sends two independent batches (cancel first, then listen), each carrying
// only the keys it owns.
func TestExecuteConfigListen_CancelAndListenBatchesDoNotCrossContaminate(t *testing.T) {
	p := &scriptedProxy{
		reply: func(req *rpc_request.ConfigBatchListenRequest) (rpc_response.IResponse, error) {
			return batchListenSuccess(), nil
		},
	}
	client := newExecutorTestClient(p)

	cancelParam := vo.ConfigParam{DataId: "cd", Group: "cg", OnChange: noopOnChange}
	listenParam := vo.ConfigParam{DataId: "ld", Group: "lg", OnChange: noopOnChange}
	require.NoError(t, client.ListenConfig(cancelParam))
	require.NoError(t, client.ListenConfig(listenParam))
	require.NoError(t, client.CancelListenConfig(cancelParam))

	client.executeConfigListen()

	sent := p.sentSnapshot()
	require.Len(t, sent, 2, "both a cancel batch and a listen batch must be sent")
	assert.False(t, sent[0].Listen, "the cancel batch must be sent before the listen batch")
	assert.True(t, sent[1].Listen, "the listen batch must be sent after the cancel batch")

	cancelKey := testConfigKey(t, client, "cd", "cg")
	listenKey := testConfigKey(t, client, "ld", "lg")

	containsKey := func(req *rpc_request.ConfigBatchListenRequest, key string) bool {
		for _, ctx := range req.ConfigListenContexts {
			if util.GetConfigCacheKey(ctx.DataId, ctx.Group, ctx.Tenant) == key {
				return true
			}
		}
		return false
	}

	assert.True(t, containsKey(sent[0], cancelKey), "cancel batch must contain the cancelled key")
	assert.False(t, containsKey(sent[0], listenKey), "cancel batch must not contain the active key")
	assert.True(t, containsKey(sent[1], listenKey), "listen batch must contain the active key")
	assert.False(t, containsKey(sent[1], cancelKey), "listen batch must not contain the cancelled key")
}

// TestExecuteConfigListen_ChangedKeyNotifiesAllListeners verifies that a key
// reported in ChangedConfigs is refreshed via refreshContentAndCheck, which
// in turn drives cacheData.notifyListeners, and that every listener
// registered on that key is notified of the change in a single round.
func TestExecuteConfigListen_ChangedKeyNotifiesAllListeners(t *testing.T) {
	got := make(chan string, 2)
	p := &scriptedProxy{}
	p.reply = func(req *rpc_request.ConfigBatchListenRequest) (rpc_response.IResponse, error) {
		return batchListenSuccess(model.ConfigContext{DataId: "d4", Group: "g4"}), nil
	}
	p.queryConfigFn = func(dataId, group, tenant string, timeout uint64, notify bool, client *ConfigClient) (*rpc_response.ConfigQueryResponse, error) {
		return &rpc_response.ConfigQueryResponse{
			Response: &rpc_response.Response{Success: true},
			Content:  "new-content",
		}, nil
	}
	client := newExecutorTestClient(p)

	for i := 0; i < 2; i++ {
		idx := i
		require.NoError(t, client.ListenConfig(vo.ConfigParam{
			DataId: "d4", Group: "g4",
			OnChange: func(ns, g, d, data string) { got <- fmt.Sprintf("%d:%s", idx, data) },
		}))
	}

	client.executeConfigListen()

	received := map[string]bool{}
	deadline := time.After(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case v := <-got:
			received[v] = true
		case <-deadline:
			t.Fatalf("timed out waiting for listener notifications, got so far: %v", received)
		}
	}
	assert.True(t, received["0:new-content"], "listener 0 must be notified of the change")
	assert.True(t, received["1:new-content"], "listener 1 must be notified of the change")
}

// TestExecuteConfigListen_ReviveDuringCancelSendPreventsRemoval simulates a
// ListenConfig racing in while the Listen=false cancel batch for the same key
// is in flight: by the time the response comes back the entry has already
// been revived, so removeIfDiscarded must refuse to delete it.
func TestExecuteConfigListen_ReviveDuringCancelSendPreventsRemoval(t *testing.T) {
	var client *ConfigClient
	param := vo.ConfigParam{DataId: "d5", Group: "g5", OnChange: noopOnChange}

	p := &scriptedProxy{}
	p.reply = func(req *rpc_request.ConfigBatchListenRequest) (rpc_response.IResponse, error) {
		if !req.Listen {
			// Simulate a concurrent ListenConfig reviving the entry while
			// the cancel batch is still in flight.
			require.NoError(t, client.ListenConfig(param))
		}
		return batchListenSuccess(), nil
	}
	client = newExecutorTestClient(p)

	require.NoError(t, client.ListenConfig(param))
	require.NoError(t, client.CancelListenConfig(param))

	client.executeConfigListen()

	key := testConfigKey(t, client, "d5", "g5")
	cd, ok := client.holder.get(key)
	require.True(t, ok, "revived entry must not be removed by the in-flight cancel batch")
	cd.mu.Lock()
	defer cd.mu.Unlock()
	assert.False(t, cd.discard, "revived entry must not be discarded")
}

// TestListenConfigOnClosedClientReturnsError verifies ListenConfig rejects
// registration once the client has been closed (#904 semantics: isClosed
// must be checked under client.mutex at the ListenConfig entry point rather
// than silently registering a listener that will never be served again), and
// that the error is the exported sentinel so callers can errors.Is against it
// instead of matching on error text.
func TestListenConfigOnClosedClientReturnsError(t *testing.T) {
	client := createConfigClientTest()
	client.CloseClient()

	err := client.ListenConfig(vo.ConfigParam{DataId: "d", Group: "g", OnChange: noopOnChange})
	require.Error(t, err, "ListenConfig on a closed client must return an error")
	assert.True(t, errors.Is(err, ErrConfigClientClosed), "error must be (or wrap) ErrConfigClientClosed, got: %v", err)
}

// TestListenConfigCommitRollsBackWhenCloseWinsRace deterministically
// exercises the exact linearization gap Finding 1 fixes: ListenConfig's
// isClosed check was read-then-released under client.mutex, but the commit
// that follows (getOrCreate + reviveAndAddListener, including the seed's disk
// I/O) ran outside that lock. A CloseClient landing in that window used to
// register a live listener on an already-shut-down client (whose listen
// executor goroutine has already exited for good) while ListenConfig still
// reported success -- a "ghost listener" that can never be served again.
//
// Forcing that interleaving through a real concurrent ListenConfig call is
// inherently racy (see TestListenConfigRaceWithCloseClientNeverLeavesLiveListenerOnClosedClient
// below for the statistical version and why it can't be made exact without a
// test-only hook). This test instead drives the same two steps ListenConfig
// performs -- commit, then the post-commit closed re-check -- directly, with
// CloseClient() deterministically sequenced in between, and asserts the
// re-check (client.rejectIfClosedAfterCommit) detects the closed client and
// rolls the commit back via cData.markDiscard(). This fails to compile/RED
// against the pre-fix code, which had no such re-check at all.
func TestListenConfigCommitRollsBackWhenCloseWinsRace(t *testing.T) {
	client := createConfigClientTest()
	p := vo.ConfigParam{DataId: "d", Group: "g", OnChange: noopOnChange}
	clientConfig, err := client.GetClientConfig()
	require.NoError(t, err)
	key := util.GetConfigCacheKey(p.DataId, p.Group, clientConfig.NamespaceId)

	cData := client.holder.getOrCreate(key, func() *cacheData {
		return &cacheData{dataId: p.DataId, group: p.Group, tenant: clientConfig.NamespaceId}
	})
	// Step 1: the commit ListenConfig performs before its post-commit check.
	cData.reviveAndAddListener(p.OnChange)
	// Step 2: CloseClient lands immediately afterward -- the exact window
	// Finding 1 closes.
	client.CloseClient()

	// Step 3: the same post-commit re-check ListenConfig runs.
	err = client.rejectIfClosedAfterCommit(cData)
	require.ErrorIs(t, err, ErrConfigClientClosed)

	cData.mu.Lock()
	defer cData.mu.Unlock()
	assert.True(t, cData.discard, "a commit landing on an already-closed client must be rolled back")
	assert.Empty(t, cData.listeners, "no live listener may remain once the commit is rolled back")
}

// TestListenConfigRaceWithCloseClientNeverLeavesLiveListenerOnClosedClient is
// a race exerciser, not the ghost-listener regression guard: that semantics
// (a commit landing after close must be rolled back) is pinned deterministically
// by TestListenConfigCommitRollsBackWhenCloseWinsRace above. This test hammers
// many fresh client/key pairs, racing the real ListenConfig against
// CloseClient on each, purely so the locking added for Finding 1 gets
// exercised under -race (any data race there would still be a real bug this
// catches). It previously also asserted a percentage threshold on how often a
// live listener survived a nil-error ListenConfig -- that "benign occurrence"
// rate (the post-commit re-check legitimately observing isClosed==false a few
// instructions before a racing CloseClient's own critical section completes,
// which is correct, unproblematic behavior, not a ghost listener) is
// machine-dependent and flaked in CI (52/500 = 10.4%, against a 10%
// threshold). That threshold assertion is removed; only scheduling-independent
// invariants are checked now:
//
//  1. discard==true && len(listeners)>0 must never be observed on the entry,
//     during the race or after it settles (the Task-3 invariant already
//     covered for Cancel/Listen races by
//     TestListenCancelConcurrentNeverLeavesDiscardedWithListeners -- checked
//     again here since the Finding 1 rollback also calls markDiscard()).
//  2. Once CloseClient has definitely returned, one further ListenConfig call
//     on the same key must fail with an error satisfying errors.Is(err,
//     ErrConfigClientClosed), and must not grow the entry's listener count.
func TestListenConfigRaceWithCloseClientNeverLeavesLiveListenerOnClosedClient(t *testing.T) {
	const iterations = 500

	for i := 0; i < iterations; i++ {
		client := createConfigClientTest()
		p := vo.ConfigParam{
			DataId:   fmt.Sprintf("race-close-d-%d", i),
			Group:    "race-close-g",
			OnChange: noopOnChange,
		}

		clientConfig, err := client.GetClientConfig()
		require.NoError(t, err)
		key := util.GetConfigCacheKey(p.DataId, p.Group, clientConfig.NamespaceId)
		listenerCount := func() int {
			cData, ok := client.holder.get(key)
			if !ok {
				return 0
			}
			cData.mu.Lock()
			defer cData.mu.Unlock()
			return len(cData.listeners)
		}

		var inconsistent atomic.Bool
		stop := make(chan struct{})
		var observer sync.WaitGroup
		observer.Add(1)
		go func() {
			defer observer.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if cData, ok := client.holder.get(key); ok {
					cData.mu.Lock()
					if cData.discard && len(cData.listeners) > 0 {
						inconsistent.Store(true)
					}
					cData.mu.Unlock()
				}
			}
		}()

		var wg sync.WaitGroup
		wg.Add(2)
		start := make(chan struct{})
		go func() {
			defer wg.Done()
			<-start
			_ = client.ListenConfig(p)
		}()
		go func() {
			defer wg.Done()
			<-start
			client.CloseClient()
		}()
		close(start)
		wg.Wait()
		close(stop)
		observer.Wait()

		assert.False(t, inconsistent.Load(),
			"entry must never be observed with discard==true and non-empty listeners")

		// Both goroutines have finished, so CloseClient has definitely
		// completed: the client is now closed for good.
		client.mutex.Lock()
		closed := client.isClosed
		client.mutex.Unlock()
		require.True(t, closed, "CloseClient must have completed by now")

		before := listenerCount()
		err = client.ListenConfig(p)
		require.Error(t, err, "ListenConfig on a definitely-closed client must fail")
		assert.True(t, errors.Is(err, ErrConfigClientClosed),
			"error must be (or wrap) ErrConfigClientClosed, got: %v", err)
		assert.Equal(t, before, listenerCount(),
			"a rejected ListenConfig on a closed client must not grow the listener count")
	}
}

// TestCancelListenConfigOnClosedClientDoesNotPanic verifies CancelListenConfig
// remains safe to call after the client has been closed.
func TestCancelListenConfigOnClosedClientDoesNotPanic(t *testing.T) {
	client := createConfigClientTest()
	p := vo.ConfigParam{DataId: "d", Group: "g", OnChange: noopOnChange}
	require.NoError(t, client.ListenConfig(p))
	client.CloseClient()

	assert.NotPanics(t, func() {
		err := client.CancelListenConfig(p)
		assert.NoError(t, err)
	})
}

// TestAsyncNotifyListenConfigDoesNotLeakAfterClose is a regression test for
// asyncNotifyListenConfig spawning a goroutine that would block forever
// trying to send on client.listenExecute once the listen executor loop
// (startInternal) has already returned via <-client.ctx.Done() after
// CloseClient -- nothing ever receives from listenExecute again past that
// point, so an unconditional send used to leak one goroutine per call
// (e.g. every CancelListenConfig after close, since it calls
// asyncNotifyListenConfig unconditionally). The send is now selected against
// <-client.ctx.Done() too, so each spawned goroutine exits as soon as the
// client is closed instead of leaking.
func TestAsyncNotifyListenConfigDoesNotLeakAfterClose(t *testing.T) {
	client := createConfigClientTest()
	p := vo.ConfigParam{DataId: "d", Group: "g", OnChange: noopOnChange}
	require.NoError(t, client.ListenConfig(p))
	client.CloseClient()

	runtime.GC()
	before := runtime.NumGoroutine()

	const calls = 200
	for i := 0; i < calls; i++ {
		// CancelListenConfig calls asyncNotifyListenConfig unconditionally
		// whenever the key is known; exercise the same path CloseClient
		// leaves callers with.
		_ = client.CancelListenConfig(p)
	}

	require.Eventually(t, func() bool {
		runtime.GC()
		return runtime.NumGoroutine() <= before+5
	}, time.Second, 10*time.Millisecond,
		"asyncNotifyListenConfig goroutines must not leak once the client is closed")
}

// TestCloseClientTwiceDoesNotPanic verifies CloseClient is idempotent.
func TestCloseClientTwiceDoesNotPanic(t *testing.T) {
	client := createConfigClientTest()
	assert.NotPanics(t, func() {
		client.CloseClient()
		client.CloseClient()
	})
}

// ---------------------------------------------------------------------------
// Async delivery regression tests (review round-1 P1): user callbacks must
// never run on the listen executor goroutine, or one slow callback blocks
// every other config's queries, notifications and cancel batches.
// ---------------------------------------------------------------------------

// scriptedRounds replies each executor round with the next scripted response
// and empty (no-change) success afterwards.
type scriptedRounds struct {
	mu     sync.Mutex
	rounds []*rpc_response.ConfigChangeBatchListenResponse
}

func (s *scriptedRounds) reply(req *rpc_request.ConfigBatchListenRequest) (rpc_response.IResponse, error) {
	if !req.Listen {
		return batchListenSuccess(), nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.rounds) == 0 {
		return batchListenSuccess(), nil
	}
	next := s.rounds[0]
	s.rounds = s.rounds[1:]
	return next, nil
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v: %s", timeout, msg)
}

// TestBlockingCallbackDoesNotBlockOtherConfigs drives the REAL executor loop
// (startInternal) and verifies that config A's blocking callback does not
// prevent config B's change (arriving in a later round) from being delivered.
func TestBlockingCallbackDoesNotBlockOtherConfigs(t *testing.T) {
	proxy := &scriptedProxy{}
	client := newExecutorTestClient(proxy)
	defer client.cancel()

	clientConfig, err := client.GetClientConfig()
	require.NoError(t, err)
	ns := clientConfig.NamespaceId

	rounds := &scriptedRounds{rounds: []*rpc_response.ConfigChangeBatchListenResponse{
		batchListenSuccess(model.ConfigContext{DataId: "block-a", Group: "g", Tenant: ns}),
		batchListenSuccess(model.ConfigContext{DataId: "block-b", Group: "g", Tenant: ns}),
	}}
	proxy.reply = rounds.reply
	proxy.queryConfigFn = func(dataId, group, tenant string, timeout uint64, notify bool, c *ConfigClient) (*rpc_response.ConfigQueryResponse, error) {
		return &rpc_response.ConfigQueryResponse{
			Response: &rpc_response.Response{Success: true},
			Content:  "content-" + dataId,
		}, nil
	}

	aEntered := make(chan struct{})
	aRelease := make(chan struct{})
	require.NoError(t, client.ListenConfig(vo.ConfigParam{DataId: "block-a", Group: "g", OnChange: func(_, _, _, _ string) {
		close(aEntered)
		<-aRelease
	}}))
	bGot := make(chan string, 4)
	require.NoError(t, client.ListenConfig(vo.ConfigParam{DataId: "block-b", Group: "g", OnChange: func(_, _, _, data string) {
		bGot <- data
	}}))

	client.startInternal()
	client.asyncNotifyListenConfig() // round 1: A changes, callback blocks

	select {
	case <-aEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("A's callback never entered")
	}

	client.asyncNotifyListenConfig() // round 2: B changes

	// B must be delivered while A is still blocked.
	select {
	case data := <-bGot:
		assert.Equal(t, "content-block-b", data)
	case <-time.After(5 * time.Second):
		close(aRelease)
		t.Fatal("B's callback never fired while A was blocked: blocking callback stalls the executor loop")
	}
	close(aRelease)
}

// TestCancelProceedsWhileCallbackBlocked verifies that CancelListenConfig's
// listen=false batch is still sent by the executor while another config's
// callback is blocked.
func TestCancelProceedsWhileCallbackBlocked(t *testing.T) {
	proxy := &scriptedProxy{}
	client := newExecutorTestClient(proxy)
	defer client.cancel()

	clientConfig, err := client.GetClientConfig()
	require.NoError(t, err)
	ns := clientConfig.NamespaceId

	rounds := &scriptedRounds{rounds: []*rpc_response.ConfigChangeBatchListenResponse{
		batchListenSuccess(model.ConfigContext{DataId: "block-a2", Group: "g", Tenant: ns}),
	}}
	proxy.reply = rounds.reply
	proxy.queryConfigFn = func(dataId, group, tenant string, timeout uint64, notify bool, c *ConfigClient) (*rpc_response.ConfigQueryResponse, error) {
		return &rpc_response.ConfigQueryResponse{
			Response: &rpc_response.Response{Success: true},
			Content:  "content-" + dataId,
		}, nil
	}

	aEntered := make(chan struct{})
	aRelease := make(chan struct{})
	require.NoError(t, client.ListenConfig(vo.ConfigParam{DataId: "block-a2", Group: "g", OnChange: func(_, _, _, _ string) {
		close(aEntered)
		<-aRelease
	}}))
	require.NoError(t, client.ListenConfig(vo.ConfigParam{DataId: "cancel-c", Group: "g", OnChange: noopOnChange}))

	client.startInternal()
	client.asyncNotifyListenConfig()

	select {
	case <-aEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("A's callback never entered")
	}

	require.NoError(t, client.CancelListenConfig(vo.ConfigParam{DataId: "cancel-c", Group: "g"}))

	waitUntil(t, 5*time.Second, func() bool {
		for _, req := range proxy.sentSnapshot() {
			if req.Listen {
				continue
			}
			for _, lc := range req.ConfigListenContexts {
				if lc.DataId == "cancel-c" {
					return true
				}
			}
		}
		return false
	}, "listen=false batch for cancel-c never sent while A's callback was blocked")
	close(aRelease)
}

// TestBellStormDoesNotAccumulateGoroutines verifies asyncNotifyListenConfig
// does not spawn one blocked goroutine per pending notification while the
// executor is busy.
func TestBellStormDoesNotAccumulateGoroutines(t *testing.T) {
	client := newExecutorTestClient(&scriptedProxy{})
	defer client.cancel()
	// no executor loop running at all: worst case for pending bells

	runtime.GC()
	before := runtime.NumGoroutine()
	for i := 0; i < 40; i++ {
		client.asyncNotifyListenConfig()
	}
	time.Sleep(200 * time.Millisecond)
	after := runtime.NumGoroutine()
	assert.LessOrEqual(t, after-before, 5,
		"bell sends must coalesce, not pile up one goroutine per notification (before=%d after=%d)", before, after)
}
