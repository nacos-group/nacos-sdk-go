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

package rpc_response

import "github.com/nacos-group/nacos-sdk-go/v2/model"

type ConfigChangeBatchListenResponse struct {
	*Response
	ChangedConfigs []model.ConfigContext `json:"changedConfigs"`
}

func (c *ConfigChangeBatchListenResponse) GetResponseType() string {
	return "ConfigChangeBatchListenResponse"
}

type ConfigQueryResponse struct {
	*Response
	Content          string `json:"content"`
	EncryptedDataKey string `json:"encryptedDataKey"`
	ContentType      string `json:"contentType"`
	Md5              string `json:"md5"`
	LastModified     int64  `json:"lastModified"`
	IsBeta           bool   `json:"isBeta"`
	Tag              bool   `json:"tag"`
}

func (c *ConfigQueryResponse) GetResponseType() string {
	return "ConfigQueryResponse"
}

type ConfigPublishResponse struct {
	*Response
}

func (c *ConfigPublishResponse) GetResponseType() string {
	return "ConfigPublishResponse"
}

type ConfigRemoveResponse struct {
	*Response
}

func (c *ConfigRemoveResponse) GetResponseType() string {
	return "ConfigRemoveResponse"
}
