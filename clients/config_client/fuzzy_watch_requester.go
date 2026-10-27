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
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_response"
)

// ErrConfigFuzzyWatchNotSupported is returned by RegisterWatcher when the
// connected server does not advertise the fuzzyWatch ability (2.x servers).
// Java 3.2.3 uses the single AbilityKey.SERVER_FUZZY_WATCH ("fuzzyWatch") for
// both naming and config, so the naming constant is reused.
var ErrConfigFuzzyWatchNotSupported = errors.New("config fuzzy watch is not supported by the connected nacos server (requires nacos 3.x)")

// ConfigFuzzyWatchServerError is a non-success ConfigFuzzyWatchResponse
// surfaced as an error with the server's errorCode preserved, so the reconcile
// worker can tell capacity rejections (suppress + notify load watchers) from
// transient failures (backoff + retry).
type ConfigFuzzyWatchServerError struct {
	ErrorCode int
	Message   string
}

func (e *ConfigFuzzyWatchServerError) Error() string {
	return fmt.Sprintf("config fuzzy watch server error, errorCode:%d message:%s", e.ErrorCode, e.Message)
}

// configFuzzyWatchRequester is the production ConfigFuzzyWatchRequester. Its
// transport and ability probe are function fields so ConfigProxy can bind them
// to the real rpc client (auth/comm headers injected by requestProxy) while
// tests substitute fakes.
type configFuzzyWatchRequester struct {
	send             func(rpc_request.IRequest) (rpc_response.IResponse, error)
	abilitySupported func() bool
}

var _ ConfigFuzzyWatchRequester = (*configFuzzyWatchRequester)(nil)

// newConfigFuzzyWatchRequester binds a requester to the proxy and the client's
// rpc channel.
func newConfigFuzzyWatchRequester(cp *ConfigProxy, client *ConfigClient) *configFuzzyWatchRequester {
	return &configFuzzyWatchRequester{
		send: func(req rpc_request.IRequest) (rpc_response.IResponse, error) {
			return cp.requestProxy(cp.getRpcClient(client), req, cp.clientConfig.TimeoutMs)
		},
		// Bounded wait covers the window where the SetupAck push has not
		// landed yet right after (re)connect.
		abilitySupported: func() bool {
			return cp.getRpcClient(client).IsAbilitySupportedByServer(constant.ABILITY_KEY_SERVER_FUZZY_WATCH, 2*time.Second)
		},
	}
}

func (r *configFuzzyWatchRequester) CheckFuzzyWatchSupported() error {
	if !r.abilitySupported() {
		return ErrConfigFuzzyWatchNotSupported
	}
	return nil
}

// SendFuzzyWatchRequest sends one WATCH/CANCEL_WATCH request on behalf of the
// reconcile worker. A non-success reply keeps its server errorCode via
// *ConfigFuzzyWatchServerError.
func (r *configFuzzyWatchRequester) SendFuzzyWatchRequest(groupKeyPattern, watchType string, receivedGroupKeys []string, isInitializing bool) error {
	request := rpc_request.NewConfigFuzzyWatchRequest(groupKeyPattern, watchType, receivedGroupKeys, isInitializing)
	response, err := r.send(request)
	if err != nil {
		return err
	}
	if response == nil {
		return errors.Errorf("config fuzzy watch pattern:%s watchType:%s got nil response", groupKeyPattern, watchType)
	}
	if !response.IsSuccess() {
		return &ConfigFuzzyWatchServerError{ErrorCode: response.GetErrorCode(), Message: response.GetMessage()}
	}
	return nil
}
