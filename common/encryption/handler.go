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
	"github.com/alibabacloud-go/tea/tea"
	dkms_api "github.com/aliyun/alibabacloud-dkms-gcs-go-sdk/openapi"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/pkg/errors"
	"strings"
)

type HandlerParam struct {
	DataId           string `json:"dataId"`  //required
	Content          string `json:"content"` //required
	EncryptedDataKey string `json:"encryptedDataKey"`
	PlainDataKey     string `json:"plainDataKey"`
	KeyId            string `json:"keyId"`
}

type Plugin interface {
	Encrypt(*HandlerParam) error
	Decrypt(*HandlerParam) error
	AlgorithmName() string
	GenerateSecretKey(*HandlerParam) (string, error)
	EncryptSecretKey(*HandlerParam) (string, error)
	DecryptSecretKey(*HandlerParam) (string, error)
}

type Handler interface {
	EncryptionHandler(*HandlerParam) error
	DecryptionHandler(*HandlerParam) error
	RegisterPlugin(Plugin) error
	GetHandlerName() string
}

func NewKmsHandler() Handler {
	return newKmsHandler()
}

func newKmsHandler() *KmsHandler {
	kmsHandler := &KmsHandler{
		encryptionPlugins: make(map[string]Plugin, 2),
	}
	logger.Debug("successfully create encryption KmsHandler")
	return kmsHandler
}

func RegisterConfigEncryptionKmsPlugins(encryptionHandler Handler, clientConfig constant.ClientConfig) {
	innerKmsClient, err := innerNewKmsClient(clientConfig)
	if err == nil && innerKmsClient == nil {
		err = errors.New("create kms client failed.")
	}
	if err != nil {
		logger.Error(err)
	}
	if err := encryptionHandler.RegisterPlugin(&KmsAes128Plugin{kmsPlugin{kmsClient: innerKmsClient}}); err != nil {
		logger.Errorf("failed to register encryption plugin[%s] to %s", KmsAes128AlgorithmName, encryptionHandler.GetHandlerName())
	} else {
		logger.Debugf("successfully register encryption plugin[%s] to %s", KmsAes128AlgorithmName, encryptionHandler.GetHandlerName())
	}
	if err := encryptionHandler.RegisterPlugin(&KmsAes256Plugin{kmsPlugin{kmsClient: innerKmsClient}}); err != nil {
		logger.Errorf("failed to register encryption plugin[%s] to %s", KmsAes256AlgorithmName, encryptionHandler.GetHandlerName())
	} else {
		logger.Debugf("successfully register encryption plugin[%s] to %s", KmsAes256AlgorithmName, encryptionHandler.GetHandlerName())
	}
	if err := encryptionHandler.RegisterPlugin(&KmsBasePlugin{kmsPlugin{kmsClient: innerKmsClient}}); err != nil {
		logger.Errorf("failed to register encryption plugin[%s] to %s", KmsAlgorithmName, encryptionHandler.GetHandlerName())
	} else {
		logger.Debugf("successfully register encryption plugin[%s] to %s", KmsAlgorithmName, encryptionHandler.GetHandlerName())
	}
}

type KmsHandler struct {
	encryptionPlugins map[string]Plugin
}

func (d *KmsHandler) EncryptionHandler(param *HandlerParam) error {
	if err := d.encryptionParamCheck(*param); err != nil {
		return err
	}
	plugin, err := d.getPluginByDataIdPrefix(param.DataId)
	if err != nil {
		return err
	}
	plainSecretKey, err := plugin.GenerateSecretKey(param)
	if err != nil {
		return err
	}
	param.PlainDataKey = plainSecretKey
	return plugin.Encrypt(param)
}

func (d *KmsHandler) DecryptionHandler(param *HandlerParam) error {
	if err := d.decryptionParamCheck(*param); err != nil {
		return err
	}
	plugin, err := d.getPluginByDataIdPrefix(param.DataId)
	if err != nil {
		return err
	}
	plainSecretkey, err := plugin.DecryptSecretKey(param)
	if err != nil {
		return err
	}
	param.PlainDataKey = plainSecretkey
	return plugin.Decrypt(param)
}

func (d *KmsHandler) getPluginByDataIdPrefix(dataId string) (Plugin, error) {
	var (
		matchedCount  int
		matchedPlugin Plugin
	)
	for k, v := range d.encryptionPlugins {
		if strings.Contains(dataId, k) {
			if len(k) > matchedCount {
				matchedCount = len(k)
				matchedPlugin = v
			}
		}
	}
	if matchedPlugin == nil {
		return matchedPlugin, PluginNotFoundError
	}
	return matchedPlugin, nil
}

func (d *KmsHandler) RegisterPlugin(plugin Plugin) error {
	if _, v := d.encryptionPlugins[plugin.AlgorithmName()]; v {
		logger.Warnf("encryption algorithm [%s] has already registered to defaultHandler, will be update", plugin.AlgorithmName())
	} else {
		logger.Debugf("register encryption algorithm [%s] to defaultHandler", plugin.AlgorithmName())
	}
	d.encryptionPlugins[plugin.AlgorithmName()] = plugin
	return nil
}

func (d *KmsHandler) GetHandlerName() string {
	return KmsHandlerName
}

func (d *KmsHandler) encryptionParamCheck(param HandlerParam) error {
	if err := d.dataIdParamCheck(param.DataId); err != nil {
		return DataIdParamCheckError
	}
	if err := d.contentParamCheck(param.Content); err != nil {
		return ContentParamCheckError
	}
	return nil
}

func (d *KmsHandler) decryptionParamCheck(param HandlerParam) error {
	return d.encryptionParamCheck(param)
}

func (d *KmsHandler) keyIdParamCheck(keyId string) error {
	if len(keyId) == 0 {
		return fmt.Errorf("cipher dataId using kmsService need to set keyId, but keyId is nil")
	}
	return nil
}

func (d *KmsHandler) dataIdParamCheck(dataId string) error {
	if !strings.Contains(dataId, CipherPrefix) {
		return fmt.Errorf("dataId prefix should start with: %s", CipherPrefix)
	}
	return nil
}

func (d *KmsHandler) contentParamCheck(content string) error {
	if len(content) == 0 {
		return fmt.Errorf("content need to encrypt is nil")
	}
	return nil
}

func innerNewKmsClient(clientConfig constant.ClientConfig) (kmsClient *KmsClient, err error) {
	switch clientConfig.KMSVersion {
	case constant.KMSv1, constant.DEFAULT_KMS_VERSION:
		kmsClient, err = newKmsV1Client(clientConfig)
	case constant.KMSv3:
		kmsClient, err = newKmsV3Client(clientConfig)
	default:
		err = fmt.Errorf("init kms client failed. unknown kms version:%s\n", clientConfig.KMSVersion)
	}
	return kmsClient, err
}

func newKmsV1Client(clientConfig constant.ClientConfig) (*KmsClient, error) {
	return NewKmsV1ClientWithAccessKey(clientConfig.RegionId, clientConfig.AccessKey, clientConfig.SecretKey)
}

func newKmsV3Client(clientConfig constant.ClientConfig) (*KmsClient, error) {
	return NewKmsV3ClientWithConfig(&dkms_api.Config{
		Protocol:         tea.String("https"),
		Endpoint:         tea.String(clientConfig.KMSv3Config.Endpoint),
		ClientKeyContent: tea.String(clientConfig.KMSv3Config.ClientKeyContent),
		Password:         tea.String(clientConfig.KMSv3Config.Password),
	}, clientConfig.KMSv3Config.CaContent)
}
