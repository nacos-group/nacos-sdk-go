package config_client

import (
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/golang/mock/gomock"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/mock"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strconv"
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
	TimeoutMs:      10000,
	ListenInterval: 20000,
	BeatInterval:   10000,
}

var clientConfigTestWithTenant = constant.ClientConfig{
	TimeoutMs:      10000,
	ListenInterval: 20000,
	BeatInterval:   10000,
	NamespaceId:    "tenant",
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

func cretateConfigClientHttpTestWithTenant(mockHttpAgent http_agent.IHttpAgent) ConfigClient {
	nc := nacos_client.NacosClient{}
	nc.SetServerConfig([]constant.ServerConfig{serverConfigTest})
	nc.SetClientConfig(clientConfigTestWithTenant)
	nc.SetHttpAgent(mockHttpAgent)
	client, _ := NewConfigClient(&nc)
	return client
}

func Test_GetConfig(t *testing.T) {

	client := cretateConfigClientTest()
	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "hello world!222222"})

	assert.Nil(t, err)
	assert.True(t, success)

	content, err := client.GetConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "group"})

	assert.Nil(t, err)
	assert.Equal(t, "hello world!222222", content)
}

func Test_SearchConfig(t *testing.T) {
	client := cretateConfigClientTest()
	configPage, err := client.SearchConfig(vo.SearchConfigParm{
		Search:   "accurate",
		DataId:   "",
		Group:    "DEFAULT_GROUP",
		PageNo:   1,
		PageSize: 10,
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, configPage)
	assert.NotEmpty(t, configPage.PageItems)
}

func Test_GetConfigWithErrorResponse_401(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no security"), nil)
	result, err := client.GetConfig(configParamTest)
	assert.Nil(t, err)
	fmt.Printf("result:%s \n", result)
}

func Test_GetConfigWithErrorResponse_404(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(404, ""), nil)
	reslut, err := client.GetConfig(configParamTest)
	assert.NotNil(t, err)
	assert.Equal(t, "", reslut)
	fmt.Println(err.Error())
}

func Test_GetConfigWithErrorResponse_403(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTest.TimeoutMs),
		gomock.Eq(configParamMapTest),
	).Times(3).Return(http_agent.FakeHttpResponse(403, ""), nil)
	reslut, err := client.GetConfig(configParamTest)
	assert.NotNil(t, err)
	assert.Equal(t, "", reslut)
	fmt.Println(err.Error())
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
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no security"), nil)
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

	client := cretateConfigClientTest()

	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "hello world2!"})

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
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no security"), nil)
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

	client := cretateConfigClientTest()

	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "hello world!"})

	assert.Nil(t, err)
	assert.True(t, success)

	success, err = client.DeleteConfig(vo.ConfigParam{
		DataId: "dataId",
		Group:  "group"})

	assert.Nil(t, err)
	assert.True(t, success)
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
	).Times(3).Return(http_agent.FakeHttpResponse(401, "no security"), nil)
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
	var err error
	var success bool
	ch := make(chan string)
	go func() {
		err = client.ListenConfig(vo.ConfigParam{
			DataId: "dataId",
			Group:  "group",
			OnChange: func(namespace, group, dataId, data string) {
				fmt.Println("group:" + group + ", dataId:" + dataId + ", data:" + data)
				ch <- data
			},
		})
		assert.Nil(t, err)
	}()

	time.Sleep(2 * time.Second)

	success, err = client.PublishConfig(vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "abc"})

	assert.Nil(t, err)
	assert.Equal(t, true, success)
	select {
	case content := <-ch:
		fmt.Println("content:" + content)
	case <-time.After(10 * time.Second):
		fmt.Println("timeout")
		assert.Errorf(t, errors.New("timeout"), "timeout")
	}
}

// listenConfigTask

func Test_listenConfigTask_NoChange(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	//var headerTest = map[string][]string{}

	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTestWithTenant(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(gomock.Eq(requests.POST),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs/listener"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTestWithTenant.ListenInterval),
		gomock.Eq(map[string]string{
			"Listening-Configs": "dataIdgroup9a0364b9e99bb480dd25e1f0284c8555tenant",
		}),
	).Times(1).Return(http_agent.FakeHttpResponse(200, ""), nil)

	changeCount := 0
	client.listenConfigTask(clientConfigTestWithTenant, vo.ConfigParam{
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
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTestWithTenant(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(
		gomock.Eq(requests.POST),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs/listener"),
		gomock.AssignableToTypeOf(http.Header{}),
		gomock.Eq(clientConfigTestWithTenant.ListenInterval),
		gomock.Eq(map[string]string{
			"Listening-Configs": "dataIdgroup9a0364b9e99bb480dd25e1f0284c8555tenant",
		}),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "dataId%02group%02tenant%01"), nil)

	mockHttpAgent.EXPECT().Request(
		gomock.Eq(http.MethodGet),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs"),
		gomock.AssignableToTypeOf(headerTest),
		gomock.Eq(clientConfigTestWithTenant.TimeoutMs),
		gomock.Eq(map[string]string{
			"dataId": "dataId",
			"group":  "group",
			"tenant": "tenant",
		}),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "content2"), nil)

	_ = client.SetHttpAgent(mockHttpAgent)
	_ = client.SetClientConfig(clientConfigTestWithTenant)
	_ = client.SetServerConfig(serverConfigsTest)
	configData := ""
	client.listenConfigTask(clientConfigTestWithTenant, vo.ConfigParam{
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
	mockHttpAgent := mock.NewMockIHttpAgent(controller)
	client := cretateConfigClientHttpTest(mockHttpAgent)
	mockHttpAgent.EXPECT().Request(
		gomock.Eq(http.MethodPost),
		gomock.Eq("http://console.nacos.io:80/nacos/v1/cs/configs/listener"),
		gomock.AssignableToTypeOf(headerTest),
		gomock.Eq(clientConfigTest.ListenInterval),
		gomock.Eq(map[string]string{
			"Listening-Configs": "dataIdgroup9a0364b9e99bb480dd25e1f0284c8555",
		}),
	).Times(1).Return(http_agent.FakeHttpResponse(200, "dataId%02group%01"), nil)

	mockHttpAgent.EXPECT().Request(
		gomock.Eq(http.MethodGet),
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
	configData := ""
	client.listenConfigTask(clientConfigTest, vo.ConfigParam{
		DataId:  "dataId",
		Group:   "group",
		Content: "content",
		OnChange: func(namespace, group, dataId, data string) {
			configData = data
		},
	})

	assert.Equal(t, "content2", configData)
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
	assert.Nil(t, err)
}

func Test_AddConfigToListenWithRepeatAdd(t *testing.T) {
	client := cretateConfigClientTest()
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

// CancelListenConfig
func TestCancelListenConfig(t *testing.T) {
	client := cretateConfigClientTest()
	var err error
	var success bool
	var context,context1 string
	listenConfigParam := vo.ConfigParam{
		DataId: "dataId",
		Group:  "group",
		OnChange: func(namespace, group, dataId, data string) {
			fmt.Println("group:" + group + ", dataId:" + dataId + ", data:" + data)
			context = data
		},
		ListenCloseChan: make(chan struct{}, 1),
	}

	listenConfigParam1 := vo.ConfigParam{
		DataId: "dataId1",
		Group:  "group1",
		OnChange: func(namespace, group, dataId, data string) {
			fmt.Println("group1:" + group + ", dataId1:" + dataId + ", data:" + data)
			context1 = data
		},
		ListenCloseChan: make(chan struct{}, 1),
	}
	go func() {
		err = client.ListenConfig(listenConfigParam)
		assert.Nil(t, err)
	}()

	go func() {
		err = client.ListenConfig(listenConfigParam1)
		assert.Nil(t, err)
	}()
	
	fmt.Println("Start listening")
	for i := 1; i <= 5; i++ {
		time.Sleep(2 * time.Second)
		success, err = client.PublishConfig(vo.ConfigParam{
			DataId:  "dataId",
			Group:   "group",
			Content: "abcd" + strconv.Itoa(i)})

		success, err = client.PublishConfig(vo.ConfigParam{
			DataId:  "dataId1",
			Group:   "group1",
			Content: "abcd" + strconv.Itoa(i)})
		if i == 3 {
			client.CancelListenConfig(&listenConfigParam)
			fmt.Println("Cancel listen config")
		}
		assert.Nil(t, err)
		assert.Equal(t, true, success)
	}
	time.Sleep(2 * time.Second)
	assert.Equal(t,"abcd3",context)
	assert.Equal(t,"abcd5",context1)
}
