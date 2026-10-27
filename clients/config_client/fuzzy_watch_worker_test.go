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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v3/model"
	"github.com/nacos-group/nacos-sdk-go/v3/vo"
)

const dispatchPattern = "public>>DEFAULT_GROUP>>app*"

func dispatchKey(dataId string) string { return buildConfigGroupKey(dataId, "DEFAULT_GROUP", "public") }

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition not met within 5s")
}

type eventSink struct {
	mu     sync.Mutex
	events []model.ConfigFuzzyWatchChangeEvent
}

func (s *eventSink) cb(ev model.ConfigFuzzyWatchChangeEvent) {
	s.mu.Lock()
	s.events = append(s.events, ev)
	s.mu.Unlock()
}

func (s *eventSink) snapshot() []model.ConfigFuzzyWatchChangeEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]model.ConfigFuzzyWatchChangeEvent(nil), s.events...)
}

// applyForTest stands in for Task 4's HandleSync/HandleChangeNotify: it
// updates receivedGroupKeys, bumps syncVersion and enqueues to every entry,
// exactly like the naming holder's applyChange + enqueue.
func applyForTest(c *fuzzyWatchContext, dataId, changedType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := dispatchKey(dataId)
	switch changedType {
	case constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG:
		c.receivedGroupKeys[key] = struct{}{}
	case constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG:
		delete(c.receivedGroupKeys, key)
	}
	c.syncVersion++
	ev := model.ConfigFuzzyWatchChangeEvent{DataId: dataId, Group: "DEFAULT_GROUP", NamespaceId: "public",
		ChangedType: changedType, SyncType: constant.FUZZY_WATCH_RESOURCE_CHANGED}
	c.enqueueLocked(notifyTask{event: &ev, targets: c.snapshotEntriesLocked()})
}

// reconcileOnce is one bell-driven reconcile pass.
func reconcileOnce(c *fuzzyWatchContext) {
	c.mu.Lock()
	c.enqueueLocked(c.syncWatchersLocked()...)
	c.mu.Unlock()
}

func registerDispatch(t *testing.T, h *ConfigFuzzyWatchHolder, fn func(model.ConfigFuzzyWatchChangeEvent)) (*fuzzyWatchContext, uint64) {
	t.Helper()
	id, err := h.RegisterWatcher(&vo.ConfigFuzzyWatchParam{DataIdPattern: "app*", Group: "DEFAULT_GROUP", OnChange: fn})
	require.NoError(t, err)
	ctx, ok := h.get(dispatchPattern)
	require.True(t, ok)
	return ctx, id
}

func idle(c *fuzzyWatchContext) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.draining && len(c.pending) == 0
}

func syncedLen(c *fuzzyWatchContext) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries[0].syncedKeys)
}

func TestGroupKeyRoundTrip(t *testing.T) {
	k := buildConfigGroupKey("a+b", "g%1", "ns")
	d, g, n, err := parseConfigGroupKey(k)
	require.NoError(t, err)
	assert.Equal(t, []string{"a+b", "g%1", "ns"}, []string{d, g, n})
	_, _, _, err = parseConfigGroupKey("only+two")
	assert.Error(t, err)
}

// Semantic 1: syncedKeys advances only after the callback returned normally;
// a panic leaves the key unsynced and diff-sync re-delivers it.
func TestPanickingCallbackIsRetriedByDiffSync(t *testing.T) {
	h := newTestConfigHolder()
	var calls atomic.Int32
	sink := &eventSink{}
	ctx, _ := registerDispatch(t, h, func(ev model.ConfigFuzzyWatchChangeEvent) {
		if calls.Add(1) == 1 {
			panic("boom")
		}
		sink.cb(ev)
	})
	applyForTest(ctx, "app1", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	waitFor(t, func() bool { return calls.Load() >= 1 && idle(ctx) })
	assert.Zero(t, syncedLen(ctx), "a panicked delivery must not be recorded as synced")

	waitFor(t, func() bool {
		reconcileOnce(ctx)
		return syncedLen(ctx) == 1
	})
	evs := sink.snapshot()
	require.Len(t, evs, 1)
	assert.Equal(t, constant.FUZZY_WATCH_DIFF_SYNC_NOTIFY, evs[0].SyncType)
	assert.Equal(t, "app1", evs[0].DataId)
}

// Semantic 2: a task queued before Cancel must not invoke the removed watcher.
func TestCancelInvalidatesQueuedCallbacks(t *testing.T) {
	h := newTestConfigHolder()
	var count atomic.Int32
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	ctx, id := registerDispatch(t, h, func(model.ConfigFuzzyWatchChangeEvent) {
		count.Add(1)
		entered <- struct{}{}
		<-release
	})
	applyForTest(ctx, "app1", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	<-entered
	applyForTest(ctx, "app2", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	h.RemoveWatcherByID(dispatchPattern, id)
	close(release)
	waitFor(t, func() bool { return idle(ctx) })
	assert.Equal(t, int32(1), count.Load())
}

// Semantic 3: a full queue drops tasks, rings the bell, and diff-sync heals.
// Passes loop (as bell wake-ups would): healing tasks can themselves be
// dropped by a limit-1 queue, so single-pass convergence is not the guarantee.
func TestQueueDropsRingBellAndHealByDiffSync(t *testing.T) {
	h := newTestConfigHolder()
	h.pendingLimitForTest(1)
	sink := &eventSink{}
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var first atomic.Bool
	ctx, _ := registerDispatch(t, h, func(ev model.ConfigFuzzyWatchChangeEvent) {
		sink.cb(ev)
		if first.CompareAndSwap(false, true) {
			entered <- struct{}{}
			<-release
		}
	})
	applyForTest(ctx, "app0", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	<-entered
	select { // clear any ring from registration
	case <-h.bell:
	default:
	}
	for _, d := range []string{"app1", "app2", "app3"} {
		applyForTest(ctx, d, constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	}
	select {
	case <-h.bell:
	default:
		t.Fatal("dropping a task must ring the bell via onDrop")
	}
	ctx.mu.Lock()
	assert.LessOrEqual(t, len(ctx.pending), 1)
	assert.Len(t, ctx.receivedGroupKeys, 4, "received state is complete despite drops")
	ctx.mu.Unlock()
	close(release)
	waitFor(t, func() bool {
		reconcileOnce(ctx)
		return syncedLen(ctx) == 4
	})
}

// Semantic 4: an empty diff observed mid-drain must not stamp syncVersion.
func TestQueueDropsHealEvenWhenReconcileRunsMidDrain(t *testing.T) {
	h := newTestConfigHolder()
	h.pendingLimitForTest(1)
	sink := &eventSink{}
	var calls atomic.Int32
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	ctx, _ := registerDispatch(t, h, func(ev model.ConfigFuzzyWatchChangeEvent) {
		sink.cb(ev)
		if calls.Add(1) == 1 {
			entered <- struct{}{}
			<-release
		}
	})
	applyForTest(ctx, "app1", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	<-entered
	applyForTest(ctx, "app2", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)    // fills queue
	applyForTest(ctx, "app1", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG) // dropped
	applyForTest(ctx, "app2", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG) // dropped

	ctx.mu.Lock()
	require.True(t, ctx.draining)
	require.Empty(t, ctx.receivedGroupKeys)
	require.Empty(t, ctx.entries[0].syncedKeys)
	ctx.enqueueLocked(ctx.syncWatchersLocked()...)
	assert.NotEqual(t, ctx.syncVersion, ctx.entries[0].syncVersion, "mid-drain empty diff must not stamp")
	ctx.mu.Unlock()

	close(release)
	waitFor(t, func() bool { return idle(ctx) })
	waitFor(t, func() bool {
		reconcileOnce(ctx)
		return syncedLen(ctx) == 0
	})
	deletes := 0
	for _, e := range sink.snapshot() {
		if e.ChangedType == constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG {
			deletes++
			assert.Equal(t, constant.FUZZY_WATCH_DIFF_SYNC_NOTIFY, e.SyncType)
		}
	}
	assert.Equal(t, 2, deletes, "both dropped DELETEs are replayed by diff-sync")

	// At rest with an empty diff the version is finally stamped.
	waitFor(t, func() bool { return idle(ctx) })
	reconcileOnce(ctx)
	ctx.mu.Lock()
	assert.Equal(t, ctx.syncVersion, ctx.entries[0].syncVersion)
	ctx.mu.Unlock()
}

// Semantic 5: callbacks run on the drain goroutine, serially; a blocked
// callback never blocks the reconcile pass or the enqueuer.
func TestSlowCallbackDoesNotBlockReconcileAndIsSerial(t *testing.T) {
	h := newTestConfigHolder()
	var running, maxRunning atomic.Int32
	var done atomic.Int32
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	ctx, _ := registerDispatch(t, h, func(model.ConfigFuzzyWatchChangeEvent) {
		n := running.Add(1)
		for {
			m := maxRunning.Load()
			if n <= m || maxRunning.CompareAndSwap(m, n) {
				break
			}
		}
		if done.Load() == 0 {
			entered <- struct{}{}
			<-release
		}
		running.Add(-1)
		done.Add(1)
	})
	applyForTest(ctx, "app1", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	<-entered
	returned := make(chan struct{})
	go func() {
		applyForTest(ctx, "app2", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
		reconcileOnce(ctx)
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("reconcile/enqueue blocked behind a slow callback")
	}
	close(release)
	waitFor(t, func() bool { return done.Load() == 2 })
	assert.Equal(t, int32(1), maxRunning.Load(), "one watcher's callbacks are serial")
}

func TestDeliverySkipsPerWatcherDuplicates(t *testing.T) {
	h := newTestConfigHolder()
	sink := &eventSink{}
	ctx, _ := registerDispatch(t, h, sink.cb)
	applyForTest(ctx, "app1", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	waitFor(t, func() bool { return len(sink.snapshot()) == 1 })
	ctx.mu.Lock()
	dup := model.ConfigFuzzyWatchChangeEvent{DataId: "app1", Group: "DEFAULT_GROUP", NamespaceId: "public",
		ChangedType: constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, SyncType: constant.FUZZY_WATCH_DIFF_SYNC_NOTIFY}
	ctx.enqueueLocked(notifyTask{event: &dup, targets: ctx.snapshotEntriesLocked()})
	ctx.mu.Unlock()
	// Queue order is FIFO: once the sentinel is delivered the duplicate was already judged.
	applyForTest(ctx, "app2", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	waitFor(t, func() bool { return len(sink.snapshot()) >= 2 && idle(ctx) })
	evs := sink.snapshot()
	require.Len(t, evs, 2, "duplicate ADD skipped per watcher")
	assert.Equal(t, "app2", evs[1].DataId)
}

func TestDeliverySkipsDeleteForKeyWatcherNeverSaw(t *testing.T) {
	h := newTestConfigHolder()
	sink := &eventSink{}
	ctx, _ := registerDispatch(t, h, sink.cb)
	ctx.mu.Lock()
	del := model.ConfigFuzzyWatchChangeEvent{DataId: "app1", Group: "DEFAULT_GROUP", NamespaceId: "public",
		ChangedType: constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG, SyncType: constant.FUZZY_WATCH_DIFF_SYNC_NOTIFY}
	ctx.enqueueLocked(notifyTask{event: &del, targets: ctx.snapshotEntriesLocked()})
	ctx.mu.Unlock()
	applyForTest(ctx, "app2", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	waitFor(t, func() bool { return len(sink.snapshot()) >= 1 && idle(ctx) })
	evs := sink.snapshot()
	require.Len(t, evs, 1)
	assert.Equal(t, "app2", evs[0].DataId)
}

// The real CONFIG_CHANGED -> ADD_CONFIG-only-if-new normalization lives in
// applyChange (Task 4), per Java ClientWorker's mapping. deliver only knows
// ADD/DELETE, so a stray CONFIG_CHANGED task is skipped defensively.
func TestDeliverSkipsUnexpectedConfigChangedType(t *testing.T) {
	h := newTestConfigHolder()
	sink := &eventSink{}
	ctx, _ := registerDispatch(t, h, sink.cb)
	ctx.mu.Lock()
	ev := model.ConfigFuzzyWatchChangeEvent{DataId: "app1", Group: "DEFAULT_GROUP", NamespaceId: "public",
		ChangedType: constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_CONFIG_CHANGED, SyncType: constant.FUZZY_WATCH_RESOURCE_CHANGED}
	ctx.enqueueLocked(notifyTask{event: &ev, targets: ctx.snapshotEntriesLocked()})
	ctx.mu.Unlock()
	applyForTest(ctx, "app2", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG) // FIFO sentinel
	waitFor(t, func() bool { return len(sink.snapshot()) >= 1 && idle(ctx) })
	evs := sink.snapshot()
	require.Len(t, evs, 1, "CONFIG_CHANGED must not reach the callback")
	assert.Equal(t, "app2", evs[0].DataId)
	assert.Equal(t, 1, syncedLen(ctx), "only the sentinel ADD is recorded")
}

func TestLoadEventDeliveredToLiveWatchersOnly(t *testing.T) {
	h := newTestConfigHolder()
	loads := make(chan model.ConfigFuzzyWatchLoadEvent, 2)
	id, err := h.RegisterWatcher(&vo.ConfigFuzzyWatchParam{DataIdPattern: "app*", Group: "DEFAULT_GROUP",
		OnChange: func(model.ConfigFuzzyWatchChangeEvent) {}, OnLoadEvent: func(e model.ConfigFuzzyWatchLoadEvent) { loads <- e }})
	require.NoError(t, err)
	ctx, _ := h.get(dispatchPattern)
	ctx.mu.Lock()
	ctx.enqueueLocked(notifyTask{loadEvent: &model.ConfigFuzzyWatchLoadEvent{Pattern: dispatchPattern, ErrorCode: 50311}, targets: ctx.snapshotEntriesLocked()})
	ctx.mu.Unlock()
	assert.Equal(t, 50311, (<-loads).ErrorCode)
	ctx.mu.Lock()
	removedEntry := ctx.entries[0]
	ctx.mu.Unlock()
	h.RemoveWatcherByID(dispatchPattern, id)
	ctx.mu.Lock()
	ctx.enqueueLocked(notifyTask{loadEvent: &model.ConfigFuzzyWatchLoadEvent{Pattern: dispatchPattern}, targets: []*watcherEntry{removedEntry}})
	ctx.mu.Unlock()
	waitFor(t, func() bool { return idle(ctx) })
	assert.Empty(t, loads)
}

func TestRegisterReplaysKnownMatchesToNewWatcherOnly(t *testing.T) {
	h := newTestConfigHolder()
	first := &eventSink{}
	ctx, _ := registerDispatch(t, h, first.cb)
	applyForTest(ctx, "app1", constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG)
	waitFor(t, func() bool { return len(first.snapshot()) == 1 })
	second := &eventSink{}
	_, err := h.RegisterWatcher(&vo.ConfigFuzzyWatchParam{DataIdPattern: "app*", Group: "DEFAULT_GROUP", OnChange: second.cb})
	require.NoError(t, err)
	waitFor(t, func() bool { return len(second.snapshot()) == 1 })
	ev := second.snapshot()[0]
	assert.Equal(t, constant.FUZZY_WATCH_INIT_NOTIFY, ev.SyncType)
	assert.Equal(t, constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, ev.ChangedType)
	assert.Equal(t, "app1", ev.DataId)
	assert.Len(t, first.snapshot(), 1)
}

// Documents the epoch guard: removing an already-removed watcher on an
// already-discarded context must not bump epoch again.
func TestSecondRemoveOnDiscardedContextKeepsEpoch(t *testing.T) {
	h := newTestConfigHolder()
	ctx, id := registerDispatch(t, h, func(model.ConfigFuzzyWatchChangeEvent) {})
	h.RemoveWatcherByID(dispatchPattern, id)
	ctx.mu.Lock()
	require.True(t, ctx.discard)
	epoch := ctx.epoch
	ctx.mu.Unlock()
	h.RemoveWatcherByID(dispatchPattern, id)
	ctx.mu.Lock()
	assert.Equal(t, epoch, ctx.epoch)
	ctx.mu.Unlock()
}
