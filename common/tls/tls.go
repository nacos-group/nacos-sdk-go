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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

// NewTLS returns a config structure is used to configure a TLS client
func NewTLS(c constant.TLSConfig) (tc *tls.Config, err error) {
	tc = &tls.Config{}
	if len(c.CertFile) > 0 && len(c.KeyFile) > 0 {
		cert, err := certificate(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, err
		}
		tc.Certificates = []tls.Certificate{*cert}
	}

	if len(c.CaFile) <= 0 {
		tc.InsecureSkipVerify = true
		return tc, nil
	}
	if len(c.ServerNameOverride) > 0 {
		tc.ServerName = c.ServerNameOverride
	}
	tc.RootCAs, err = rootCert(c.CaFile)
	return
}

func rootCert(caFile string) (*x509.CertPool, error) {
	b, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(b) {
		return nil, fmt.Errorf("credentials: failed to append certificates")
	}
	return cp, nil
}

func certificate(certFile, keyFile string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}
