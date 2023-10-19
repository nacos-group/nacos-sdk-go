package filter

import (
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	nacos_inner_encryption "github.com/nacos-group/nacos-sdk-go/v2/common/encryption"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
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
		})
	}
	return defaultConfigEncryptionFilter
}

func (d *DefaultConfigEncryptionFilter) DoFilter(param *vo.ConfigParam) error {
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
