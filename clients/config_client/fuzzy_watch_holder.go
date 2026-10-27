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
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/nacos-group/nacos-sdk-go/v3/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v3/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v3/model"
	"github.com/nacos-group/nacos-sdk-go/v3/vo"
)

// ErrConfigFuzzyWatchClientClosed is returned by RegisterWatcher after the
// holder's Shutdown has run: the reconcile worker is stopped, so a new watch
// could never be established.
var ErrConfigFuzzyWatchClientClosed = errors.New("config fuzzy watch rejected: the nacos client is already closed")

// ConfigFuzzyWatchRequester is the transport the reconcile worker (Task 4)
// sends WATCH/CANCEL_WATCH requests through, injected via SetRequester after
// construction. CheckFuzzyWatchSupported returns nil when the connected server
// advertises the fuzzyWatch ability and otherwise the error RegisterWatcher
// must surface verbatim (the real requester returns its own
// ErrConfigFuzzyWatchNotSupported sentinel).
type ConfigFuzzyWatchRequester interface {
	SendFuzzyWatchRequest(groupKeyPattern, watchType string, receivedGroupKeys []string, isInitializing bool) error
	CheckFuzzyWatchSupported() error
}

// watcherEntry pairs a user callback with the id RegisterWatcher handed back
// to its caller. The id, not the callback's code pointer, identifies a
// registration (distinct method values can share a code pointer in Go).
// syncedKeys / syncVersion are per-watcher delivery state consumed by the
// dispatch queue (Task 3); this task only initializes them.
type watcherEntry struct {
	id          uint64
	fn          func(model.ConfigFuzzyWatchChangeEvent)
	onLoadEvent func(model.ConfigFuzzyWatchLoadEvent)
	syncedKeys  map[string]struct{} // guarded by ctx.mu
	syncVersion uint64

	// removed flips true (under ctx.mu) when RemoveWatcherByID takes the entry
	// out of ctx.entries, so queued-but-undelivered tasks (Task 3) can skip it.
	removed bool
}

// notifyTask is one dispatch queue element: an event (or load event) together
// with the watcher snapshot that must observe it. Queuing whole tasks keeps
// per-pattern delivery order. event and loadEvent are mutually exclusive.
type notifyTask struct {
	event     *model.ConfigFuzzyWatchChangeEvent
	loadEvent *model.ConfigFuzzyWatchLoadEvent
	targets   []*watcherEntry
}

// fuzzyWatchContext is the per-pattern desired/observed state. Its own mutex
// guards every field. Lock order is always h.mu -> ctx.mu.
//
// epoch increments on every desired-state mutation (reconnect reset, a discard
// flip in either direction) and lets the worker detect, when a WATCH RPC
// returns, whether the desired state it read before sending is still current;
// if not, the response must not flip consistentWithServer to true.
// discard marks desired state "not watched": set when the last watcher is
// removed, cleared if a new watcher revives the pattern before the in-flight
// CANCEL is confirmed. initDone is closed exactly once (Task 3's HandleSync on
// FINISH) and lets MatchedGroupKeys block until the initial sync completes.
type fuzzyWatchContext struct {
	pattern           string
	receivedGroupKeys map[string]struct{}
	entries           []*watcherEntry
	initialized       bool
	syncVersion       uint64

	consistentWithServer bool
	discard              bool
	initDone             chan struct{}
	lastOverLimitNotify  time.Time
	epoch                uint64

	// Dispatch queue state, driven by Task 3 (enqueue/drain/deliver). onDrop
	// must be non-blocking and lock-free; it is wired to h.Bell.
	pending      []notifyTask
	draining     bool
	pendingLimit int
	onDrop       func()

	mu sync.Mutex
}

func newFuzzyWatchContext(pattern string, h *ConfigFuzzyWatchHolder) *fuzzyWatchContext {
	return &fuzzyWatchContext{
		pattern:           pattern,
		receivedGroupKeys: make(map[string]struct{}),
		pendingLimit:      h.pendingLimit,
		onDrop:            h.Bell,
		initDone:          make(chan struct{}),
	}
}

// ConfigFuzzyWatchHolder tracks every config fuzzy watch pattern the client
// has registered (desired state). It is the config-side twin of naming's
// FuzzyWatchServiceListHolder; this file holds the state machine only - the
// dispatch queue (Task 3) and the reconcile worker / RPC sending (Task 4) are
// layered on top.
type ConfigFuzzyWatchHolder struct {
	namespace string
	patterns  cache.ConcurrentMap // groupKeyPattern -> *fuzzyWatchContext
	mu        sync.Mutex          // serializes pattern create/remove and linearizes Shutdown

	// nextCallbackID gives every registration its own identity to cancel by.
	// atomic.Uint64 keeps 64-bit alignment on 32-bit platforms.
	nextCallbackID atomic.Uint64

	requester   ConfigFuzzyWatchRequester
	requesterMu sync.RWMutex

	bell     chan struct{} // cap 1: rings coalesce naturally
	stopCh   chan struct{}
	stopOnce sync.Once
	started  atomic.Bool

	// Reconcile cadence (consumed by Task 4's worker). Fields, not consts, so
	// tests can inject short intervals; defaults mirror the Java client.
	now               func() time.Time
	pollInterval      time.Duration // 5s
	allSyncInterval   time.Duration // 3min
	overLimitSuppress time.Duration // 60s
	failureBackoff    time.Duration // 1s
	lastAllSync       time.Time

	// pendingLimit bounds each context's dispatch queue (Task 3); copied into
	// a context at creation.
	pendingLimit int // 1024
}

// NewConfigFuzzyWatchHolder creates an empty holder for the given namespace.
// It does not start the worker: call Start, or drive it manually in tests.
func NewConfigFuzzyWatchHolder(namespace string) *ConfigFuzzyWatchHolder {
	return &ConfigFuzzyWatchHolder{
		namespace:         namespace,
		patterns:          cache.NewConcurrentMap(),
		pendingLimit:      1024,
		bell:              make(chan struct{}, 1),
		stopCh:            make(chan struct{}),
		now:               time.Now,
		pollInterval:      5 * time.Second,
		allSyncInterval:   3 * time.Minute,
		overLimitSuppress: 60 * time.Second,
		failureBackoff:    time.Second,
	}
}

func (h *ConfigFuzzyWatchHolder) get(pattern string) (*fuzzyWatchContext, bool) {
	v, ok := h.patterns.Get(pattern)
	if !ok {
		return nil, false
	}
	return v.(*fuzzyWatchContext), true
}

// Patterns returns the currently registered groupKeyPatterns.
func (h *ConfigFuzzyWatchHolder) Patterns() []string {
	return h.patterns.Keys()
}

// ReceivedGroupKeys returns the groupKeys currently known to match pattern.
// It is sent back to the server on redo so the server can compute a diff.
func (h *ConfigFuzzyWatchHolder) ReceivedGroupKeys(pattern string) []string {
	ctx, ok := h.get(pattern)
	if !ok {
		return nil
	}
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	keys := make([]string, 0, len(ctx.receivedGroupKeys))
	for k := range ctx.receivedGroupKeys {
		keys = append(keys, k)
	}
	return keys
}

// SetRequester wires the transport used for WATCH/CANCEL_WATCH RPCs and the
// ability check. requesterMu serializes it against concurrent RegisterWatcher.
func (h *ConfigFuzzyWatchHolder) SetRequester(r ConfigFuzzyWatchRequester) {
	h.requesterMu.Lock()
	defer h.requesterMu.Unlock()
	h.requester = r
}

func (h *ConfigFuzzyWatchHolder) getRequester() ConfigFuzzyWatchRequester {
	h.requesterMu.RLock()
	defer h.requesterMu.RUnlock()
	return h.requester
}

// Bell rings the reconcile worker without blocking; a pending ring coalesces.
func (h *ConfigFuzzyWatchHolder) Bell() {
	select {
	case h.bell <- struct{}{}:
	default:
	}
}

// Start launches the reconcile worker goroutine. Idempotent.
func (h *ConfigFuzzyWatchHolder) Start() {
	if !h.started.CompareAndSwap(false, true) {
		return
	}
	go h.loop()
}

// Shutdown stops the worker. Idempotent. Closing stopCh under h.mu is what
// linearizes shutdown with RegisterWatcher's commit: a registration that
// re-checks stopCh while holding h.mu either observes the close (and rejects)
// or commits strictly before it.
func (h *ConfigFuzzyWatchHolder) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stopOnce.Do(func() { close(h.stopCh) })
}

func (h *ConfigFuzzyWatchHolder) loop() {
	for {
		select {
		case <-h.stopCh:
			return
		case <-h.bell:
		case <-time.After(h.pollInterval):
		}
		h.executeSync()
	}
}

// executeSync is the single sender of WATCH/CANCEL RPCs. STUB: implemented by
// Task 4 (the holder never sends in this task).
func (h *ConfigFuzzyWatchHolder) executeSync() {}

// pendingLimitForTest shrinks the dispatch queue bound for contexts created afterwards.
func (h *ConfigFuzzyWatchHolder) pendingLimitForTest(n int) { h.pendingLimit = n }

// replayKnownMatchesLocked replays already-known matches to a freshly added
// watcher only (INIT_NOTIFY / ADD_CONFIG). Caller holds ctx.mu.
func (c *fuzzyWatchContext) replayKnownMatchesLocked(entry *watcherEntry) {
	replay := make([]notifyTask, 0, len(c.receivedGroupKeys))
	for groupKey := range c.receivedGroupKeys {
		dataId, group, namespace, err := parseConfigGroupKey(groupKey)
		if err != nil {
			continue
		}
		ev := model.ConfigFuzzyWatchChangeEvent{DataId: dataId, Group: group, NamespaceId: namespace,
			ChangedType: constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, SyncType: constant.FUZZY_WATCH_INIT_NOTIFY}
		replay = append(replay, notifyTask{event: &ev, targets: []*watcherEntry{entry}})
	}
	c.enqueueLocked(replay...)
}

// configGroupKeySplitter joins dataId/group/namespace into a groupKey
// (Java GroupKey.getKeyTenant: dataId+group+tenant).
const configGroupKeySplitter = "+"

var (
	groupKeyEscaper   = strings.NewReplacer("%", "%25", "+", "%2B")
	groupKeyUnescaper = strings.NewReplacer("%2B", "+", "%25", "%")
)

func buildConfigGroupKey(dataId, group, namespace string) string {
	return groupKeyEscaper.Replace(dataId) + configGroupKeySplitter + groupKeyEscaper.Replace(group) +
		configGroupKeySplitter + groupKeyEscaper.Replace(namespace)
}

// parseConfigGroupKey splits a groupKey (dataId+group+namespace) into parts.
func parseConfigGroupKey(groupKey string) (dataId, group, namespace string, err error) {
	parts := strings.Split(groupKey, configGroupKeySplitter)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", errors.Errorf("groupKey %q is not dataId%sgroup%snamespace", groupKey,
			configGroupKeySplitter, configGroupKeySplitter)
	}
	return groupKeyUnescaper.Replace(parts[0]), groupKeyUnescaper.Replace(parts[1]), groupKeyUnescaper.Replace(parts[2]), nil
}

func (c *fuzzyWatchContext) snapshotEntriesLocked() []*watcherEntry {
	out := make([]*watcherEntry, len(c.entries))
	copy(out, c.entries)
	return out
}

// enqueueLocked appends tasks, dropping any that would push pending past
// pendingLimit (each drop rings onDrop so the reconcile worker heals the gap
// via syncWatchersLocked), and starts the drain goroutine when none is
// running. Caller must hold c.mu. At most one drain goroutine exists per
// context, so per-pattern order is preserved and callbacks never run on the
// caller's (RPC handler / reconcile worker) goroutine.
func (c *fuzzyWatchContext) enqueueLocked(tasks ...notifyTask) {
	if len(tasks) == 0 {
		return
	}
	for _, task := range tasks {
		if c.pendingLimit > 0 && len(c.pending) >= c.pendingLimit {
			logger.Warnf("config fuzzy watch pattern:%s dispatch queue full (%d), dropping notification", c.pattern, c.pendingLimit)
			if c.onDrop != nil {
				c.onDrop()
			}
			continue
		}
		c.pending = append(c.pending, task)
	}
	if len(c.pending) > 0 && !c.draining {
		c.draining = true
		go c.drain()
	}
}

func (c *fuzzyWatchContext) drain() {
	for {
		c.mu.Lock()
		if len(c.pending) == 0 {
			c.draining = false
			c.mu.Unlock()
			return
		}
		batch := c.pending
		c.pending = nil
		c.mu.Unlock()
		for _, task := range batch {
			c.deliver(task)
		}
	}
}

// deliver dispatches one task to its snapshotted targets. The per-watcher
// dedupe decision and the syncedKeys update happen at delivery time under
// c.mu. A removed watcher is re-checked immediately before invoking. syncedKeys
// advances only after the callback returns normally, so a panic leaves the key
// unsynced and diff-sync re-delivers it. Only ADD_CONFIG / DELETE_CONFIG tasks
// are delivered; any other changedType is skipped (applyChange normalizes
// CONFIG_CHANGED to ADD_CONFIG-only-if-new before enqueueing).
func (c *fuzzyWatchContext) deliver(task notifyTask) {
	if task.event == nil && task.loadEvent == nil {
		return
	}
	for _, entry := range task.targets {
		if task.loadEvent != nil {
			c.mu.Lock()
			removed := entry.removed
			c.mu.Unlock()
			if !removed && entry.onLoadEvent != nil {
				safeInvoke(func() { entry.onLoadEvent(*task.loadEvent) })
			}
			continue
		}
		ev := *task.event
		key := buildConfigGroupKey(ev.DataId, ev.Group, ev.NamespaceId)
		c.mu.Lock()
		skip := entry.removed
		if !skip {
			switch ev.ChangedType {
			case constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG:
				_, seen := entry.syncedKeys[key]
				skip = seen
			case constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG:
				_, seen := entry.syncedKeys[key]
				skip = !seen
			default:
				// Only ADD_CONFIG / DELETE_CONFIG are valid here; CONFIG_CHANGED
				// is normalized upstream (applyChange). Skip defensively.
				logger.Warnf("config fuzzy watch pattern:%s skips unexpected changedType:%s for %s", c.pattern, ev.ChangedType, key)
				skip = true
			}
		}
		c.mu.Unlock()
		if skip {
			continue
		}
		if !safeInvoke(func() { entry.fn(ev) }) {
			continue
		}
		c.mu.Lock()
		switch ev.ChangedType {
		case constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG:
			entry.syncedKeys[key] = struct{}{}
		case constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG:
			delete(entry.syncedKeys, key)
		}
		c.mu.Unlock()
	}
}

// safeInvoke isolates user-callback panics and reports whether fn completed
// without panicking.
func safeInvoke(fn func()) (completed bool) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("config fuzzy watch callback panic recovered: %v\n%s", r, string(debug.Stack()))
		}
	}()
	fn()
	return true
}

// BuildConfigFuzzyWatchPattern builds the groupKeyPattern the server matches
// against. Verified against Nacos 3.2.3 FuzzyGroupKeyPattern.generatePattern
// (common/src/main/java/com/alibaba/nacos/common/utils/FuzzyGroupKeyPattern.java):
//
//	fixNamespace>>groupPattern>>resourcePattern   (resourcePattern = dataId pattern)
//
// with a blank namespace defaulting to "public". A blank group defaults to
// DEFAULT_GROUP here (Java rejects a blank group; the SDK defaults it).
func BuildConfigFuzzyWatchPattern(namespace, dataIdPattern, group string) string {
	if namespace == "" {
		namespace = constant.DEFAULT_NAMESPACE_ID
	}
	if group == "" {
		group = constant.DEFAULT_GROUP
	}
	return namespace + constant.FUZZY_WATCH_PATTERN_SPLITTER + group +
		constant.FUZZY_WATCH_PATTERN_SPLITTER + dataIdPattern
}

// RegisterWatcher adds one watcher purely locally (reviving a pending-cancel
// pattern if needed) and rings the worker. The server RPC outcome never
// affects the registration. The pattern is BuildConfigFuzzyWatchPattern of the
// holder's namespace, param.DataIdPattern and param.Group.
func (h *ConfigFuzzyWatchHolder) RegisterWatcher(param *vo.ConfigFuzzyWatchParam) (uint64, error) {
	if param == nil {
		return 0, errors.New("config fuzzy watch param cannot be nil!")
	}
	if param.DataIdPattern == "" {
		return 0, errors.New("dataIdPattern cannot be empty!")
	}
	if param.OnChange == nil {
		return 0, errors.New("onChange cannot be nil!")
	}
	// Fast-fail before the ability check, which may block for its network
	// timeout; the authoritative closed check is the one under h.mu below.
	select {
	case <-h.stopCh:
		return 0, ErrConfigFuzzyWatchClientClosed
	default:
	}
	r := h.getRequester()
	if r == nil {
		return 0, errors.New("config fuzzy watch requester is not set")
	}
	if err := r.CheckFuzzyWatchSupported(); err != nil {
		return 0, err
	}
	pattern := BuildConfigFuzzyWatchPattern(h.namespace, param.DataIdPattern, param.Group)

	h.mu.Lock()
	// Shutdown closes stopCh while holding h.mu, so this re-check linearizes
	// the commit with shutdown (no ghost handle after the worker is gone).
	select {
	case <-h.stopCh:
		h.mu.Unlock()
		return 0, ErrConfigFuzzyWatchClientClosed
	default:
	}
	var ctx *fuzzyWatchContext
	if v, ok := h.patterns.Get(pattern); ok {
		ctx = v.(*fuzzyWatchContext)
	} else {
		ctx = newFuzzyWatchContext(pattern, h)
		h.patterns.Set(pattern, ctx)
	}
	id := h.nextCallbackID.Add(1)
	ctx.mu.Lock()
	if ctx.discard {
		ctx.discard = false // revive a pattern whose CANCEL may be in flight
		ctx.epoch++         // invalidate any in-flight send's stale desired-state snapshot
	}
	entry := &watcherEntry{id: id, fn: param.OnChange, onLoadEvent: param.OnLoadEvent, syncedKeys: make(map[string]struct{})}
	ctx.entries = append(ctx.entries, entry)
	ctx.replayKnownMatchesLocked(entry)
	ctx.mu.Unlock()
	h.mu.Unlock()
	h.Bell()
	return id, nil
}

// RemoveWatcherByID removes exactly one registration. When the last watcher
// goes, the pattern flips to discard (desired state: not watched) and the
// worker sends the server-side CANCEL; local state stays until the server
// confirms, so pushes keep being handled meanwhile.
func (h *ConfigFuzzyWatchHolder) RemoveWatcherByID(pattern string, id uint64) {
	h.mu.Lock()
	ctx, ok := h.get(pattern)
	if !ok {
		h.mu.Unlock()
		return
	}
	ctx.mu.Lock()
	kept := ctx.entries[:0:0]
	for _, e := range ctx.entries {
		if e.id != id {
			kept = append(kept, e)
		} else {
			e.removed = true
		}
	}
	ctx.entries = kept
	if len(ctx.entries) == 0 && !ctx.discard {
		ctx.discard = true
		ctx.consistentWithServer = false
		ctx.epoch++ // invalidate any in-flight WATCH send's stale desired-state snapshot
	}
	ctx.mu.Unlock()
	h.mu.Unlock()
	h.Bell()
}

// ResetConsistenceStatus marks every pattern out of sync with the server -
// called on reconnect, since a fresh connection has no server-side WATCH state
// - and rings the worker to re-send them all. Bumping epoch invalidates any
// WATCH send in flight on the old connection.
func (h *ConfigFuzzyWatchHolder) ResetConsistenceStatus() {
	for _, pattern := range h.Patterns() {
		if ctx, ok := h.get(pattern); ok {
			ctx.mu.Lock()
			ctx.consistentWithServer = false
			ctx.epoch++
			ctx.mu.Unlock()
		}
	}
	h.Bell()
}

// MatchedGroupKeys blocks until pattern's initial sync finishes (or waitCtx is
// done) and then returns the current matched-groupKey snapshot.
func (h *ConfigFuzzyWatchHolder) MatchedGroupKeys(waitCtx context.Context, pattern string) ([]string, error) {
	pctx, ok := h.get(pattern)
	if !ok {
		return nil, errors.Errorf("pattern %s is not being fuzzy watched", pattern)
	}
	select {
	case <-pctx.initDone:
		return h.ReceivedGroupKeys(pattern), nil
	case <-waitCtx.Done():
		return nil, waitCtx.Err()
	}
}

// matchPattern reports whether a config (namespace/group/dataId) is covered by
// a groupKeyPattern (namespace>>groupPattern>>dataIdPattern). Namespace must
// be exact (blank means "public", as in Java); group and dataId match by
// itemMatch.
func matchPattern(pattern, namespace, group, dataId string) bool {
	parts := strings.Split(pattern, constant.FUZZY_WATCH_PATTERN_SPLITTER)
	if len(parts) != 3 {
		return false
	}
	if namespace == "" {
		namespace = constant.DEFAULT_NAMESPACE_ID
	}
	return parts[0] == namespace && itemMatch(parts[1], group) && itemMatch(parts[2], dataId)
}

// itemMatch mirrors com.alibaba.nacos.common.utils.FuzzyGroupKeyPattern.itemMatched
// (exact, "*", prefix "x*", suffix "*x", contains "*x*"). "*x*" must be tested
// before the prefix/suffix cases since it also satisfies both.
// https://github.com/alibaba/nacos/blob/develop/common/src/main/java/com/alibaba/nacos/common/utils/FuzzyGroupKeyPattern.java
func itemMatch(pattern, value string) bool {
	switch {
	case pattern == "*":
		return true
	case strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") && len(pattern) >= 2:
		return strings.Contains(value, pattern[1:len(pattern)-1])
	case strings.HasPrefix(pattern, "*"):
		return strings.HasSuffix(value, pattern[1:])
	case strings.HasSuffix(pattern, "*"):
		return strings.HasPrefix(value, pattern[:len(pattern)-1])
	default:
		return pattern == value
	}
}
