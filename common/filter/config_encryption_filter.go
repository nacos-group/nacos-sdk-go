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

package filter

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	nacos_inner_encryption "github.com/nacos-group/nacos-sdk-go/v2/common/encryption"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"strings"
	"sync"
)

const (
	defaultConfigEncryptionFilterName = "defaultConfigEncryptionFilter"
)

var (
	initDefaultConfigEncryptionFilterOnce = &sync.Once{}
	defaultConfigEncryptionFilter         IConfigFilter
)

type DefaultConfigEncryptionFilter struct {
}

func GetDefaultConfigEncryptionFilter() IConfigFilter {
	if defaultConfigEncryptionFilter == nil {
		initDefaultConfigEncryptionFilterOnce.Do(func() {
			defaultConfigEncryptionFilter = &DefaultConfigEncryptionFilter{}
			logger.Infof("successfully create ConfigFilter[%s]", defaultConfigEncryptionFilter.GetFilterName())
		})
	}
	return defaultConfigEncryptionFilter
}

func (d *DefaultConfigEncryptionFilter) DoFilter(param *vo.ConfigParam) error {
	if !strings.HasPrefix(param.DataId, nacos_inner_encryption.CipherPrefix) {
		return nil
	}
	if nacos_inner_encryption.GetDefaultKmsClient() == nil {
		return fmt.Errorf("kms client hasn't inited, can't publish config dataId start with: %s", nacos_inner_encryption.CipherPrefix)
	}
	if param.UsageType == vo.RequestType {
		encryptionParam := &nacos_inner_encryption.HandlerParam{
			DataId:  param.DataId,
			Content: param.Content,
			KeyId:   param.KmsKeyId,
		}
		if len(encryptionParam.KeyId) == 0 && nacos_inner_encryption.GetDefaultKmsClient().GetKmsVersion() == constant.KMSv1 {
			encryptionParam.KeyId = nacos_inner_encryption.GetDefaultKMSv1KeyId()
		}
		if err := nacos_inner_encryption.GetDefaultHandler().EncryptionHandler(encryptionParam); err != nil {
			return err
		}
		param.Content = encryptionParam.Content
		param.EncryptedDataKey = encryptionParam.EncryptedDataKey

	} else if param.UsageType == vo.ResponseType {
		decryptionParam := &nacos_inner_encryption.HandlerParam{
			DataId:           param.DataId,
			Content:          param.Content,
			EncryptedDataKey: param.EncryptedDataKey,
		}
		if err := nacos_inner_encryption.GetDefaultHandler().DecryptionHandler(decryptionParam); err != nil {
			return err
		}
		param.Content = decryptionParam.Content
	}
	return nil
}

func (d *DefaultConfigEncryptionFilter) GetOrder() int {
	return 0
}

func (d *DefaultConfigEncryptionFilter) GetFilterName() string {
	return defaultConfigEncryptionFilterName
}
