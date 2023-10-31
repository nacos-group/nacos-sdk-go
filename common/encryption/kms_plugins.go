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
	inner_encoding "github.com/nacos-group/nacos-sdk-go/v2/common/encoding"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"strings"
)

func init() {
	if err := GetDefaultHandler().RegisterPlugin(&KmsAes128Plugin{}); err != nil {
		logger.Errorf("failed to register encryption plugin[%s] to defaultHandler", KmsAes128AlgorithmName)
	} else {
		logger.Infof("successfully register encryption plugin[%s] to defaultHandler", KmsAes128AlgorithmName)
	}
	if err := GetDefaultHandler().RegisterPlugin(&KmsAes256Plugin{}); err != nil {
		logger.Errorf("failed to register encryption plugin[%s] to defaultHandler", KmsAes256AlgorithmName)
	} else {
		logger.Infof("successfully register encryption plugin[%s] to defaultHandler", KmsAes256AlgorithmName)
	}
	if err := GetDefaultHandler().RegisterPlugin(&KmsBasePlugin{}); err != nil {
		logger.Errorf("failed to register encryption plugin[%s] to defaultHandler", KmsAlgorithmName)
	} else {
		logger.Infof("successfully register encryption plugin[%s] to defaultHandler", KmsAlgorithmName)

	}
}

type kmsPlugin struct {
}

func (k *kmsPlugin) Encrypt(param *HandlerParam) error {
	if len(param.Content) == 0 {
		return nil
	}
	if len(param.PlainDataKey) == 0 {
		return EmptyPlainDataKeyError
	}
	secretKeyBase64Decoded, err := inner_encoding.DecodeBase64(inner_encoding.DecodeString2Utf8Bytes(param.PlainDataKey))
	if err != nil {
		return err
	}
	contentUtf8Bytes := inner_encoding.DecodeString2Utf8Bytes(param.Content)
	encryptedContent, err := AesEcbPkcs5PaddingEncrypt(contentUtf8Bytes, secretKeyBase64Decoded)
	if err != nil {
		return err
	}
	contentBase64Encoded, err := inner_encoding.EncodeBase64(encryptedContent)
	if err != nil {
		return err
	}
	param.Content = inner_encoding.EncodeUtf8Bytes2String(contentBase64Encoded)
	return nil
}

func (k *kmsPlugin) Decrypt(param *HandlerParam) error {
	if len(param.Content) == 0 {
		return nil
	}
	if len(param.PlainDataKey) == 0 {
		return EmptyPlainDataKeyError
	}
	secretKeyBase64Decoded, err := inner_encoding.DecodeBase64(inner_encoding.DecodeString2Utf8Bytes(param.PlainDataKey))
	if err != nil {
		return err
	}
	contentBase64Decoded, err := inner_encoding.DecodeBase64(inner_encoding.DecodeString2Utf8Bytes(param.Content))
	if err != nil {
		return err
	}
	decryptedContent, err := AesEcbPkcs5PaddingDecrypt(contentBase64Decoded, secretKeyBase64Decoded)
	if err != nil {
		return err
	}
	param.Content = inner_encoding.EncodeUtf8Bytes2String(decryptedContent)
	return nil
}

func (k *kmsPlugin) AlgorithmName() string {
	return ""
}

func (k *kmsPlugin) GenerateSecretKey(param *HandlerParam) (string, error) {
	return "", nil
}

func (k *kmsPlugin) EncryptSecretKey(param *HandlerParam) (string, error) {
	if err := keyIdParamCheck(param.KeyId); err != nil {
		return "", err
	}
	if len(param.PlainDataKey) == 0 {
		return "", nil
	}
	encryptedDataKey, err := GetDefaultKmsClient().Encrypt(param.PlainDataKey, param.KeyId)
	if err != nil {
		return "", err
	}
	if len(encryptedDataKey) == 0 {
		return "", EmptyEncryptedDataKeyError
	}
	param.EncryptedDataKey = encryptedDataKey
	return encryptedDataKey, nil
}

func (k *kmsPlugin) DecryptSecretKey(param *HandlerParam) (string, error) {
	if len(param.EncryptedDataKey) == 0 {
		return "", nil
	}
	plainDataKey, err := GetDefaultKmsClient().Decrypt(param.EncryptedDataKey)
	if err != nil {
		return "", err
	}
	if len(plainDataKey) == 0 {
		return "", EmptyPlainDataKeyError
	}
	param.PlainDataKey = plainDataKey
	return plainDataKey, nil
}

type KmsAes128Plugin struct {
	kmsPlugin
}

func (k *KmsAes128Plugin) Encrypt(param *HandlerParam) error {
	return k.kmsPlugin.Encrypt(param)
}

func (k *KmsAes128Plugin) Decrypt(param *HandlerParam) error {
	return k.kmsPlugin.Decrypt(param)
}

func (k *KmsAes128Plugin) AlgorithmName() string {
	return KmsAes128AlgorithmName
}

func (k *KmsAes128Plugin) GenerateSecretKey(param *HandlerParam) (string, error) {
	if err := keyIdParamCheck(param.KeyId); err != nil {
		return "", err
	}
	plainSecretKey, encryptedSecretKey, err := GetDefaultKmsClient().GenerateDataKey(param.KeyId, kmsAes128KeySpec)
	if err != nil {
		return "", err
	}
	param.PlainDataKey = plainSecretKey
	param.EncryptedDataKey = encryptedSecretKey
	if len(param.PlainDataKey) == 0 {
		return "", EmptyPlainDataKeyError
	}
	if len(param.EncryptedDataKey) == 0 {
		return "", EmptyEncryptedDataKeyError
	}
	return plainSecretKey, nil
}

func (k *KmsAes128Plugin) EncryptSecretKey(param *HandlerParam) (string, error) {
	return k.kmsPlugin.EncryptSecretKey(param)
}

func (k *KmsAes128Plugin) DecryptSecretKey(param *HandlerParam) (string, error) {
	return k.kmsPlugin.DecryptSecretKey(param)
}

type KmsAes256Plugin struct {
	kmsPlugin
}

func (k *KmsAes256Plugin) Encrypt(param *HandlerParam) error {
	return k.kmsPlugin.Encrypt(param)

}

func (k *KmsAes256Plugin) Decrypt(param *HandlerParam) error {
	return k.kmsPlugin.Decrypt(param)
}

func (k *KmsAes256Plugin) AlgorithmName() string {
	return KmsAes256AlgorithmName
}

func (k *KmsAes256Plugin) GenerateSecretKey(param *HandlerParam) (string, error) {
	if err := keyIdParamCheck(param.KeyId); err != nil {
		return "", err
	}
	plainSecretKey, encryptedSecretKey, err := GetDefaultKmsClient().GenerateDataKey(param.KeyId, kmsAes256KeySpec)
	if err != nil {
		return "", err
	}
	param.PlainDataKey = plainSecretKey
	param.EncryptedDataKey = encryptedSecretKey
	if len(param.PlainDataKey) == 0 {
		return "", EmptyPlainDataKeyError
	}
	if len(param.EncryptedDataKey) == 0 {
		return "", EmptyEncryptedDataKeyError
	}
	return plainSecretKey, nil
}

func (k *KmsAes256Plugin) EncryptSecretKey(param *HandlerParam) (string, error) {
	return k.kmsPlugin.EncryptSecretKey(param)
}

func (k *KmsAes256Plugin) DecryptSecretKey(param *HandlerParam) (string, error) {
	return k.kmsPlugin.DecryptSecretKey(param)
}

type KmsBasePlugin struct {
}

func (k *KmsBasePlugin) Encrypt(param *HandlerParam) error {
	if err := keyIdParamCheck(param.KeyId); err != nil {
		return err
	}
	if len(param.Content) == 0 {
		return nil
	}
	encryptedContent, err := GetDefaultKmsClient().Encrypt(param.Content, param.KeyId)
	if err != nil {
		return err
	}
	param.Content = encryptedContent
	return nil
}

func (k *KmsBasePlugin) Decrypt(param *HandlerParam) error {
	if len(param.Content) == 0 {
		return nil
	}
	plainContent, err := GetDefaultKmsClient().Decrypt(param.Content)
	if err != nil {
		return err
	}
	param.Content = plainContent
	return nil
}

func (k *KmsBasePlugin) AlgorithmName() string {
	return KmsAlgorithmName
}

func (k *KmsBasePlugin) GenerateSecretKey(param *HandlerParam) (string, error) {
	return "", nil
}

func (k *KmsBasePlugin) EncryptSecretKey(param *HandlerParam) (string, error) {
	return "", nil
}

func (k *KmsBasePlugin) DecryptSecretKey(param *HandlerParam) (string, error) {
	return "", nil
}

func keyIdParamCheck(keyId string) error {
	if len(strings.TrimSpace(keyId)) == 0 {
		return KeyIdParamCheckError
	}
	return nil
}
