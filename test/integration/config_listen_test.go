//go:build integration

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

package integration

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nacos-group/nacos-sdk-go/v3/clients"
	"github.com/nacos-group/nacos-sdk-go/v3/util"
	"github.com/nacos-group/nacos-sdk-go/v3/vo"
)

// TestIntegrationConfigMultiListener verifies that two independent listeners
// registered on the same dataId/group both receive a subsequently published
// change -- ListenConfig must append listeners rather than replace them.
func TestIntegrationConfigMultiListener(t *testing.T) {
	client, err := clients.NewConfigClient(clientParam(t))
	require.NoError(t, err, "create config client")
	defer client.CloseClient()

	dataId := fmt.Sprintf("it-config-multi-%d", time.Now().UnixNano())
	defer func() {
		_, _ = client.DeleteConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
	}()

	first := make(chan string, 1)
	second := make(chan string, 1)

	require.NoError(t, client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  testGroup,
		OnChange: func(namespace, group, dataId, data string) {
			select {
			case first <- data:
			default:
			}
		},
	}), "first listener")

	require.NoError(t, client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  testGroup,
		OnChange: func(namespace, group, dataId, data string) {
			select {
			case second <- data:
			default:
			}
		},
	}), "second listener")

	ok, err := client.PublishConfig(vo.ConfigParam{DataId: dataId, Group: testGroup, Content: "multi-v1"})
	require.NoError(t, err, "publish config")
	require.True(t, ok, "publish config should return true")

	deadline := time.After(waitTimeout)
	var gotFirst, gotSecond bool
	for !gotFirst || !gotSecond {
		select {
		case data := <-first:
			assert.Equal(t, "multi-v1", data)
			gotFirst = true
		case data := <-second:
			assert.Equal(t, "multi-v1", data)
			gotSecond = true
		case <-deadline:
			t.Fatalf("timed out after %s waiting for both listeners to fire (first=%v second=%v)",
				waitTimeout, gotFirst, gotSecond)
		}
	}

	_ = client.CancelListenConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
}

// TestIntegrationConfigCancelStopsPush is a regression test for #629: once
// CancelListenConfig has been called for a key, further server-side changes
// to that key must not trigger the (now cancelled) OnChange callback.
func TestIntegrationConfigCancelStopsPush(t *testing.T) {
	client, err := clients.NewConfigClient(clientParam(t))
	require.NoError(t, err, "create config client")
	defer client.CloseClient()

	dataId := fmt.Sprintf("it-config-cancel-%d", time.Now().UnixNano())
	defer func() {
		_, _ = client.DeleteConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
	}()

	changes := make(chan string, 4)
	require.NoError(t, client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  testGroup,
		OnChange: func(namespace, group, dataId, data string) {
			select {
			case changes <- data:
			default:
			}
		},
	}))

	ok, err := client.PublishConfig(vo.ConfigParam{DataId: dataId, Group: testGroup, Content: "cancel-v1"})
	require.NoError(t, err, "publish config v1")
	require.True(t, ok)

	select {
	case data := <-changes:
		assert.Equal(t, "cancel-v1", data)
	case <-time.After(waitTimeout):
		t.Fatalf("timed out after %s waiting for first change notification", waitTimeout)
	}

	require.NoError(t, client.CancelListenConfig(vo.ConfigParam{DataId: dataId, Group: testGroup}))

	// This pins the user-visible contract: once CancelListenConfig returns, no
	// further OnChange callback fires for the key, and it exercises the full
	// cancel path end-to-end against a real server. CancelListenConfig clears
	// the entry's listeners synchronously, so the no-callback assertion below
	// does not depend on the Listen=false batch actually reaching the server
	// -- that server-side unsubscription is covered separately by the
	// executor unit tests asserting the Listen=false request stream, plus
	// probe verification on 2.5.2/3.2.0. The settle sleep here just gives the
	// executor round (bell via asyncNotifyListenConfig, or the 5s poll
	// fallback) a chance to complete, so this run also exercises the
	// listen=false send + reap path rather than skipping it entirely.
	time.Sleep(6 * time.Second)

	ok, err = client.PublishConfig(vo.ConfigParam{DataId: dataId, Group: testGroup, Content: "cancel-v2"})
	require.NoError(t, err, "publish config v2")
	require.True(t, ok)

	select {
	case data := <-changes:
		t.Fatalf("received unexpected callback after cancel: %q", data)
	case <-time.After(10 * time.Second):
		// expected: no callback within the window
	}
}

// TestIntegrationConfigCasPublish is a regression test for #727: PublishConfig
// with a stale CasMd5 must fail without mutating the stored content, and
// publishing with the current content's md5 must succeed.
func TestIntegrationConfigCasPublish(t *testing.T) {
	client, err := clients.NewConfigClient(clientParam(t))
	require.NoError(t, err, "create config client")
	defer client.CloseClient()

	dataId := fmt.Sprintf("it-config-cas-%d", time.Now().UnixNano())
	defer func() {
		_, _ = client.DeleteConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
	}()

	baseContent := "cas-base"
	ok, err := client.PublishConfig(vo.ConfigParam{DataId: dataId, Group: testGroup, Content: baseContent})
	require.NoError(t, err, "publish base config")
	require.True(t, ok)

	eventually(t, "base content becomes readable", func() bool {
		got, err := client.GetConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
		return err == nil && got == baseContent
	})

	// Wrong CasMd5: publish must fail and leave the stored content untouched.
	ok, err = client.PublishConfig(vo.ConfigParam{
		DataId:  dataId,
		Group:   testGroup,
		Content: "cas-should-not-apply",
		CasMd5:  "0000000000000000000000000000000",
	})
	assert.Error(t, err, "publish with a stale CasMd5 should fail")
	assert.False(t, ok, "publish with a stale CasMd5 should report false")

	got, err := client.GetConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
	require.NoError(t, err, "get config after failed cas publish")
	assert.Equal(t, baseContent, got, "content must be unchanged after a failed cas publish")

	// Correct CasMd5 (matching the currently stored content): publish must
	// succeed and update the content.
	updatedContent := "cas-updated"
	ok, err = client.PublishConfig(vo.ConfigParam{
		DataId:  dataId,
		Group:   testGroup,
		Content: updatedContent,
		CasMd5:  util.Md5(baseContent),
	})
	require.NoError(t, err, "publish with the current CasMd5 should succeed")
	require.True(t, ok)

	eventually(t, "updated content becomes readable", func() bool {
		got, err := client.GetConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
		return err == nil && got == updatedContent
	})
}

// readinessURLs are the health endpoints to poll after restarting the
// container: Nacos 2.x and 3.x expose readiness under different paths, so
// both are tried and either responding with 200 is accepted.
func readinessURLs() []string {
	base := fmt.Sprintf("http://%s:%s", envOr("NACOS_SERVER_IP", "127.0.0.1"), envOr("NACOS_SERVER_PORT", "8848"))
	return []string{
		base + "/nacos/v1/console/health/readiness",
		base + "/nacos/v3/admin/core/state/readiness",
	}
}

func waitServerReady(t *testing.T, timeout time.Duration) {
	t.Helper()
	httpClient := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, u := range readinessURLs() {
			resp, err := httpClient.Get(u)
			if err != nil {
				continue
			}
			status := resp.StatusCode
			resp.Body.Close()
			if status == http.StatusOK {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("server did not become ready within %s", timeout)
}

// TestIntegrationConfigListenSurvivesRestart is a regression test for #694:
// a config listener registered before a server restart must keep receiving
// pushes once the server (and the client's underlying gRPC connection) has
// recovered.
func TestIntegrationConfigListenSurvivesRestart(t *testing.T) {
	container := os.Getenv("NACOS_CONTAINER_NAME")
	if container == "" {
		t.Skip("NACOS_CONTAINER_NAME not set; skipping restart test")
	}

	client, err := clients.NewConfigClient(clientParam(t))
	require.NoError(t, err, "create config client")
	defer client.CloseClient()

	dataId := fmt.Sprintf("it-config-restart-%d", time.Now().UnixNano())
	defer func() {
		_, _ = client.DeleteConfig(vo.ConfigParam{DataId: dataId, Group: testGroup})
	}()

	changes := make(chan string, 4)
	require.NoError(t, client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  testGroup,
		OnChange: func(namespace, group, dataId, data string) {
			select {
			case changes <- data:
			default:
			}
		},
	}))

	ok, err := client.PublishConfig(vo.ConfigParam{DataId: dataId, Group: testGroup, Content: "restart-v1"})
	require.NoError(t, err, "publish before restart")
	require.True(t, ok)

	select {
	case data := <-changes:
		assert.Equal(t, "restart-v1", data)
	case <-time.After(waitTimeout):
		t.Fatalf("timed out after %s waiting for pre-restart change notification", waitTimeout)
	}

	out, err := exec.Command("docker", "restart", container).CombinedOutput()
	require.NoError(t, err, "docker restart %s: %s", container, out)

	waitServerReady(t, 180*time.Second)

	// Publish may need a few retries while the client's own gRPC stream is
	// still reconnecting after the restart.
	publishDeadline := time.Now().Add(60 * time.Second)
	published := false
	for time.Now().Before(publishDeadline) {
		if ok, err := client.PublishConfig(vo.ConfigParam{DataId: dataId, Group: testGroup, Content: "restart-v2"}); err == nil && ok {
			published = true
			break
		}
		time.Sleep(2 * time.Second)
	}
	require.True(t, published, "publish after restart should eventually succeed")

	select {
	case data := <-changes:
		assert.Equal(t, "restart-v2", data)
	case <-time.After(60 * time.Second):
		t.Fatal("client did not receive the post-restart config change within 60s")
	}
}
