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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

func AesEcbPkcs5PaddingEncrypt(plainContent, key []byte) (retBytes []byte, err error) {
	if len(plainContent) == 0 {
		return nil, nil
	}
	aesCipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	pkcs5PaddingBytes := PKCS5Padding(plainContent, aesCipherBlock.BlockSize())
	return BlockEncrypt(pkcs5PaddingBytes, aesCipherBlock)
}

func AesEcbPkcs5PaddingDecrypt(cipherContent, key []byte) (retBytes []byte, err error) {
	if len(cipherContent) == 0 {
		return nil, nil
	}
	aesCipherBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	decryptBytes, err := BlockDecrypt(cipherContent, aesCipherBlock)
	if err != nil {
		return nil, err
	}
	retBytes = PKCS5UnPadding(decryptBytes)
	return retBytes, nil
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func BlockEncrypt(src []byte, b cipher.Block) (dst []byte, err error) {
	if len(src)%b.BlockSize() != 0 {
		return nil, fmt.Errorf("input not full blocks")
	}
	buf := make([]byte, b.BlockSize())
	for i := 0; i < len(src); i += b.BlockSize() {
		b.Encrypt(buf, src[i:i+b.BlockSize()])
		dst = append(dst, buf...)
	}
	return
}

func BlockDecrypt(src []byte, b cipher.Block) (dst []byte, err error) {
	if len(src)%b.BlockSize() != 0 {
		return nil, fmt.Errorf("input not full blocks")
	}
	buf := make([]byte, b.BlockSize())
	for i := 0; i < len(src); i += b.BlockSize() {
		b.Decrypt(buf, src[i:i+b.BlockSize()])
		dst = append(dst, buf...)
	}
	return
}
