package configs

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConfigs_NewConfig(t *testing.T) {
	c := New("123.4.56.78", 0, "", "")
	assert.NotNil(t, c)
	cf := &Config{}
	newCf := c.newConfig(cf)
	assert.NotNil(t, newCf)
	assert.NotEmpty(t, newCf.md5Sum)
	assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", newCf.md5Sum)
}

func TestConfigs_Register(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)
	var wg sync.WaitGroup
	cf := &Config{
		Tenant: "tenantId_123",
		Group: "group_123",
		DataId: "dataId123",
	}
	err := c.OnChange(&wg, cf)
	assert.Nil(t, err)
}

func TestConfigs_Register_Running(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)
	var wg sync.WaitGroup
	cf := &Config{
		Tenant: "tenantId_123",
		Group: "group_123",
		DataId: "dataId123",
	}
	cf.running = true

	c.configsMap.Store("dataId123", cf)
	err := c.OnChange(&wg, cf)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "running"))
}

func TestConfigs_Stop(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)
	var wg sync.WaitGroup
	cf := &Config{
		Tenant: "tenantId_123",
		Group: "group_123",
		DataId: "dataId123",
	}
	err := c.OnChange(&wg, cf)
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)

	err = c.Stop(&wg)
	assert.Nil(t, err)
	wg.Wait()
}

func TestConfigs_Stop_Failed(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)
	var wg sync.WaitGroup
	cf := &Configs{}
	c.configsMap.Store("dataId123", cf)
	err := c.Stop(&wg)
	assert.NotNil(t, err)
}

func TestConfigs_GetConfig(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)

	cf := &Config{}
	c.configsMap.Store("dataId1", cf)
	v := c.getConfig("dataId1")
	assert.NotNil(t, v)
	assert.Equal(t, "123.4.56.78", c.Host)
}

func TestConfigs_GetConfig_NotFound(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)

	cf := &Config{}
	c.configsMap.Store("dataId1", cf)
	v := c.getConfig("dataId2")
	assert.Nil(t, v)
}

func TestConfigs_BuildRetieveUrl(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)

	cf := &Config{}
	v := c.buildRetrieveUrl(cf)
	assert.NotNil(t, v)
	assert.Equal(t, "https://123.4.56.78:8888/nacos/v1/cs/configs", v)
}

func TestConfigs_BuildRetieveUrl_Customized(t *testing.T) {
	rp := &ResourcePaths{
		ResourceListen: "/abc/1/listener",
		ResourceGet: "/abd/2",
		ResourcePost: "/abe/3",
		ResourceDelete: "/abf/4",
	}
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123").WithResourcePaths(rp)
	assert.NotNil(t, c)

	cf := &Config{}
	v := c.buildRetrieveUrl(cf)
	assert.NotNil(t, v)
	assert.Equal(t, "https://123.4.56.78:8888/abd/2", v)
}

func TestConfigs_BuildListenUrl(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)

	cf := &Config{}
	v := c.buildListenUrl(cf)
	assert.NotNil(t, v)
	assert.Equal(t, "https://123.4.56.78:8888/nacos/v1/cs/configs/listener", v)
}

func TestConfigs_BuildListenUrl_Customized(t *testing.T) {
	rp := &ResourcePaths{
		ResourceListen: "/abc/1/listener",
		ResourceGet: "/abd/2",
		ResourcePost: "/abe/3",
		ResourceDelete: "/abf/4",
	}
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123").WithResourcePaths(rp)
	assert.NotNil(t, c)

	cf := &Config{}
	v := c.buildListenUrl(cf)
	assert.NotNil(t, v)
	assert.Equal(t, "https://123.4.56.78:8888/abc/1/listener", v)
}

func TestConfigs_BuildRequest_Get(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)

	cf := &Config{
		Tenant: "tenantId_123",
		Group: "group_123",
	}

	v := c.buildRetrieveUrl(cf)
	assert.NotNil(t, v)
	req, cancel, err := c.buildRequest(cf, "GET", v, "", -1)
	assert.Nil(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, cancel)
	assert.Equal(t, fmt.Sprintf("%s:%d", c.Host, c.Port), req.Host)
	assert.Equal(t, "GET", req.Method)
	assert.NotNil(t, req.Header.Get("Spas-Signature"))
	assert.NotEmpty(t, req.Header.Get("Spas-Signature"))
}

func TestConfigs_BuildRequest_Post(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)

	cf := &Config{
		Tenant: "tenantId_123",
		Group: "group_123",
	}

	v := c.buildRetrieveUrl(cf)
	assert.NotNil(t, v)
	req, cancel, err := c.buildRequest(cf, "POST", v, "abc", 10)
	assert.Nil(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, cancel)
	assert.Equal(t, fmt.Sprintf("%s:%d", c.Host, c.Port), req.Host)
	assert.Equal(t, "POST", req.Method)
	assert.NotNil(t, req.Header.Get("Spas-Signature"))
	assert.NotEmpty(t, req.Header.Get("Spas-Signature"))
	assert.Equal(t, "10", req.Header.Get("longpullingtimeout"))
}

func TestConfigs_BuildListenRequest(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)

	cf := &Config{
		Tenant: "tenantId_123",
		Group: "group_123",
	}

	req, cancel, err := c.buildListenRequest(cf)
	assert.Nil(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, cancel)
	assert.Equal(t, fmt.Sprintf("%s:%d", c.Host, c.Port), req.Host)
	assert.Equal(t, "POST", req.Method)
	assert.NotNil(t, req.Header.Get("Spas-Signature"))
	assert.NotEmpty(t, req.Header.Get("Spas-Signature"))
	assert.Equal(t, "30000", req.Header.Get("longpullingtimeout"))
}

func TestConfigs_BuildRetrieveRequest(t *testing.T) {
	c := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	assert.NotNil(t, c)

	cf := &Config{
		Tenant: "tenantId_123",
		Group: "group_123",
		DataId: "dataId123",
	}

	req, cancel, err := c.buildRetrieveRequest(cf)
	assert.Nil(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, cancel)
	assert.Equal(t, fmt.Sprintf("%s:%d", c.Host, c.Port), req.Host)
	assert.Equal(t, "GET", req.Method)
	assert.NotNil(t, req.Header.Get("Spas-Signature"))
	assert.NotEmpty(t, req.Header.Get("Spas-Signature"))
}