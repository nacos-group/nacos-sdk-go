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
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
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

// Environment variable key for overriding the client IP.
// This takes highest priority, useful for containerized environments (e.g., K8s Pod spec).
const EnvNacosClientLocalIP = "NACOS_CLIENT_LOCAL_IP"

var (
	resolvedIP   string
	ipOnce       sync.Once
	configuredIP atomic.Value // string; set via SetClientIPFromConfig before first LocalIP() call
)

// SetClientIPFromConfig stores the IP from ClientConfig.ClientIP.
// Typically called by the client factory during initialization before any LocalIP() invocation.
// This value has second priority after the NACOS_CLIENT_LOCAL_IP environment variable.
// Uses atomic.Value so concurrent calls and concurrent LocalIP() reads are race-free.
func SetClientIPFromConfig(ip string) {
	configuredIP.Store(ip)
}

func getConfiguredIP() string {
	v := configuredIP.Load()
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// LocalIP returns the client IP address. Resolution happens exactly once with the following priority:
//  1. NACOS_CLIENT_LOCAL_IP environment variable (highest, zero code change needed)
//  2. ClientConfig.ClientIP set via WithClientIP option
//  3. Auto-detected from network interfaces (first non-loopback IPv4)
func LocalIP() string {
	ipOnce.Do(func() {
		// Priority 1: environment variable
		if envIP := os.Getenv(EnvNacosClientLocalIP); envIP != "" {
			resolvedIP = envIP
			logger.Infof("Using client IP from environment variable %s: %s", EnvNacosClientLocalIP, resolvedIP)
			return
		}
		// Priority 2: ClientConfig.ClientIP
		if cfg := getConfiguredIP(); cfg != "" {
			resolvedIP = cfg
			logger.Infof("Using client IP from ClientConfig: %s", resolvedIP)
			return
		}
		// Priority 3: auto-detect from network interfaces
		resolvedIP = detectLocalIP()
		if resolvedIP != "" {
			logger.Infof("Local IP:%s", resolvedIP)
		}
	})
	return resolvedIP
}

// detectLocalIP scans network interfaces and returns the first non-loopback IPv4 address.
func detectLocalIP() string {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		logger.Errorf("get Interfaces failed,err:%+v", err)
		return ""
	}

	for i := 0; i < len(netInterfaces); i++ {
		if ((netInterfaces[i].Flags & net.FlagUp) != 0) && ((netInterfaces[i].Flags & net.FlagLoopback) == 0) {
			addrs, err := netInterfaces[i].Addrs()
			if err != nil {
				logger.Errorf("get InterfaceAddress failed,err:%+v", err)
				return ""
			}
			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}
	return ""
}

// resetLocalIPForTesting resets the IP resolution state. FOR TESTING ONLY.
// Lowercase intentionally: this helper must NOT be part of the public API,
// otherwise external callers could wipe an already-resolved IP at runtime.
func resetLocalIPForTesting() {
	resolvedIP = ""
	configuredIP = atomic.Value{}
	ipOnce = sync.Once{}
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

// get status code by response,default is NA
func GetStatusCode(response *http.Response) string {
	var statusCode string
	if response != nil {
		statusCode = strconv.Itoa(response.StatusCode)
	} else {
		statusCode = "NA"
	}
	return statusCode
}

func DeepCopyMap(params map[string]string) map[string]string {
	result := make(map[string]string, len(params))
	for k, v := range params {
		result[k] = v
	}
	return result
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
