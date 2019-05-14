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
	"encoding/json"
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
	Host            string // the server host
	Port            int    // the server port
	AccessKeyId     string
	AccessKeySecret string
	Group           string
	Tenant          string
	DataId          string
	Cfg             interface{} // pointer of a struct that used for json unmarshal
	md5Sum          string
	running         bool
	cancel          context.CancelFunc
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
	resourcePaths *ResourcePaths
	configsMap    sync.Map
	lock          sync.RWMutex
}

// New returns a *Configs
func New() *Configs {
	return &Configs{
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

// Register will start a goroutine for listener
func (c *Configs) Register(wg *sync.WaitGroup, cf *Config) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	v := c.getConfig(cf.DataId)
	if v == nil {
		v = &Config{
			Host:            cf.Host,
			Port:            cf.Port,
			AccessKeyId:     cf.AccessKeyId,
			AccessKeySecret: cf.AccessKeySecret,
			Group:           cf.Group,
			Tenant:          cf.Tenant,
			DataId:          cf.DataId,
			Cfg:             cf.Cfg,
			md5Sum:          MD5(""),
			running:         false,
		}
		c.configsMap.Store(v.DataId, v)
	}

	// listener for dataId is running
	if v.running {
		return nil
	}

	v.running = true
	wg.Add(1)
	go c.listen(wg, cf)
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

	err = json.Unmarshal(v, cf.Cfg)
	if err != nil {
		return err
	}
	return nil
}

func (c *Configs) buildRetrieveRequest(cf *Config) (*http.Request, context.CancelFunc, error) {
	urlHttps, err := c.buildRetrieveUrl(cf)
	if err != nil {
		return nil, nil, err
	}

	urlHttps = fmt.Sprintf("%s?tenant=%s&dataId=%s&group=%s", urlHttps, cf.Tenant, cf.DataId, cf.Group)

	return c.buildRequest(cf, "GET", urlHttps, "", -1)
}

func (c *Configs) buildListenRequest(cf *Config) (*http.Request, context.CancelFunc, error) {
	urlHttps, err := c.buildListenUrl(cf)
	if err != nil {
		return nil, nil, err
	}

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
	signature := HmacSHA1(raw, cf.AccessKeySecret)

	req.Header.Set("Spas-AccessKey", cf.AccessKeyId)
	req.Header.Set("Spas-Signature", signature)
	if longPullingTimeout >= 0 {
		req.Header.Set("longPullingTimeout", strconv.Itoa(longPullingTimeout))
	}
	req.Header.Set("timeStamp", strconv.FormatInt(now, 10))
	return req, cancel, nil
}

func (c *Configs) buildRetrieveUrl(cf *Config) (string, error) {
	urlHttps := fmt.Sprintf("https://%s:%d%s", cf.Host, cf.Port, c.resourcePaths.ResourceGet)
	return urlHttps, nil
}

func (c *Configs) buildListenUrl(cf *Config) (string, error) {
	urlHttps := fmt.Sprintf("https://%s:%d%s", cf.Host, cf.Port, c.resourcePaths.ResourceListen)
	return urlHttps, nil
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
