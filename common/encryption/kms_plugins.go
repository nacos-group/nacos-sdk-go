package encryption

import (
	inner_encoding "github.com/nacos-group/nacos-sdk-go/v2/common/encoding"
)

func init() {
	_ = GetDefaultHandler().RegisterPlugin(&KmsAes128Plugin{})
	_ = GetDefaultHandler().RegisterPlugin(&KmsAes256Plugin{})
	_ = GetDefaultHandler().RegisterPlugin(&KmsBasePlugin{})
}

type kmsPlugin struct {
}

func (k *kmsPlugin) Encrypt(param *HandlerParam) error {
	if len(param.Content) == 0 {
		return nil
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
	if len(param.PlainDataKey) == 0 {
		return "", nil
	}
	encryptedDataKey, err := GetDefaultKmsClient().Encrypt(param.PlainDataKey, param.KeyId)
	if err != nil {
		return "", err
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
	plainSecretKey, encryptedSecretKey, err := GetDefaultKmsClient().GenerateDataKey(param.KeyId, kmsAes128KeySpec)
	if err != nil {
		return "", err
	}
	param.PlainDataKey = plainSecretKey
	param.EncryptedDataKey = encryptedSecretKey
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
	plainSecretKey, encryptedSecretKey, err := GetDefaultKmsClient().GenerateDataKey(param.KeyId, kmsAes256KeySpec)
	if err != nil {
		return "", err
	}
	param.PlainDataKey = plainSecretKey
	param.EncryptedDataKey = encryptedSecretKey
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
	encryptedContent, err := GetDefaultKmsClient().Encrypt(param.Content, param.KeyId)
	if err != nil {
		return err
	}
	param.Content = encryptedContent
	return nil
}

func (k *KmsBasePlugin) Decrypt(param *HandlerParam) error {
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
