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
	"encoding/base64"
	"fmt"
)

// Ciphertext envelope compatible with the dkms-transfer bridge SDK:
// base64std( keyVersionId[36] + iv[12] + raw ).
const (
	kmsEktIdLength            = 36                            // dkms-transfer@v0.1.8 sdk/decrypt.go EktIdLength
	kmsGcmIvLength            = 12                            // dkms-transfer@v0.1.8 sdk/decrypt.go GcmIvLength
	kmsMigrationKeyVersionHdr = "x-kms-migrationkeyversionid" // dkms-transfer@v0.1.8 sdk/decrypt.go MigrationKeyVersionIdKey
)

// sealKmsEnvelope concatenates keyVersionId, iv and raw, then base64-encodes the result.
// keyVersionId must be exactly kmsEktIdLength bytes and iv exactly kmsGcmIvLength bytes.
func sealKmsEnvelope(keyVersionId, iv, raw []byte) (string, error) {
	if len(keyVersionId) != kmsEktIdLength {
		return "", fmt.Errorf("kms envelope: keyVersionId must be %d bytes, got %d", kmsEktIdLength, len(keyVersionId))
	}
	if len(iv) != kmsGcmIvLength {
		return "", fmt.Errorf("kms envelope: iv must be %d bytes, got %d", kmsGcmIvLength, len(iv))
	}
	plain := make([]byte, 0, kmsEktIdLength+kmsGcmIvLength+len(raw))
	plain = append(plain, keyVersionId...)
	plain = append(plain, iv...)
	plain = append(plain, raw...)
	return base64.StdEncoding.EncodeToString(plain), nil
}

// openKmsEnvelope base64-decodes blob and splits it into keyVersionId, iv and raw.
// Returns an error for invalid base64 or when the decoded data is shorter than
// kmsEktIdLength+kmsGcmIvLength bytes. An empty raw section is valid.
func openKmsEnvelope(blob string) (keyVersionId, iv, raw []byte, err error) {
	plain, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("kms envelope: invalid base64: %w", err)
	}
	headerLen := kmsEktIdLength + kmsGcmIvLength
	if len(plain) < headerLen {
		return nil, nil, nil, fmt.Errorf("kms envelope: decoded length %d is shorter than header %d", len(plain), headerLen)
	}
	keyVersionId = plain[:kmsEktIdLength]
	iv = plain[kmsEktIdLength:headerLen]
	raw = plain[headerLen:]
	return keyVersionId, iv, raw, nil
}
