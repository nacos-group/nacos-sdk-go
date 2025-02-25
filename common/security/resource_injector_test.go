package security

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_NamingResourceInjector_doInject(t *testing.T) {
	namingResourceInjector := NamingResourceInjector{}
	resource := BuildNamingResource("testNamespace", "testGroup", "testServiceName")
	t.Run("test_doInject_v4_sts", func(t *testing.T) {
		ramContext := RamContext{
			AccessKey:            "testAccessKey",
			SecretKey:            "testSecretKey",
			SecurityToken:        "testSecurityToken",
			EphemeralAccessKeyId: true,
			SignatureRegionId:    "testSignatureRegionId",
		}
		param := map[string]string{}
		namingResourceInjector.doInject(resource, ramContext, param)
		assert.Equal(t, param[NAMING_AK_FILED], ramContext.AccessKey)
		assert.Equal(t, param[SECURITY_TOKEN_HEADER], ramContext.SecurityToken)
		assert.Equal(t, param[SIGNATURE_VERSION_HEADER], SIGNATURE_VERSION_V4)
		assert.NotEmpty(t, param["signature"])
	})

	t.Run("test_doInject", func(t *testing.T) {
		ramContext := RamContext{
			AccessKey: "testAccessKey",
			SecretKey: "testSecretKey",
		}
		param := map[string]string{}
		namingResourceInjector.doInject(resource, ramContext, param)
		assert.Equal(t, param[NAMING_AK_FILED], ramContext.AccessKey)
		assert.Empty(t, param[SECURITY_TOKEN_HEADER])
		assert.Empty(t, param[SIGNATURE_VERSION_HEADER])
		assert.NotEmpty(t, param["signature"])
	})
}

func Test_NamingResourceInjector_getGroupedServiceName(t *testing.T) {
	namingResourceInjector := NamingResourceInjector{}
	t.Run("test_getGroupedServiceName", func(t *testing.T) {
		resource := BuildNamingResource("testNamespace", "testGroup", "testServiceName")
		assert.Equal(t, namingResourceInjector.getGroupedServiceName(resource), "testGroup@@testServiceName")
	})
	t.Run("test_getGroupedServiceName_without_group", func(t *testing.T) {
		resource := BuildNamingResource("testNamespace", "", "testServiceName")
		assert.Equal(t, namingResourceInjector.getGroupedServiceName(resource), "testServiceName")
	})
}

func Test_ConfigResourceInjector_doInject(t *testing.T) {
	configResourceInjector := ConfigResourceInjector{}
	resource := BuildConfigResource("testTenant", "testGroup", "testDataId")
	t.Run("test_doInject_v4_sts", func(t *testing.T) {
		ramContext := RamContext{
			AccessKey:            "testAccessKey",
			SecretKey:            "testSecretKey",
			SecurityToken:        "testSecurityToken",
			EphemeralAccessKeyId: true,
			SignatureRegionId:    "testSignatureRegionId",
		}
		param := map[string]string{}
		configResourceInjector.doInject(resource, ramContext, param)
		assert.Equal(t, param[CONFIG_AK_FILED], ramContext.AccessKey)
		assert.Equal(t, param[SECURITY_TOKEN_HEADER], ramContext.SecurityToken)
		assert.Equal(t, param[SIGNATURE_VERSION_HEADER], SIGNATURE_VERSION_V4)
		assert.NotEmpty(t, param[SIGNATURE_HEADER])
		assert.NotEmpty(t, param[TIMESTAMP_HEADER])
	})

	t.Run("test_doInject", func(t *testing.T) {
		ramContext := RamContext{
			AccessKey: "testAccessKey",
			SecretKey: "testSecretKey",
		}
		param := map[string]string{}
		configResourceInjector.doInject(resource, ramContext, param)
		assert.Equal(t, param[CONFIG_AK_FILED], ramContext.AccessKey)
		assert.Empty(t, param[SECURITY_TOKEN_HEADER])
		assert.Empty(t, param[SIGNATURE_VERSION_HEADER])
		assert.NotEmpty(t, param[SIGNATURE_HEADER])
		assert.NotEmpty(t, param[TIMESTAMP_HEADER])
	})
}

func Test_ConfigResourceInjector_getResourceName(t *testing.T) {
	configResourceInjector := ConfigResourceInjector{}
	t.Run("test_getGroupedServiceName", func(t *testing.T) {
		resource := BuildConfigResource("testTenant", "testGroup", "testDataId")
		assert.Equal(t, configResourceInjector.getResourceName(resource), "testTenant+testGroup")
	})
	t.Run("test_getGroupedServiceName_without_group", func(t *testing.T) {
		resource := BuildConfigResource("testTenant", "", "testDataId")
		assert.Equal(t, configResourceInjector.getResourceName(resource), "testTenant+")
	})
}
