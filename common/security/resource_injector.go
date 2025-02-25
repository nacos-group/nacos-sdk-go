package security

import (
	"fmt"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
)

type ResourceInjector interface {
	doInject(resource RequestResource, ramContext RamContext, param map[string]string)
}

const (
	CONFIG_AK_FILED          string = "Spas-AccessKey"
	NAMING_AK_FILED          string = "ak"
	SECURITY_TOKEN_HEADER    string = "Spas-SecurityToken"
	SIGNATURE_VERSION_HEADER string = "signatureVersion"
	SIGNATURE_VERSION_V4     string = "v4"
	SERVICE_INFO_SPLITER     string = "@@"
	TIMESTAMP_HEADER         string = "Timestamp"
	SIGNATURE_HEADER         string = "Spas-Signature"
)

type NamingResourceInjector struct {
}

func (n *NamingResourceInjector) doInject(resource RequestResource, ramContext RamContext, param map[string]string) {
	param[NAMING_AK_FILED] = ramContext.AccessKey
	if ramContext.EphemeralAccessKeyId {
		param[SECURITY_TOKEN_HEADER] = ramContext.SecurityToken
	}
	secretKey := trySignatureWithV4(ramContext, param)
	signatures := n.calculateSignature(resource, secretKey, ramContext)
	for k, v := range signatures {
		param[k] = v
	}
}

func (n *NamingResourceInjector) calculateSignature(resource RequestResource, secretKey string, ramContext RamContext) map[string]string {
	var result = make(map[string]string, 4)
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
	param[CONFIG_AK_FILED] = ramContext.AccessKey
	if ramContext.EphemeralAccessKeyId {
		param[SECURITY_TOKEN_HEADER] = ramContext.SecurityToken
	}
	secretKey := trySignatureWithV4(ramContext, param)
	signatures := c.calculateSignature(resource, secretKey, ramContext)
	for k, v := range signatures {
		param[k] = v
	}
}

func (c *ConfigResourceInjector) calculateSignature(resource RequestResource, secretKey string, ramContext RamContext) map[string]string {
	var result = make(map[string]string, 4)
	resourceName := c.getResourceName(resource)
	signHeaders := c.getSignHeaders(resourceName, secretKey)
	for k, v := range signHeaders {
		result[k] = v
	}
	return result
}

func (c *ConfigResourceInjector) getResourceName(resource RequestResource) string {
	if resource.namespace != "" {
		return resource.namespace + "+" + resource.group
	} else {
		return resource.group
	}
}
func (c *ConfigResourceInjector) getSignHeaders(resource, secretKey string) map[string]string {
	header := make(map[string]string, 4)
	timeStamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	header[TIMESTAMP_HEADER] = timeStamp
	if secretKey != "" {
		var signature string
		if strings.TrimSpace(resource) == "" {
			signature = signWithHmacSha1Encrypt(timeStamp, secretKey)
		} else {
			signature = signWithHmacSha1Encrypt(resource+"+"+timeStamp, secretKey)
		}
		header[SIGNATURE_HEADER] = signature
	}
	return header
}

func trySignatureWithV4(ramContext RamContext, param map[string]string) string {
	if ramContext.SignatureRegionId == "" {
		return ramContext.SecretKey
	}
	signatureV4, err := finalSigningKeyStringWithDefaultInfo(ramContext.SecretKey, ramContext.SignatureRegionId)
	if err != nil {
		logger.Errorf("get v4 signatrue error: %v", err)
		return ramContext.SecretKey
	}
	param[SIGNATURE_VERSION_HEADER] = SIGNATURE_VERSION_V4
	return signatureV4
}
