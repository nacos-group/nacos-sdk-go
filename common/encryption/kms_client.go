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

package encryption

import (
	"fmt"
	"net/http"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	kms20160120 "github.com/alibabacloud-go/kms-20160120/v3/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
	dkms_api "github.com/aliyun/alibabacloud-dkms-gcs-go-sdk/openapi"
	dkms_transfer "github.com/aliyun/alibabacloud-dkms-transfer-go-sdk/sdk"
	"github.com/dbsyk/nacos-sdk-go/v2/common/constant"
	"github.com/dbsyk/nacos-sdk-go/v2/common/logger"
	"github.com/pkg/errors"
)

type KmsClient interface {
	Decrypt(cipherContent string) (string, error)
	Encrypt(content string, keyId string) (string, error)
	GenerateDataKey(keyId, keySpec string) (string, string, error)
	GetKmsVersion() constant.KMSVersion
	setKmsVersion(constant.KMSVersion)
}

type TransferKmsClient struct {
	*dkms_transfer.KmsTransferClient
	kmsVersion constant.KMSVersion
}

func NewKmsV1ClientWithAccessKey(regionId, ak, sk string) (*TransferKmsClient, error) {
	var rErr error
	if rErr = checkKmsV1InitParam(regionId, ak, sk); rErr != nil {
		return nil, rErr
	}
	kmsClient, err := newKmsV1ClientWithAccessKey(regionId, ak, sk)
	if err != nil {
		rErr = errors.Wrap(err, "init kms v1 client with ak/sk failed")
	} else {
		kmsClient.setKmsVersion(constant.KMSv1)
	}
	return kmsClient, rErr
}

func checkKmsV1InitParam(regionId, ak, sk string) error {
	if len(regionId) == 0 {
		return EmptyRegionKmsV1ClientInitError
	}
	if len(ak) == 0 {
		return EmptyAkKmsV1ClientInitError
	}
	if len(sk) == 0 {
		return EmptySkKmsV1ClientInitError
	}
	return nil
}

func checkKmsRamInitParam(endpoint, ak, sk string) error {
	if len(endpoint) == 0 {
		return EmptyEndpointKmsRamClientInitError
	}
	if len(ak) == 0 {
		return EmptyAkKmsV1ClientInitError
	}
	if len(sk) == 0 {
		return EmptySkKmsV1ClientInitError
	}
	return nil
}

func NewKmsV3ClientWithConfig(config *dkms_api.Config, caVerify string) (*TransferKmsClient, error) {
	var rErr error
	if rErr = checkKmsV3InitParam(config, caVerify); rErr != nil {
		return nil, rErr
	}
	kmsClient, err := newKmsV3ClientWithConfig(config)
	if err != nil {
		rErr = errors.Wrap(err, "init kms v3 client with config failed")
	} else {
		if len(strings.TrimSpace(caVerify)) != 0 {
			logger.Debugf("set kms client Ca with content: %s\n", caVerify[:len(caVerify)/maskUnit32Width])
			kmsClient.SetVerify(caVerify)
		} else {
			kmsClient.SetHTTPSInsecure(true)
		}
		kmsClient.setKmsVersion(constant.KMSv3)
	}
	return kmsClient, rErr
}

func checkKmsV3InitParam(config *dkms_api.Config, caVerify string) error {
	if len(*config.Endpoint) == 0 {
		return EmptyEndpointKmsV3ClientInitError
	}
	if len(*config.Password) == 0 {
		return EmptyPasswordKmsV3ClientInitError
	}
	if len(*config.ClientKeyContent) == 0 {
		return EmptyClientKeyContentKmsV3ClientInitError
	}
	if len(caVerify) == 0 {
		return EmptyCaVerifyKmsV3ClientInitError
	}
	return nil
}

func newKmsV1ClientWithAccessKey(regionId, ak, sk string) (*TransferKmsClient, error) {
	logger.Debugf("init kms client with region:[%s], ak:[%s]xxx, sk:[%s]xxx\n",
		regionId, ak[:len(ak)/maskUnit8Width], sk[:len(sk)/maskUnit8Width])
	return newKmsClient(regionId, ak, sk, nil)
}

func newKmsV3ClientWithConfig(config *dkms_api.Config) (*TransferKmsClient, error) {
	logger.Debugf("init kms client with endpoint:[%s], clientKeyContent:[%s], password:[%s]\n",
		config.Endpoint, (*config.ClientKeyContent)[:len(*config.ClientKeyContent)/maskUnit8Width],
		(*config.Password)[:len(*config.Password)/maskUnit8Width])
	return newKmsClient("", "", "", config)
}

func newKmsClient(regionId, ak, sk string, config *dkms_api.Config) (*TransferKmsClient, error) {
	client, err := dkms_transfer.NewClientWithAccessKey(regionId, ak, sk, config)
	if err != nil {
		return nil, err
	}
	return &TransferKmsClient{
		KmsTransferClient: client,
	}, nil
}

func (kmsClient *TransferKmsClient) GetKmsVersion() constant.KMSVersion {
	return kmsClient.kmsVersion
}

func (kmsClient *TransferKmsClient) setKmsVersion(kmsVersion constant.KMSVersion) {
	logger.Debug("successfully set kms client version to " + kmsVersion)
	kmsClient.kmsVersion = kmsVersion
}

func (kmsClient *TransferKmsClient) GenerateDataKey(keyId, keySpec string) (string, string, error) {
	generateDataKeyRequest := kms.CreateGenerateDataKeyRequest()
	generateDataKeyRequest.Scheme = kmsScheme
	generateDataKeyRequest.AcceptFormat = kmsAcceptFormat
	generateDataKeyRequest.KeyId = keyId
	generateDataKeyRequest.KeySpec = keySpec
	generateDataKeyResponse, err := kmsClient.KmsTransferClient.GenerateDataKey(generateDataKeyRequest)
	if err != nil {
		return "", "", err
	}
	return generateDataKeyResponse.Plaintext, generateDataKeyResponse.CiphertextBlob, nil
}

func (kmsClient *TransferKmsClient) Decrypt(cipherContent string) (string, error) {
	request := kms.CreateDecryptRequest()
	request.Method = http.MethodPost
	request.Scheme = kmsScheme
	request.AcceptFormat = kmsAcceptFormat
	request.CiphertextBlob = cipherContent
	response, err := kmsClient.KmsTransferClient.Decrypt(request)
	if err != nil {
		return "", fmt.Errorf("kms decrypt failed: %v", err)
	}
	return response.Plaintext, nil
}

func (kmsClient *TransferKmsClient) Encrypt(content, keyId string) (string, error) {
	request := kms.CreateEncryptRequest()
	request.Method = http.MethodPost
	request.Scheme = kmsScheme
	request.AcceptFormat = kmsAcceptFormat
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

type RamKmsClient struct {
	*kms20160120.Client
	kmsVersion constant.KMSVersion
	runtime    *util.RuntimeOptions
}

func NewKmsRamClient(kmsConfig *constant.KMSConfig, regionId, ak, sk string) (*RamKmsClient, error) {
	if kmsConfig == nil || len(kmsConfig.Endpoint) == 0 {
		if err := checkKmsV1InitParam(regionId, ak, sk); err != nil {
			return nil, err
		}
		KmsV1Config := &openapi.Config{}
		KmsV1Config.AccessKeyId = tea.String(ak)
		KmsV1Config.AccessKeySecret = tea.String(sk)
		KmsV1Config.RegionId = tea.String(regionId)
		_result, _err := kms20160120.NewClient(KmsV1Config)
		if _err != nil {
			return nil, _err
		}
		_ramClient := &RamKmsClient{
			Client:     _result,
			kmsVersion: constant.KMSv1,
			runtime:    &util.RuntimeOptions{},
		}
		return _ramClient, nil
	}
	if err := checkKmsRamInitParam(kmsConfig.Endpoint, ak, sk); err != nil {
		return nil, err
	}
	config := &openapi.Config{}
	config.AccessKeyId = tea.String(ak)
	config.AccessKeySecret = tea.String(sk)
	if len(regionId) != 0 {
		config.RegionId = tea.String(regionId)
	}
	config.Endpoint = tea.String(kmsConfig.Endpoint)
	config.Ca = tea.String(kmsConfig.CaContent)
	runtimeOption := &util.RuntimeOptions{}
	if len(kmsConfig.CaContent) == 0 {
		runtimeOption.IgnoreSSL = tea.Bool(true)
	}
	if kmsConfig.OpenSSL == "true" {
		runtimeOption.IgnoreSSL = tea.Bool(false)
	} else if kmsConfig.OpenSSL == "false" {
		runtimeOption.IgnoreSSL = tea.Bool(true)
	}
	_result, _err := kms20160120.NewClient(config)
	if _err != nil {
		return nil, _err
	}
	_ramClient := &RamKmsClient{
		Client:     _result,
		kmsVersion: constant.KMSv3,
		runtime:    runtimeOption,
	}
	return _ramClient, nil
}

func (kmsClient *RamKmsClient) GetKmsVersion() constant.KMSVersion {
	return kmsClient.kmsVersion
}

func (kmsClient *RamKmsClient) setKmsVersion(kmsVersion constant.KMSVersion) {
	logger.Debug("successfully set kms client version to " + kmsVersion)
	kmsClient.kmsVersion = kmsVersion
}

func (kmsClient *RamKmsClient) GenerateDataKey(keyId, keySpec string) (string, string, error) {
	request := &kms20160120.GenerateDataKeyRequest{
		KeyId:   tea.String(keyId),
		KeySpec: tea.String(keySpec),
	}

	_body, _err := kmsClient.Client.GenerateDataKeyWithOptions(request, kmsClient.runtime)

	if _err != nil {
		return "", "", _err
	}
	return *_body.Body.Plaintext, *_body.Body.CiphertextBlob, nil
}

func (kmsClient *RamKmsClient) Decrypt(cipherContent string) (string, error) {
	request := &kms20160120.DecryptRequest{
		CiphertextBlob: tea.String(cipherContent),
	}

	_body, _err := kmsClient.Client.DecryptWithOptions(request, kmsClient.runtime)
	if _err != nil {
		return "", _err
	}
	return *_body.Body.Plaintext, nil
}

func (kmsClient *RamKmsClient) Encrypt(content, keyId string) (string, error) {
	request := &kms20160120.EncryptRequest{
		Plaintext: tea.String(content),
		KeyId:     tea.String(keyId),
	}
	_body, _err := kmsClient.Client.EncryptWithOptions(request, kmsClient.runtime)
	if _err != nil {
		return "", _err
	}
	return *_body.Body.CiphertextBlob, nil
}
