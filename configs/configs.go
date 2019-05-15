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
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	logger          Logger
}

// Logger is used to log error messages.
type Logger interface {
	Println(v ...interface{})
}

// New returns a *Configs
func New(host string, port int, accessKeyId, AccessKeySecret string) *Configs {
	cs := &Configs{
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
	_ = cs.SetLogger(log.New(os.Stdout, "\r\n", 0))
	return cs
}

// WithResourcePaths can specify the *ResourcePaths
func (cs *Configs) WithResourcePaths(rp *ResourcePaths) *Configs {
	cs.resourcePaths = rp
	return cs
}

// SetLogger sets the logger that implemented interface configs.Logger
// The initial logger is os.Stderr.
func (cs *Configs) SetLogger(logger Logger) error {
	if logger == nil {
		return errors.New("logger is nil")
	}
	cs.logger = logger
	return nil
}

// OnChange will start a goroutine for listener
func (cs *Configs) OnChange(wg *sync.WaitGroup, cf *Config) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	v := cs.getConfig(cf)
	if v == nil {
		v = cs.newConfig(cf)
	}

	// listener for dataId is running
	if v.running {
		return fmt.Errorf("listener for dataId:%s is running", cf.DataId)
	}

	v.running = true
	wg.Add(1)
	go cs.listen(wg, v)
	return nil
}

func (cs *Configs) newConfig(cf *Config) *Config {
	newCf := &Config{
		Group:    cf.Group,
		Tenant:   cf.Tenant,
		DataId:   cf.DataId,
		OnChange: cf.OnChange,
		md5Sum:   MD5(""),
		running:  false,
	}
	cs.configsMap.Store(newCf.key(), newCf)
	return newCf
}

// Stop will stop all listener
func (cs *Configs) Stop(wg *sync.WaitGroup) error {
	cs.logger.Println("stopping all listeners")
	var failedConfigs []string
	cs.configsMap.Range(func(key, value interface{}) bool {
		if cf, ok := value.(*Config); ok {
			cf.cancel()
			cf.running = false
			cs.logger.Println(cf.key() + " stopped")
		} else {
			failedConfigs = append(failedConfigs, key.(string))
		}
		return true
	})

	if len(failedConfigs) > 0 {
		cs.logger.Println(fmt.Sprintf("stop failed list: %v", failedConfigs))
		return fmt.Errorf("stop listener failed: %v", failedConfigs)
	}
	cs.logger.Println("all stopped")
	return nil
}

func (cs *Configs) listen(wg *sync.WaitGroup, cf *Config) {
	defer wg.Done()

	failCount := 0
	for {
		if !cf.running || failCount > 10 {
			cs.logger.Println("stop listener")
			break
		}

		client := cs.newHttpClient()
		req, cancel, err := cs.buildListenRequest(cf)
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

		if err := cs.retrieveConfig(cf); err != nil {
			failCount++
			time.Sleep(3 * time.Second)
			continue
		}
	}
}

func (cs *Configs) newHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
	return client
}

func (cs *Configs) retrieveConfig(cf *Config) error {
	req, _, err := cs.buildRetrieveRequest(cf)
	if err != nil {
		return err
	}
	client := cs.newHttpClient()
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

func (cs *Configs) buildRetrieveRequest(cf *Config) (*http.Request, context.CancelFunc, error) {
	urlHttps := cs.buildRetrieveUrl(cf)
	urlHttps = fmt.Sprintf("%s?tenant=%s&dataId=%s&group=%s", urlHttps, cf.Tenant, cf.DataId, cf.Group)

	return cs.buildRequest(cf, "GET", urlHttps, "", -1)
}

func (cs *Configs) buildListenRequest(cf *Config) (*http.Request, context.CancelFunc, error) {
	urlHttps := cs.buildListenUrl(cf)
	bodyValue := fmt.Sprintf("%s%s%s%s%s%s%s%s",
		cf.DataId, fieldSeparator,
		cf.Group, fieldSeparator,
		cf.md5Sum, fieldSeparator,
		cf.Tenant, configSeparator)
	body := "Probe-Modify-Request=" + bodyValue

	return cs.buildRequest(cf, "POST", urlHttps, body, 30000)
}

func (cs *Configs) buildRequest(
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
	req = req.WithContext(ctx)

	now := time.Now().UnixNano() / 1e6

	raw := cf.Tenant + "+" + cf.Group + "+" + strconv.FormatInt(now, 10)
	signature := HmacSHA1(raw, cs.AccessKeySecret)

	req.Header.Set("Spas-AccessKey", cs.AccessKeyId)
	req.Header.Set("Spas-Signature", signature)
	if longPullingTimeout >= 0 {
		req.Header.Set("longPullingTimeout", strconv.Itoa(longPullingTimeout))
	}
	req.Header.Set("timeStamp", strconv.FormatInt(now, 10))
	return req, cancel, nil
}

func (cs *Configs) buildRetrieveUrl(cf *Config) string {
	urlHttps := fmt.Sprintf("https://%s:%d%s", cs.Host, cs.Port, cs.resourcePaths.ResourceGet)
	return urlHttps
}

func (cs *Configs) buildListenUrl(cf *Config) string {
	urlHttps := fmt.Sprintf("https://%s:%d%s", cs.Host, cs.Port, cs.resourcePaths.ResourceListen)
	return urlHttps
}

func (cs *Configs) getConfig(cf *Config) *Config {
	if v, found := cs.configsMap.Load(cf.key()); found {
		if cfg, ok := v.(*Config); ok {
			return cfg
		}
		return nil
	}
	return nil
}

func (c *Config) key() string {
	return fmt.Sprintf("%s_%s_%s", c.Tenant, c.Group, c.DataId)
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
