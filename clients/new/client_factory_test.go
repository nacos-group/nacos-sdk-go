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

package new

import (
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"testing"
)

func TestCreateNameClient(t *testing.T) {
	serverConfigs := &[]constant.ServerConfig{
		*constant.NewServerConfig("console.nacos.io", 80),
	}

	clientConfig := &constant.ClientConfig{
		NamespaceId:         "e525eafa-f7d7-4029-83d9-008937f9d468", //namespace id
		TimeoutMs:           5000,
		ListenInterval:      10000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		RotateTime:          "1h",
		MaxAge:              3,
		LogLevel:            "debug",
	}

	_, err := CreateNamingClient(&constant.Config{
		ServerConfigs: serverConfigs,
		ClientConfig:  clientConfig},
	)

	if err != nil {
		panic(err)
	}
}
