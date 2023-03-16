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
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/stretchr/testify/assert"
)

func TestLimiter(t *testing.T) {
	client := createConfigClientTest()
	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "default-group",
		Content: "hello world"})

	assert.Nil(t, err)
	assert.True(t, success)

	for i := 0; i <= 10; i++ {
		content, err := client.GetConfig(vo.ConfigParam{
			DataId: localConfigTest.DataId,
			Group:  "default-group"})
		if i > 4 {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, "hello world", content)
		}
	}
}
