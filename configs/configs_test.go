package configs

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetConfig(t *testing.T) {
	c := New()
	assert.NotNil(t, c)

	cf := &Config{
		Host: "123.4.56.78",
	}
	c.configsMap.Store("dataId1", cf)
	v := c.getConfig("dataId1")
	assert.NotNil(t, v)
	assert.Equal(t, "123.4.56.78", v.Host)
}

func TestBuildRetieveUrl(t *testing.T) {
	c := New()
	assert.NotNil(t, c)

	cf := &Config{
		Host: "123.4.56.78",
		Port: 8888,
	}
	v, err := c.buildRetrieveUrl(cf)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "https://123.4.56.78:8888/nacos/v1/cs/configs", v)
}

func TestBuildRetieveUrl_Customized(t *testing.T) {
	rp := &ResourcePaths{
		ResourceListen: "/abc/1/listener",
		ResourceGet: "/abd/2",
		ResourcePost: "/abe/3",
		ResourceDelete: "/abf/4",
	}
	c := New().WithResourcePaths(rp)
	assert.NotNil(t, c)

	cf := &Config{
		Host: "123.4.56.78",
		Port: 8888,
	}
	v, err := c.buildRetrieveUrl(cf)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "https://123.4.56.78:8888/abd/2", v)
}

func TestBuildListenUrl(t *testing.T) {
	c := New()
	assert.NotNil(t, c)

	cf := &Config{
		Host: "123.4.56.78",
		Port: 8888,
	}
	v, err := c.buildListenUrl(cf)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "https://123.4.56.78:8888/nacos/v1/cs/configs/listener", v)
}

func TestBuildListenUrl_Customized(t *testing.T) {
	rp := &ResourcePaths{
		ResourceListen: "/abc/1/listener",
		ResourceGet: "/abd/2",
		ResourcePost: "/abe/3",
		ResourceDelete: "/abf/4",
	}
	c := New().WithResourcePaths(rp)
	assert.NotNil(t, c)

	cf := &Config{
		Host: "123.4.56.78",
		Port: 8888,
	}
	v, err := c.buildListenUrl(cf)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "https://123.4.56.78:8888/abc/1/listener", v)
}

func TestBuildRequest_Get(t *testing.T) {
	c := New()
	assert.NotNil(t, c)

	cf := &Config{
		Host: "123.4.56.78",
		Port: 8888,
		AccessKeyId: "access_key_id_123",
		AccessKeySecret: "access_key_secret_123",
		Tenant: "tenantId_123",
		Group: "group_123",
	}

	v, err := c.buildRetrieveUrl(cf)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	req, cancel, err := c.buildRequest(cf, "GET", v, "", -1)
	assert.Nil(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, cancel)
	assert.Equal(t, fmt.Sprintf("%s:%d", cf.Host, cf.Port), req.Host)
	assert.Equal(t, "GET", req.Method)
	assert.NotNil(t, req.Header.Get("Spas-Signature"))
	assert.NotEmpty(t, req.Header.Get("Spas-Signature"))
}

func TestBuildRequest_Post(t *testing.T) {
	c := New()
	assert.NotNil(t, c)

	cf := &Config{
		Host: "123.4.56.78",
		Port: 8888,
		AccessKeyId: "access_key_id_123",
		AccessKeySecret: "access_key_secret_123",
		Tenant: "tenantId_123",
		Group: "group_123",
	}

	v, err := c.buildRetrieveUrl(cf)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	req, cancel, err := c.buildRequest(cf, "POST", v, "abc", 10)
	assert.Nil(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, cancel)
	assert.Equal(t, fmt.Sprintf("%s:%d", cf.Host, cf.Port), req.Host)
	assert.Equal(t, "POST", req.Method)
	assert.NotNil(t, req.Header.Get("Spas-Signature"))
	assert.NotEmpty(t, req.Header.Get("Spas-Signature"))
	assert.Equal(t, "10", req.Header.Get("longpullingtimeout"))
}

func TestBuildListenRequest(t *testing.T) {
	c := New()
	assert.NotNil(t, c)

	cf := &Config{
		Host: "123.4.56.78",
		Port: 8888,
		AccessKeyId: "access_key_id_123",
		AccessKeySecret: "access_key_secret_123",
		Tenant: "tenantId_123",
		Group: "group_123",
	}

	req, cancel, err := c.buildListenRequest(cf)
	assert.Nil(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, cancel)
	assert.Equal(t, fmt.Sprintf("%s:%d", cf.Host, cf.Port), req.Host)
	assert.Equal(t, "POST", req.Method)
	assert.NotNil(t, req.Header.Get("Spas-Signature"))
	assert.NotEmpty(t, req.Header.Get("Spas-Signature"))
	assert.Equal(t, "30000", req.Header.Get("longpullingtimeout"))
}

func TestBuildRetrieveRequest(t *testing.T) {
	c := New()
	assert.NotNil(t, c)

	cf := &Config{
		Host: "123.4.56.78",
		Port: 8888,
		AccessKeyId: "access_key_id_123",
		AccessKeySecret: "access_key_secret_123",
		Tenant: "tenantId_123",
		Group: "group_123",
		DataId: "dataId123",
	}

	req, cancel, err := c.buildRetrieveRequest(cf)
	assert.Nil(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, cancel)
	assert.Equal(t, fmt.Sprintf("%s:%d", cf.Host, cf.Port), req.Host)
	assert.Equal(t, "GET", req.Method)
	assert.NotNil(t, req.Header.Get("Spas-Signature"))
	assert.NotEmpty(t, req.Header.Get("Spas-Signature"))
}