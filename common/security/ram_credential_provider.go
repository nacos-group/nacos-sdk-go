package security

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/aliyun/aliyun-secretsmanager-client-go/sdk"
	"github.com/aliyun/credentials-go/credentials"
	"github.com/dbsyk/nacos-sdk-go/v2/common/constant"
)

const (
	ENV_PREFIX                  string = "ALIBABA_CLOUD_"
	ACCESS_KEY_ID_KEY           string = ENV_PREFIX + "ACCESS_KEY_ID"
	ACCESS_KEY_SECRET_KEY       string = ENV_PREFIX + "ACCESS_KEY_SECRET"
	SECURITY_TOKEN_KEY          string = ENV_PREFIX + "SECURITY_TOKEN"
	SIGNATURE_REGION_ID_KEY     string = ENV_PREFIX + "SIGNATURE_REGION_ID"
	RAM_ROLE_NAME_KEY           string = ENV_PREFIX + "RAM_ROLE_NAME"
	ROLE_ARN_KEY                string = ENV_PREFIX + "ROLE_ARN"
	ROLE_SESSION_NAME_KEY       string = ENV_PREFIX + "ROLE_SESSION_NAME"
	ROLE_SESSION_EXPIRATION_KEY string = ENV_PREFIX + "ROLE_SESSION_EXPIRATION"
	POLICY_KEY                  string = ENV_PREFIX + "POLICY"
	OIDC_PROVIDER_ARN_KEY       string = ENV_PREFIX + "OIDC_PROVIDER_ARN"
	OIDC_TOKEN_FILE_KEY         string = ENV_PREFIX + "OIDC_TOKEN_FILE"
	CREDENTIALS_URI_KEY         string = ENV_PREFIX + "CREDENTIALS_URI"
	SECRET_NAME_KEY             string = ENV_PREFIX + "SECRET_NAME"
)

func GetNacosProperties(property string, envKey string) string {
	if property != "" {
		return property
	} else {
		return os.Getenv(envKey)
	}
}

type RamCredentialProvider interface {
	MatchProvider() bool
	Init() error
	GetCredentialsForNacosClient() RamContext
}

type AccessKeyCredentialProvider struct {
	clientConfig      constant.ClientConfig
	accessKey         string
	secretKey         string
	signatureRegionId string
}

func (provider *AccessKeyCredentialProvider) MatchProvider() bool {
	accessKey := GetNacosProperties(provider.clientConfig.AccessKey, ACCESS_KEY_ID_KEY)
	secretKey := GetNacosProperties(provider.clientConfig.SecretKey, ACCESS_KEY_SECRET_KEY)
	return accessKey != "" && secretKey != ""
}

func (provider *AccessKeyCredentialProvider) Init() error {
	provider.accessKey = GetNacosProperties(provider.clientConfig.AccessKey, ACCESS_KEY_ID_KEY)
	provider.secretKey = GetNacosProperties(provider.clientConfig.SecretKey, ACCESS_KEY_SECRET_KEY)
	if provider.clientConfig.RamConfig != nil {
		provider.signatureRegionId = GetNacosProperties(provider.clientConfig.RamConfig.SignatureRegionId, SIGNATURE_REGION_ID_KEY)
	} else {
		provider.signatureRegionId = ""
	}
	return nil
}

func (provider *AccessKeyCredentialProvider) GetCredentialsForNacosClient() RamContext {
	ramContext := RamContext{
		AccessKey:         provider.accessKey,
		SecretKey:         provider.secretKey,
		SignatureRegionId: provider.signatureRegionId,
	}
	return ramContext
}

type AutoRotateCredentialProvider struct {
	clientConfig             constant.ClientConfig
	secretManagerCacheClient *sdk.SecretManagerCacheClient
	secretName               string
	signatureRegionId        string
}

func (provider *AutoRotateCredentialProvider) MatchProvider() bool {
	if provider.clientConfig.RamConfig == nil {
		return false
	}
	secretName := GetNacosProperties(provider.clientConfig.RamConfig.SecretName, SECRET_NAME_KEY)
	return secretName != ""
}

func (provider *AutoRotateCredentialProvider) Init() error {
	secretName := GetNacosProperties(provider.clientConfig.RamConfig.SecretName, SECRET_NAME_KEY)
	client, err := sdk.NewClient()
	if err != nil {
		return err
	}
	provider.secretManagerCacheClient = client
	provider.secretName = secretName
	provider.signatureRegionId = GetNacosProperties(provider.clientConfig.RamConfig.SignatureRegionId, SIGNATURE_REGION_ID_KEY)
	return nil
}

func (provider *AutoRotateCredentialProvider) GetCredentialsForNacosClient() RamContext {
	if provider.secretManagerCacheClient == nil || provider.secretName == "" {
		return RamContext{}
	}
	secretInfo, err := provider.secretManagerCacheClient.GetSecretInfo(provider.secretName)
	if err != nil {
		return RamContext{}
	}
	var m map[string]string
	err = json.Unmarshal([]byte(secretInfo.SecretValue), &m)
	if err != nil {
		return RamContext{}
	}
	accessKeyId := m["AccessKeyId"]
	accessKeySecret := m["AccessKeySecret"]
	ramContext := RamContext{
		AccessKey:            accessKeyId,
		SecretKey:            accessKeySecret,
		SignatureRegionId:    provider.signatureRegionId,
		EphemeralAccessKeyId: false,
	}
	return ramContext
}

type StsTokenCredentialProvider struct {
	clientConfig      constant.ClientConfig
	accessKey         string
	secretKey         string
	securityToken     string
	signatureRegionId string
}

func (provider *StsTokenCredentialProvider) MatchProvider() bool {
	accessKey := GetNacosProperties(provider.clientConfig.AccessKey, ACCESS_KEY_ID_KEY)
	secretKey := GetNacosProperties(provider.clientConfig.SecretKey, ACCESS_KEY_SECRET_KEY)
	if provider.clientConfig.RamConfig == nil {
		return false
	}
	stsToken := GetNacosProperties(provider.clientConfig.RamConfig.SecurityToken, SECURITY_TOKEN_KEY)
	return accessKey != "" && secretKey != "" && stsToken != ""
}

func (provider *StsTokenCredentialProvider) Init() error {
	provider.accessKey = GetNacosProperties(provider.clientConfig.AccessKey, ACCESS_KEY_ID_KEY)
	provider.secretKey = GetNacosProperties(provider.clientConfig.SecretKey, ACCESS_KEY_SECRET_KEY)
	provider.securityToken = GetNacosProperties(provider.clientConfig.RamConfig.SecurityToken, SECURITY_TOKEN_KEY)
	provider.signatureRegionId = GetNacosProperties(provider.clientConfig.RamConfig.SignatureRegionId, SIGNATURE_REGION_ID_KEY)
	return nil
}

func (provider *StsTokenCredentialProvider) GetCredentialsForNacosClient() RamContext {
	ramContext := RamContext{
		AccessKey:            provider.accessKey,
		SecretKey:            provider.secretKey,
		SecurityToken:        provider.securityToken,
		SignatureRegionId:    provider.signatureRegionId,
		EphemeralAccessKeyId: true,
	}
	return ramContext
}

type EcsRamRoleCredentialProvider struct {
	clientConfig      constant.ClientConfig
	credentialClient  credentials.Credential
	signatureRegionId string
}

func (provider *EcsRamRoleCredentialProvider) MatchProvider() bool {
	if provider.clientConfig.RamConfig == nil {
		return false
	}
	ramRoleName := GetNacosProperties(provider.clientConfig.RamConfig.RamRoleName, RAM_ROLE_NAME_KEY)
	return ramRoleName != ""
}

func (provider *EcsRamRoleCredentialProvider) Init() error {
	ramRoleName := GetNacosProperties(provider.clientConfig.RamConfig.RamRoleName, RAM_ROLE_NAME_KEY)
	credentialsConfig := new(credentials.Config).SetType("ecs_ram_role").SetRoleName(ramRoleName)
	credentialClient, err := credentials.NewCredential(credentialsConfig)
	if err != nil {
		return err
	}
	provider.credentialClient = credentialClient
	provider.signatureRegionId = GetNacosProperties(provider.clientConfig.RamConfig.SignatureRegionId, SIGNATURE_REGION_ID_KEY)
	return nil
}

func (provider *EcsRamRoleCredentialProvider) GetCredentialsForNacosClient() RamContext {
	if provider.credentialClient == nil {
		return RamContext{}
	}
	credential, err := provider.credentialClient.GetCredential()
	if err != nil {
		return RamContext{}
	}
	ramContext := RamContext{
		AccessKey:            *credential.AccessKeyId,
		SecretKey:            *credential.AccessKeySecret,
		SecurityToken:        *credential.SecurityToken,
		SignatureRegionId:    provider.signatureRegionId,
		EphemeralAccessKeyId: true,
	}
	return ramContext
}

type RamRoleArnCredentialProvider struct {
	clientConfig      constant.ClientConfig
	credentialClient  credentials.Credential
	signatureRegionId string
}

func (provider *RamRoleArnCredentialProvider) MatchProvider() bool {
	if provider.clientConfig.RamConfig == nil {
		return false
	}
	accessKey := GetNacosProperties(provider.clientConfig.AccessKey, ACCESS_KEY_ID_KEY)
	secretKey := GetNacosProperties(provider.clientConfig.SecretKey, ACCESS_KEY_SECRET_KEY)
	roleArn := GetNacosProperties(provider.clientConfig.RamConfig.RoleArn, ROLE_ARN_KEY)
	roleSessionName := GetNacosProperties(provider.clientConfig.RamConfig.RoleSessionName, ROLE_SESSION_NAME_KEY)
	oidcProviderArn := GetNacosProperties(provider.clientConfig.RamConfig.OIDCProviderArn, OIDC_PROVIDER_ARN_KEY)
	return accessKey == "" && secretKey == "" && roleArn != "" && roleSessionName != "" && oidcProviderArn == ""
}

func (provider *RamRoleArnCredentialProvider) Init() error {
	accessKey := GetNacosProperties(provider.clientConfig.AccessKey, ACCESS_KEY_ID_KEY)
	secretKey := GetNacosProperties(provider.clientConfig.SecretKey, ACCESS_KEY_SECRET_KEY)
	roleArn := GetNacosProperties(provider.clientConfig.RamConfig.RoleArn, ROLE_ARN_KEY)
	roleSessionName := GetNacosProperties(provider.clientConfig.RamConfig.RoleSessionName, ROLE_SESSION_NAME_KEY)
	credentialsConfig := new(credentials.Config).SetType("ram_role_arn").
		SetAccessKeyId(accessKey).SetAccessKeySecret(secretKey).
		SetRoleArn(roleArn).SetRoleSessionName(roleSessionName)
	if roleSessionExpiration := GetNacosProperties(strconv.Itoa(provider.clientConfig.RamConfig.RoleSessionExpiration), ROLE_SESSION_EXPIRATION_KEY); roleSessionExpiration != "" {
		if roleSessionExpirationTime, err := strconv.Atoi(roleSessionExpiration); err == nil {
			if roleSessionExpirationTime == 0 {
				roleSessionExpirationTime = 3600
			}
			credentialsConfig.SetRoleSessionExpiration(roleSessionExpirationTime)
		}
	}
	policy := GetNacosProperties(provider.clientConfig.RamConfig.Policy, POLICY_KEY)
	if policy != "" {
		credentialsConfig.SetPolicy(policy)
	}
	credentialClient, err := credentials.NewCredential(credentialsConfig)
	if err != nil {
		return err
	}
	provider.credentialClient = credentialClient
	provider.signatureRegionId = GetNacosProperties(provider.clientConfig.RamConfig.SignatureRegionId, SIGNATURE_REGION_ID_KEY)
	return nil
}

func (provider *RamRoleArnCredentialProvider) GetCredentialsForNacosClient() RamContext {
	if provider.credentialClient == nil {
		return RamContext{}
	}
	credential, err := provider.credentialClient.GetCredential()
	if err != nil {
		return RamContext{}
	}
	return RamContext{
		AccessKey:            *credential.AccessKeyId,
		SecretKey:            *credential.AccessKeySecret,
		SecurityToken:        *credential.SecurityToken,
		SignatureRegionId:    provider.signatureRegionId,
		EphemeralAccessKeyId: true,
	}
}

type OIDCRoleArnCredentialProvider struct {
	clientConfig      constant.ClientConfig
	credentialClient  credentials.Credential
	signatureRegionId string
}

func (provider *OIDCRoleArnCredentialProvider) MatchProvider() bool {
	if provider.clientConfig.RamConfig == nil {
		return false
	}
	roleArn := GetNacosProperties(provider.clientConfig.RamConfig.RoleArn, ROLE_ARN_KEY)
	roleSessionName := GetNacosProperties(provider.clientConfig.RamConfig.RoleSessionName, ROLE_SESSION_NAME_KEY)
	oidcProviderArn := GetNacosProperties(provider.clientConfig.RamConfig.OIDCProviderArn, OIDC_PROVIDER_ARN_KEY)
	oidcTokenFile := GetNacosProperties(provider.clientConfig.RamConfig.OIDCTokenFilePath, OIDC_TOKEN_FILE_KEY)
	return roleArn != "" && roleSessionName != "" && oidcProviderArn != "" && oidcTokenFile != ""
}

func (provider *OIDCRoleArnCredentialProvider) Init() error {
	ramRoleArn := GetNacosProperties(provider.clientConfig.RamConfig.RoleArn, ROLE_ARN_KEY)
	roleSessionName := GetNacosProperties(provider.clientConfig.RamConfig.RoleSessionName, ROLE_SESSION_NAME_KEY)
	oidcProviderArn := GetNacosProperties(provider.clientConfig.RamConfig.OIDCProviderArn, OIDC_PROVIDER_ARN_KEY)
	oidcTokenFilePath := GetNacosProperties(provider.clientConfig.RamConfig.OIDCTokenFilePath, OIDC_TOKEN_FILE_KEY)
	credentialsConfig := new(credentials.Config).SetType("oidc_role_arn").
		SetRoleArn(ramRoleArn).SetRoleSessionName(roleSessionName).
		SetOIDCProviderArn(oidcProviderArn).SetOIDCTokenFilePath(oidcTokenFilePath)
	if roleSessionExpiration := GetNacosProperties(strconv.Itoa(provider.clientConfig.RamConfig.RoleSessionExpiration), ROLE_SESSION_EXPIRATION_KEY); roleSessionExpiration != "" {
		if roleSessionExpirationTime, err := strconv.Atoi(roleSessionExpiration); err == nil {
			if roleSessionExpirationTime == 0 {
				roleSessionExpirationTime = 3600
			}
			credentialsConfig.SetRoleSessionExpiration(roleSessionExpirationTime)
		}
	}
	policy := GetNacosProperties(provider.clientConfig.RamConfig.Policy, POLICY_KEY)
	if policy != "" {
		credentialsConfig.SetPolicy(policy)
	}
	credentialClient, err := credentials.NewCredential(credentialsConfig)
	if err != nil {
		return err
	}
	provider.credentialClient = credentialClient
	provider.signatureRegionId = GetNacosProperties(provider.clientConfig.RamConfig.SignatureRegionId, SIGNATURE_REGION_ID_KEY)
	return nil
}

func (provider *OIDCRoleArnCredentialProvider) GetCredentialsForNacosClient() RamContext {
	if provider.credentialClient == nil {
		return RamContext{}
	}
	credential, err := provider.credentialClient.GetCredential()
	if err != nil {
		return RamContext{}
	}
	return RamContext{
		AccessKey:            *credential.AccessKeyId,
		SecretKey:            *credential.AccessKeySecret,
		SecurityToken:        *credential.SecurityToken,
		SignatureRegionId:    provider.signatureRegionId,
		EphemeralAccessKeyId: true,
	}
}

type CredentialsURICredentialProvider struct {
	clientConfig      constant.ClientConfig
	credentialClient  credentials.Credential
	signatureRegionId string
}

func (provider *CredentialsURICredentialProvider) MatchProvider() bool {
	if provider.clientConfig.RamConfig == nil {
		return false
	}
	credentialsURI := GetNacosProperties(provider.clientConfig.RamConfig.CredentialsURI, CREDENTIALS_URI_KEY)
	return credentialsURI != ""
}

func (provider *CredentialsURICredentialProvider) Init() error {
	credentialsURI := GetNacosProperties(provider.clientConfig.RamConfig.CredentialsURI, CREDENTIALS_URI_KEY)
	credentialsConfig := new(credentials.Config).SetType("credentials_uri").SetURLCredential(credentialsURI)
	credentialClient, err := credentials.NewCredential(credentialsConfig)
	if err != nil {
		return err
	}
	provider.credentialClient = credentialClient
	provider.signatureRegionId = GetNacosProperties(provider.clientConfig.RamConfig.SignatureRegionId, SIGNATURE_REGION_ID_KEY)
	return nil
}

func (provider *CredentialsURICredentialProvider) GetCredentialsForNacosClient() RamContext {
	if provider.credentialClient == nil {
		return RamContext{}
	}
	if provider.credentialClient == nil {
		return RamContext{}
	}
	credential, err := provider.credentialClient.GetCredential()
	if err != nil {
		return RamContext{}
	}
	return RamContext{
		AccessKey:            *credential.AccessKeyId,
		SecretKey:            *credential.AccessKeySecret,
		SecurityToken:        *credential.SecurityToken,
		SignatureRegionId:    provider.signatureRegionId,
		EphemeralAccessKeyId: true,
	}
}
