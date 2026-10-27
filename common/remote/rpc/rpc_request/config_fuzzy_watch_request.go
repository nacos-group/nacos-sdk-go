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

package rpc_request

import (
	"strconv"
	"time"

	"github.com/nacos-group/nacos-sdk-proto/go/config"
	"google.golang.org/protobuf/proto"

	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
)

// ConfigFuzzyWatchRequest registers or cancels a config fuzzy watch pattern.
// The boolean isInitializing has no @JsonProperty override on the Java side,
// so Jackson derives the wire name "initializing"; sdk-proto beta.9 carries
// json_name=initializing for the proto field, and the legacy tag matches.
type ConfigFuzzyWatchRequest struct {
	*Request
	GroupKeyPattern   string   `json:"groupKeyPattern"`
	ReceivedGroupKeys []string `json:"receivedGroupKeys"`
	WatchType         string   `json:"watchType"`
	IsInitializing    bool     `json:"initializing"`
}

func NewConfigFuzzyWatchRequest(pattern, watchType string, received []string, initializing bool) *ConfigFuzzyWatchRequest {
	return &ConfigFuzzyWatchRequest{
		Request:           &Request{Headers: make(map[string]string, 8)},
		GroupKeyPattern:   pattern,
		ReceivedGroupKeys: received,
		WatchType:         watchType,
		IsInitializing:    initializing,
	}
}

func (r *ConfigFuzzyWatchRequest) GetRequestType() string {
	return constant.CONFIG_FUZZY_WATCH_REQUEST_NAME
}

// GetStringToSign: a pattern request has no single dataId/group to sign
// beyond the timestamp (same as NamingFuzzyWatchRequest).
func (r *ConfigFuzzyWatchRequest) GetStringToSign() string {
	return strconv.FormatInt(time.Now().Unix()*1000, 10)
}

func (r *ConfigFuzzyWatchRequest) ProtoMessage() proto.Message {
	return &config.ConfigFuzzyWatchRequest{
		RequestId:         r.RequestId,
		GroupKeyPattern:   r.GroupKeyPattern,
		ReceivedGroupKeys: r.ReceivedGroupKeys,
		WatchType:         r.WatchType,
		IsInitializing:    r.IsInitializing,
	}
}

// ConfigFuzzyWatchChangeNotifyRequest is a server push for one config change
// under a watched pattern. It has no syncType on the wire.
type ConfigFuzzyWatchChangeNotifyRequest struct {
	*Request
	GroupKey   string `json:"groupKey"`
	ChangeType string `json:"changeType"`
}

func (r *ConfigFuzzyWatchChangeNotifyRequest) GetRequestType() string {
	return constant.CONFIG_FUZZY_WATCH_CHANGE_NOTIFY_REQUEST_NAME
}

type ConfigFuzzyWatchSyncContext struct {
	GroupKey    string `json:"groupKey"`
	ChangedType string `json:"changedType"`
}

// ConfigFuzzyWatchSyncRequest is a server push syncing (a batch of) the
// group keys matching a watched pattern.
type ConfigFuzzyWatchSyncRequest struct {
	*Request
	GroupKeyPattern string                        `json:"groupKeyPattern"`
	SyncType        string                        `json:"syncType"`
	Contexts        []ConfigFuzzyWatchSyncContext `json:"contexts"`
	TotalBatch      int                           `json:"totalBatch"`
	CurrentBatch    int                           `json:"currentBatch"`
}

func (r *ConfigFuzzyWatchSyncRequest) GetRequestType() string {
	return constant.CONFIG_FUZZY_WATCH_SYNC_REQUEST_NAME
}
