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
	"sync"
)

// ConfigFuzzyWatchHandle identifies one FuzzyWatchConfig registration.
// Cancellation is watcher-scoped: Cancel removes only this registration, and
// the server-side watch is torn down by the reconcile worker after the
// pattern's last registration is gone.
type ConfigFuzzyWatchHandle struct {
	holder  *ConfigFuzzyWatchHolder
	pattern string
	id      uint64
	once    sync.Once
}

// Pattern returns the resolved groupKeyPattern (namespace>>group>>dataId).
func (h *ConfigFuzzyWatchHandle) Pattern() string { return h.pattern }

// Cancel removes this registration. Idempotent.
func (h *ConfigFuzzyWatchHandle) Cancel() {
	h.once.Do(func() { h.holder.RemoveWatcherByID(h.pattern, h.id) })
}

// MatchedGroupKeys blocks until the pattern's initial server sync finishes,
// then returns the matched groupKeys (dataId+group+namespace). Cancel the
// context to bound the wait.
func (h *ConfigFuzzyWatchHandle) MatchedGroupKeys(ctx context.Context) ([]string, error) {
	return h.holder.MatchedGroupKeys(ctx, h.pattern)
}
