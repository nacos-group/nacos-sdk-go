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

import "fmt"

const (
	CipherPrefix = "cipher-"

	KmsAes128AlgorithmName = "cipher-kms-aes-128"
	KmsAes256AlgorithmName = "cipher-kms-aes-256"
	KmsAlgorithmName       = "cipher"

	kmsAes128KeySpec = "AES_128"
	kmsAes256KeySpec = "AES_256"

	kmsScheme       = "https"
	kmsAcceptFormat = "XML"

	kmsCipherAlgorithm = "AES/ECB/PKCS5Padding"

	maskUnit8Width  = 8
	maskUnit32Width = 32

	KmsHandlerName = "KmsHandler"
)

var (
	DataIdParamCheckError  = fmt.Errorf("dataId prefix should start with: %s", CipherPrefix)
	ContentParamCheckError = fmt.Errorf("content need to encrypt is nil")
	KeyIdParamCheckError   = fmt.Errorf("keyId is nil, need to be set")
)

var (
	PluginNotFoundError = fmt.Errorf("cannot find encryption plugin by dataId prefix")
)

var (
	EmptyEncryptedDataKeyError = fmt.Errorf("empty encrypted data key error")
	EmptyPlainDataKeyError     = fmt.Errorf("empty plain data key error")
	EmptyContentError          = fmt.Errorf("encrypt empty content error")
)

var (
	EmptyRegionKmsV1ClientInitError = fmt.Errorf("init kmsV1 client failed with empty region")
	EmptyAkKmsV1ClientInitError     = fmt.Errorf("init kmsV1 client failed with empty ak")
	EmptySkKmsV1ClientInitError     = fmt.Errorf("init kmsV1 client failed with empty sk")

	EmptyEndpointKmsV3ClientInitError         = fmt.Errorf("init kmsV3 client failed with empty endpoint")
	EmptyPasswordKmsV3ClientInitError         = fmt.Errorf("init kmsV3 client failed with empty password")
	EmptyClientKeyContentKmsV3ClientInitError = fmt.Errorf("init kmsV3 client failed with empty client key content")
	EmptyCaVerifyKmsV3ClientInitError         = fmt.Errorf("init kmsV3 client failed with empty ca verify")
)
