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
	"testing"

	"github.com/stretchr/testify/assert"
)

type object struct {
	Name     string            `param:"name"`
	Likes    []string          `param:"likes"`
	Metadata map[string]string `param:"metadata"`
	Age      uint64            `param:"age"`
	Healthy  bool              `param:"healthy"`
	Money    int               `param:"money"`
}

func TestTransformObject2Param(t *testing.T) {
	assert.Equal(t, map[string]string{}, TransformObject2Param(nil))
	obj := object{
		Name:  "code",
		Likes: []string{"a", "b"},
		Metadata: map[string]string{
			"M1": "m1",
		},
		Age:     10,
		Healthy: true,
		Money:   10,
	}
	params := TransformObject2Param(&obj)
	assert.Equal(t, map[string]string{
		"name":     "code",
		"metadata": `{"M1":"m1"}`,
		"likes":    "a,b",
		"age":      "10",
		"money":    "10",
		"healthy":  "true",
	}, params)
}
