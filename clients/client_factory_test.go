package clients

import (
	"errors"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
)

func TestNewNocosClientError(t *testing.T) {
	_, herr := setConfigHistory(map[string]interface{}{})
	_, err := setConfig(vo.NacosClientParam{})
	assert.Equal(t, herr.Error(), err.Error())
}

func TestSetConfigClient(t *testing.T) {
	sc, cc := getTestScAndCC()
	hnocosClient, err := setConfigHistory(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		assert.Error(t, err)
	}
	nacosClient, err := setConfig(vo.NacosClientParam{
		ClientConfig:  &cc,
		ServerConfigs: sc,
	})
	if err != nil {
		assert.Error(t, err)
	}

	// test client config Equal
	hcc, err := hnocosClient.GetClientConfig()
	if err != nil {
		assert.Error(t, err)
	}
	ncc, err := nacosClient.GetClientConfig()
	if err != nil {
		assert.Error(t, err)
	}
	assertEquelClientConfig(t, ncc, cc)
	assertEquelClientConfig(t, hcc, ncc)

	// test server config Equal
	hsc, err := hnocosClient.GetServerConfig()
	if err != nil {
		assert.Error(t, err)
	}
	nsc, err := nacosClient.GetServerConfig()
	if err != nil {
		assert.Error(t, err)
	}
	assertEquelServerConfigs(t, nsc, sc)
	assertEquelServerConfigs(t, hsc, nsc)
}

func getTestScAndCC() ([]constant.ServerConfig, constant.ClientConfig) {
	sc := []constant.ServerConfig{
		*constant.NewServerConfig(
			"console.nacos.io",
			80,
			constant.WithScheme("http"),
			constant.WithContextPath("/nacos")),
	}

	cc := *constant.NewClientConfig(
		constant.WithNamespaceId("e525eafa-f7d7-4029-83d9-008937f9d468"),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithRotateTime("1h"),
		constant.WithMaxAge(3),
		constant.WithLogLevel("debug"),
	)
	return sc, cc
}

func assertEquelClientConfig(t *testing.T, hcc, cc constant.ClientConfig) {
	assert.Equal(t, hcc.TimeoutMs, cc.TimeoutMs)
	assert.Equal(t, hcc.Endpoint, cc.Endpoint)
	assert.Equal(t, hcc.LogLevel, cc.LogLevel)
	assert.Equal(t, hcc.BeatInterval, cc.BeatInterval)
	assert.Equal(t, hcc.UpdateThreadNum, cc.UpdateThreadNum)
	assert.Equal(t, hcc.RotateTime, cc.RotateTime)
	assert.Equal(t, hcc.LogDir, cc.LogDir)
	assert.Equal(t, hcc.CacheDir, cc.CacheDir)
	assert.Equal(t, hcc.MaxAge, cc.MaxAge)
	assert.Equal(t, hcc.NotLoadCacheAtStart, cc.NotLoadCacheAtStart)
	assert.Equal(t, hcc.UpdateCacheWhenEmpty, cc.UpdateCacheWhenEmpty)
	assert.Equal(t, hcc.Username, cc.Username)
	assert.Equal(t, hcc.Password, cc.Password)
	assert.Equal(t, hcc.OpenKMS, cc.OpenKMS)
	assert.Equal(t, hcc.NamespaceId, cc.NamespaceId)
	assert.Equal(t, hcc.Username, cc.Username)
	assert.Equal(t, hcc.RegionId, cc.RegionId)
	assert.Equal(t, hcc.AccessKey, cc.AccessKey)
	assert.Equal(t, hcc.SecretKey, cc.SecretKey)
}

func assertEquelServerConfigs(t *testing.T, hsc, sc []constant.ServerConfig) {
	assert.Len(t, hsc, len(sc))
	if len(hsc) != len(sc) {
		return
	}
	for i := 0; i < len(hsc); i++ {
		assert.Equal(t, hsc[i].IpAddr, sc[i].IpAddr)
		assert.Equal(t, hsc[i].Port, sc[i].Port)
		assert.Equal(t, hsc[i].ContextPath, sc[i].ContextPath)
		assert.Equal(t, hsc[i].Scheme, sc[i].Scheme)
	}
}

func setConfigHistory(properties map[string]interface{}) (iClient nacos_client.INacosClient, err error) {
	client := nacos_client.NacosClient{}
	if clientConfigTmp, exist := properties[constant.KEY_CLIENT_CONFIG]; exist {
		if clientConfig, ok := clientConfigTmp.(constant.ClientConfig); ok {
			err = client.SetClientConfig(clientConfig)
			if err != nil {
				return nil, err
			}
		}
	} else {
		_ = client.SetClientConfig(constant.ClientConfig{
			TimeoutMs:    10 * 1000,
			BeatInterval: 5 * 1000,
		})
	}
	if serverConfigTmp, exist := properties[constant.KEY_SERVER_CONFIGS]; exist {
		if serverConfigs, ok := serverConfigTmp.([]constant.ServerConfig); ok {
			err = client.SetServerConfig(serverConfigs)
			if err != nil {
				return nil, err
			}
		}
	} else {
		clientConfig, _ := client.GetClientConfig()
		if len(clientConfig.Endpoint) <= 0 {
			err = errors.New("server configs not found in properties")
			return
		}
		client.SetServerConfig([]constant.ServerConfig{})
	}

	iClient = &client

	return
}
