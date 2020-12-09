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

package constant

import (
	"os"

	"github.com/nacos-group/nacos-sdk-go/common/file"
)

func NewClientConfig(opts ...ClientOption) *ClientConfig {
	clientConfig := &ClientConfig{
		TimeoutMs:            10 * 1000,
		BeatInterval:         5 * 1000,
		OpenKMS:              false,
		CacheDir:             file.GetCurrentPath() + string(os.PathSeparator) + "cache",
		UpdateThreadNum:      20,
		NotLoadCacheAtStart:  false,
		UpdateCacheWhenEmpty: false,
		LogDir:               file.GetCurrentPath() + string(os.PathSeparator) + "log",
		RotateTime:           "24h",
		MaxAge:               3,
		LogLevel:             "info",
	}

	for _, opt := range opts {
		opt(clientConfig)
	}

	return clientConfig
}

// ClientOption ...
type ClientOption func(*ClientConfig)
