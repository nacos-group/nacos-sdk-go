package config_client

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/mock"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

/**
*
* @description :
*
* @author : codezhang
*
* @create : 2019-01-16 21:01
**/

var clientConfigTest = constant.ClientConfig{
	TimeoutMs:      20000,
	ListenInterval: 10000,
	BeatInterval:   10000,
}

var serverConfigTest = constant.ServerConfig{
	ContextPath: "/nacos",
	Port:        80,
	IpAddr:      "console.nacos.io",
}

var configParamMapTest = map[string]string{
	"dataId": "dataId",
	"group":  "group",
}

var configParamTest = vo.ConfigParam{
	DataId: "dataId",
	Group:  "group",
}

var localConfigTest = vo.ConfigParam{
	DataId:  "dataId",
	Group:   "group",
	Content: "content",
}

var localConfigMapTest = map[string]string{
	"dataId":  "dataId",
	"group":   "group",
	"content": "content",
}

var headerTest = map[string][]string{
	"Content-Type": {"application/x-www-form-urlencoded"},
}

var serverConfigsTest = []constant.ServerConfig{serverConfigTest}

var httpAgentTest = mock.MockIHttpAgent{}

func cretateConfigClientTest() ConfigClient {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, _ := NewConfigClient(&nc)
	return client
}

func cretateConfigClientHttpTest(mockHttpAgent http_agent.IHttpAgent) ConfigClient {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTest)
	nc.SetHttpAgent(mockHttpAgent)
	client, _ := NewConfigClient(&nc)
	return client
}

// sync

func Test_SyncWithoutClientConfig(t *testing.T) {
	client := cretateConfigClientTest()
	_, _, _, err := client.sync()
	assert.NotNil(t, err)
}

func Test_SyncWithoutServerConfig(t *testing.T) {
	client := cretateConfigClientTest()
	_ = client.SetClientConfig(clientConfigTest)
	_, _, _, err := client.sync()
	assert.NotNil(t, err)
}

func Test_SyncWithoutHttpAgent(t *testing.T) {
	client := cretateConfigClientTest()
	_ = client.SetServerConfig(serverConfigsTest)
	_ = client.SetClientConfig(clientConfigTest)
	_, _, _, err := client.sync()
	assert.NotNil(t, err)
}

func Test_Sync(t *testing.T) {
	client := cretateConfigClientTest()
	_ = client.SetClientConfig(clientConfigTest)
	_ = client.SetServerConfig(serverConfigsTest)
	_ = client.SetHttpAgent(&httpAgentTest)
	_, _, _, err := client.sync()
	assert.Nil(t, err)
}

// GetConfigContent

func Test_GetConfigContentWithoutDataId(t *testing.T) {
	client := cretateConfigClientTest()
	_, err := client.GetConfigContent("", "Test")
	assert.NotNil(t, err)
}

func Test_GetConfigContentWithoutGroup(t *testing.T) {
	client := cretateConfigClientTest()
	_, err := client.GetConfigContent("Test", "")
	assert.NotNil(t, err)
}

func Test_GetConfigContentWithoutLocalConfig(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	client := cretateConfigClientTest()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	_ = client.SetHttpAgent(mockHttpAgent)
	_ = client.SetClientConfig(clientConfigTest)
	_ = client.SetServerConfig(serverConfigsTest)
	mockHttpAgent.EXPECT().Get(
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.Nil(),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "content"), nil)
	content, err := client.GetConfigContent("dataId", "group")
	assert.Nil(t, err)
	assert.Equal(t, "content", content)
}

func Test_GetConfigContentWithLocalConfig(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	client := cretateConfigClientTest()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	_ = client.SetHttpAgent(mockHttpAgent)
	_ = client.SetClientConfig(clientConfigTest)
	_ = client.SetServerConfig(serverConfigsTest)
	client.localConfigs = []vo.ConfigParam{
		localConfigTest,
	}
	content, err := client.GetConfigContent("dataId", "group")
	assert.Nil(t, err)
	assert.Equal(t, "content", content)
}

// GetConfig

func Test_GetConfigWithoutDataId(t *testing.T) {
	client := cretateConfigClientTest()
	_, err := client.GetConfigContent("", "Test")
	assert.NotNil(t, err)
}

func Test_GetConfigWithoutGroup(t *testing.T) {
	client := cretateConfigClientTest()
	_, err := client.GetConfigContent("Test", "")
	assert.NotNil(t, err)
}

func Test_GetConfig(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)

	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "content"), nil)
	content, err := client.GetConfig(configParamTest)
	assert.Nil(t, err)
	assert.Equal(t, "content", content)
}

func Test_GetConfigWithErrorResponse(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no auth"), nil)
	_, err := client.GetConfig(configParamTest)
	assert.NotNil(t, err)
}

func Test_GetConfigWithCache(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)

	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "content"), nil)
	content, err := client.GetConfig(configParamTest)
	assert.Nil(t, err)
	assert.Equal(t, "content", content)

	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no auth"), nil)
	content, err = client.GetConfig(configParamTest)
	assert.Nil(t, err)
	assert.Equal(t, "content", content)
}

// PublishConfig

func Test_PublishConfigWithoutDataId(t *testing.T) {
	client := cretateConfigClientTest()
	_, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "",
		Group:   "group",
		Content: "content",
	})
	assert.NotNil(t, err)
}

func Test_PublishConfigWithoutGroup(t *testing.T) {
	client := cretateConfigClientTest()
	_, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "",
		Content: "content",
	})
	assert.NotNil(t, err)
}

func Test_PublishConfigWithoutContent(t *testing.T) {
	client := cretateConfigClientTest()
	_, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "",
	})
	assert.NotNil(t, err)
}

func Test_PublishConfig(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodPost),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(localConfigMapTest),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "true"), nil)
	success, err := client.PublishConfig(localConfigTest)
	assert.Nil(t, err)
	assert.True(t, success)
}

func Test_PublishConfigWithErrorResponse(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodPost),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(localConfigMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no auth"), nil)
	success, err := client.PublishConfig(localConfigTest)
	assert.NotNil(t, err)
	assert.True(t, !success)
}

func Test_PublishConfigWithErrorResponse_200(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodPost),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(localConfigMapTest),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "false"), nil)
	success, err := client.PublishConfig(localConfigTest)
	assert.NotNil(t, err)
	assert.True(t, !success)
}

// DeleteConfig

func Test_DeleteConfig(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)

	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodDelete),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "true"), nil)
	success, err := client.DeleteConfig(configParamTest)
	assert.Nil(t, err)
	assert.Equal(t, true, success)
}

func Test_DeleteConfigWithErrorResponse_200(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)

	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodDelete),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "false"), nil)
	success, err := client.DeleteConfig(configParamTest)
	assert.NotNil(t, err)
	assert.Equal(t, false, success)
}

func Test_DeleteConfigWithErrorResponse_401(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)

	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodDelete),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no auth"), nil)
	success, err := client.DeleteConfig(configParamTest)
	assert.NotNil(t, err)
	assert.Equal(t, false, success)
}

func Test_DeleteConfigWithoutDataId(t *testing.T) {
	client := cretateConfigClientTest()
	success, err := client.DeleteConfig(vo.ConfigParam{
		DataId: "",
		Group:  "group",
	})
	assert.NotNil(t, err)
	assert.Equal(t, false, success)
}

func Test_DeleteConfigWithoutGroup(t *testing.T) {
	client := cretateConfigClientTest()
	success, err := client.DeleteConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "",
	})
	assert.NotNil(t, err)
	assert.Equal(t, false, success)
}

// ListenConfig

func TestListenConfig(t *testing.T) {
	client := cretateConfigClientTest()
	client.listening = true

	err := client.ListenConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "group",
		OnChange: func(namespace, group, dataId, data string) {
			fmt.Print("group:" + group + ", dataId:" + dataId + ", data:" + data)
		},
	})

	time.Sleep(1000000 * time.Second)

	assert.True(t, client.listening)
	assert.NotNil(t, err)
}

// listenConfigTask

func Test_listenConfigTask_NoChange(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Post(
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs/listener"),
		gomock.AssignableToTypeOf(headerTest),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(map[string]string{
			"Listening-Configs": "dataIdgroup9a0364b9e99bb480dd25e1f0284c8555tenant",
		}),
	).Times(1).Return(http_agent.FakeHttpResponse(200, ""), nil)

	client.listening = true
	changeCount := 0
	client.listenConfigTask(clientConfigTest, serverConfigsTest, mockHttpAgent, vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "content",
		OnChange: func(namespace, group, dataId, data string) {
			changeCount = changeCount + 1
		}})

	assert.Equal(t, 0, changeCount)
}

func Test_listenConfigTask_Change_WithTenant(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	client := cretateConfigClientTest()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	mockHttpAgent.EXPECT().Post(
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs/listener"),
		gomock.AssignableToTypeOf(headerTest),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(map[string]string{
			"Listening-Configs": "dataIdgroup9a0364b9e99bb480dd25e1f0284c8555tenant",
		}),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "dataId%02group%02tenant%01"), nil)

	mockHttpAgent.EXPECT().Get(
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(headerTest),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(map[string]string{
			"dataId": "dataId",
			"group":  "group",
			"tenant": "tenant",
		}),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "content2"), nil)

	_ = client.SetHttpAgent(mockHttpAgent)
	_ = client.SetClientConfig(clientConfigTest)
	_ = client.SetServerConfig(serverConfigsTest)
	client.listening = true
	configData := ""
	client.listenConfigTask(clientConfigTest, serverConfigsTest, mockHttpAgent, vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "content",
		OnChange: func(namespace, group, dataId, data string) {
			configData = data
		}})

	assert.Equal(t, "content2", configData)
}

func Test_listenConfigTask_Change_NoTenant(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	client := cretateConfigClientTest()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	mockHttpAgent.EXPECT().Post(
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs/listener"),
		gomock.AssignableToTypeOf(headerTest),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(map[string]string{
			"Listening-Configs": "dataIdgroup9a0364b9e99bb480dd25e1f0284c8555",
		}),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "dataId%02group%01"), nil)

	mockHttpAgent.EXPECT().Get(
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(headerTest),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(map[string]string{
			"dataId": "dataId",
			"group":  "group",
		}),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "content2"), nil)

	_ = client.SetHttpAgent(mockHttpAgent)
	_ = client.SetClientConfig(clientConfigTest)
	_ = client.SetServerConfig(serverConfigsTest)
	client.listening = true
	configData := ""
	client.listenConfigTask(clientConfigTest, serverConfigsTest, mockHttpAgent, vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "content",
		OnChange: func(namespace, group, dataId, data string) {
			configData = data
		},
	})

	assert.Equal(t, "content2", configData)
}

func Test_listenConfigTaskWithoutLocalConfigs(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	client := cretateConfigClientTest()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client.listening = true
	client.listenConfigTask(clientConfigTest, serverConfigsTest, mockHttpAgent,
		vo.ConfigParam{
			DataId:  "dataId",
			Group:   "group",
			Content: "content",
		})
	assert.Equal(t, false, client.listening)
}

// listen

func Test_listen(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	client := cretateConfigClientTest()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	path := "http://console.nacos.io:80/nacos/v1/cs/configs/listener"
	param := map[string]string{
		"Listening-Configs": "dataIdgroup9a0364b9e99bb480dd25e1f0284c8555tenant",
	}
	changedString := "dataId%02group%02tenant%01"
	mockHttpAgent.EXPECT().Post(
		gomock.Eq(path),
		gomock.AssignableToTypeOf(headerTest),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(param),
	).Times(1).Return(http_agent.FakeHttpResponse(200, changedString), nil)

	_ = client.SetHttpAgent(mockHttpAgent)

	changed, err := listen(mockHttpAgent, path, clientConfigTest.TimeoutMs, clientConfigTest.ListenInterval, param)
	assert.Equal(t, changed, changedString)
	assert.Nil(t, err)
}

func Test_listenWithErrorResponse_401(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	client := cretateConfigClientTest()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	path := "http://console.nacos.io:80/nacos/v1/cs/configs/listener"
	param := map[string]string{
		"Listening-Configs": "dataIdgroup9a0364b9e99bb480dd25e1f0284c8555tenant",
	}
	mockHttpAgent.EXPECT().Post(
		gomock.Eq(path),
		gomock.AssignableToTypeOf(headerTest),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(param),
	).Times(1).Return(http_agent.FakeHttpResponse(401, "no auth"), nil)

	_ = client.SetHttpAgent(mockHttpAgent)

	_, err := listen(mockHttpAgent, path, clientConfigTest.TimeoutMs, clientConfigTest.ListenInterval, param)
	assert.NotNil(t, err)
}

// StopListenConfig

func Test_StopListenConfig(t *testing.T) {
	client := cretateConfigClientTest()
	client.listening = true
	client.StopListenConfig()
	assert.True(t, !client.listening)
}

// AddConfigToListen
func Test_AddConfigToListenWithNotListening(t *testing.T) {
	client := cretateConfigClientTest()
	configs := []vo.ConfigParam{{
		DataId:  "dataId",
		Group:   "group",
		Content: "content",
	}}
	err := client.AddConfigToListen(configs)
	assert.NotNil(t, err)
}

func Test_AddConfigToListenWithRepeatAdd(t *testing.T) {
	client := cretateConfigClientTest()
	client.listening = true
	configs := []vo.ConfigParam{{
		DataId: "dataId",
		Group:  "group",
	}}
	addConfigs := []vo.ConfigParam{{
		DataId: "dataId",
		Group:  "group",
	}, {
		DataId: "dataId2",
		Group:  "group",
	}, {
		DataId: "dataId3",
		Group:  "group",
	}}
	client.localConfigs = configs
	err := client.AddConfigToListen(addConfigs)
	assert.Nil(t, err)
	assert.Equal(t, addConfigs, client.localConfigs)
}

func Test_AddConfigToListen(t *testing.T) {
	client := cretateConfigClientTest()
	client.listening = true
	configs := []vo.ConfigParam{{
		DataId: "dataId",
		Group:  "group",
	}}
	addConfigs := []vo.ConfigParam{
		{
			DataId: "dataId2",
			Group:  "group",
		}, {
			DataId: "dataId3",
			Group:  "group",
		}}
	resultConfigs := []vo.ConfigParam{{
		DataId: "dataId",
		Group:  "group",
	}, {
		DataId: "dataId2",
		Group:  "group",
	}, {
		DataId: "dataId3",
		Group:  "group",
	}}
	client.localConfigs = configs
	err := client.AddConfigToListen(addConfigs)
	assert.Nil(t, err)
	assert.Equal(t, resultConfigs, client.localConfigs)
}
