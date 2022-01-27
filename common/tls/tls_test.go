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

package tls

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/stretchr/testify/assert"
)

var (
	testCaCrt = []byte(`-----BEGIN CERTIFICATE-----
MIICojCCAYoCCQDbLXd9WTa7rTANBgkqhkiG9w0BAQsFADATMREwDwYDVQQDDAh3
ZS1hcy1jYTAeFw0yMTA4MDQxNTE3MTlaFw00ODEyMjAxNTE3MTlaMBMxETAPBgNV
BAMMCHdlLWFzLWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAyc0D
ca+T6zwVnroauoVYQvPEx6R2jxmEgmCEclXegmO0rJ+23nP63nhgDvLN2Yv4olmv
d+eh1WfsnmqfdtqUcooZQIYZHWw5jWSYygZOwUWzfIclVFcyfkZnP7qTMGjYPn9Y
hOfdgSIh1c/DXrKFu1VQd9p3DevUD+ImAbxYJW4SMgYvliooPABbFU/sm3ZrHPwb
Ik8U1KlGHoYtw8KslD0INTfOOEYfQToeZtoAkoajykyteYYbI0kNVYBr2W3AOEXt
/QQkj/kAa1o8YKrVkufvi90UI/53SnJa0o5TDzXJCAu4jg4Xfpq0tVogFuEamMeI
f2R4JL77flG41nqN2QIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQCOYw0C4wJTbHly
LRmR7lJiLDkdObVKyQM6UUrbY62h9Fu02vYI9a8sj4xVSr3JKTXWBUwXSDqTUqgr
+9+sWoWxGwHbDINVHSsy2vnZlhGFkRkdWv3qgAPOn1Dc2ZMVzNEXRnmLFc1X2Ir/
niuk4cqSKwFE4IoJz9CHDDOlJzowimTwD6ReIrJDhi0pEFE6YtBVfRF5XPvz3AyG
mIFTX9LPRmCBnRi7We9cea+zuFarbjU6qDtf9jDfWANz1Gv6OHf0oM6BoCJ0jp0b
tJ5yJe4OCybgpb5bMZygBkGWozeQ5I/XzhkswNN0jVXeC3e0UWLscYvsgPVAM1kH
vZvo/wBG
-----END CERTIFICATE-----`)

	testClientCrt = []byte(`-----BEGIN CERTIFICATE-----
MIICozCCAYsCCQDaqEi3maR5ojANBgkqhkiG9w0BAQUFADATMREwDwYDVQQDDAh3
ZS1hcy1jYTAeFw0yMTA4MDQxNTE4MzFaFw0yMjA4MDQxNTE4MzFaMBQxEjAQBgNV
BAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALPN
fcRRVPcinxUhS+Q1pvL6nZk7Gg7vcyoOqdX68R/iG/by/dL/wKbYzi2gyMyNN7Q8
o1+74inrsS5KMXAuyxrurdudWaoS9eWFzrd9r98+47kxUVwye5R4leT9+NCiI0mp
HOqEkOFv+X//kkCtCpDVj/XkfZJ+UrJHgAHvL9me/v5yUvLDDgu7/cdGU1tRwGQc
zabzk8SkvoQjMWaZ+R1eIWfeg9lYFyWQyYJhZkAqlBhlyDHw3FfrfPKjrxsI6uDq
ACeptHUuMZu6H6EzKDJtnx5DhSrwXTOwEcsOzTl60Hb2wT154CGi+7VbaZDGl6Uf
ZBkVAiSZvNCDJ0NGa9kCAwEAATANBgkqhkiG9w0BAQUFAAOCAQEAqjg4WeFbUcYC
Ko5R6UNTEYUvLYk45hks/Cmu5Mdqe3tFbPsr3EVdqd+zFJrINQQTlhZx14stJjIO
b3+eRX7Rldow7AI9Q2dyCQUoWiYmmco//4Mx1jObN2wMUd7tavhwg8RNdps1Yly2
/l0Vj2OhNDFnApqAiHZ0NGuCW7CLBvuD2XFCPZLCYFv0aQTw/Vr0+hHvNApmFYn0
4wiveiWUf98KKrp93lbzskAd3OZmoNx4bIo/J2Arcz0KzuliBgXDcGPOb4YLRLgs
QBhpY/VCGRat52Ys+sm6l/+2Cv+C2mHhn6V4BNVifaZIgOofXwzO4vaQO9sNTZ4f
qqJ4/s4nDg==
-----END CERTIFICATE-----`)

	testClientKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAs819xFFU9yKfFSFL5DWm8vqdmTsaDu9zKg6p1frxH+Ib9vL9
0v/AptjOLaDIzI03tDyjX7viKeuxLkoxcC7LGu6t251ZqhL15YXOt32v3z7juTFR
XDJ7lHiV5P340KIjSakc6oSQ4W/5f/+SQK0KkNWP9eR9kn5SskeAAe8v2Z7+/nJS
8sMOC7v9x0ZTW1HAZBzNpvOTxKS+hCMxZpn5HV4hZ96D2VgXJZDJgmFmQCqUGGXI
MfDcV+t88qOvGwjq4OoAJ6m0dS4xm7ofoTMoMm2fHkOFKvBdM7ARyw7NOXrQdvbB
PXngIaL7tVtpkMaXpR9kGRUCJJm80IMnQ0Zr2QIDAQABAoIBAFMiakpBSMXT7jY4
5Pwpin3CPuhAmXXaZSdHDGPx2VdiloeCJrZOpmb+y6XxN6bMjLr7Zpa3KoUzgwLi
LyWtnR9gyGZIxNKMXcG4MrJInO7eBzDziqjUdqtZbgUpIMhmj2ZZmRMeJFb4DSaP
prHc0IvTEvMgqKb5XYcs5BUA4OD/ihXvpW7GN4c/+iakJN0UBTI0/P/bTiYkERN8
ousw8UebrODyUbI20rqL/UO8UyOPIyifFpU29nvn/57Z42TzU+BqdxmJG5VJCR7w
lvJhA+jkldorHksHm1Z/9qDMj2UIvuPTJoa6L2t1utJRFgE+27QKHQ9Nv7IjbkUr
gdHO/QECgYEA5TneVmplJ4ARg8JIFYBotArcSj4f+Z2pZ36KO2CMklnnTFy2tAuw
766yxvZULCU5hr9AwlQwGkqt99o50a8WP4HBGjZ27r6CPODbvZvF7JnsFsD8z178
H52GNMO626KogDrC6aoJnYPJQAY+wGhR6Gg05goGoiKdUnhBOzKuEfECgYEAyM3O
TUch0FTBmKGA7IWGRL9bQBpw13UtZOokm5g5zTg+yDsQZ5BZCIgycZf77zQmIpbZ
TJe8xeFBI8fjfAF+UAfvzwwc4b3dSmD/jUSrv2gcKfCff2wZw8c8sYwbqNpSeX3l
Y2m7VJj5fw8I2vMOKjzISNKX55qNc8BGUuLiEGkCgYEA1Gb93ccyuhpSoGuLDdlx
q7sQiv7r9AmiqpK3lfON7iK+T6TtawIWTtHrOK+SKWHI31IiuK7377TZZPvibai2
jdw2yYpERE9lMPIOy7AnA2lROXhUCfdy2fzGGehwIgqj5kYMzCXSSRGPjvL6fKFt
nFPLCImrwdsfOgbSMv9wCpECgYA1n2fxEQbBmHCebrp77ug9IZCfnK/3iW4W3cPq
3QrKd7OkSsmFrnFoKt61oO2BIj7wy7G5l2esvAtmH7Hq4oc1nfj3JHft/ILEowR7
WBQ5J/claAFfyKFUu7bEfvK/85VEpk8Ebi69V6CAwqYNugxVUSf28m3oRkhx2a2t
4rKVyQKBgBYWALJLO3YwzpdelzVJiOPatVrphQarUKafsE32u/iBBvJwfpYpkclh
kJ4wLmJAMU8VAhvfSh6T8z8us9z3znONoUI0z6GzwbjROFRtd2WiffXvgcKfTacu
q9K53Jum9GDmkbUODa77sWR1zQsdrqSKywcjP/6FYXU9RMDqKUpm
-----END RSA PRIVATE KEY-----`)
)

func Test_NewTLS(t *testing.T) {
	dir, err := ioutil.TempDir("", "tls-test")
	if err != nil {
		t.Errorf(err.Error())
	}
	defer os.RemoveAll(dir)

	caPath, crtPath, keyPath := filepath.Join(dir, "ca.crt"), filepath.Join(dir, "client.crt"), filepath.Join(dir, "client.key")
	ioutil.WriteFile(caPath, testCaCrt, 0666)
	ioutil.WriteFile(crtPath, testClientCrt, 0666)
	ioutil.WriteFile(keyPath, testClientKey, 0666)

	t.Run("TestNoAuth", func(t *testing.T) {
		cfg, err := NewTLS(constant.SkipVerifyConfig)
		assert.Nil(t, err)
		assert.Equal(t, true, cfg.InsecureSkipVerify)
		assert.Nil(t, cfg.RootCAs)
		assert.Nil(t, cfg.Certificates)

	})

	t.Run("TestClientAuth", func(t *testing.T) {
		cfg, err := NewTLS(*constant.NewTLSConfig(
			constant.WithCA(caPath, ""),
		))
		assert.Nil(t, err)
		assert.Equal(t, false, cfg.InsecureSkipVerify)
		assert.NotNil(t, cfg.RootCAs)
		assert.Nil(t, cfg.Certificates)

	})

	t.Run("TestServerAuth", func(t *testing.T) {
		cfg, err := NewTLS(*constant.NewTLSConfig(
			constant.WithCA(caPath, ""),
			constant.WithCertificate(crtPath, keyPath),
		))
		assert.Nil(t, err)
		assert.Equal(t, false, cfg.InsecureSkipVerify)
		assert.NotNil(t, cfg.RootCAs)
		assert.NotNil(t, cfg.Certificates)
	})
}
