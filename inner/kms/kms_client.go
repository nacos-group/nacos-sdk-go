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
	logger.Infof("init kms client with region:[%s], ak:[%s], sk:[%s]\n", regionId, ak, sk)
	return newKmsClient(regionId, ak, sk, nil)
}

func NewKmsV3ClientWithConfig(config *dkms_api.Config) (*KmsClient, error) {
	logger.Infof("init kms client with endpoint:[%s], clientKeyContent:[%s], password:[%s]\n",
		config.Endpoint, config.ClientKeyContent, config.Password)
	return newKmsClient("", "", "", config)
}

func newKmsClient(regionId, ak, sk string, config *dkms_api.Config) (*KmsClient, error) {
	client, err := dkms_transfer.NewClientWithAccessKey(regionId, ak, sk, config)
	if err != nil {
		return nil, err
	}
	return &KmsClient{
		KmsTransferClient: client,
	}, nil
}

func (kmsClient *KmsClient) GetKmsVersion() constant.KMSVersion {
	return kmsClient.kmsVersion
}

func (kmsClient *KmsClient) SetKmsVersion(kmsVersion constant.KMSVersion) {
	logger.Info("successfully change kms client version to " + kmsVersion)
	kmsClient.kmsVersion = kmsVersion
}

func GetDefaultKMSv1KeyId() string {
	return constant.MSE_KMSv1_DEFAULT_KEY_ID
}
