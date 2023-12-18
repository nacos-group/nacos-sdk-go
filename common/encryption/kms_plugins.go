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
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	inner_encoding "github.com/nacos-group/nacos-sdk-go/v2/common/encoding"
	"strings"
)

type kmsPlugin struct {
	kmsClient *KmsClient
}

func (k *kmsPlugin) Encrypt(param *HandlerParam) error {
	err := k.encryptionParamCheck(*param)
	if err != nil {
		return err
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
	err := k.decryptionParamCheck(*param)
	if err != nil {
		return err
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
	var keyId string
	var err error
	if keyId, err = k.keyIdParamCheck(param.KeyId); err != nil {
		return "", err
	}
	if len(param.PlainDataKey) == 0 {
		return "", EmptyPlainDataKeyError
	}
	encryptedDataKey, err := k.kmsClient.Encrypt(param.PlainDataKey, keyId)
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
		return "", EmptyEncryptedDataKeyError
	}
	plainDataKey, err := k.kmsClient.Decrypt(param.EncryptedDataKey)
	if err != nil {
		return "", err
	}
	if len(plainDataKey) == 0 {
		return "", EmptyPlainDataKeyError
	}
	param.PlainDataKey = plainDataKey
	return plainDataKey, nil
}

func (k *kmsPlugin) encryptionParamCheck(param HandlerParam) error {
	if err := k.plainDataKeyParamCheck(param.PlainDataKey); err != nil {
		return KeyIdParamCheckError
	}
	if err := k.contentParamCheck(param.Content); err != nil {
		return ContentParamCheckError
	}
	return nil
}

func (k *kmsPlugin) decryptionParamCheck(param HandlerParam) error {
	return k.encryptionParamCheck(param)
}

func (k *kmsPlugin) plainDataKeyParamCheck(plainDataKey string) error {
	if len(plainDataKey) == 0 {
		return EmptyPlainDataKeyError
	}
	return nil
}

func (k *kmsPlugin) dataIdParamCheck(dataId string) error {
	if !strings.Contains(dataId, CipherPrefix) {
		return fmt.Errorf("dataId prefix should start with: %s", CipherPrefix)
	}
	return nil
}

func (k *kmsPlugin) keyIdParamCheck(keyId string) (string, error) {
	if len(strings.TrimSpace(keyId)) == 0 {
		if k.kmsClient.GetKmsVersion() == constant.KMSv1 {
			return GetDefaultKMSv1KeyId(), nil
		}
		return "", KeyIdParamCheckError
	}
	return keyId, nil
}

func (k *kmsPlugin) contentParamCheck(content string) error {
	if len(content) == 0 {
		return fmt.Errorf("content need to encrypt is nil")
	}
	return nil
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
	var keyId string
	var err error
	if keyId, err = k.keyIdParamCheck(param.KeyId); err != nil {
		return "", err
	}
	plainSecretKey, encryptedSecretKey, err := k.kmsClient.GenerateDataKey(keyId, kmsAes128KeySpec)
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
	var keyId string
	var err error
	if keyId, err = k.keyIdParamCheck(param.KeyId); err != nil {
		return "", err
	}
	plainSecretKey, encryptedSecretKey, err := k.kmsClient.GenerateDataKey(keyId, kmsAes256KeySpec)
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
	kmsPlugin
}

func (k *KmsBasePlugin) Encrypt(param *HandlerParam) error {
	var keyId string
	var err error
	if keyId, err = k.keyIdParamCheck(param.KeyId); err != nil {
		return err
	}
	if len(param.Content) == 0 {
		return EmptyContentError
	}
	encryptedContent, err := k.kmsClient.Encrypt(param.Content, keyId)
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
	plainContent, err := k.kmsClient.Decrypt(param.Content)
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
