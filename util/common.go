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

package util

import (
	"encoding/json"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/logger"
	"github.com/nacos-group/nacos-sdk-go/model"
)

func CurrentMillis() int64 {
	return time.Now().UnixNano() / 1e6
}

func JsonToService(result string) *model.Service {
	var service model.Service
	err := json.Unmarshal([]byte(result), &service)
	if err != nil {
		logger.Errorf("failed to unmarshal json string:%s err:%+v", result, err)
		return nil
	}
	if len(service.Hosts) == 0 {
		logger.Warnf("instance list is empty,json string:%s", result)
	}
	return &service

}
func ToJsonString(object interface{}) string {
	js, _ := json.Marshal(object)
	return string(js)
}

func GetGroupName(serviceName string, groupName string) string {
	return groupName + constant.SERVICE_INFO_SPLITER + serviceName
}

func GetServiceCacheKey(serviceName string, clusters string) string {
	if clusters == "" {
		return serviceName
	}
	return serviceName + constant.SERVICE_INFO_SPLITER + clusters
}

func GetConfigCacheKey(dataId string, group string, tenant string) string {
	return dataId + constant.CONFIG_INFO_SPLITER + group + constant.CONFIG_INFO_SPLITER + tenant
}

var (
	localIP     = ""
	privateCIDR []*net.IPNet
)

func LocalIP() string {
	if localIP != "" {
		return localIP
	}

	faces, err := getFaces()
	if err != nil {
		return ""
	}

	for _, address := range faces {
		ipNet, ok := address.(*net.IPNet)
		if !ok || ipNet.IP.To4() == nil || isFilteredIP(ipNet.IP) {
			continue
		}

		localIP = ipNet.IP.String()
		break
	}

	if localIP != "" {
		logger.Infof("Local IP:%s", localIP)
	}

	return localIP
}

// SetFilterNetNumberAndMask ipMask like 127.0.0.0/8
func SetFilterNetNumberAndMask(ipMask ...string) error {
	var (
		err   error
		ipNet *net.IPNet
	)
	for _, b := range ipMask {
		_, ipNet, err = net.ParseCIDR(b)
		if err != nil {
			return err
		}
		privateCIDR = append(privateCIDR, ipNet)
	}
	return err
}

// getFaces return addresses from interfaces that is up
func getFaces() ([]net.Addr, error) {
	var upAddrs []net.Addr

	interfaces, err := net.Interfaces()
	if err != nil {
		logger.Errorf("get Interfaces failed,err:%+v", err)
		return nil, err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if (iface.Flags & net.FlagLoopback) != 0 {
			continue
		}

		addresses, err := iface.Addrs()
		if err != nil {
			logger.Errorf("get InterfaceAddress failed,err:%+v", err)
			return nil, err
		}

		upAddrs = append(upAddrs, addresses...)
	}

	return upAddrs, nil
}

func isFilteredIP(ip net.IP) bool {
	for _, privateIP := range privateCIDR {
		if privateIP.Contains(ip) {
			return true
		}
	}
	return false
}

func GetDurationWithDefault(metadata map[string]string, key string, defaultDuration time.Duration) time.Duration {
	data, ok := metadata[key]
	if ok {
		value, err := strconv.ParseInt(data, 10, 64)
		if err != nil {
			logger.Errorf("key:%s is not a number", key)
			return defaultDuration
		}
		return time.Duration(value)
	}
	return defaultDuration
}

func GetUrlFormedMap(source map[string]string) (urlEncoded string) {
	urlEncoder := url.Values{}
	for key, value := range source {
		urlEncoder.Add(key, value)
	}
	urlEncoded = urlEncoder.Encode()
	return
}

func DeepCopyMap(params map[string]string) map[string]string {
	result := make(map[string]string, len(params))
	for k, v := range params {
		result[k] = v
	}
	return result
}
