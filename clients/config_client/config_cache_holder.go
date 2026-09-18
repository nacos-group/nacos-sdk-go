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

	"github.com/nacos-group/nacos-sdk-go/v3/common/filter"
	"github.com/nacos-group/nacos-sdk-go/v3/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v3/vo"
)

// cacheData holds the listen state for a single dataId/group/tenant tuple.
// It is always referenced by pointer so that listeners registered against
// the same key observe/update a single shared instance.
type cacheData struct {
	mu sync.Mutex

	dataId, group, tenant                       string
	content, contentType, md5, encryptedDataKey string
	taskId                                      int
	isSyncWithServer                            bool
	isInitializing                              bool
	discard                                     bool
	listeners                                   []*listenerWrap
}

// listenerWrap pairs a user listener callback with the md5 watermark it has
// last been notified with, so distinct listeners on the same key can be
// notified independently. inFlight (guarded by the owning cacheData's mu)
// marks a delivery goroutine currently running this listener's callback:
// while set, notify rounds skip the wrap, so a single listener is never
// invoked concurrently with itself and a slow callback consumes exactly one
// goroutine instead of one per change.
type listenerWrap struct {
	listener    vo.Listener
	lastCallMd5 string
	inFlight    bool
}

// configCacheHolder is the pointer-based replacement for the previous
// cache.ConcurrentMap-backed by-value storage. Entries are never replaced in
// place; callers mutate the returned *cacheData under its own mu.
type configCacheHolder struct {
	mu      sync.RWMutex
	entries map[string]*cacheData
}

func newConfigCacheHolder() *configCacheHolder {
	return &configCacheHolder{
		entries: make(map[string]*cacheData),
	}
}

// get returns the entry for key, if any.
func (h *configCacheHolder) get(key string) (*cacheData, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	cData, ok := h.entries[key]
	return cData, ok
}

// getOrCreate returns the existing entry for key, reviving it (discard=false)
// if it was previously cancelled. If no entry exists yet, seed() is invoked
// to build one, which is then stored and returned. seed is only invoked while
// creating a brand-new entry.
func (h *configCacheHolder) getOrCreate(key string, seed func() *cacheData) *cacheData {
	h.mu.Lock()
	defer h.mu.Unlock()
	if cData, ok := h.entries[key]; ok {
		cData.mu.Lock()
		cData.discard = false
		cData.mu.Unlock()
		return cData
	}
	cData := seed()
	h.entries[key] = cData
	return cData
}

// snapshot returns all entries currently stored. Order is unspecified.
func (h *configCacheHolder) snapshot() []*cacheData {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]*cacheData, 0, len(h.entries))
	for _, cData := range h.entries {
		result = append(result, cData)
	}
	return result
}

// removeIfDiscarded removes the entry for key only if it is still marked as
// discarded and has no listeners attached, re-checking both conditions while
// holding the lock to avoid racing with a concurrent revive/addListener.
func (h *configCacheHolder) removeIfDiscarded(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	cData, ok := h.entries[key]
	if !ok {
		return
	}
	cData.mu.Lock()
	shouldRemove := cData.discard && len(cData.listeners) == 0
	cData.mu.Unlock()
	if shouldRemove {
		delete(h.entries, key)
	}
}

// count returns the number of entries currently stored.
func (h *configCacheHolder) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.entries)
}

// addListener appends a listener to the entry, recording initialMd5 as its
// starting watermark so it is only notified once content actually changes
// relative to that baseline.
func (c *cacheData) addListener(l vo.Listener, initialMd5 string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, &listenerWrap{listener: l, lastCallMd5: initialMd5})
}

// reviveAndAddListener atomically (re)activates the entry and appends l as a
// new listener, in a single critical section: discard is reasserted to
// false, isInitializing is set to true, and the listener is appended using
// the entry's md5 (read under the same lock) as its initial watermark.
//
// This exists because getOrCreate's own revive step (discard=false) and a
// subsequent, separately-locked addListener call are two distinct critical
// sections; a concurrent CancelListenConfig/markDiscard could interleave
// between them and leave the entry with discard==true and a non-empty
// listeners slice -- an inconsistent state that downstream reap logic must
// never observe. Folding revive + append into one lock closes that window:
// whichever of markDiscard/reviveAndAddListener runs last atomically
// determines the resulting (discard, listeners) pair.
func (c *cacheData) reviveAndAddListener(l vo.Listener) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.discard = false
	c.isInitializing = true
	c.listeners = append(c.listeners, &listenerWrap{listener: l, lastCallMd5: c.md5})
}

// notifyListeners delivers the entry's current content to every listener
// whose watermark (lastCallMd5) differs from the entry's md5. The snapshot
// (md5, content, encryptedDataKey, dataId/group/tenant, and the set of
// not-yet-caught-up wraps) is taken under c.mu, which is released before any
// callback runs -- callbacks must never run while holding the lock, since a
// slow or panicking listener would otherwise block every other operation on
// this entry (addListener, markDiscard, a concurrent notifyListeners round).
//
// The content handed to listeners is the filter chain's output, not the raw
// cacheData content: this preserves the pre-existing decryption path for
// cipher- dataIds (the same DoFilters(&vo.ConfigParam{..., UsageType:
// ResponseType}) semantics the old executeListener used). If the filter
// chain errors, the entire round is skipped without advancing any
// watermark, so the next round retries.
//
// Each wrap's watermark is advanced to the snapshotted md5 (not a fresh
// re-read -- content may change concurrently with these callbacks) only if
// its callback returns normally: a panicking listener leaves its own
// watermark untouched so the same content is redelivered to it in a later
// round, while every other listener on the same key still receives this
// round's notification.
//
// Callbacks run on their own delivery goroutine, one per lagging wrap, never
// on the caller: the caller is the single listen executor goroutine, and a
// slow callback running inline there would stall every other config's
// queries, notifications and cancel batches (the pre-v3 code launched
// callbacks with `go` for the same reason). Fan-out is bounded per listener
// by the inFlight flag -- a wrap whose previous delivery has not returned is
// skipped, so a slow listener holds exactly one goroutine no matter how many
// changes arrive meanwhile. When a delivery completes and the wrap's
// watermark still trails the entry's current md5 (a newer change landed
// while the callback ran, or the callback panicked), wake is invoked (if
// non-nil) so the executor promptly re-notifies instead of waiting out the
// poll interval.
func (c *cacheData) notifyListeners(chain filter.IConfigFilterChain, wake func()) {
	c.mu.Lock()
	dataId, group, tenant := c.dataId, c.group, c.tenant
	content := c.content
	encryptedDataKey := c.encryptedDataKey
	md5 := c.md5
	toNotify := make([]*listenerWrap, 0, len(c.listeners))
	for _, lw := range c.listeners {
		if lw.lastCallMd5 != md5 && !lw.inFlight {
			lw.inFlight = true
			toNotify = append(toNotify, lw)
		}
	}
	c.mu.Unlock()

	if len(toNotify) == 0 {
		return
	}

	clearInFlight := func() {
		c.mu.Lock()
		for _, lw := range toNotify {
			lw.inFlight = false
		}
		c.mu.Unlock()
	}

	param := &vo.ConfigParam{
		DataId:           dataId,
		Content:          content,
		EncryptedDataKey: encryptedDataKey,
		UsageType:        vo.ResponseType,
	}
	if err := chain.DoFilters(param); err != nil {
		logger.Errorf("do filters failed ,dataId=%s,group=%s,tenant=%s,err:%+v ", dataId, group, tenant, err)
		clearInFlight()
		return
	}
	decryptedContent := param.Content

	for _, lw := range toNotify {
		go c.deliverAndAdvance(lw, tenant, group, dataId, decryptedContent, md5, wake)
	}
}

// deliverAndAdvance invokes lw's listener on the current (delivery)
// goroutine and, only if it returns normally, advances lw.lastCallMd5 to
// md5. The inFlight flag is cleared in the same critical section as the
// watermark write, so the wrap becomes eligible for the next round only
// once its final state for this round is visible. If the wrap still trails
// the entry's current md5 afterwards -- because a newer change arrived
// while the callback ran, or because the callback panicked and the
// watermark stayed put -- wake is invoked to trigger a prompt re-notify.
func (c *cacheData) deliverAndAdvance(lw *listenerWrap, tenant, group, dataId, content, md5 string, wake func()) {
	delivered := callListenerSafely(lw.listener, tenant, group, dataId, content)
	c.mu.Lock()
	lw.inFlight = false
	if delivered {
		lw.lastCallMd5 = md5
	}
	lagging := lw.lastCallMd5 != c.md5 && !c.discard
	c.mu.Unlock()
	if lagging && wake != nil {
		wake()
	}
}

// callListenerSafely invokes listener, recovering from any panic so a single
// misbehaving listener cannot take down the executor goroutine or prevent
// other listeners on the same key from being notified this round. It
// reports whether the callback returned normally.
func callListenerSafely(listener vo.Listener, tenant, group, dataId, content string) (succeeded bool) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("listener panic recovered, dataId=%s, group=%s, tenant=%s, panic=%v", dataId, group, tenant, r)
			succeeded = false
		}
	}()
	listener(tenant, group, dataId, content)
	succeeded = true
	return
}

// markDiscard flags the entry as cancelled: it stops being treated as synced
// and its listeners are dropped. The entry itself is left in the holder for
// removeIfDiscarded (or a future revive via getOrCreate) to reconcile.
func (c *cacheData) markDiscard() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.discard = true
	c.listeners = nil
	c.isSyncWithServer = false
}
