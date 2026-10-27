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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/model"
	"github.com/nacos-group/nacos-sdk-go/v3/vo"
)

type sentFuzzyRequest struct {
	pattern      string
	watchType    string
	received     []string
	initializing bool
}

// scriptedRequester records every send and lets a test script each outcome and
// run a hook while the "RPC" is in flight.
type scriptedRequester struct {
	mu      sync.Mutex
	sent    []sentFuzzyRequest
	results []error // consumed in order; exhausted -> nil
	hook    func(req sentFuzzyRequest)
}

func (r *scriptedRequester) SendFuzzyWatchRequest(pattern, watchType string, received []string, initializing bool) error {
	req := sentFuzzyRequest{pattern, watchType, append([]string(nil), received...), initializing}
	r.mu.Lock()
	r.sent = append(r.sent, req)
	var err error
	if len(r.results) > 0 {
		err = r.results[0]
		r.results = r.results[1:]
	}
	hook := r.hook
	r.mu.Unlock()
	if hook != nil {
		hook(req)
	}
	return err
}

func (r *scriptedRequester) CheckFuzzyWatchSupported() error { return nil }

func (r *scriptedRequester) requests() []sentFuzzyRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]sentFuzzyRequest(nil), r.sent...)
}

func newReconcileHolder(t *testing.T) (*ConfigFuzzyWatchHolder, *scriptedRequester) {
	t.Helper()
	h := NewConfigFuzzyWatchHolder("public")
	h.failureBackoff = time.Millisecond
	req := &scriptedRequester{}
	h.SetRequester(req)
	return h, req
}

func consistent(c *fuzzyWatchContext) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.consistentWithServer
}

func syncContexts(keys ...string) []rpc_request.ConfigFuzzyWatchSyncContext {
	out := make([]rpc_request.ConfigFuzzyWatchSyncContext, 0, len(keys))
	for _, k := range keys {
		out = append(out, rpc_request.ConfigFuzzyWatchSyncContext{GroupKey: dispatchKey(k),
			ChangedType: constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG})
	}
	return out
}

func TestExecuteSyncWatchCarriesReceivedKeysSnapshot(t *testing.T) {
	h, req := newReconcileHolder(t)
	sink := &eventSink{}
	ctx, _ := registerDispatch(t, h, sink.cb)
	h.HandleSync(dispatchPattern, constant.FUZZY_WATCH_INIT_NOTIFY, syncContexts("app1", "app2"), 1, 1)

	h.executeSync()

	sent := req.requests()
	require.Len(t, sent, 1)
	assert.Equal(t, constant.FUZZY_WATCH_TYPE_WATCH, sent[0].watchType)
	assert.Equal(t, dispatchPattern, sent[0].pattern)
	assert.ElementsMatch(t, []string{dispatchKey("app1"), dispatchKey("app2")}, sent[0].received)
	assert.True(t, sent[0].initializing, "initial WATCH is sent before FINISH")
	assert.True(t, consistent(ctx))
}

func TestExecuteSyncConsistentFastPathSkipsSecondSend(t *testing.T) {
	h, req := newReconcileHolder(t)
	sink := &eventSink{}
	registerDispatch(t, h, sink.cb)
	h.executeSync()
	h.executeSync()
	assert.Len(t, req.requests(), 1, "consistent pattern is not re-sent before allSyncInterval")
}

func TestExecuteSyncStaleSuccessDoesNotMarkConsistent(t *testing.T) {
	h, req := newReconcileHolder(t)
	sink := &eventSink{}
	ctx, _ := registerDispatch(t, h, sink.cb)
	// Reconnect lands while the WATCH is in flight: epoch moves on.
	req.hook = func(sentFuzzyRequest) { h.ResetConsistenceStatus() }

	h.executeSync()

	assert.False(t, consistent(ctx), "stale response must not flip consistent")
	req.hook = nil
	h.executeSync()
	assert.Len(t, req.requests(), 2, "newer desired state gets its own send")
	assert.True(t, consistent(ctx))
}

func TestExecuteSyncDiscardedEmptySendsCancelEvenWhenConsistent(t *testing.T) {
	h, req := newReconcileHolder(t)
	sink := &eventSink{}
	ctx, id := registerDispatch(t, h, sink.cb)
	h.executeSync() // WATCH -> consistent
	require.True(t, consistent(ctx))

	h.RemoveWatcherByID(dispatchPattern, id)
	h.executeSync()

	sent := req.requests()
	require.Len(t, sent, 2)
	assert.Equal(t, constant.FUZZY_WATCH_TYPE_CANCEL_WATCH, sent[1].watchType)
	_, exists := h.get(dispatchPattern)
	assert.False(t, exists, "confirmed CANCEL removes the pattern")
}

func TestExecuteSyncRegisterDuringCancelRevivesPattern(t *testing.T) {
	h, req := newReconcileHolder(t)
	sink := &eventSink{}
	ctx, id := registerDispatch(t, h, sink.cb)
	h.executeSync()
	h.RemoveWatcherByID(dispatchPattern, id)
	req.hook = func(r sentFuzzyRequest) {
		if r.watchType == constant.FUZZY_WATCH_TYPE_CANCEL_WATCH {
			_, err := h.RegisterWatcher(&vo.ConfigFuzzyWatchParam{DataIdPattern: "app*", Group: "DEFAULT_GROUP", OnChange: sink.cb})
			assert.NoError(t, err)
		}
	}

	h.executeSync()

	got, exists := h.get(dispatchPattern)
	require.True(t, exists, "revived pattern must not be deleted by the CANCEL confirmation")
	assert.Same(t, ctx, got)
	req.hook = nil
	h.executeSync()
	sent := req.requests()
	assert.Equal(t, constant.FUZZY_WATCH_TYPE_WATCH, sent[len(sent)-1].watchType, "revived pattern is re-WATCHed")
}

func TestExecuteSyncOverLimitMarksInconsistentAndSuppressesLoadEvents(t *testing.T) {
	h, req := newReconcileHolder(t)
	var loads []model.ConfigFuzzyWatchLoadEvent
	var mu sync.Mutex
	var cnt atomic.Int32
	_, err := h.RegisterWatcher(&vo.ConfigFuzzyWatchParam{DataIdPattern: "app*", Group: "DEFAULT_GROUP",
		OnChange: func(model.ConfigFuzzyWatchChangeEvent) {},
		OnLoadEvent: func(e model.ConfigFuzzyWatchLoadEvent) {
			mu.Lock()
			loads = append(loads, e)
			mu.Unlock()
			cnt.Add(1)
		}})
	require.NoError(t, err)
	ctx, _ := h.get(dispatchPattern)
	clock := time.Unix(1000, 0)
	h.now = func() time.Time { return clock }
	over := &ConfigFuzzyWatchServerError{ErrorCode: constant.ERROR_CODE_FUZZY_WATCH_MATCH_COUNT_OVER_LIMIT}
	req.results = []error{over, over, over}

	h.executeSync()
	waitFor(t, func() bool { return cnt.Load() == 1 })
	assert.False(t, consistent(ctx))

	clock = clock.Add(10 * time.Second)
	h.executeSync() // inside the 60s window: still retried, but no new load event
	waitFor(t, func() bool { return idle(ctx) })
	assert.Equal(t, int32(1), cnt.Load())
	assert.Len(t, req.requests(), 2, "over-limit pattern stays inconsistent so every poll retries it")

	clock = clock.Add(61 * time.Second)
	h.executeSync()
	waitFor(t, func() bool { return cnt.Load() == 2 })
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, model.ConfigFuzzyWatchLoadEvent{Pattern: dispatchPattern, ErrorCode: constant.ERROR_CODE_FUZZY_WATCH_MATCH_COUNT_OVER_LIMIT}, loads[0])
}

func TestExecuteSyncTransientErrorBacksOffThenRetries(t *testing.T) {
	h, req := newReconcileHolder(t)
	sink := &eventSink{}
	ctx, _ := registerDispatch(t, h, sink.cb)
	<-h.bell // drain RegisterWatcher's ring so the retry bell is observable
	req.results = []error{errors.New("boom")}

	start := time.Now()
	h.executeSync()
	assert.GreaterOrEqual(t, time.Since(start), h.failureBackoff, "backoff elapsed before returning")
	assert.False(t, consistent(ctx))
	select {
	case <-h.bell:
	default:
		t.Fatal("transient failure must ring the bell for a retry")
	}
	h.executeSync()
	assert.Len(t, req.requests(), 2)
	assert.True(t, consistent(ctx))
}

func TestExecuteSyncTransientBackoffAbortsOnShutdown(t *testing.T) {
	h, req := newReconcileHolder(t)
	h.failureBackoff = time.Hour
	sink := &eventSink{}
	registerDispatch(t, h, sink.cb)
	req.results = []error{errors.New("boom")}
	done := make(chan struct{})
	go func() { h.executeSync(); close(done) }()
	waitFor(t, func() bool { return len(req.requests()) == 1 })
	h.Shutdown()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("executeSync did not abort backoff on shutdown")
	}
}

func TestExecuteSyncWithoutRequesterIsNoop(t *testing.T) {
	h := NewConfigFuzzyWatchHolder("public")
	h.executeSync() // must not panic
}

// ---- P7: real ingestion path (HandleSync / HandleChangeNotify -> applyChange) ----

func TestChangeNotifyConfigChangedNormalizesToAddOnlyIfNew(t *testing.T) {
	h, _ := newReconcileHolder(t)
	sink := &eventSink{}
	ctx, _ := registerDispatch(t, h, sink.cb)

	h.HandleChangeNotify(dispatchKey("app1"), constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_CONFIG_CHANGED)
	waitFor(t, func() bool { return len(sink.snapshot()) == 1 })
	ev := sink.snapshot()[0]
	assert.Equal(t, model.ConfigFuzzyWatchChangeEvent{DataId: "app1", Group: "DEFAULT_GROUP", NamespaceId: "public",
		ChangedType: constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, SyncType: constant.FUZZY_WATCH_RESOURCE_CHANGED}, ev)

	// Known key: a further CONFIG_CHANGED / ADD is a no-op (Java: only if newly received).
	h.HandleChangeNotify(dispatchKey("app1"), constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_CONFIG_CHANGED)
	h.HandleChangeNotify(dispatchKey("app1"), constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	// Sentinel: FIFO delivery means once app2 arrives, any extra app1 event would be visible.
	h.HandleChangeNotify(dispatchKey("app2"), constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	waitFor(t, func() bool { return len(sink.snapshot()) == 2 })
	assert.Equal(t, "app2", sink.snapshot()[1].DataId)
	assert.ElementsMatch(t, []string{dispatchKey("app1"), dispatchKey("app2")}, h.ReceivedGroupKeys(dispatchPattern))
	_ = ctx
}

func TestChangeNotifyDeleteOnlyIfReceived(t *testing.T) {
	h, _ := newReconcileHolder(t)
	sink := &eventSink{}
	registerDispatch(t, h, sink.cb)

	h.HandleChangeNotify(dispatchKey("app1"), constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG) // never received
	h.HandleChangeNotify(dispatchKey("app2"), constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	h.HandleChangeNotify(dispatchKey("app2"), constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG)
	waitFor(t, func() bool { return len(sink.snapshot()) == 2 })
	evs := sink.snapshot()
	assert.Equal(t, constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, evs[0].ChangedType)
	assert.Equal(t, constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG, evs[1].ChangedType)
	assert.Equal(t, "app2", evs[1].DataId)
	assert.Empty(t, h.ReceivedGroupKeys(dispatchPattern))
}

func TestChangeNotifyOnlyReachesMatchingPatternAndIgnoresMalformedKey(t *testing.T) {
	h, _ := newReconcileHolder(t)
	sink := &eventSink{}
	registerDispatch(t, h, sink.cb)

	h.HandleChangeNotify(dispatchKey("other1"), constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	h.HandleChangeNotify("not-a-group-key", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	h.HandleChangeNotify(dispatchKey("app1"), constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	waitFor(t, func() bool { return len(sink.snapshot()) == 1 })
	assert.Equal(t, "app1", sink.snapshot()[0].DataId)
	assert.Equal(t, []string{dispatchKey("app1")}, h.ReceivedGroupKeys(dispatchPattern))
}

func TestHandleSyncInitDeliversInitNotifyThenFinishCompletesGate(t *testing.T) {
	h, _ := newReconcileHolder(t)
	sink := &eventSink{}
	registerDispatch(t, h, sink.cb)

	h.HandleSync(dispatchPattern, constant.FUZZY_WATCH_INIT_NOTIFY, syncContexts("app1", "app2"), 2, 1)
	waitFor(t, func() bool { return len(sink.snapshot()) == 2 })
	for _, ev := range sink.snapshot() {
		assert.Equal(t, constant.FUZZY_WATCH_INIT_NOTIFY, ev.SyncType)
		assert.Equal(t, constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, ev.ChangedType)
	}

	gateCtx, cancel := contextWithShortTimeout()
	_, err := h.MatchedGroupKeys(gateCtx, dispatchPattern)
	cancel()
	require.Error(t, err, "gate stays closed before FINISH")

	h.HandleSync(dispatchPattern, constant.FINISH_FUZZY_WATCH_INIT_NOTIFY, nil, 2, 2)
	h.HandleSync(dispatchPattern, constant.FINISH_FUZZY_WATCH_INIT_NOTIFY, nil, 2, 2) // repeat after reconnect: no double close panic
	keys, err := h.MatchedGroupKeys(context.Background(), dispatchPattern)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{dispatchKey("app1"), dispatchKey("app2")}, keys)
	assert.Len(t, sink.snapshot(), 2, "FINISH emits no events")
}

func TestHandleSyncSkipsConfigChangedAndUnknownPatternAndMalformedKey(t *testing.T) {
	h, _ := newReconcileHolder(t)
	sink := &eventSink{}
	registerDispatch(t, h, sink.cb)

	h.HandleSync("public>>DEFAULT_GROUP>>nope*", constant.FUZZY_WATCH_INIT_NOTIFY, syncContexts("app1"), 1, 1) // unknown: no create
	_, exists := h.get("public>>DEFAULT_GROUP>>nope*")
	assert.False(t, exists)

	h.HandleSync(dispatchPattern, constant.FUZZY_WATCH_DIFF_SYNC_NOTIFY, []rpc_request.ConfigFuzzyWatchSyncContext{
		{GroupKey: dispatchKey("app1"), ChangedType: constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_CONFIG_CHANGED}, // Java warn-skips
		{GroupKey: "malformed", ChangedType: constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG},
		{GroupKey: dispatchKey("app2"), ChangedType: constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG},
	}, 1, 1)
	waitFor(t, func() bool { return len(sink.snapshot()) == 1 })
	assert.Equal(t, "app2", sink.snapshot()[0].DataId)
	assert.Equal(t, constant.FUZZY_WATCH_DIFF_SYNC_NOTIFY, sink.snapshot()[0].SyncType)
	assert.Equal(t, []string{dispatchKey("app2")}, h.ReceivedGroupKeys(dispatchPattern))
}

func contextWithShortTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Millisecond)
}
