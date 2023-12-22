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
	nacos_inner_encryption "github.com/nacos-group/nacos-sdk-go/v2/common/encryption"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/pkg/errors"
	"strings"
)

const (
	defaultConfigEncryptionFilterName = "defaultConfigEncryptionFilter"
)

var (
	noNeedEncryptionError = errors.New("dataId doesn't need to encrypt/decrypt.")
)

type DefaultConfigEncryptionFilter struct {
	handler nacos_inner_encryption.Handler
}

func NewDefaultConfigEncryptionFilter(handler nacos_inner_encryption.Handler) IConfigFilter {
	return &DefaultConfigEncryptionFilter{handler}
}

func (d *DefaultConfigEncryptionFilter) DoFilter(param *vo.ConfigParam) error {
	if err := d.paramCheck(*param); err != nil {
		if errors.Is(err, noNeedEncryptionError) {
			return nil
		}
	}
	if param.UsageType == vo.RequestType {
		encryptionParam := &nacos_inner_encryption.HandlerParam{
			DataId:  param.DataId,
			Content: param.Content,
			KeyId:   param.KmsKeyId,
		}
		if err := d.handler.EncryptionHandler(encryptionParam); err != nil {
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
		if err := d.handler.DecryptionHandler(decryptionParam); err != nil {
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
func (d *DefaultConfigEncryptionFilter) paramCheck(param vo.ConfigParam) error {
	if !strings.HasPrefix(param.DataId, nacos_inner_encryption.CipherPrefix) ||
		len(strings.TrimSpace(param.Content)) == 0 {
		return noNeedEncryptionError
	}
	return nil
}
