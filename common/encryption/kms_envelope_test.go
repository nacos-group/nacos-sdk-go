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
	"crypto/rand"
	"encoding/base64"
	"testing"
)

// TestKmsEnvelopeGoldenVector pins the wire format against an expectation
// built independently of sealKmsEnvelope/openKmsEnvelope:
// base64std( keyVersionId[36] + iv[12] + raw ).
func TestKmsEnvelopeGoldenVector(t *testing.T) {
	keyVersionId := []byte("0123456789abcdefghijklmnopqrstuvwxyz") // 36 bytes
	iv := []byte("ABCDEFGHIJKL")                                   // 12 bytes
	raw := []byte("hello-raw-ciphertext")

	if len(keyVersionId) != kmsEktIdLength || len(iv) != kmsGcmIvLength {
		t.Fatalf("fixture lengths wrong: kv=%d iv=%d", len(keyVersionId), len(iv))
	}

	var expectedPlain []byte
	expectedPlain = append(expectedPlain, keyVersionId...)
	expectedPlain = append(expectedPlain, iv...)
	expectedPlain = append(expectedPlain, raw...)
	expected := base64.StdEncoding.EncodeToString(expectedPlain)

	sealed, err := sealKmsEnvelope(keyVersionId, iv, raw)
	if err != nil {
		t.Fatalf("sealKmsEnvelope returned error: %v", err)
	}
	if sealed != expected {
		t.Fatalf("sealed envelope mismatch:\n got  %q\n want %q", sealed, expected)
	}

	gotKv, gotIv, gotRaw, err := openKmsEnvelope(expected)
	if err != nil {
		t.Fatalf("openKmsEnvelope returned error: %v", err)
	}
	if !bytes.Equal(gotKv, keyVersionId) {
		t.Errorf("keyVersionId mismatch: got %q want %q", gotKv, keyVersionId)
	}
	if !bytes.Equal(gotIv, iv) {
		t.Errorf("iv mismatch: got %q want %q", gotIv, iv)
	}
	if !bytes.Equal(gotRaw, raw) {
		t.Errorf("raw mismatch: got %q want %q", gotRaw, raw)
	}
}

// TestKmsEnvelopeRoundTrip seals and opens random material and checks identity.
func TestKmsEnvelopeRoundTrip(t *testing.T) {
	for i := 0; i < 16; i++ {
		kv := make([]byte, kmsEktIdLength)
		iv := make([]byte, kmsGcmIvLength)
		raw := make([]byte, 1+i*37)
		for _, b := range [][]byte{kv, iv, raw} {
			if _, err := rand.Read(b); err != nil {
				t.Fatalf("rand.Read: %v", err)
			}
		}

		blob, err := sealKmsEnvelope(kv, iv, raw)
		if err != nil {
			t.Fatalf("iteration %d: seal error: %v", i, err)
		}
		gotKv, gotIv, gotRaw, err := openKmsEnvelope(blob)
		if err != nil {
			t.Fatalf("iteration %d: open error: %v", i, err)
		}
		if !bytes.Equal(gotKv, kv) || !bytes.Equal(gotIv, iv) || !bytes.Equal(gotRaw, raw) {
			t.Fatalf("iteration %d: round-trip mismatch", i)
		}
	}
}

// TestKmsEnvelopeEmptyRaw checks that an empty raw section is a valid envelope.
func TestKmsEnvelopeEmptyRaw(t *testing.T) {
	kv := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	iv := []byte("ABCDEFGHIJKL")

	blob, err := sealKmsEnvelope(kv, iv, nil)
	if err != nil {
		t.Fatalf("seal with empty raw returned error: %v", err)
	}
	if want := base64.StdEncoding.EncodeToString(append(append([]byte{}, kv...), iv...)); blob != want {
		t.Fatalf("empty raw envelope mismatch: got %q want %q", blob, want)
	}

	gotKv, gotIv, gotRaw, err := openKmsEnvelope(blob)
	if err != nil {
		t.Fatalf("open of empty-raw envelope returned error: %v", err)
	}
	if !bytes.Equal(gotKv, kv) || !bytes.Equal(gotIv, iv) || len(gotRaw) != 0 {
		t.Fatalf("empty-raw round-trip mismatch: kv=%q iv=%q raw=%q", gotKv, gotIv, gotRaw)
	}
}

// TestKmsEnvelopeSealErrors covers keyVersionId/iv length violations.
func TestKmsEnvelopeSealErrors(t *testing.T) {
	validKv := make([]byte, kmsEktIdLength)
	validIv := make([]byte, kmsGcmIvLength)
	raw := []byte("x")

	cases := []struct {
		name string
		kv   []byte
		iv   []byte
	}{
		{"keyVersionId too short", make([]byte, kmsEktIdLength-1), validIv},
		{"keyVersionId too long", make([]byte, kmsEktIdLength+1), validIv},
		{"keyVersionId empty", nil, validIv},
		{"iv too short", validKv, make([]byte, kmsGcmIvLength-1)},
		{"iv too long", validKv, make([]byte, kmsGcmIvLength+1)},
		{"iv empty", validKv, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := sealKmsEnvelope(c.kv, c.iv, raw); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

// TestKmsEnvelopeOpenErrors covers invalid base64 and truncated envelopes.
func TestKmsEnvelopeOpenErrors(t *testing.T) {
	short := base64.StdEncoding.EncodeToString(make([]byte, kmsEktIdLength+kmsGcmIvLength-1)) // 47 bytes
	cases := []struct {
		name string
		blob string
	}{
		{"invalid base64", "!!!not-base64!!!"},
		{"empty string decodes to zero bytes", ""},
		{"decoded length below 48", short},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, _, _, err := openKmsEnvelope(c.blob); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}
