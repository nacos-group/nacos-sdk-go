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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nacos-group/nacos-sdk-go/v3/model"
	"github.com/nacos-group/nacos-sdk-go/v3/vo"
)

const testConfigPattern = "public>>DEFAULT_GROUP>>app*"

// fakeConfigFuzzyRequester stands in for Task 4's real requester. supportErr
// is returned verbatim by CheckFuzzyWatchSupported; abilityGate/abilityEntered
// let a test park RegisterWatcher inside the ability check.
type fakeConfigFuzzyRequester struct {
	supportErr     error
	abilityGate    chan struct{}
	abilityEntered chan struct{}
}

func (f *fakeConfigFuzzyRequester) SendFuzzyWatchRequest(pattern, watchType string, receivedGroupKeys []string, isInitializing bool) error {
	return nil
}

func (f *fakeConfigFuzzyRequester) CheckFuzzyWatchSupported() error {
	if f.abilityEntered != nil {
		select {
		case f.abilityEntered <- struct{}{}:
		default:
		}
	}
	if f.abilityGate != nil {
		<-f.abilityGate
	}
	return f.supportErr
}

func newTestConfigHolder() *ConfigFuzzyWatchHolder {
	h := NewConfigFuzzyWatchHolder("public")
	h.SetRequester(&fakeConfigFuzzyRequester{})
	return h
}

func testParam(dataIdPattern string) *vo.ConfigFuzzyWatchParam {
	return &vo.ConfigFuzzyWatchParam{
		DataIdPattern: dataIdPattern,
		OnChange:      func(model.ConfigFuzzyWatchChangeEvent) {},
	}
}

func TestBuildConfigFuzzyWatchPattern(t *testing.T) {
	// Java FuzzyGroupKeyPattern.generatePattern: fixNamespace>>groupPattern>>dataIdPattern
	assert.Equal(t, "public>>DEFAULT_GROUP>>app*", BuildConfigFuzzyWatchPattern("public", "app*", "DEFAULT_GROUP"))
	assert.Equal(t, "ns1>>g>>*", BuildConfigFuzzyWatchPattern("ns1", "*", "g"))
	// blank namespace -> public, blank group -> DEFAULT_GROUP
	assert.Equal(t, "public>>DEFAULT_GROUP>>app*", BuildConfigFuzzyWatchPattern("", "app*", ""))
}

func TestRegisterCreatesContext(t *testing.T) {
	h := newTestConfigHolder()
	id, err := h.RegisterWatcher(testParam("app*"))
	require.NoError(t, err)
	require.NotZero(t, id)
	assert.Equal(t, []string{testConfigPattern}, h.Patterns())
	ctx, ok := h.get(testConfigPattern)
	require.True(t, ok)
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	assert.Len(t, ctx.entries, 1)
	assert.Equal(t, id, ctx.entries[0].id)
	assert.False(t, ctx.discard)
	assert.False(t, ctx.consistentWithServer)
}

func TestRegisterValidatesParam(t *testing.T) {
	h := newTestConfigHolder()
	_, err := h.RegisterWatcher(nil)
	assert.Error(t, err)
	_, err = h.RegisterWatcher(&vo.ConfigFuzzyWatchParam{DataIdPattern: "app*"})
	assert.Error(t, err, "OnChange is required")
	_, err = h.RegisterWatcher(&vo.ConfigFuzzyWatchParam{OnChange: func(model.ConfigFuzzyWatchChangeEvent) {}})
	assert.Error(t, err, "DataIdPattern is required")
	assert.Empty(t, h.Patterns())
}

func TestTwoWatchersShareContext(t *testing.T) {
	h := newTestConfigHolder()
	id1, err := h.RegisterWatcher(testParam("app*"))
	require.NoError(t, err)
	id2, err := h.RegisterWatcher(testParam("app*"))
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, h.Patterns(), 1)
	ctx, _ := h.get(testConfigPattern)
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	assert.Len(t, ctx.entries, 2)
}

func TestRemoveWatcherByIDRemovesOnlyOne(t *testing.T) {
	h := newTestConfigHolder()
	id1, _ := h.RegisterWatcher(testParam("app*"))
	id2, _ := h.RegisterWatcher(testParam("app*"))
	ctx, _ := h.get(testConfigPattern)
	ctx.mu.Lock()
	epochBefore := ctx.epoch
	ctx.mu.Unlock()

	h.RemoveWatcherByID(testConfigPattern, id1)

	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	require.Len(t, ctx.entries, 1)
	assert.Equal(t, id2, ctx.entries[0].id)
	assert.False(t, ctx.discard, "a remaining watcher keeps the pattern desired")
	assert.Equal(t, epochBefore, ctx.epoch, "partial removal does not change desired state")
}

func TestRemoveLastWatcherSetsDiscard(t *testing.T) {
	h := newTestConfigHolder()
	id, _ := h.RegisterWatcher(testParam("app*"))
	ctx, _ := h.get(testConfigPattern)
	ctx.mu.Lock()
	ctx.consistentWithServer = true
	e := ctx.entries[0]
	ctx.mu.Unlock()

	h.RemoveWatcherByID(testConfigPattern, id)

	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	assert.Empty(t, ctx.entries)
	assert.True(t, ctx.discard)
	assert.False(t, ctx.consistentWithServer)
	assert.True(t, e.removed)
	// the local context is kept until the worker confirms CANCEL (Task 4)
	_, ok := h.get(testConfigPattern)
	assert.True(t, ok)
}

func TestRemoveWatcherByIDUnknownIsNoop(t *testing.T) {
	h := newTestConfigHolder()
	h.RemoveWatcherByID("public>>g>>nope", 42) // must not panic
	id, _ := h.RegisterWatcher(testParam("app*"))
	h.RemoveWatcherByID(testConfigPattern, id+100)
	ctx, _ := h.get(testConfigPattern)
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	assert.Len(t, ctx.entries, 1)
	assert.False(t, ctx.discard)
}

func TestReviveClearsDiscardAndBumpsEpoch(t *testing.T) {
	h := newTestConfigHolder()
	id, _ := h.RegisterWatcher(testParam("app*"))
	h.RemoveWatcherByID(testConfigPattern, id)
	ctx, _ := h.get(testConfigPattern)
	ctx.mu.Lock()
	epochDiscarded := ctx.epoch
	ctx.mu.Unlock()

	_, err := h.RegisterWatcher(testParam("app*"))
	require.NoError(t, err)

	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	assert.False(t, ctx.discard)
	assert.Equal(t, epochDiscarded+1, ctx.epoch)
	assert.Len(t, ctx.entries, 1)
}

func TestEpochBumpsOnDesiredStateChanges(t *testing.T) {
	h := newTestConfigHolder()
	epoch := func() uint64 {
		ctx, _ := h.get(testConfigPattern)
		ctx.mu.Lock()
		defer ctx.mu.Unlock()
		return ctx.epoch
	}
	id, _ := h.RegisterWatcher(testParam("app*"))
	e0 := epoch()

	h.ResetConsistenceStatus()
	e1 := epoch()
	assert.Equal(t, e0+1, e1, "reconnect reset")

	h.RemoveWatcherByID(testConfigPattern, id)
	e2 := epoch()
	assert.Equal(t, e1+1, e2, "discard flip")

	_, _ = h.RegisterWatcher(testParam("app*"))
	e3 := epoch()
	assert.Equal(t, e2+1, e3, "revive flip")
}

func TestResetConsistenceStatusClearsConsistentFlagAndBells(t *testing.T) {
	h := newTestConfigHolder()
	_, _ = h.RegisterWatcher(testParam("app*"))
	_, _ = h.RegisterWatcher(testParam("db*"))
	for _, p := range h.Patterns() {
		ctx, _ := h.get(p)
		ctx.mu.Lock()
		ctx.consistentWithServer = true
		ctx.mu.Unlock()
	}
	select { // drain the ring RegisterWatcher left behind
	case <-h.bell:
	default:
	}

	h.ResetConsistenceStatus()

	for _, p := range h.Patterns() {
		ctx, _ := h.get(p)
		ctx.mu.Lock()
		assert.False(t, ctx.consistentWithServer, p)
		ctx.mu.Unlock()
	}
	select {
	case <-h.bell:
	default:
		t.Fatal("ResetConsistenceStatus must ring the worker")
	}
}

func TestBellCoalesces(t *testing.T) {
	h := newTestConfigHolder()
	for i := 0; i < 10; i++ {
		h.Bell() // must never block
	}
	assert.Len(t, h.bell, 1)
}

func TestRegisterAbilityErrorPropagatesWithoutState(t *testing.T) {
	h := NewConfigFuzzyWatchHolder("public")
	sentinel := errors.New("server too old")
	h.SetRequester(&fakeConfigFuzzyRequester{supportErr: sentinel})
	_, err := h.RegisterWatcher(testParam("app*"))
	assert.ErrorIs(t, err, sentinel)
	assert.Empty(t, h.Patterns())
}

func TestRegisterWithoutRequesterFails(t *testing.T) {
	h := NewConfigFuzzyWatchHolder("public")
	_, err := h.RegisterWatcher(testParam("app*"))
	assert.Error(t, err)
	assert.Empty(t, h.Patterns())
}

func TestRegisterFailsAfterShutdown(t *testing.T) {
	h := newTestConfigHolder()
	id, err := h.RegisterWatcher(testParam("app*"))
	require.NoError(t, err)
	require.NotZero(t, id)

	h.Shutdown()
	h.Shutdown() // idempotent

	_, err = h.RegisterWatcher(testParam("other*"))
	assert.ErrorIs(t, err, ErrConfigFuzzyWatchClientClosed)
	_, ok := h.get("public>>DEFAULT_GROUP>>other*")
	assert.False(t, ok, "a rejected registration must not create pattern state")
}

func TestShutdownDuringAbilityCheckRejectsRegistration(t *testing.T) {
	h := NewConfigFuzzyWatchHolder("public")
	r := &fakeConfigFuzzyRequester{abilityGate: make(chan struct{}), abilityEntered: make(chan struct{}, 1)}
	h.SetRequester(r)

	type result struct {
		id  uint64
		err error
	}
	done := make(chan result, 1)
	go func() {
		id, err := h.RegisterWatcher(testParam("app*"))
		done <- result{id, err}
	}()
	select {
	case <-r.abilityEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("registration never reached the ability check")
	}
	h.Shutdown()
	close(r.abilityGate)

	select {
	case res := <-done:
		assert.ErrorIs(t, res.err, ErrConfigFuzzyWatchClientClosed)
		assert.Zero(t, res.id)
	case <-time.After(2 * time.Second):
		t.Fatal("registration did not return")
	}
	assert.Empty(t, h.Patterns())
}

// Deterministic invariant under concurrency: every RegisterWatcher either
// returns a nonzero id and its entry is present in the holder, or returns
// ErrConfigFuzzyWatchClientClosed and left no trace. A registration that
// succeeds after Shutdown has returned is the violation being hunted.
func TestShutdownRegisterRaceHammer(t *testing.T) {
	const workers = 16
	h := newTestConfigHolder()

	type result struct {
		id  uint64
		err error
	}
	results := make([]result, workers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			id, err := h.RegisterWatcher(testParam("app*"))
			results[i] = result{id, err}
		}(i)
	}
	shutdownReturned := make(chan struct{})
	go func() {
		<-start
		h.Shutdown()
		close(shutdownReturned)
	}()
	close(start)
	wg.Wait()
	<-shutdownReturned

	committed := 0
	for _, r := range results {
		if r.err != nil {
			require.ErrorIs(t, r.err, ErrConfigFuzzyWatchClientClosed)
			require.Zero(t, r.id)
			continue
		}
		require.NotZero(t, r.id)
		committed++
	}
	entries := 0
	if ctx, ok := h.get(testConfigPattern); ok {
		ctx.mu.Lock()
		entries = len(ctx.entries)
		ctx.mu.Unlock()
	}
	assert.Equal(t, committed, entries, "entries must equal successful registrations")

	// after Shutdown returned, nothing may commit
	_, err := h.RegisterWatcher(testParam("app*"))
	assert.ErrorIs(t, err, ErrConfigFuzzyWatchClientClosed)
}

func TestRegisterRemoveHammerKeepsInvariants(t *testing.T) {
	h := newTestConfigHolder()
	const workers = 8
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				id, err := h.RegisterWatcher(testParam("app*"))
				require.NoError(t, err)
				h.ResetConsistenceStatus()
				h.RemoveWatcherByID(testConfigPattern, id)
			}
		}()
	}
	wg.Wait()
	ctx, ok := h.get(testConfigPattern)
	require.True(t, ok)
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	assert.Empty(t, ctx.entries)
	assert.True(t, ctx.discard, "all watchers gone => desired state is not-watched")
}

func TestMatchedGroupKeysBlocksUntilInitDone(t *testing.T) {
	h := newTestConfigHolder()
	_, err := h.MatchedGroupKeys(context.Background(), testConfigPattern)
	assert.Error(t, err, "unknown pattern")

	_, _ = h.RegisterWatcher(testParam("app*"))
	shortCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = h.MatchedGroupKeys(shortCtx, testConfigPattern)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	ctx, _ := h.get(testConfigPattern)
	ctx.mu.Lock()
	ctx.receivedGroupKeys["app1+DEFAULT_GROUP+public"] = struct{}{}
	ctx.initialized = true
	close(ctx.initDone)
	ctx.mu.Unlock()

	keys, err := h.MatchedGroupKeys(context.Background(), testConfigPattern)
	require.NoError(t, err)
	assert.Equal(t, []string{"app1+DEFAULT_GROUP+public"}, keys)
}

func TestItemMatchFiveModes(t *testing.T) {
	cases := []struct {
		name, pattern, value string
		want                 bool
	}{
		{"exact hit", "app", "app", true},
		{"exact miss", "app", "apps", false},
		{"all", "*", "anything", true},
		{"prefix hit", "app*", "app.yaml", true},
		{"prefix miss", "app*", "db.yaml", false},
		{"suffix hit", "*.yaml", "app.yaml", true},
		{"suffix miss", "*.yaml", "app.yml", false},
		{"contains hit", "*pp.*", "app.yaml", true},
		{"contains miss", "*zz*", "app.yaml", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, itemMatch(c.pattern, c.value))
		})
	}
}

func TestMatchPatternSlotOrder(t *testing.T) {
	// namespace>>group>>dataId: group is slot 2, dataId slot 3.
	assert.True(t, matchPattern("public>>DEFAULT_GROUP>>app*", "public", "DEFAULT_GROUP", "app.yaml"))
	assert.True(t, matchPattern("public>>*GROUP>>*.yaml", "public", "DEFAULT_GROUP", "app.yaml"))
	assert.False(t, matchPattern("public>>app*>>DEFAULT_GROUP", "public", "DEFAULT_GROUP", "app.yaml"), "slots must not be swapped")
	assert.False(t, matchPattern("other>>*>>*", "public", "DEFAULT_GROUP", "app.yaml"), "namespace is exact")
	assert.False(t, matchPattern("public>>*", "public", "DEFAULT_GROUP", "app.yaml"), "malformed pattern")
	assert.True(t, matchPattern("public>>*>>*", "", "g", "d"), "blank namespace means public")
}
