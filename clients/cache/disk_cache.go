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
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/nacos-group/nacos-sdk-go/v2/common/file"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/pkg/errors"
)

var (
	fileNotExistError = errors.New("file not exist")
)

func GetFileName(cacheKey, cacheDir string) string {
	return cacheDir + string(os.PathSeparator) + cacheKey
}

func GetEncryptedDataKeyDir(cacheDir string) string {
	return cacheDir + string(os.PathSeparator) + ENCRYPTED_DATA_KEY_FILE_NAME
}

func GetConfigEncryptedDataKeyFileName(cacheKey, cacheDir string) string {
	return GetEncryptedDataKeyDir(cacheDir) + string(os.PathSeparator) + cacheKey
}

func GetConfigFailOverContentFileName(cacheKey, cacheDir string) string {
	return GetFileName(cacheKey, cacheDir) + FAILOVER_FILE_SUFFIX
}

func GetConfigFailOverEncryptedDataKeyFileName(cacheKey, cacheDir string) string {
	return GetConfigEncryptedDataKeyFileName(cacheKey, cacheDir) + FAILOVER_FILE_SUFFIX
}

func WriteServicesToFile(service *model.Service, cacheKey, cacheDir string) {
	err := file.MkdirIfNecessary(cacheDir)
	if err != nil {
		logger.Errorf("mkdir cacheDir failed,cacheDir:%s,err:", cacheDir, err)
		return
	}
	bytes, _ := json.Marshal(service)
	domFileName := GetFileName(cacheKey, cacheDir)
	err = os.WriteFile(domFileName, bytes, 0666)
	if err != nil {
		logger.Errorf("failed to write name cache:%s ,value:%s ,err:%v", domFileName, string(bytes), err)
	}
}

func ReadServicesFromFile(cacheDir string) map[string]model.Service {
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		logger.Errorf("read cacheDir:%s failed!err:%+v", cacheDir, err)
		return nil
	}
	serviceMap := map[string]model.Service{}
	for _, f := range files {
		fileName := GetFileName(f.Name(), cacheDir)
		b, err := os.ReadFile(fileName)
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

func WriteConfigToFile(cacheKey string, cacheDir string, content string) error {
	err := file.MkdirIfNecessary(cacheDir)
	if err != nil {
		errMsg := fmt.Sprintf("make dir failed, dir path %s, err: %v.", cacheDir, err)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}
	err = writeConfigToFile(GetFileName(cacheKey, cacheDir), content, ConfigContent)
	if err != nil {
		logger.Error(err)
		return err
	}
	return nil
}

func WriteEncryptedDataKeyToFile(cacheKey string, cacheDir string, content string) error {
	err := file.MkdirIfNecessary(GetEncryptedDataKeyDir(cacheDir))
	if err != nil {
		errMsg := fmt.Sprintf("make dir failed, dir path %s, err: %v.", cacheDir, err)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}
	err = writeConfigToFile(GetConfigEncryptedDataKeyFileName(cacheKey, cacheDir), content, ConfigEncryptedDataKey)
	if err != nil {
		logger.Error(err)
		return err
	}
	return nil
}

func writeConfigToFile(fileName string, content string, fileType ConfigCachedFileType) error {
	if len(strings.TrimSpace(content)) == 0 {
		// delete config snapshot
		if err := os.Remove(fileName); err != nil {
			if err != syscall.ENOENT {
				logger.Debug(fmt.Sprintf("no need to delete %s cache file, file path %s, file doesn't exist.", fileType, fileName))
				return nil
			}
			errMsg := fmt.Sprintf("failed to delete %s cache file, file path %s, err:%v", fileType, fileName, err)
			return errors.New(errMsg)
		}
	}
	err := os.WriteFile(fileName, []byte(content), 0666)
	if err != nil {
		errMsg := fmt.Sprintf("failed to write %s cache file, file name: %s, value: %s, err:%v", fileType, fileName, content, err)
		return errors.New(errMsg)
	}
	return nil
}

func ReadEncryptedDataKeyFromFile(cacheKey string, cacheDir string) (string, error) {
	content, err := readConfigFromFile(GetConfigEncryptedDataKeyFileName(cacheKey, cacheDir), ConfigEncryptedDataKey)
	if err != nil {
		if errors.Is(err, fileNotExistError) {
			logger.Warn(err)
			return "", nil
		}
	}
	return content, nil
}

func ReadConfigFromFile(cacheKey string, cacheDir string) (string, error) {
	return readConfigFromFile(GetFileName(cacheKey, cacheDir), ConfigEncryptedDataKey)
}

func readConfigFromFile(fileName string, fileType ConfigCachedFileType) (string, error) {
	if !file.IsExistFile(fileName) {
		errMsg := fmt.Sprintf("read cache file %s failed. cause file doesn't exist, file path: %s.", fileType, fileName)
		return "", errors.Wrap(fileNotExistError, errMsg)
	}
	b, err := os.ReadFile(fileName)
	if err != nil {
		errMsg := fmt.Sprintf("get %s from cache failed, filePath:%s, error:%v ", fileType, fileName, err)
		return "", errors.New(errMsg)
	}
	return string(b), nil
}

// GetFailover , get failover content
func GetFailover(key, dir string) string {
	filePath := GetConfigFailOverContentFileName(key, dir)
	return getFailOverConfig(filePath, ConfigContent)
}

func GetFailoverEncryptedDataKey(key, dir string) string {
	filePath := GetConfigFailOverEncryptedDataKeyFileName(key, dir)
	return getFailOverConfig(filePath, ConfigEncryptedDataKey)
}

func getFailOverConfig(filePath string, fileType ConfigCachedFileType) string {
	if !file.IsExistFile(filePath) {
		errMsg := fmt.Sprintf("read %s failed. cause file doesn't exist, file path: %s.", fileType, filePath)
		logger.Error(errMsg)
		return ""
	}
	logger.Warnf("reading failover %s from path:%s", fileType, filePath)
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		logger.Errorf("fail to read failover %s from %s", fileType, filePath)
		return ""
	}
	return string(fileContent)
}
