package main

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func main() {
	//os.Setenv("nacos.remote.client.grpc.timeout", "1000")
	sc := []constant.ServerConfig{
		*constant.NewServerConfig("nacos.test.infra.ww5sawfyut0k.bitsvc.io", 8848),
		//*constant.NewServerConfig("10.18.1.4", 8848),
	}

	//create ClientConfig
	cc := constant.NewClientConfig(
		constant.WithNamespaceId(""),
		constant.WithTimeoutMs(1000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithUsername("nacos"),
		constant.WithPassword("nacos"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
		constant.WithAppName("local-golang-idea"),
		constant.WithDisableUseSnapShot(false),
	)

	// create config client
	s := time.Now()
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  cc,
			ServerConfigs: sc,
		},
	)
	fmt.Println("建立client耗时: ", time.Since(s))

	if err != nil {
		panic(err)
	}

	//fmt.Println(os.Getenv("xxx") == "true")

	var content string
	s = time.Now()

	content, err = client.GetConfig(vo.ConfigParam{
		DataId: "option-zone.yaml",
		Group:  "ROUTE_STRATEGY",
	})

	logger.Info(content)
	fmt.Println("配置内容->" + content)

	fmt.Println("第一次获取配置耗时: ", time.Since(s))
	//fmt.Println(cost, "=====>>>>>>>>>")

	if err != nil {
		println("获取配置失败", err)
		return
	}

	s = time.Now()

	content, err = client.GetConfig(vo.ConfigParam{
		DataId: "option-zone.yaml",
		Group:  "ROUTE_STRATEGY",
	})

	fmt.Println("第二次次获取配置耗时: ", time.Since(s))
	//fmt.Println("config")
	//fmt.Println(content)
	//fmt.Println("获取配置:", float64(time.Now().UnixMilli()-s.UnixMilli()), " ms")
	//
	//time.Sleep(100 * time.Millisecond)

	s = time.Now()
	err = client.ListenConfig(vo.ConfigParam{
		DataId: "option-zone.yaml",
		Group:  "ROUTE_STRATEGY",
		OnChange: func(namespace, group, dataId, data string) {
			fmt.Println("收到内容变更推送事件", data)
		},
	})
	if err != nil {
		fmt.Println("监听error", err)
		return
	}

	fmt.Println("监控耗时:", time.Since(s))
	for {
		time.Sleep(10 * time.Second)
		s = time.Now()

		content, err = client.GetConfig(vo.ConfigParam{
			DataId: "option-zone.yaml",
			Group:  "ROUTE_STRATEGY",
		})

		fmt.Println(content)
		fmt.Println("获取配置耗时: ", time.Since(s))
	}
}
