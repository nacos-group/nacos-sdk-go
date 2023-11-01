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

package util

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckInstanceIsLegal(t *testing.T) {

	//beat.timeout and delete.timeout should more than beat.interval
	t.Run("beat.interval-case1", func(t *testing.T) {
		pass, err := CheckInstanceIsLegal(model.Instance{
			Metadata: map[string]string{
				"preserved.heart.beat.timeout":  "2000",
				"preserved.ip.delete.timeout":   "3000",
				"preserved.heart.beat.interval": "4000",
			},
		})
		fmt.Println(err)
		assert.Equal(t, false, pass)
	})

	//clusterName should match ^[0-9a-zA-Z-]+$
	t.Run("clusterName-case2", func(t *testing.T) {
		pass, err := CheckInstanceIsLegal(model.Instance{
			ClusterName: ">clusterName",
		})
		fmt.Println(err)
		assert.Equal(t, false, pass)
	})

	//success-case
	t.Run("success-case", func(t *testing.T) {
		pass, err := CheckInstanceIsLegal(model.Instance{
			ClusterName: "clusterName1230ab",
			Metadata: map[string]string{
				"preserved.heart.beat.timeout":  "2000",
				"preserved.ip.delete.timeout":   "3000",
				"preserved.heart.beat.interval": "1000",
			},
		})
		assert.Equal(t, true, pass, err)
	})
}
