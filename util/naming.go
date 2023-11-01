package util

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/common/nacos_error"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"regexp"
	"strconv"
)
import "github.com/nacos-group/nacos-sdk-go/v2/common/constant"

// CheckInstanceIsLegal Verify whether the instance params are legal
func CheckInstanceIsLegal(instance model.Instance) (bool, error) {
	if getInstanceHeartBeatTimeOut(instance.Metadata) < getInstanceHeartBeatInterval(instance.Metadata) ||
		getIpDeleteTimeout(instance.Metadata) < getInstanceHeartBeatInterval(instance.Metadata) {
		return false, nacos_error.NewNacosError(strconv.Itoa(nacos_error.INVALID_PARAM),
			"Instance 'heart beat interval' must less than 'heart beat timeout' and 'ip delete timeout'.", nil)
	}

	if len(instance.ClusterName) > 0 {
		if match, _ := regexp.MatchString(constant.CLUSTER_NAME_PATTERN_STRING, instance.ClusterName); !match {
			return false, nacos_error.NewNacosError(strconv.Itoa(nacos_error.INVALID_PARAM),
				"Instance 'clusterName' should be characters with only 0-9a-zA-Z-. (current: %s)", nil)
		}
	}
	return true, nil
}

func getInstanceHeartBeatInterval(metaData map[string]string) int64 {
	return getMetaDataByKeyWithDefaultInt64(metaData, constant.KEY_PRESERVED_HEART_BEAT_INTERVAL,
		constant.DEFAULT_HEART_BEAT_INTERVAL)
}

func getInstanceHeartBeatTimeOut(metaData map[string]string) int64 {
	return getMetaDataByKeyWithDefaultInt64(metaData, constant.KEY_PRESERVED_HEART_BEAT_TIMEOUT,
		constant.DEFAULT_HEART_BEAT_TIMEOUT)
}

func getIpDeleteTimeout(metaData map[string]string) int64 {
	return getMetaDataByKeyWithDefaultInt64(metaData, constant.KEY_PRESERVED_IP_DELETE_TIMEOUT,
		constant.DEFAULT_IP_DELETE_TIMEOUT)
}

func getMetaDataByKeyWithDefaultStr(metaData map[string]string, key string, defaultValue string) string {
	if metaData == nil || len(metaData) == 0 {
		return defaultValue
	}
	if value, ok := metaData[key]; ok {
		return value
	} else {
		return defaultValue
	}
}

func getMetaDataByKeyWithDefaultInt64(metaData map[string]string, key string, defaultValue int64) int64 {
	if metaData == nil || len(metaData) == 0 {
		return defaultValue
	}
	if value, ok := metaData[key]; ok {
		valueInt64, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("[%s]:%s can not parse to int64!", key, value))
		}
		return valueInt64
	} else {
		return defaultValue
	}
}
