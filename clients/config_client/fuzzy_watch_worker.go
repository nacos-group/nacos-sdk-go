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
	"time"

	"github.com/pkg/errors"

	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v3/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/model"
)

// syncWatchersLocked heals per-watcher drift (dropped notifications, watchers
// added between pushes): any watcher whose syncVersion lags gets targeted
// ADD/DELETE diff events; watchers already in sync just have their version
// stamped. Caller holds c.mu.
func (c *fuzzyWatchContext) syncWatchersLocked() []notifyTask {
	var tasks []notifyTask
	for _, entry := range c.entries {
		if entry.syncVersion == c.syncVersion {
			continue
		}
		var diff []notifyTask
		for key := range c.receivedGroupKeys {
			if _, seen := entry.syncedKeys[key]; !seen {
				if task, ok := diffTask(key, constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, entry); ok {
					diff = append(diff, task)
				}
			}
		}
		for key := range entry.syncedKeys {
			if _, exists := c.receivedGroupKeys[key]; !exists {
				if task, ok := diffTask(key, constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG, entry); ok {
					diff = append(diff, task)
				}
			}
		}
		// While a drain is in flight, syncedKeys is only an intermediate
		// delivery state: queued or in-flight callbacks will still mutate it
		// after we release c.mu. An empty diff observed mid-drain therefore
		// proves nothing - stamping here would let dropped DELETE/ADD tasks
		// escape all future diff-sync passes. Stamp only at rest.
		if len(diff) == 0 && !c.draining {
			entry.syncVersion = c.syncVersion
		} else {
			tasks = append(tasks, diff...)
		}
	}
	return tasks
}

func diffTask(groupKey, changedType string, entry *watcherEntry) (notifyTask, bool) {
	dataId, group, namespace, err := parseConfigGroupKey(groupKey)
	if err != nil {
		return notifyTask{}, false
	}
	ev := model.ConfigFuzzyWatchChangeEvent{DataId: dataId, Group: group, NamespaceId: namespace,
		ChangedType: changedType, SyncType: constant.FUZZY_WATCH_DIFF_SYNC_NOTIFY}
	return notifyTask{event: &ev, targets: []*watcherEntry{entry}}, true
}

// executeSync is the single sender of WATCH/CANCEL RPCs. Serializing every
// lifecycle transition through this one goroutine is what removes the
// register/cancel/reconnect races by construction (Java parity:
// ConfigFuzzyWatchGroupKeyHolder.executeConfigFuzzyWatch).
func (h *ConfigFuzzyWatchHolder) executeSync() {
	r := h.getRequester()
	if r == nil {
		return
	}
	now := h.now()
	needAllSync := now.Sub(h.lastAllSync) >= h.allSyncInterval
	for _, pattern := range h.Patterns() {
		ctx, ok := h.get(pattern)
		if !ok {
			continue
		}
		ctx.mu.Lock()
		// A discarded, now-empty context always needs its CANCEL sent: the
		// consistentWithServer fast path below is for an established WATCH,
		// and must never short-circuit a pending CANCEL, or the CANCEL would
		// wait up to allSyncInterval behind a stale "consistent" flag left
		// over from the WATCH that preceded the last watcher's removal.
		discardedEmpty := ctx.discard && len(ctx.entries) == 0
		if ctx.consistentWithServer && !discardedEmpty {
			ctx.enqueueLocked(ctx.syncWatchersLocked()...)
			if !needAllSync {
				ctx.mu.Unlock()
				continue
			}
		}
		watchType := constant.FUZZY_WATCH_TYPE_WATCH
		if discardedEmpty {
			watchType = constant.FUZZY_WATCH_TYPE_CANCEL_WATCH
		}
		keys := make([]string, 0, len(ctx.receivedGroupKeys))
		for k := range ctx.receivedGroupKeys {
			keys = append(keys, k)
		}
		isInitializing := !ctx.initialized
		// epoch is snapshotted together with the rest of the desired state
		// that this RPC represents, so the response handler below can tell
		// whether desired state moved on while the RPC was in flight.
		epoch := ctx.epoch
		ctx.mu.Unlock()

		err := r.SendFuzzyWatchRequest(pattern, watchType, keys, isInitializing)
		switch {
		case err == nil:
			if watchType == constant.FUZZY_WATCH_TYPE_CANCEL_WATCH {
				h.removePatternIfDiscarded(pattern)
			} else {
				ctx.mu.Lock()
				// Only apply the success if desired state has not moved on
				// since the snapshot this WATCH was sent from (e.g. a
				// concurrent RemoveWatcherByID or ResetConsistenceStatus) -
				// otherwise this stale response must not mark the pattern
				// consistent; the newer desired state gets its own send.
				if ctx.epoch == epoch {
					ctx.consistentWithServer = true
					ctx.lastOverLimitNotify = time.Time{}
				}
				ctx.mu.Unlock()
			}
		case isOverLimitError(err):
			h.notifyOverLimit(ctx, pattern, err)
		default:
			logger.Warnf("config fuzzy watch pattern:%s %s failed, will retry: %v", pattern, watchType, err)
			// Java parity: brief in-loop backoff, then ring to retry. Blocking
			// the worker briefly is acceptable at this cadence and keeps a
			// single sender.
			select {
			case <-h.stopCh:
				return
			case <-time.After(h.failureBackoff):
			}
			h.Bell()
		}
	}
	if needAllSync {
		h.lastAllSync = now
	}
}

// isOverLimitError reports a server capacity rejection. Java 3.2.3
// ErrorCode.FUZZY_WATCH_PATTERN_OVER_LIMIT(50310) / ..._MATCH_COUNT_OVER_LIMIT(50311)
// are shared by naming and config, so the same constants apply.
func isOverLimitError(err error) bool {
	var serverErr *ConfigFuzzyWatchServerError
	if !errors.As(err, &serverErr) {
		return false
	}
	return serverErr.ErrorCode == constant.ERROR_CODE_FUZZY_WATCH_PATTERN_OVER_LIMIT ||
		serverErr.ErrorCode == constant.ERROR_CODE_FUZZY_WATCH_MATCH_COUNT_OVER_LIMIT
}

// notifyOverLimit fans a capacity rejection out to the pattern's load
// watchers, suppressed to one notification per overLimitSuppress window. The
// pattern is marked inconsistent unconditionally (independent of the
// suppression window) so the regular poll keeps retrying it - without this,
// an over-limit hit against a previously consistent pattern (its match count
// grew past the server limit between polls) would otherwise wait out the
// full allSyncInterval before the next WATCH attempt.
func (h *ConfigFuzzyWatchHolder) notifyOverLimit(ctx *fuzzyWatchContext, pattern string, err error) {
	var serverErr *ConfigFuzzyWatchServerError
	errors.As(err, &serverErr)
	logger.Warnf("config fuzzy watch pattern:%s suppressed by server capacity limit: %v", pattern, err)
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.consistentWithServer = false
	if !ctx.lastOverLimitNotify.IsZero() && h.now().Sub(ctx.lastOverLimitNotify) < h.overLimitSuppress {
		return
	}
	ctx.lastOverLimitNotify = h.now()
	load := model.ConfigFuzzyWatchLoadEvent{Pattern: pattern, ErrorCode: serverErr.ErrorCode}
	ctx.enqueueLocked(notifyTask{loadEvent: &load, targets: ctx.snapshotEntriesLocked()})
}

// removePatternIfDiscarded finalizes a confirmed CANCEL, but only if the
// pattern is still discarded and empty: a watcher registered while the CANCEL
// was in flight revives the pattern, which then gets re-WATCHed next round.
func (h *ConfigFuzzyWatchHolder) removePatternIfDiscarded(pattern string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ctx, ok := h.get(pattern)
	if !ok {
		return
	}
	ctx.mu.Lock()
	remove := ctx.discard && len(ctx.entries) == 0
	ctx.mu.Unlock()
	if remove {
		h.patterns.Remove(pattern)
	}
}

// applyChange mutates receivedGroupKeys for a single groupKey and reports the
// event to fire, if any. Called with c.mu held. Mirrors Java
// ConfigFuzzyWatchGroupKeyHolder: ADD_CONFIG and CONFIG_CHANGED both become an
// ADD_CONFIG event and only when the key is newly received (a changed config
// the watcher already knows fires nothing); DELETE_CONFIG fires only for a key
// that was received. This is the sole producer of events for deliver, which
// only understands ADD_CONFIG / DELETE_CONFIG. syncVersion is bumped on every
// actual change so the reconcile worker can tell a stale watcher apart.
func (c *fuzzyWatchContext) applyChange(groupKey, dataId, group, namespace, changedType, syncType string) (model.ConfigFuzzyWatchChangeEvent, bool) {
	normalized := constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG
	switch changedType {
	case constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG, constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_CONFIG_CHANGED:
		if _, ok := c.receivedGroupKeys[groupKey]; ok {
			return model.ConfigFuzzyWatchChangeEvent{}, false
		}
		c.receivedGroupKeys[groupKey] = struct{}{}
	case constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG:
		if _, ok := c.receivedGroupKeys[groupKey]; !ok {
			return model.ConfigFuzzyWatchChangeEvent{}, false
		}
		delete(c.receivedGroupKeys, groupKey)
		normalized = constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG
	default:
		logger.Warnf("config fuzzy watch ignores unknown changedType:%s groupKey:%s", changedType, groupKey)
		return model.ConfigFuzzyWatchChangeEvent{}, false
	}
	c.syncVersion++
	return model.ConfigFuzzyWatchChangeEvent{DataId: dataId, Group: group, NamespaceId: namespace,
		ChangedType: normalized, SyncType: syncType}, true
}

// HandleSync processes a server-pushed ConfigFuzzyWatchSyncRequest: a batch of
// matched configs for pattern. INIT / DIFF batches add or remove keys and
// enqueue callbacks for async dispatch; a FINISH batch just marks the initial
// sync complete (no events). A sync for an unknown pattern is dropped, never
// create-on-demand, so a late batch cannot resurrect a canceled pattern. As in
// Java, contexts whose changedType is not ADD_CONFIG/DELETE_CONFIG are skipped.
func (h *ConfigFuzzyWatchHolder) HandleSync(pattern, syncType string, contexts []rpc_request.ConfigFuzzyWatchSyncContext, totalBatch, currentBatch int) {
	ctx, ok := h.get(pattern)
	if !ok {
		logger.Warnf("config fuzzy watch received sync for unknown pattern:%s (already canceled or never registered), ignoring", pattern)
		return
	}

	if syncType == constant.FINISH_FUZZY_WATCH_INIT_NOTIFY {
		ctx.mu.Lock()
		// FINISH can arrive more than once across reconnects; closing an
		// already-closed channel panics.
		if !ctx.initialized {
			ctx.initialized = true
			close(ctx.initDone)
		}
		ctx.mu.Unlock()
		logger.Infof("config fuzzy watch pattern:%s initial sync finished, batch %d/%d", pattern, currentBatch, totalBatch)
		return
	}

	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	var tasks []notifyTask
	targets := ctx.snapshotEntriesLocked()
	for _, item := range contexts {
		if item.ChangedType != constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG &&
			item.ChangedType != constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG {
			logger.Warnf("config fuzzy watch sync skips unsupported changedType:%s groupKey:%s", item.ChangedType, item.GroupKey)
			continue
		}
		dataId, group, namespace, err := parseConfigGroupKey(item.GroupKey)
		if err != nil {
			logger.Warnf("config fuzzy watch skips malformed groupKey:%s changedType:%s err:%v", item.GroupKey, item.ChangedType, err)
			continue
		}
		if ev, fired := ctx.applyChange(item.GroupKey, dataId, group, namespace, item.ChangedType, syncType); fired {
			ev := ev
			tasks = append(tasks, notifyTask{event: &ev, targets: targets})
		}
	}
	ctx.enqueueLocked(tasks...)
}

// HandleChangeNotify processes a server-pushed ConfigFuzzyWatchChangeNotifyRequest:
// one config added / changed / deleted after the initial sync. The request
// carries no pattern, so the groupKey is matched against every registered
// pattern. Events carry syncType FUZZY_WATCH_RESOURCE_CHANGED.
func (h *ConfigFuzzyWatchHolder) HandleChangeNotify(groupKey, changedType string) {
	dataId, group, namespace, err := parseConfigGroupKey(groupKey)
	if err != nil {
		logger.Warnf("config fuzzy watch change-notify skips malformed groupKey:%s changedType:%s err:%v", groupKey, changedType, err)
		return
	}
	for _, pattern := range h.Patterns() {
		if !matchPattern(pattern, namespace, group, dataId) {
			continue
		}
		ctx, ok := h.get(pattern)
		if !ok {
			continue
		}
		ctx.mu.Lock()
		if ev, fired := ctx.applyChange(groupKey, dataId, group, namespace, changedType, constant.FUZZY_WATCH_RESOURCE_CHANGED); fired {
			ctx.enqueueLocked(notifyTask{event: &ev, targets: ctx.snapshotEntriesLocked()})
		}
		ctx.mu.Unlock()
	}
}
