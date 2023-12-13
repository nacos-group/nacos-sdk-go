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

type Listener func(namespace, group, dataId, data string)

type ConfigParam struct {
	DataId           string    `param:"dataId"`  //required
	Group            string    `param:"group"`   //required
	Content          string    `param:"content"` //required
	Tag              string    `param:"tag"`
	AppName          string    `param:"appName"`
	BetaIps          string    `param:"betaIps"`
	CasMd5           string    `param:"casMd5"`
	Type             string    `param:"type"`
	SrcUser          string    `param:"srcUser"`
	EncryptedDataKey string    `param:"encryptedDataKey"`
	KmsKeyId         string    `param:"kmsKeyId"`
	UsageType        UsageType `param:"usageType"`
	OnChange         func(namespace, group, dataId, data string)
}

func (this *ConfigParam) DeepCopy() *ConfigParam {
	if this == nil {
		return nil
	}
	result := new(ConfigParam)
	*result = *this
	return result
}

type UsageType string

const (
	RequestType  UsageType = "RequestType"
	ResponseType UsageType = "ResponseType"
)

type SearchConfigParam struct {
	Search   string `param:"search"`
	DataId   string `param:"dataId"`
	Group    string `param:"group"`
	Tag      string `param:"tag"`
	AppName  string `param:"appName"`
	PageNo   int    `param:"pageNo"`
	PageSize int    `param:"pageSize"`
}
