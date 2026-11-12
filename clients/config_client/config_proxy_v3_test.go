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

// v1ConfigListResponse is what the v1/v2 config list API answers: the same page
// metadata as v3 plus the whole content of every config.
const v1ConfigListResponse = `{
  "totalCount": 1,
  "pageNumber": 1,
  "pagesAvailable": 1,
  "pageItems": [
    {
      "id": "13",
      "dataId": "app.yaml",
      "group": "DEFAULT_GROUP",
      "content": "server:\n  port: 8080\n",
      "md5": "cccccccccccccccccccccccccccccccc",
      "tenant": "custom-ns",
      "appname": "higress"
    }
  ]
}`

func newSearchConfigTestProxy(t *testing.T, mux *http.ServeMux) *ConfigProxy {
	t.Helper()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

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
	t.Cleanup(cancel)

	proxy, err := NewConfigProxy(
		ctx,
		[]constant.ServerConfig{*constant.NewServerConfig(serverURL.Hostname(), port)},
		*constant.NewClientConfig(constant.WithTimeoutMs(5000)),
		&http_agent.HttpAgent{},
	)
	if err != nil {
		t.Fatalf("create config proxy: %v", err)
	}
	return proxy.(*ConfigProxy)
}

func TestSearchConfigProxyIgnoreContentPrefersV3(t *testing.T) {
	var mu sync.Mutex
	var v1Calls int

	mux := http.NewServeMux()
	mux.HandleFunc(constant.DEFAULT_CONTEXT_PATH+constant.CONFIG_PATH, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		v1Calls++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(v1ConfigListResponse))
	})
	mux.HandleFunc(constant.DEFAULT_CONTEXT_PATH+"/v3/admin/cs/config/list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(v3CrossGroupResponse))
	})

	proxy := newSearchConfigTestProxy(t, mux)

	page, err := proxy.searchConfigProxy(vo.SearchConfigParam{
		Search:        "accurate",
		DataId:        "app.yaml",
		PageNo:        1,
		PageSize:      10,
		IgnoreContent: true,
	}, "custom-ns", "", "")
	if err != nil {
		t.Fatalf("searchConfigProxy failed: %v", err)
	}

	mu.Lock()
	calls := v1Calls
	mu.Unlock()
	if calls != 0 {
		t.Errorf("v1 API called %d times, want 0 when IgnoreContent is set and v3 is available", calls)
	}
	if len(page.PageItems) != 2 {
		t.Fatalf("unexpected page items length: %d", len(page.PageItems))
	}
	if page.PageItems[0].Group != "GROUP_A" || page.PageItems[0].Tenant != "custom-ns" {
		t.Errorf("unexpected first item: %+v", page.PageItems[0])
	}
}

func TestSearchConfigProxyIgnoreContentStripsV1Content(t *testing.T) {
	var mu sync.Mutex
	var v3Calls int

	mux := http.NewServeMux()
	mux.HandleFunc(constant.DEFAULT_CONTEXT_PATH+"/v3/admin/cs/config/list", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		v3Calls++
		mu.Unlock()
		http.Error(w, "v3 API unavailable", http.StatusNotFound)
	})
	mux.HandleFunc(constant.DEFAULT_CONTEXT_PATH+constant.CONFIG_PATH, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(v1ConfigListResponse))
	})

	proxy := newSearchConfigTestProxy(t, mux)

	page, err := proxy.searchConfigProxy(vo.SearchConfigParam{
		Search:        "accurate",
		DataId:        "app.yaml",
		PageNo:        1,
		PageSize:      10,
		IgnoreContent: true,
	}, "custom-ns", "", "")
	if err != nil {
		t.Fatalf("searchConfigProxy failed: %v", err)
	}

	mu.Lock()
	tried := v3Calls
	mu.Unlock()
	if tried == 0 {
		t.Fatal("expected the v3 API to be tried before falling back to v1")
	}
	if page.TotalCount != 1 || len(page.PageItems) != 1 {
		t.Fatalf("unexpected page: %+v", page)
	}
	item := page.PageItems[0]
	if item.Content != "" {
		t.Errorf("Content = %q, want empty when IgnoreContent is set", item.Content)
	}
	if item.DataId != "app.yaml" || item.Group != "DEFAULT_GROUP" || item.Tenant != "custom-ns" {
		t.Errorf("metadata should be kept, got %+v", item)
	}
}

func TestSearchConfigProxyKeepsContentByDefault(t *testing.T) {
	var mu sync.Mutex
	var v3Calls int

	mux := http.NewServeMux()
	mux.HandleFunc(constant.DEFAULT_CONTEXT_PATH+"/v3/admin/cs/config/list", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		v3Calls++
		mu.Unlock()
		http.Error(w, "v3 API unavailable", http.StatusNotFound)
	})
	mux.HandleFunc(constant.DEFAULT_CONTEXT_PATH+constant.CONFIG_PATH, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(v1ConfigListResponse))
	})

	proxy := newSearchConfigTestProxy(t, mux)

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
	calls := v3Calls
	mu.Unlock()
	if calls != 0 {
		t.Errorf("v3 API called %d times, want 0 when the v1 API answers", calls)
	}
	if len(page.PageItems) != 1 {
		t.Fatalf("unexpected page items length: %d", len(page.PageItems))
	}
	if got := page.PageItems[0].Content; got != "server:\n  port: 8080\n" {
		t.Errorf("Content = %q, want the config content to be kept by default", got)
	}
}
