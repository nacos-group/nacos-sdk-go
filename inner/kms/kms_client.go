package kms

import (
	dkms_api "github.com/aliyun/alibabacloud-dkms-gcs-go-sdk/openapi"
	dkms_transfer "github.com/aliyun/alibabacloud-dkms-transfer-go-sdk/sdk"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
)

type KmsClient struct {
	*dkms_transfer.KmsTransferClient
	kmsVersion constant.KMSVersion
}

func NewKmsV1ClientWithAccessKey(regionId, ak, sk string) (*KmsClient, error) {
	logger.Info("init kms v1 client with ak/sk")
	return newKmsClient(regionId, ak, sk, nil)
}

func NewKmsV3ClientWithConfig(config *dkms_api.Config) (*KmsClient, error) {
	logger.Info("init kms v3 client with config")
	return newKmsClient("", "", "", config)
}

func newKmsClient(regionId, ak, sk string, config *dkms_api.Config) (*KmsClient, error) {
	var kmsVersion constant.KMSVersion
	if config != nil {
		kmsVersion = constant.KMSv3
	} else {
		kmsVersion = constant.KMSv1
	}
	client, err := dkms_transfer.NewClientWithAccessKey(regionId, ak, sk, config)
	if err != nil {
		return nil, err
	}
	return &KmsClient{
		KmsTransferClient: client,
		kmsVersion:        kmsVersion,
	}, nil
}

func (kmsClient *KmsClient) GetKmsVersion() constant.KMSVersion {
	return kmsClient.kmsVersion
}

func GetDefaultKMSv1KeyId() string {
	return constant.MSE_KMSv1_DEFAULT_KEY_ID
}
