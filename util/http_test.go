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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodingParams(t *testing.T) {
	tag := "test-tag"
	content := "&123456!@#$"

	params := map[string]string{}
	params["tag"] = tag
	params["content"] = content

	params = EncodingParams(params)

	encodedTag := url.QueryEscape(tag)
	encodedContent := url.QueryEscape(content)
	assert.Equal(t, encodedTag, params["tag"])
	assert.Equal(t, encodedContent, params["content"])
}
