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
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

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

var localIP = ""

func LocalIP() string {
	if localIP == "" {
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
						localIP = ipnet.IP.String()
						break
					}
				}
			}
		}

		if len(localIP) > 0 {
			logger.Infof("Local IP:%s", localIP)
		}
	}
	return localIP
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

// ValidateIPAddress validates if the input is a valid IPv4 or IPv6 address
// Returns nil if valid, otherwise returns an error message
func ValidateIPAddress(input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return fmt.Errorf("input is empty")
	}

	if ip := net.ParseIP(input); ip != nil {
		return nil // Valid IP address
	}

	return fmt.Errorf("not a valid IPv4 or IPv6 address")
}

// ValidateDomain validates domain name format and returns error message
// support full URLs with http:// or https://
func ValidateDomain(input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return fmt.Errorf("input is empty")
	}

	// Check if it's a full URL with protocol
	var domain string
	if strings.HasPrefix(input, "http://") {
		domain = strings.TrimPrefix(input, "http://")
	} else if strings.HasPrefix(input, "https://") {
		domain = strings.TrimPrefix(input, "https://")
	} else {
		// Assume it's just a domain
		domain = input
	}

	// Remove path, query parameters, and fragment if present
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	// Remove port if present
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	// Basic length check
	if len(domain) > 253 {
		return fmt.Errorf("domain length exceeds 253 characters")
	}

	// Check for consecutive dots
	if strings.Contains(domain, "..") {
		return fmt.Errorf("domain contains consecutive dots")
	}

	// Split labels and validate each part
	labels := strings.Split(domain, ".")
	for i, label := range labels {
		// Label length check
		if len(label) == 0 {
			return fmt.Errorf("domain label cannot be empty")
		}
		if len(label) > 63 {
			return fmt.Errorf("domain label '%s' exceeds 63 characters", label)
		}

		// Label cannot start or end with hyphen
		if label[0] == '-' {
			return fmt.Errorf("domain label '%s' starts with hyphen", label)
		}
		if label[len(label)-1] == '-' {
			return fmt.Errorf("domain label '%s' ends with hyphen", label)
		}

		// Top-level domain cannot be all digits
		if i == len(labels)-1 && isAllDigits(label) {
			return fmt.Errorf("top-level domain '%s' cannot be all digits", label)
		}

		// Label character validation
		for _, r := range label {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-') {
				return fmt.Errorf("domain label '%s' contains invalid character '%c'", label, r)
			}
		}
	}

	return nil
}

// isAllDigits checks if string contains only digits
func isAllDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
