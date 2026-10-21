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
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// Reported in https://github.com/nacos-group/nacos-sdk-go/issues/907: the v3
// admin config list API returns groupName/namespaceId and no group/tenant or
// content fields, so the raw v1-style ConfigItem decodes empty Group/Tenant.
const v3ConfigListResponse = `{
  "code": 0,
  "message": "success",
  "data": {
    "totalCount": 2,
    "pageNumber": 1,
    "pagesAvailable": 1,
    "pageItems": [
      {
        "id": 1,
        "dataId": "ingresses.default",
        "groupName": "DEFAULT_GROUP",
        "namespaceId": "",
        "md5": "9f8a1c2b3d4e5f60718293a4b5c6d7e8",
        "type": "yaml"
      },
      {
        "id": 2,
        "dataId": "wasm-plugin-detail.default",
        "groupName": "DEFAULT_GROUP",
        "namespaceId": "public",
        "appName": "higress",
        "md5": "0a1b2c3d4e5f60718293a4b5c6d7e8f",
        "type": "json"
      }
    ]
  }
}`

func TestParseV3ConfigListResult(t *testing.T) {
	page, err := parseV3ConfigListResult([]byte(v3ConfigListResponse))
	if err != nil {
		t.Fatalf("parseV3ConfigListResult failed: %v", err)
	}
	if page.TotalCount != 2 || page.PageNumber != 1 || page.PagesAvailable != 1 {
		t.Fatalf("unexpected page header: %+v", page)
	}
	if len(page.PageItems) != 2 {
		t.Fatalf("unexpected page items length: %d", len(page.PageItems))
	}

	first := page.PageItems[0]
	if first.DataId != "ingresses.default" {
		t.Errorf("DataId = %q, want %q", first.DataId, "ingresses.default")
	}
	if first.Group != "DEFAULT_GROUP" {
		t.Errorf("Group = %q, want %q (normalized from groupName)", first.Group, "DEFAULT_GROUP")
	}
	if first.Tenant != "" {
		t.Errorf("Tenant = %q, want empty", first.Tenant)
	}

	second := page.PageItems[1]
	if second.Group != "DEFAULT_GROUP" {
		t.Errorf("Group = %q, want %q (normalized from groupName)", second.Group, "DEFAULT_GROUP")
	}
	if second.Tenant != "public" {
		t.Errorf("Tenant = %q, want %q (normalized from namespaceId)", second.Tenant, "public")
	}
	if second.Appname != "higress" {
		t.Errorf("Appname = %q, want %q", second.Appname, "higress")
	}
	if second.Md5 == "" {
		t.Error("Md5 should be carried over")
	}
	if second.Id.String() != "2" {
		t.Errorf("Id = %q, want %q", second.Id.String(), "2")
	}
}

func TestParseV3ConfigListResultEmptyItems(t *testing.T) {
	page, err := parseV3ConfigListResult([]byte(`{"code":0,"message":"success","data":{"totalCount":0,"pageNumber":1,"pagesAvailable":0,"pageItems":[]}}`))
	if err != nil {
		t.Fatalf("parseV3ConfigListResult failed: %v", err)
	}
	if page.TotalCount != 0 || len(page.PageItems) != 0 {
		t.Fatalf("unexpected empty page: %+v", page)
	}
}

func TestParseV3ConfigListResultInvalidBody(t *testing.T) {
	_, err := parseV3ConfigListResult([]byte(`not-json`))
	if err == nil {
		t.Fatal("expected error for invalid body, got nil")
	}
}

// The v1 API is unavailable here, so SearchConfig must fall back to the v3 admin
// API and still distinguish the same dataId across non-default groups in a
// custom namespace.
const v3CrossGroupResponse = `{
  "code": 0,
  "message": "success",
  "data": {
    "totalCount": 2,
    "pageNumber": 1,
    "pagesAvailable": 1,
    "pageItems": [
      {
        "id": 11,
        "dataId": "app.yaml",
        "groupName": "GROUP_A",
        "namespaceId": "custom-ns",
        "md5": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      },
      {
        "id": 12,
        "dataId": "app.yaml",
        "groupName": "GROUP_B",
        "namespaceId": "custom-ns",
        "md5": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
      }
    ]
  }
}`

func TestSearchConfigProxyV3FallbackCrossGroup(t *testing.T) {
	var mu sync.Mutex
	var v3Query url.Values
	var v1Calls int

	mux := http.NewServeMux()
	mux.HandleFunc(constant.DEFAULT_CONTEXT_PATH+constant.CONFIG_PATH, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		v1Calls++
		mu.Unlock()
		http.Error(w, "v1 API unavailable", http.StatusNotFound)
	})
	mux.HandleFunc(constant.DEFAULT_CONTEXT_PATH+"/v3/admin/cs/config/list", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		v3Query = r.URL.Query()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(v3CrossGroupResponse))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	port, err := strconv.ParseUint(serverURL.Port(), 10, 64)
	if err != nil {
		t.Fatalf("parse test server port: %v", err)
	}

	// NewConfigProxy starts a token auto-refresh goroutine bound to this context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy, err := NewConfigProxy(
		ctx,
		[]constant.ServerConfig{*constant.NewServerConfig(serverURL.Hostname(), port)},
		*constant.NewClientConfig(constant.WithTimeoutMs(5000)),
		&http_agent.HttpAgent{},
	)
	if err != nil {
		t.Fatalf("create config proxy: %v", err)
	}

	page, err := proxy.searchConfigProxy(vo.SearchConfigParam{
		Search:   "accurate",
		DataId:   "app.yaml",
		PageNo:   1,
		PageSize: 10,
	}, "custom-ns", "", "")
	if err != nil {
		t.Fatalf("searchConfigProxy failed: %v", err)
	}

	mu.Lock()
	v1Tried := v1Calls
	query := v3Query
	mu.Unlock()

	if v1Tried == 0 {
		t.Fatal("expected the v1 API to be attempted before falling back")
	}
	if got := query.Get("dataId"); got != "app.yaml" {
		t.Errorf("v3 dataId = %q, want %q", got, "app.yaml")
	}
	if got := query.Get("groupName"); got != "" {
		t.Errorf("v3 groupName = %q, want empty for cross-group search", got)
	}
	if got := query.Get("namespaceId"); got != "custom-ns" {
		t.Errorf("v3 namespaceId = %q, want %q", got, "custom-ns")
	}

	if len(page.PageItems) != 2 {
		t.Fatalf("unexpected page items length: %d", len(page.PageItems))
	}
	wantGroups := []string{"GROUP_A", "GROUP_B"}
	for i, item := range page.PageItems {
		if item.DataId != "app.yaml" {
			t.Errorf("item %d DataId = %q, want %q", i, item.DataId, "app.yaml")
		}
		if item.Group != wantGroups[i] {
			t.Errorf("item %d Group = %q, want %q", i, item.Group, wantGroups[i])
		}
		if item.Tenant != "custom-ns" {
			t.Errorf("item %d Tenant = %q, want %q", i, item.Tenant, "custom-ns")
		}
	}
}
