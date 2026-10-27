//go:build integration

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

package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nacos-group/nacos-sdk-go/v3/clients"
	"github.com/nacos-group/nacos-sdk-go/v3/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v3/model"
	"github.com/nacos-group/nacos-sdk-go/v3/vo"
)

// configEventRecorder is a mutex-guarded collector for config fuzzy watch
// events. Callbacks fire on the holder's drain goroutine, never on the test
// goroutine, so every access goes through the mutex (-race clean).
type configEventRecorder struct {
	mu     sync.Mutex
	events []model.ConfigFuzzyWatchChangeEvent
}

func (r *configEventRecorder) record(e model.ConfigFuzzyWatchChangeEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *configEventRecorder) find(changedType, dataId string) (model.ConfigFuzzyWatchChangeEvent, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.events {
		if e.ChangedType == changedType && e.DataId == dataId {
			return e, true
		}
	}
	return model.ConfigFuzzyWatchChangeEvent{}, false
}

func (r *configEventRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

func (r *configEventRecorder) syncTypes() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	for i, e := range r.events {
		out[i] = e.SyncType
	}
	return out
}

func configFuzzySuffix() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}

func publishFuzzyConfig(t *testing.T, c config_client.IConfigClient, dataId string) {
	t.Helper()
	ok, err := c.PublishConfig(vo.ConfigParam{DataId: dataId, Group: testGroup, Content: "fuzzy-v1"})
	require.NoError(t, err, "publish %s", dataId)
	require.True(t, ok, "publish %s should return true", dataId)
}

// TestIntegrationConfigFuzzyWatch: a prefix pattern watch must see ADD_CONFIG
// for a config published (by an independent client) after the watch, converge
// MatchedGroupKeys onto it, and see DELETE_CONFIG when it is removed.
// 3.x only.
func TestIntegrationConfigFuzzyWatch(t *testing.T) {
	if !isNacosV3Admin(t) {
		t.Skip("config fuzzy watch requires nacos 3.x")
	}

	watchClient, err := clients.NewConfigClient(clientParam(t))
	require.NoError(t, err, "create config client for fuzzy watch")
	defer watchClient.CloseClient()
	pubClient, err := clients.NewConfigClient(clientParam(t))
	require.NoError(t, err, "create config client for publishing")
	defer pubClient.CloseClient()

	suffix := configFuzzySuffix()
	prefix := fmt.Sprintf("it-cfz-%s-", suffix)
	dataId := prefix + "1"
	defer func() {
		_, _ = pubClient.DeleteConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
	}()

	recorder := &configEventRecorder{}
	handle, err := watchClient.FuzzyWatchConfig(&vo.ConfigFuzzyWatchParam{
		DataIdPattern: prefix + "*",
		Group:         testGroup,
		OnChange:      recorder.record,
	})
	require.NoError(t, err, "fuzzy watch config")
	defer handle.Cancel()

	// Initial sync must finish (nothing matches yet) before we publish, so
	// the ADD below is unambiguously a post-init change notify.
	initCtx, cancelInit := context.WithTimeout(context.Background(), waitTimeout)
	defer cancelInit()
	keys, err := handle.MatchedGroupKeys(initCtx)
	require.NoError(t, err, "initial sync should complete")
	assert.Empty(t, keys, "nothing matches the fresh pattern yet")

	publishFuzzyConfig(t, pubClient, dataId)

	require.Eventually(t, func() bool {
		_, found := recorder.find(constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, dataId)
		return found
	}, waitTimeout, waitInterval, "should observe ADD_CONFIG for the matching config")
	add, _ := recorder.find(constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, dataId)
	assert.Equal(t, testGroup, add.Group)

	require.Eventually(t, func() bool {
		ks, e := handle.MatchedGroupKeys(initCtx)
		return e == nil && len(ks) == 1 && strings.Contains(ks[0], dataId)
	}, waitTimeout, waitInterval, "MatchedGroupKeys should converge to exactly the published config")

	ok, err := pubClient.DeleteConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
	require.NoError(t, err, "delete config")
	require.True(t, ok)

	require.Eventually(t, func() bool {
		_, found := recorder.find(constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG, dataId)
		return found
	}, waitTimeout, waitInterval, "should observe DELETE_CONFIG after deletion")
	t.Logf("syncTypes observed: %v", recorder.syncTypes())
}

// TestIntegrationConfigFuzzyWatchSecondWatcherReplay: a second watcher on an
// already-synced pattern is replayed the existing match as ADD_CONFIG tagged
// FUZZY_WATCH_INIT_NOTIFY, without a new server-side change; the first watcher
// is not replayed to.
func TestIntegrationConfigFuzzyWatchSecondWatcherReplay(t *testing.T) {
	if !isNacosV3Admin(t) {
		t.Skip("config fuzzy watch requires nacos 3.x")
	}

	watchClient, err := clients.NewConfigClient(clientParam(t))
	require.NoError(t, err, "create config client for fuzzy watch")
	defer watchClient.CloseClient()
	pubClient, err := clients.NewConfigClient(clientParam(t))
	require.NoError(t, err, "create config client for publishing")
	defer pubClient.CloseClient()

	suffix := configFuzzySuffix()
	prefix := fmt.Sprintf("it-cfz-replay-%s-", suffix)
	dataId := prefix + "1"
	defer func() {
		_, _ = pubClient.DeleteConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
	}()

	watcher1 := &configEventRecorder{}
	handle1, err := watchClient.FuzzyWatchConfig(&vo.ConfigFuzzyWatchParam{
		DataIdPattern: prefix + "*", Group: testGroup, OnChange: watcher1.record,
	})
	require.NoError(t, err, "fuzzy watch (watcher1)")
	defer handle1.Cancel()

	// Await handle1's initial sync before publishing: the WATCH is sent
	// asynchronously by the reconcile worker, so publishing earlier could let
	// the config arrive as FUZZY_WATCH_INIT_NOTIFY instead of a post-init
	// change, making the zero-replay assertion below flaky.
	initCtx, cancelInit := context.WithTimeout(context.Background(), waitTimeout)
	defer cancelInit()
	_, err = handle1.MatchedGroupKeys(initCtx)
	require.NoError(t, err, "watcher1 initial sync should complete")

	publishFuzzyConfig(t, pubClient, dataId)

	// watcher1 observing the ADD is the externally visible proof that the
	// pattern's shared matched-key set already contains the config, which is
	// what makes watcher2's replay deterministic rather than racing the push.
	require.Eventually(t, func() bool {
		_, found := watcher1.find(constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, dataId)
		return found
	}, waitTimeout, waitInterval, "watcher1 should observe ADD_CONFIG before watcher2 registers")

	watcher2 := &configEventRecorder{}
	handle2, err := watchClient.FuzzyWatchConfig(&vo.ConfigFuzzyWatchParam{
		DataIdPattern: prefix + "*", Group: testGroup, OnChange: watcher2.record,
	})
	require.NoError(t, err, "fuzzy watch (watcher2)")
	defer handle2.Cancel()

	require.Eventually(t, func() bool {
		ev, found := watcher2.find(constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, dataId)
		return found && ev.SyncType == constant.FUZZY_WATCH_INIT_NOTIFY
	}, waitTimeout, waitInterval, "watcher2 should be replayed the existing match tagged FUZZY_WATCH_INIT_NOTIFY")

	initReplays := 0
	for _, st := range watcher1.syncTypes() {
		if st == constant.FUZZY_WATCH_INIT_NOTIFY {
			initReplays++
		}
	}
	assert.Zero(t, initReplays, "watcher1 must not receive watcher2's replay")
	t.Logf("watcher1 syncTypes: %v, watcher2 syncTypes: %v", watcher1.syncTypes(), watcher2.syncTypes())
}

// TestIntegrationConfigFuzzyWatchCancelNoResurrection: after handle.Cancel()
// a newly published matching config never reaches the canceled callback, and
// re-watching the same pattern works normally again.
func TestIntegrationConfigFuzzyWatchCancelNoResurrection(t *testing.T) {
	if !isNacosV3Admin(t) {
		t.Skip("config fuzzy watch requires nacos 3.x")
	}

	watchClient, err := clients.NewConfigClient(clientParam(t))
	require.NoError(t, err, "create config client for fuzzy watch")
	defer watchClient.CloseClient()
	pubClient, err := clients.NewConfigClient(clientParam(t))
	require.NoError(t, err, "create config client for publishing")
	defer pubClient.CloseClient()

	suffix := configFuzzySuffix()
	prefix := fmt.Sprintf("it-cfz-cancel-%s-", suffix)
	pattern := prefix + "*"
	before := prefix + "1"
	after := prefix + "2"
	rewatched := prefix + "3"
	for _, id := range []string{before, after, rewatched} {
		id := id
		defer func() {
			_, _ = pubClient.DeleteConfig(vo.ConfigParam{DataId: id, Group: testGroup})
		}()
	}

	canceled := &configEventRecorder{}
	handle, err := watchClient.FuzzyWatchConfig(&vo.ConfigFuzzyWatchParam{
		DataIdPattern: pattern, Group: testGroup, OnChange: canceled.record,
	})
	require.NoError(t, err, "fuzzy watch")

	publishFuzzyConfig(t, pubClient, before)
	require.Eventually(t, func() bool {
		_, found := canceled.find(constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, before)
		return found
	}, waitTimeout, waitInterval, "watch should observe ADD_CONFIG before cancel")

	handle.Cancel()
	// Cancel is synchronous locally (callback detached before it returns);
	// the server-side CANCEL is sent asynchronously by the reconcile worker.
	// This settle sleep only gives the server-side teardown a head start so the
	// negative window below also exercises it; it is NOT what the assertion
	// relies on - local detachment alone already guarantees silence.
	time.Sleep(3 * time.Second)
	preCancelCount := canceled.count()

	publishFuzzyConfig(t, pubClient, after)

	// Negative assertion: nothing to wait FOR, so every sample in the window
	// must find no event; one passing sample proves nothing.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, found := canceled.find(constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, after)
		require.False(t, found, "canceled callback must not observe a config published after cancel")
		time.Sleep(waitInterval)
	}
	assert.Equal(t, preCancelCount, canceled.count(), "no events of any kind should reach the canceled callback")

	// Re-watching the same pattern must work normally: it is synced afresh,
	// replaying the existing matches, then sees new changes.
	fresh := &configEventRecorder{}
	handle2, err := watchClient.FuzzyWatchConfig(&vo.ConfigFuzzyWatchParam{
		DataIdPattern: pattern, Group: testGroup, OnChange: fresh.record,
	})
	require.NoError(t, err, "re-watch same pattern")
	defer handle2.Cancel()

	require.Eventually(t, func() bool {
		_, found := fresh.find(constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, after)
		return found
	}, waitTimeout, waitInterval, "re-watch should observe the config published while canceled")

	publishFuzzyConfig(t, pubClient, rewatched)
	require.Eventually(t, func() bool {
		_, found := fresh.find(constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, rewatched)
		return found
	}, waitTimeout, waitInterval, "re-watch should observe new matching configs")
	assert.Equal(t, preCancelCount, canceled.count(), "the old canceled callback must stay silent after re-watch")
}

// TestIntegrationConfigFuzzyWatchNotSupportedOnV2: against a 2.x server (no
// fuzzyWatch ability) FuzzyWatchConfig fast-fails with a nil handle and
// ErrConfigFuzzyWatchNotSupported.
func TestIntegrationConfigFuzzyWatchNotSupportedOnV2(t *testing.T) {
	param, ok := fuzzyWatchV2ClientParam(t)
	if !ok {
		t.Skip("no nacos 2.x server available (neither the suite default nor NACOS2_SERVER_IP/NACOS2_SERVER_PORT)")
	}

	client, err := clients.NewConfigClient(param)
	require.NoError(t, err, "create config client against nacos 2.x")
	defer client.CloseClient()

	handle, err := client.FuzzyWatchConfig(&vo.ConfigFuzzyWatchParam{
		DataIdPattern: "it-cfz-notsupported-*",
		Group:         testGroup,
		OnChange:      func(model.ConfigFuzzyWatchChangeEvent) {},
	})
	assert.Nil(t, handle, "handle should be nil when the server does not support fuzzy watch")
	require.Error(t, err)
	assert.True(t, errors.Is(err, config_client.ErrConfigFuzzyWatchNotSupported),
		"error should be ErrConfigFuzzyWatchNotSupported, got: %v", err)
}
