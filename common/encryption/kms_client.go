package encryption

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
	dkms_api "github.com/aliyun/alibabacloud-dkms-gcs-go-sdk/openapi"
	dkms_transfer "github.com/aliyun/alibabacloud-dkms-transfer-go-sdk/sdk"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"sync"
)

var (
	initKmsClientOnce = &sync.Once{}
	kmsClient         *KmsClient
)

type KmsClient struct {
	*dkms_transfer.KmsTransferClient
	kmsVersion constant.KMSVersion
}

func InitDefaultKmsV1ClientWithAccessKey(regionId, ak, sk string) (*KmsClient, error) {
	var rErr error
	if GetDefaultKmsClient() == nil {
		initKmsClientOnce.Do(func() {
			client, err := NewKmsV1ClientWithAccessKey(regionId, ak, sk)
			if err != nil {
				rErr = err
			} else {
				kmsClient = client
				kmsClient.SetKmsVersion(constant.KMSv1)
			}
		})
	}
	return GetDefaultKmsClient(), rErr
}

func InitDefaultKmsV3ClientWithConfig(config *dkms_api.Config) (*KmsClient, error) {
	var rErr error
	if GetDefaultKmsClient() == nil {
		initKmsClientOnce.Do(func() {
			client, err := NewKmsV3ClientWithConfig(config)
			if err != nil {
				rErr = err
			} else {
				kmsClient = client
				kmsClient.SetKmsVersion(constant.KMSv3)
			}
		})
	}
	return GetDefaultKmsClient(), rErr
}

func GetDefaultKmsClient() *KmsClient {
	return kmsClient
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

func (kmsClient *KmsClient) GenerateDataKey(keyId, keySpec string) (string, string, error) {
	generateDataKeyRequest := kms.CreateGenerateDataKeyRequest()
	generateDataKeyRequest.Scheme = "https"
	generateDataKeyRequest.AcceptFormat = "JSON"
	generateDataKeyRequest.KeyId = keyId
	generateDataKeyRequest.KeySpec = keySpec
	generateDataKeyResponse, err := kmsClient.KmsTransferClient.GenerateDataKey(generateDataKeyRequest)
	if err != nil {
		return "", "", err
	}
	return generateDataKeyResponse.Plaintext, generateDataKeyResponse.CiphertextBlob, nil
}

func (kmsClient *KmsClient) Decrypt(cipherContent string) (string, error) {
	request := kms.CreateDecryptRequest()
	request.Method = "POST"
	request.Scheme = "https"
	request.AcceptFormat = "JSON"
	request.CiphertextBlob = cipherContent
	response, err := kmsClient.KmsTransferClient.Decrypt(request)
	if err != nil {
		return "", fmt.Errorf("kms decrypt failed: %v", err)
	}
	return response.Plaintext, nil
}

func (kmsClient *KmsClient) Encrypt(content, keyId string) (string, error) {
	request := kms.CreateEncryptRequest()
	request.Method = "POST"
	request.Scheme = "https"
	request.AcceptFormat = "JSON"
	request.Plaintext = content
	request.KeyId = keyId
	response, err := kmsClient.KmsTransferClient.Encrypt(request)
	if err != nil {
		return "", fmt.Errorf("kms encrypt failed: %v", err)
	}
	return response.CiphertextBlob, nil
}

func GetDefaultKMSv1KeyId() string {
	return constant.MSE_KMSv1_DEFAULT_KEY_ID
}
