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

package model

// FuzzyWatchChangeEvent is delivered to a FuzzyWatch callback whenever a
// service matching the watched pattern is added or deleted. ChangedType is
// one of constant.FUZZY_WATCH_CHANGED_TYPE_ADD_SERVICE /
// FUZZY_WATCH_CHANGED_TYPE_DELETE_SERVICE. SyncType identifies which server
// push produced the event: constant.FUZZY_WATCH_INIT_NOTIFY /
// FUZZY_WATCH_DIFF_SYNC_NOTIFY for a batch sync (also used for the local
// replay a late-joining watcher receives for already-known matches),
// or constant.FUZZY_WATCH_RESOURCE_CHANGED for a post-init change notify.
type FuzzyWatchChangeEvent struct {
	ServiceName string
	GroupName   string
	NamespaceId string
	ChangedType string
	SyncType    string
}

// FuzzyWatchLoadEvent is delivered to a load-event callback when the server
// reports that a fuzzy watch pattern hit a capacity limit (pattern count or
// matched-service count). Pattern is the groupKeyPattern that was rejected,
// and ErrorCode is one of constant.ERROR_CODE_FUZZY_WATCH_PATTERN_OVER_LIMIT
// or ERROR_CODE_FUZZY_WATCH_MATCH_COUNT_OVER_LIMIT.
type FuzzyWatchLoadEvent struct {
	Pattern   string
	ErrorCode int
}

// ConfigFuzzyWatchChangeEvent is delivered to a config FuzzyWatch callback
// when the matched set of configs changes: a config starts matching the
// watched pattern (ChangedType constant.CONFIG_FUZZY_WATCH_CHANGED_TYPE_ADD_CONFIG),
// or stops existing or matching it (CONFIG_FUZZY_WATCH_CHANGED_TYPE_DELETE_CONFIG).
// These are the only two ChangedType values delivered. Content changes of an
// already-matched config fire no fuzzy event; use ListenConfig for content.
type ConfigFuzzyWatchChangeEvent struct {
	DataId      string
	Group       string
	NamespaceId string
	ChangedType string
	SyncType    string
}

// ConfigFuzzyWatchLoadEvent is delivered when the server reports that a config
// fuzzy watch pattern hit a capacity limit.
type ConfigFuzzyWatchLoadEvent struct {
	Pattern   string
	ErrorCode int
}
