package encryption

import "fmt"

const (
	CipherPrefix = "cipher-"

	KmsAes128AlgorithmName = "cipher-kms-aes-128"
	KmsAes256AlgorithmName = "cipher-kms-aes-256"
	KmsAlgorithmName       = "cipher"

	kmsAes128KeySpec = "AES_128"
	kmsAes256KeySpec = "AES_256"

	kmsCipherAlgorithm = "AES/ECB/PKCS5Padding"

	maskUnit8Width  = 8
	maskUnit32Width = 32
)

var (
	DataIdParamCheckError  = fmt.Errorf("dataId prefix should start with: %s", CipherPrefix)
	ContentParamCheckError = fmt.Errorf("content need to encrypt is nil")
	KeyIdParamCheckError   = fmt.Errorf("keyId is nil, need to be set")
)

var (
	PluginNotFoundError = fmt.Errorf("cannot find encryption plugin by dataId prefix")
)
