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

package encoding

import (
	"encoding/base64"
	"unicode/utf8"
)

func DecodeString2Utf8Bytes(data string) []byte {
	resBytes := make([]byte, 0, 8)
	if len(data) == 0 {
		return resBytes
	}
	bytesLen := 0
	runes := []rune(data)
	for _, r := range runes {
		bytesLen += utf8.RuneLen(r)
	}
	resBytes = make([]byte, bytesLen)
	pos := 0
	for _, r := range runes {
		pos += utf8.EncodeRune(resBytes[pos:], r)
	}
	return resBytes
}

func EncodeUtf8Bytes2String(bytes []byte) string {
	if len(bytes) == 0 {
		return ""
	}
	var startPos, endPos int
	resRunes := make([]rune, 0, 8)
	for endPos <= len(bytes) {
		if utf8.FullRune(bytes[startPos:endPos]) {
			decodedRune, _ := utf8.DecodeRune(bytes[startPos:endPos])
			resRunes = append(resRunes, decodedRune)
			startPos = endPos
		}
		endPos++
	}
	return string(resRunes)
}

func DecodeBase64(bytes []byte) ([]byte, error) {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(bytes)))
	n, err := base64.StdEncoding.Decode(dst, bytes)
	if err != nil {
		return nil, err
	}
	return dst[:n], nil
}

func EncodeBase64(bytes []byte) ([]byte, error) {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(bytes)))
	base64.StdEncoding.Encode(dst, bytes)
	return dst[:], nil
}
