package main

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/filter"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"io/ioutil"
	"os"
	"path"
	"time"
)

var localServerConfigWithOptions = constant.NewServerConfig(
	"mse-d12e6112-p.nacos-ans.mse.aliyuncs.com",
	8848,
)

var localClientConfigWithOptions = constant.NewClientConfig(
	constant.WithTimeoutMs(10*1000),
	constant.WithBeatInterval(2*1000),
	constant.WithNotLoadCacheAtStart(true),
	constant.WithAccessKey(getFileContent(path.Join(getWDR(), "ak"))),
	constant.WithSecretKey(getFileContent(path.Join(getWDR(), "sk"))),
	constant.WithNamespaceId("791fd262-3735-40df-a605-e3236f8ff495"),
	constant.WithOpenKMS(true),
	constant.WithKMSVersion(constant.KMSv3),
	constant.WithKMSv3Config(&constant.KMSv3Config{
		ClientKeyContent: getFileContent(path.Join(getWDR(), "client_key.json")),
		Password:         getFileContent(path.Join(getWDR(), "password")),
		Endpoint:         getFileContent(path.Join(getWDR(), "endpoint")),
		CaContent:        getFileContent(path.Join(getWDR(), "ca.pem")),
	}),
	constant.WithRegionId("cn-beijing"),
)

var localConfigList = []vo.ConfigParam{
	{
		DataId:   "common-config",
		Group:    "default",
		Content:  "common",
		KmsKeyId: "key-xxx", //可以识别
	},
	{
		DataId:   "cipher-crypt",
		Group:    "default",
		Content:  "cipher",
		KmsKeyId: "key-xxx", //可以识别
	},
	{
		DataId:   "cipher-kms-aes-128-crypt",
		Group:    "default",
		Content:  "cipher-aes-128",
		KmsKeyId: "key-xxx", //可以识别
	},
	{
		DataId:   "cipher-kms-aes-256-crypt",
		Group:    "default",
		Content:  "cipher-aes-256",
		KmsKeyId: "key-xxx", //可以识别
	},
}

func main() {
	usingKMSv3ClientAndStoredByNacos()
	//onlyUsingFilters()
}

func usingKMSv3ClientAndStoredByNacos() {
	client := createConfigClient()
	if client == nil {
		panic("init ConfigClient failed")
	}

	for _, localConfig := range localConfigList {
		// to enable encrypt/decrypt, DataId should be start with "cipher-"
		configParam := vo.ConfigParam{
			DataId:   localConfig.DataId,
			Group:    localConfig.Group,
			Content:  localConfig.Content,
			KmsKeyId: localConfig.KmsKeyId,
			OnChange: func(namespace, group, dataId, data string) {
				fmt.Printf("successfully receive changed config: \n"+
					"group[%s], dataId[%s], data[%s]\n", group, dataId, data)
			},
		}

		err := client.ListenConfig(configParam)
		if err != nil {
			fmt.Printf("failed to listen: group[%s], dataId[%s] with error: %s\n",
				configParam.Group, configParam.DataId, err)
		} else {
			fmt.Printf("successfully ListenConfig: group[%s], dataId[%s]\n", configParam.Group, configParam.DataId)
		}

		published, err := client.PublishConfig(configParam)
		if published && err == nil {
			fmt.Printf("successfully publish: group[%s], dataId[%s], data[%s]\n", configParam.Group, configParam.DataId, configParam.Content)
		} else {
			fmt.Printf("failed to publish: group[%s], dataId[%s], data[%s]\n with error: %s\n",
				configParam.Group, configParam.DataId, configParam.Content, err)
		}

		//wait for config change callback to execute
		time.Sleep(2 * time.Second)

		//get config
		content, err := client.GetConfig(configParam)
		if err == nil {
			fmt.Printf("successfully get config: group[%s], dataId[%s], data[%s]\n", configParam.Group, configParam.DataId, configParam.Content)
		} else {
			fmt.Printf("failed to get config: group[%s], dataId[%s], data[%s]\n with error: %s\n",
				configParam.Group, configParam.DataId, configParam.Content, err)
		}

		if content != localConfig.Content {
			panic("publish/get encrypted config failed.")
		} else {
			fmt.Println("publish/get encrypted config success.")
		}
		//wait for config change callback to execute
		//time.Sleep(2 * time.Second)
	}
}

func onlyUsingFilters() error {
	createConfigClient()
	for _, param := range localConfigList {
		param.UsageType = vo.RequestType
		fmt.Println("param = ", param)
		if err := filter.GetDefaultConfigFilterChainManager().DoFilters(&param); err != nil {
			return err
		}
		fmt.Println("after encrypt param = ", param)
		param.UsageType = vo.ResponseType
		if err := filter.GetDefaultConfigFilterChainManager().DoFilters(&param); err != nil {
			return err
		}
		fmt.Println("after decrypt param = ", param)
	}
	return nil
}

func createConfigClient() *config_client.ConfigClient {
	nc := nacos_client.NacosClient{}
	_ = nc.SetServerConfig([]constant.ServerConfig{*localServerConfigWithOptions})
	_ = nc.SetClientConfig(*localClientConfigWithOptions)
	_ = nc.SetHttpAgent(&http_agent.HttpAgent{})
	client, err := config_client.NewConfigClient(&nc)
	if err != nil {
		logger.Errorf("create config client failed: " + err.Error())
		return nil
	}
	return client
}

func getWDR() string {
	getwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return getwd
}

func getFileContent(filePath string) string {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return string(file)
}
