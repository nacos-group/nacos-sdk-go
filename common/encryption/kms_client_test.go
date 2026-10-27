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
	"encoding/base64"
	"errors"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	dkms_util "github.com/aliyun/alibabacloud-dkms-gcs-go-sdk/openapi-util"
	dkms_sdk "github.com/aliyun/alibabacloud-dkms-gcs-go-sdk/sdk"
	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
)

var _ KmsClient = (*DedicatedKmsClient)(nil)

const testKvId = "0123456789abcdef0123456789abcdef0123" // 36 bytes

type fakeDkms struct {
	encReqs  []*dkms_sdk.EncryptRequest
	encOpts  []*dkms_util.RuntimeOptions
	encResps []*dkms_sdk.EncryptResponse
	decReq   *dkms_sdk.DecryptRequest
	decOpts  *dkms_util.RuntimeOptions
	decResp  *dkms_sdk.DecryptResponse
	gdkReq   *dkms_sdk.GenerateDataKeyRequest
	gdkOpts  *dkms_util.RuntimeOptions
	gdkResp  *dkms_sdk.GenerateDataKeyResponse
	err      error
	calls    []string
}

func (f *fakeDkms) EncryptWithOptions(r *dkms_sdk.EncryptRequest, o *dkms_util.RuntimeOptions) (*dkms_sdk.EncryptResponse, error) {
	f.calls = append(f.calls, "enc")
	f.encReqs = append(f.encReqs, r)
	f.encOpts = append(f.encOpts, o)
	if f.err != nil {
		return nil, f.err
	}
	resp := f.encResps[len(f.encReqs)-1]
	return resp, nil
}

func (f *fakeDkms) DecryptWithOptions(r *dkms_sdk.DecryptRequest, o *dkms_util.RuntimeOptions) (*dkms_sdk.DecryptResponse, error) {
	f.calls = append(f.calls, "dec")
	f.decReq, f.decOpts = r, o
	if f.err != nil {
		return nil, f.err
	}
	return f.decResp, nil
}

func (f *fakeDkms) GenerateDataKeyWithOptions(r *dkms_sdk.GenerateDataKeyRequest, o *dkms_util.RuntimeOptions) (*dkms_sdk.GenerateDataKeyResponse, error) {
	f.calls = append(f.calls, "gdk")
	f.gdkReq, f.gdkOpts = r, o
	if f.err != nil {
		return nil, f.err
	}
	return f.gdkResp, nil
}

func hdr(kv string) map[string]*string {
	return map[string]*string{kmsMigrationKeyVersionHdr: tea.String(kv)}
}

func newTestClient(f *fakeDkms) *DedicatedKmsClient {
	return &DedicatedKmsClient{client: f, kmsVersion: constant.KMSv3, verify: "ca", ignoreSSL: false}
}

func hasMigrationHdr(o *dkms_util.RuntimeOptions) bool {
	for _, h := range o.Headers {
		if tea.StringValue(h) == kmsMigrationKeyVersionHdr {
			return true
		}
	}
	return false
}

func TestDedicatedKmsClient_Encrypt(t *testing.T) {
	iv := bytes.Repeat([]byte{7}, kmsGcmIvLength)
	f := &fakeDkms{encResps: []*dkms_sdk.EncryptResponse{{Iv: iv, CiphertextBlob: []byte("cipher"), Headers: hdr(testKvId)}}}
	c := newTestClient(f)
	out, err := c.Encrypt("hello", "key-1")
	if err != nil {
		t.Fatal(err)
	}
	req := f.encReqs[0]
	if tea.StringValue(req.KeyId) != "key-1" || string(req.Plaintext) != "hello" {
		t.Fatalf("bad request: %+v", req)
	}
	if !hasMigrationHdr(f.encOpts[0]) {
		t.Fatal("missing migration header in runtime options")
	}
	if tea.StringValue(f.encOpts[0].Verify) != "ca" || tea.BoolValue(f.encOpts[0].IgnoreSSL) {
		t.Fatalf("bad verify/ignoreSSL: %+v", f.encOpts[0])
	}
	want, _ := sealKmsEnvelope([]byte(testKvId), iv, []byte("cipher"))
	if out != want {
		t.Fatalf("got %s want %s", out, want)
	}
}

func TestDedicatedKmsClient_Decrypt(t *testing.T) {
	iv := bytes.Repeat([]byte{9}, kmsGcmIvLength)
	blob, _ := sealKmsEnvelope([]byte(testKvId), iv, []byte("cipher"))
	f := &fakeDkms{decResp: &dkms_sdk.DecryptResponse{Plaintext: []byte("plain")}}
	out, err := newTestClient(f).Decrypt(blob)
	if err != nil {
		t.Fatal(err)
	}
	if out != "plain" {
		t.Fatalf("got %q", out)
	}
	r := f.decReq
	if string(r.CiphertextBlob) != "cipher" || !bytes.Equal(r.Iv, iv) {
		t.Fatalf("bad split: %+v", r)
	}
	if tea.StringValue(r.Headers[kmsMigrationKeyVersionHdr]) != testKvId {
		t.Fatalf("bad request header: %+v", r.Headers)
	}
	if !hasMigrationHdr(f.decOpts) {
		t.Fatal("missing migration header in runtime options")
	}
}

func TestDedicatedKmsClient_GenerateDataKey_TwoStep(t *testing.T) {
	iv := bytes.Repeat([]byte{5}, kmsGcmIvLength)
	dk := []byte{1, 2, 3, 4, 250, 251, 252, 253, 0, 1, 2, 3, 4, 5, 6, 7}
	f := &fakeDkms{
		gdkResp:  &dkms_sdk.GenerateDataKeyResponse{Plaintext: dk, Iv: bytes.Repeat([]byte{1}, 12), CiphertextBlob: []byte("ignored"), Headers: hdr(testKvId)},
		encResps: []*dkms_sdk.EncryptResponse{{Iv: iv, CiphertextBlob: []byte("second"), Headers: hdr("ffffffffffffffffffffffffffffffffffff")}},
	}
	plain, blob, err := newTestClient(f).GenerateDataKey("key-1", "AES_128")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 2 || f.calls[0] != "gdk" || f.calls[1] != "enc" {
		t.Fatalf("calls: %v", f.calls)
	}
	if tea.StringValue(f.gdkReq.KeyId) != "key-1" || tea.Int32Value(f.gdkReq.NumberOfBytes) != 16 {
		t.Fatalf("bad gdk req: %+v", f.gdkReq)
	}
	b64 := base64.StdEncoding.EncodeToString(dk)
	if string(f.encReqs[0].Plaintext) != b64 || tea.StringValue(f.encReqs[0].KeyId) != "key-1" {
		t.Fatalf("bad second-step encrypt req: %+v", f.encReqs[0])
	}
	if !hasMigrationHdr(f.gdkOpts) || !hasMigrationHdr(f.encOpts[0]) {
		t.Fatal("missing migration header")
	}
	if plain != b64 {
		t.Fatalf("plain %q want %q", plain, b64)
	}
	want, _ := sealKmsEnvelope([]byte(testKvId), iv, []byte("second"))
	if blob != want {
		t.Fatalf("envelope should use GDK-header kv and second encrypt iv/cipher")
	}
}

func TestDedicatedKmsClient_GenerateDataKey_KeySpec(t *testing.T) {
	for spec, n := range map[string]int32{"": 32, "AES_256": 32, "AES_128": 16} {
		f := &fakeDkms{
			gdkResp:  &dkms_sdk.GenerateDataKeyResponse{Plaintext: []byte("x"), Headers: hdr(testKvId)},
			encResps: []*dkms_sdk.EncryptResponse{{Iv: make([]byte, 12), CiphertextBlob: []byte("c")}},
		}
		if _, _, err := newTestClient(f).GenerateDataKey("k", spec); err != nil {
			t.Fatal(err)
		}
		if tea.Int32Value(f.gdkReq.NumberOfBytes) != n {
			t.Fatalf("spec %q: got %d want %d", spec, tea.Int32Value(f.gdkReq.NumberOfBytes), n)
		}
	}
	f := &fakeDkms{}
	if _, _, err := newTestClient(f).GenerateDataKey("k", "AES_999"); err == nil || len(f.calls) != 0 {
		t.Fatal("invalid keySpec must fail before any RPC")
	}
}

func TestDedicatedKmsClient_Errors(t *testing.T) {
	// Encrypt: missing migration header
	f := &fakeDkms{encResps: []*dkms_sdk.EncryptResponse{{Iv: make([]byte, 12), CiphertextBlob: []byte("c"), Headers: map[string]*string{}}}}
	if _, err := newTestClient(f).Encrypt("a", "k"); err == nil {
		t.Fatal("expected missing header error")
	}
	// GDK: missing header on GDK response
	f = &fakeDkms{
		gdkResp:  &dkms_sdk.GenerateDataKeyResponse{Plaintext: []byte("x")},
		encResps: []*dkms_sdk.EncryptResponse{{Iv: make([]byte, 12), CiphertextBlob: []byte("c"), Headers: hdr(testKvId)}},
	}
	if _, _, err := newTestClient(f).GenerateDataKey("k", "AES_256"); err == nil {
		t.Fatal("expected missing header error on GDK")
	}
	// Decrypt: bad envelope, no RPC
	f = &fakeDkms{}
	if _, err := newTestClient(f).Decrypt("!!not-base64!!"); err == nil || len(f.calls) != 0 {
		t.Fatal("expected envelope error without RPC")
	}
	if _, err := newTestClient(f).Decrypt(base64.StdEncoding.EncodeToString([]byte("short"))); err == nil || len(f.calls) != 0 {
		t.Fatal("expected short-envelope error without RPC")
	}
	// RPC errors propagate
	f = &fakeDkms{err: errors.New("boom")}
	c := newTestClient(f)
	blob, _ := sealKmsEnvelope([]byte(testKvId), make([]byte, 12), nil)
	if _, err := c.Encrypt("a", "k"); err == nil {
		t.Fatal("encrypt err")
	}
	if _, err := c.Decrypt(blob); err == nil {
		t.Fatal("decrypt err")
	}
	if _, _, err := c.GenerateDataKey("k", "AES_256"); err == nil {
		t.Fatal("gdk err")
	}
}
