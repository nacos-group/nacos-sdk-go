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

package cache

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/file"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/pkg/errors"
)

func GetFileName(cacheKey string, cacheDir string) string {
	return cacheDir + string(os.PathSeparator) + cacheKey
}

func WriteServicesToFile(service *model.Service, cacheKey, cacheDir string) {
	err := file.MkdirIfNecessary(cacheDir)
	if err != nil {
		logger.Errorf("mkdir cacheDir failed,cacheDir:%s,err:", cacheDir, err)
		return
	}
	bytes, _ := json.Marshal(service)
	domFileName := GetFileName(cacheKey, cacheDir)
	err = ioutil.WriteFile(domFileName, bytes, 0666)
	if err != nil {
		logger.Errorf("failed to write name cache:%s ,value:%s ,err:%v", domFileName, string(bytes), err)
	}
}

func ReadServicesFromFile(cacheDir string) map[string]model.Service {
	files, err := ioutil.ReadDir(cacheDir)
	if err != nil {
		logger.Errorf("read cacheDir:%s failed!err:%+v", cacheDir, err)
		return nil
	}
	serviceMap := map[string]model.Service{}
	for _, f := range files {
		fileName := GetFileName(f.Name(), cacheDir)
		b, err := ioutil.ReadFile(fileName)
		if err != nil {
			logger.Errorf("failed to read name cache file:%s,err:%v ", fileName, err)
			continue
		}

		s := string(b)
		service := util.JsonToService(s)

		if service == nil {
			continue
		}
		cacheKey := util.GetServiceCacheKey(util.GetGroupName(service.Name, service.GroupName), service.Clusters)
		serviceMap[cacheKey] = *service
	}

	logger.Infof("finish loading name cache, total: %s", strconv.Itoa(len(files)))
	return serviceMap
}

func WriteConfigToFile(cacheKey string, cacheDir string, content string) {
	file.MkdirIfNecessary(cacheDir)
	fileName := GetFileName(cacheKey, cacheDir)
	if len(content) == 0 {
		// delete config snapshot
		if err := os.Remove(fileName); err != nil {
			logger.Errorf("failed to delete config file,cache:%s ,value:%s ,err:%v", fileName, content, err)
		}
		return
	}
	err := ioutil.WriteFile(fileName, []byte(content), 0666)
	if err != nil {
		logger.Errorf("failed to write config  cache:%s ,value:%s ,err:%v", fileName, content, err)
	}

}

func ReadConfigFromFile(cacheKey string, cacheDir string) (string, error) {
	fileName := GetFileName(cacheKey, cacheDir)
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		logger.Errorf("get config from cache, cacheKey:%s, cacheDir:%s, error:%v ", cacheKey, cacheDir, err)
		return "", errors.Errorf("failed to read config cache file:%s, cacheDir:%s, err:%v ", fileName, cacheDir, err)
	}
	return string(b), nil
}

// GetFailover , get failover content
func GetFailover(key, dir string) string {
	filePath := dir + string(os.PathSeparator) + key + constant.FAILOVER_FILE_SUFFIX
	if !file.IsExistFile(filePath) {
		return ""
	}
	logger.Warnf("reading failover content from path:%s", filePath)
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.Errorf("fail to read failover content from %s", filePath)
		return ""
	}
	return string(fileContent)
}
