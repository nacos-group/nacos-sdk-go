package clients

import (
	"net"
	"reflect"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/assert"
)

func getIntranetIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func TestSetConfigClient(t *testing.T) {
	ip := getIntranetIP()
	sc := []constant.ServerConfig{
		*constant.NewServerConfig(
			ip,
			8848,
		),
	}

	cc := *constant.NewClientConfig(
		constant.WithNamespaceId("public"),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
	)

	t.Run("setConfig_error", func(t *testing.T) {
		nacosClient, err := setConfig(vo.NacosClientParam{})
		assert.Nil(t, nacosClient)
		assert.Equal(t, "server configs not found in properties", err.Error())
	})

	t.Run("setConfig_normal", func(t *testing.T) {
		// use map params setConfig
		param := getConfigParam(map[string]interface{}{
			"serverConfigs": sc,
			"clientConfig":  cc,
		})
		nacosClientFromMap, err := setConfig(param)
		assert.Nil(t, err)
		nacosClientFromStruct, err := setConfig(vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		})
		assert.Nil(t, err)
		assert.True(t, reflect.DeepEqual(nacosClientFromMap, nacosClientFromStruct))
	})
}
