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
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
)

type ConfigConnectionEventListener struct {
	client *ConfigClient
	taskId string
}

func NewConfigConnectionEventListener(client *ConfigClient, taskId string) *ConfigConnectionEventListener {
	return &ConfigConnectionEventListener{
		client: client,
		taskId: taskId,
	}
}

func (c *ConfigConnectionEventListener) OnConnected() {
	logger.Info("[ConfigConnectionEventListener] connect to config server for taskId: " + c.taskId)
	if c.client != nil {
		c.client.asyncNotifyListenConfig()
	}
}

func (c *ConfigConnectionEventListener) OnDisConnect() {
	logger.Info("[ConfigConnectionEventListener] disconnect from config server for taskId: " + c.taskId)

	if c.client != nil {
		taskIdInt, err := strconv.Atoi(c.taskId)
		if err != nil {
			logger.Errorf("[ConfigConnectionEventListener] parse taskId error: %v", err)
			return
		}

		items := c.client.cacheMap.Items()
		for key, v := range items {
			if data, ok := v.(cacheData); ok {
				if data.taskId == taskIdInt {
					data.isSyncWithServer = false
					c.client.cacheMap.Set(key, data)
				}
			}
		}
	}
}
