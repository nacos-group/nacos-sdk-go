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

package model

import "encoding/json"

type ConfigItem struct {
	Id      json.Number `param:"id"`
	DataId  string      `param:"dataId"`
	Group   string      `param:"group"`
	Content string      `param:"content"`
	Md5     string      `param:"md5"`
	Tenant  string      `param:"tenant"`
	Appname string      `param:"appname"`
}
type ConfigPage struct {
	TotalCount     int          `param:"totalCount"`
	PageNumber     int          `param:"pageNumber"`
	PagesAvailable int          `param:"pagesAvailable"`
	PageItems      []ConfigItem `param:"pageItems"`
}

type ConfigListenContext struct {
	Group  string `json:"group"`
	Md5    string `json:"md5"`
	DataId string `json:"dataId"`
	Tenant string `json:"tenant"`
}

type ConfigContext struct {
	Group  string `json:"group"`
	DataId string `json:"dataId"`
	Tenant string `json:"tenant"`
}

// ConfigPageResult is the response envelope of the v3 admin config list API.
//
// Deprecated: the SDK decodes the v3 admin config list response internally, so
// this type is no longer used. It is retained for source compatibility with
// callers that still reference it and may be removed in a future release.
type ConfigPageResult struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    ConfigPage `json:"data"`
}
