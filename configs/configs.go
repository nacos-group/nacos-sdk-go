// Copyright 2019 alibaba cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Since: v0.1.0
// Author: github.com/atlanssia
package configs

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	fieldSeparator          = "%02"
	configSeparator         = "%01"
	baseResource            = "/nacos/v1/cs/configs"
	defaultListenerResource = baseResource + "/listener"
)

// Config defines the config information that used for retrieve from server
type Config struct {
	Group    string
	Tenant   string // in ACM, the namespace is the tenant
	DataId   string
	OnChange func(namespace, group, dataId string, data []byte)
	md5Sum   string
	running  bool
	cancel   context.CancelFunc
}

// ResourcePaths defines the resources for request, default for nacos resources
// ref: https://nacos.io/zh-cn/docs/open-API.html
type ResourcePaths struct {
	ResourceGet    string // the target resource for GET method, ex: /nacos/v1/cs/configs
	ResourcePost   string // the target resource for POST method, ex: /nacos/v1/cs/configs
	ResourceDelete string // the target resource for DELETE method, ex: /nacos/v1/cs/configs
	ResourceListen string // the target resource for Listen, ex: /nacos/v1/cs/configs/listener
}

// Configs contains all configs
type Configs struct {
	Host            string // the server host
	Port            int    // the server port
	AccessKeyId     string
	AccessKeySecret string
	resourcePaths   *ResourcePaths
	configsMap      sync.Map
	lock            sync.RWMutex
}

// New returns a *Configs
func New(host string, port int, accessKeyId, AccessKeySecret string) *Configs {
	return &Configs{
		Host:            host,
		Port:            port,
		AccessKeyId:     accessKeyId,
		AccessKeySecret: AccessKeySecret,
		resourcePaths: &ResourcePaths{
			ResourceGet:    baseResource,
			ResourcePost:   baseResource,
			ResourceDelete: baseResource,
			ResourceListen: defaultListenerResource,
		},
	}
}

// WithResourcePaths can specify the *ResourcePaths
func (c *Configs) WithResourcePaths(rp *ResourcePaths) *Configs {
	c.resourcePaths = rp
	return c
}

// OnChange will start a goroutine for listener
func (c *Configs) OnChange(wg *sync.WaitGroup, cf *Config) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	v := c.getConfig(cf.DataId)
	if v == nil {
		v = c.newConfig(cf)
	}

	// listener for dataId is running
	if v.running {
		return fmt.Errorf("listener for dataId:%s is running", cf.DataId)
	}

	v.running = true
	wg.Add(1)
	go c.listen(wg, v)
	return nil
}

func (c *Configs) newConfig(cf *Config) *Config {
	newCf := &Config{
		Group:    cf.Group,
		Tenant:   cf.Tenant,
		DataId:   cf.DataId,
		OnChange: cf.OnChange,
		md5Sum:   MD5(""),
		running:  false,
	}
	c.configsMap.Store(newCf.DataId, newCf)
	return newCf
}

// Stop will stop all listener
func (c *Configs) Stop(wg *sync.WaitGroup) error {
	var failedDataId []string
	c.configsMap.Range(func(key, value interface{}) bool {
		if cf, ok := value.(*Config); ok {
			cf.cancel()
			cf.running = false
		} else {
			failedDataId = append(failedDataId, key.(string))
		}
		return true
	})

	if len(failedDataId) > 0 {
		return fmt.Errorf("stop listener failed: %v", failedDataId)
	}

	return nil
}

func (c *Configs) listen(wg *sync.WaitGroup, cf *Config) {
	defer wg.Done()

	failCount := 0
	for {
		if !cf.running || failCount > 10 {
			break
		}

		client := c.newHttpClient()
		req, cancel, err := c.buildListenRequest(cf)
		if err != nil {
			failCount++
			time.Sleep(1 * time.Second)
			continue
		}
		cf.cancel = cancel

		// do request
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			failCount++
			time.Sleep(1 * time.Second)
			continue
		}

		v, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			failCount++
			time.Sleep(1 * time.Second)
			continue
		}

		if len(v) == 0 {
			continue
		}

		if err := c.retrieveConfig(cf); err != nil {
			failCount++
			time.Sleep(3 * time.Second)
			continue
		}
	}
}

func (c *Configs) newHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
	return client
}

func (c *Configs) retrieveConfig(cf *Config) error {
	req, _, err := c.buildRetrieveRequest(cf)
	if err != nil {
		return err
	}
	client := c.newHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	v, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	cf.OnChange(cf.Tenant, cf.Group, cf.DataId, v)

	return nil
}

func (c *Configs) buildRetrieveRequest(cf *Config) (*http.Request, context.CancelFunc, error) {
	urlHttps := c.buildRetrieveUrl(cf)
	urlHttps = fmt.Sprintf("%s?tenant=%s&dataId=%s&group=%s", urlHttps, cf.Tenant, cf.DataId, cf.Group)

	return c.buildRequest(cf, "GET", urlHttps, "", -1)
}

func (c *Configs) buildListenRequest(cf *Config) (*http.Request, context.CancelFunc, error) {
	urlHttps := c.buildListenUrl(cf)
	bodyValue := fmt.Sprintf("%s%s%s%s%s%s%s%s",
		cf.DataId, fieldSeparator,
		cf.Group, fieldSeparator,
		cf.md5Sum, fieldSeparator,
		cf.Tenant, configSeparator)
	body := "Probe-Modify-Request=" + bodyValue

	return c.buildRequest(cf, "POST", urlHttps, body, 30000)
}

func (c *Configs) buildRequest(
	cf *Config,
	method, urlStr, body string,
	longPullingTimeout int) (*http.Request, context.CancelFunc, error) {

	var bd io.Reader = nil
	if body != "" {
		bd = bytes.NewBuffer([]byte(body))
	}

	req, err := http.NewRequest(method, urlStr, bd)
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	req.WithContext(ctx)

	now := time.Now().UnixNano() / 1e6

	raw := cf.Tenant + "+" + cf.Group + "+" + strconv.FormatInt(now, 10)
	signature := HmacSHA1(raw, c.AccessKeySecret)

	req.Header.Set("Spas-AccessKey", c.AccessKeyId)
	req.Header.Set("Spas-Signature", signature)
	if longPullingTimeout >= 0 {
		req.Header.Set("longPullingTimeout", strconv.Itoa(longPullingTimeout))
	}
	req.Header.Set("timeStamp", strconv.FormatInt(now, 10))
	return req, cancel, nil
}

func (c *Configs) buildRetrieveUrl(cf *Config) string {
	urlHttps := fmt.Sprintf("https://%s:%d%s", c.Host, c.Port, c.resourcePaths.ResourceGet)
	return urlHttps
}

func (c *Configs) buildListenUrl(cf *Config) string {
	urlHttps := fmt.Sprintf("https://%s:%d%s", c.Host, c.Port, c.resourcePaths.ResourceListen)
	return urlHttps
}

func (c *Configs) getConfig(dataId string) *Config {
	if v, found := c.configsMap.Load(dataId); found {
		if cfg, ok := v.(*Config); ok {
			return cfg
		}
		return nil
	}
	return nil
}

// HmacSHA1 returns the base64 encoded string hashed by Hmac and SHA1
func HmacSHA1(text, key string) string {
	algorithm := hmac.New(sha1.New, []byte(key))
	return hash2base64(algorithm, text)
}

// MD5 returns the md5 hashed string
func MD5(text string) string {
	algorithm := md5.New()
	return hash2string(algorithm, text)
}

func hash2base64(algorithm hash.Hash, text string) string {
	algorithm.Write([]byte(text))
	return base64.StdEncoding.EncodeToString(algorithm.Sum(nil))
}

func hash2string(algorithm hash.Hash, text string) string {
	algorithm.Write([]byte(text))
	return hex.EncodeToString(algorithm.Sum(nil))
}
