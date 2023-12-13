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

package vo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigParamDeepCopy(t *testing.T) {
	t.Run("config param deep copy", func(t *testing.T) {
		param := &ConfigParam{
			DataId:           "dataId",
			Group:            "",
			Content:          "common content",
			Tag:              "",
			AppName:          "",
			BetaIps:          "",
			CasMd5:           "",
			Type:             "",
			SrcUser:          "",
			EncryptedDataKey: "",
			KmsKeyId:         "",
			UsageType:        RequestType,
			OnChange: func(namespace, group, dataId, data string) {
				//do nothing
			},
		}
		paramDeepCopied := param.DeepCopy()

		assert.Equal(t, param.DataId, paramDeepCopied.DataId)
		assert.Equal(t, param.Content, paramDeepCopied.Content)
		assert.NotEqual(t, &param.OnChange, &paramDeepCopied.OnChange)
		assert.NotEqual(t, param, paramDeepCopied)
	})
}
