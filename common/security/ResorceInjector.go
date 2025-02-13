package security

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"maps"
	"strings"
	"time"
)

type ResourceInjector interface {
	doInject(resource RequestResource, ramContext RamContext, param map[string]string)
}

const (
	NAMING_AK_FILED          string = "ak"
	SECURITY_TOKEN_HEADER    string = "Spas-SecurityToken"
	SIGNATURE_VERSION_HEADER string = "signatureVersion"
	SIGNATURE_VERSION_V4     string = "v4"
	SERVICE_INFO_SPLITER     string = "@@"
)

type NamingResourceInjector struct {
}

func (n *NamingResourceInjector) doInject(resource RequestResource, ramContext RamContext, param map[string]string) {
	param["ak"] = ramContext.AccessKey
	if ramContext.EphemeralAccessKeyId {
		param[SECURITY_TOKEN_HEADER] = ramContext.SecurityToken
	}
	secretKey := trySignatureWithV4(ramContext, param)
	signatures := n.calculateSignature(resource, secretKey, ramContext)
	maps.Copy(param, signatures)
}

func (n *NamingResourceInjector) calculateSignature(resource RequestResource, secretKey string, ramContext RamContext) map[string]string {
	var result = make(map[string]string)
	signData := n.getSignData(n.getGroupedServiceName(resource))
	signature, err := Sign(signData, secretKey)
	if err != nil {
		logger.Errorf("get v4 signatrue error: %v", err)
		return result
	}
	result["signature"] = signature
	result["data"] = signData
	return result
}

func (n *NamingResourceInjector) getGroupedServiceName(resource RequestResource) string {
	if strings.Contains(resource.resource, SERVICE_INFO_SPLITER) || resource.group == "" {
		return resource.resource
	}
	return resource.group + SERVICE_INFO_SPLITER + resource.resource
}

func (n *NamingResourceInjector) getSignData(serviceName string) string {
	if serviceName != "" {
		return fmt.Sprintf("%d%s%s", time.Now().UnixMilli(), SERVICE_INFO_SPLITER, serviceName)
	}
	return fmt.Sprintf("%d", time.Now().UnixMilli())
}

type ConfigResourceInjector struct {
}

func (c *ConfigResourceInjector) doInject(resource RequestResource, ramContext RamContext, param map[string]string) {
	param["Spas-AccessKey"] = ramContext.AccessKey
	if ramContext.EphemeralAccessKeyId {
		param[SECURITY_TOKEN_HEADER] = ramContext.SecurityToken
	}
	secretKey := trySignatureWithV4(ramContext, param)
	signatures := c.calculateSignature(resource, secretKey, ramContext)
	maps.Copy(param, signatures)
}

func (c *ConfigResourceInjector) calculateSignature(resource RequestResource, secretKey string, ramContext RamContext) map[string]string {
	var result = make(map[string]string)
	resourceName := c.getResourceName(resource)
	signHeaders := getSignHeaders(resourceName, secretKey)
	maps.Copy(result, signHeaders)
	return result
}

func (c *ConfigResourceInjector) getResourceName(resource RequestResource) string {
	if resource.namespace != "" && resource.group != "" {
		return resource.namespace + "+" + resource.group
	}
	if resource.group != "" {
		return resource.group
	}
	if resource.namespace != "" {
		return resource.namespace
	}
	return ""
}

func trySignatureWithV4(ramContext RamContext, param map[string]string) string {
	if ramContext.SignatrueRegionId == "" {
		return ramContext.SecretKey
	}
	signatureV4, err := finalSigningKeyStringWithDefaultInfo(ramContext.SecretKey, ramContext.SignatrueRegionId)
	if err != nil {
		logger.Errorf("get v4 signatrue error: %v", err)
		return ramContext.SecretKey
	}
	param[SIGNATURE_VERSION_HEADER] = SIGNATURE_VERSION_V4
	return signatureV4
}
