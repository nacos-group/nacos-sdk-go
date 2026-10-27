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
	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
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
