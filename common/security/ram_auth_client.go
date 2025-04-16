package security

import (
	"github.com/dbsyk/nacos-sdk-go/v2/common/constant"
	"github.com/pkg/errors"
)

type RamContext struct {
	SignatureRegionId    string
	AccessKey            string
	SecretKey            string
	SecurityToken        string
	EphemeralAccessKeyId bool
}

type RamAuthClient struct {
	clientConfig           constant.ClientConfig
	ramCredentialProviders []RamCredentialProvider
	resourceInjector       map[string]ResourceInjector
	matchedProvider        RamCredentialProvider
}

func NewRamAuthClient(clientCfg constant.ClientConfig) *RamAuthClient {
	var providers = []RamCredentialProvider{
		&RamRoleArnCredentialProvider{
			clientConfig: clientCfg,
		},
		&EcsRamRoleCredentialProvider{
			clientConfig: clientCfg,
		},
		&OIDCRoleArnCredentialProvider{
			clientConfig: clientCfg,
		},
		&CredentialsURICredentialProvider{
			clientConfig: clientCfg,
		},
		&AutoRotateCredentialProvider{
			clientConfig: clientCfg,
		},
		&StsTokenCredentialProvider{
			clientConfig: clientCfg,
		},
		&AccessKeyCredentialProvider{
			clientConfig: clientCfg,
		},
	}
	injectors := map[string]ResourceInjector{
		REQUEST_TYPE_NAMING: &NamingResourceInjector{},
		REQUEST_TYPE_CONFIG: &ConfigResourceInjector{},
	}
	return &RamAuthClient{
		clientConfig:           clientCfg,
		ramCredentialProviders: providers,
		resourceInjector:       injectors,
	}
}

func NewRamAuthClientWithProvider(clientCfg constant.ClientConfig, ramCredentialProvider RamCredentialProvider) *RamAuthClient {
	ramAuthClient := NewRamAuthClient(clientCfg)
	if ramCredentialProvider != nil {
		ramAuthClient.ramCredentialProviders = append(ramAuthClient.ramCredentialProviders, ramCredentialProvider)
	}

	return ramAuthClient
}

func (rac *RamAuthClient) Login() (bool, error) {
	for _, provider := range rac.ramCredentialProviders {
		if provider.MatchProvider() {
			rac.matchedProvider = provider
			break
		}
	}

	if rac.matchedProvider == nil {
		return false, errors.Errorf("no matched provider")
	}
	err := rac.matchedProvider.Init()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (rac *RamAuthClient) GetSecurityInfo(resource RequestResource) map[string]string {
	var securityInfo = make(map[string]string, 4)
	if rac.matchedProvider == nil {
		return securityInfo
	}
	ramContext := rac.matchedProvider.GetCredentialsForNacosClient()
	rac.resourceInjector[resource.requestType].doInject(resource, ramContext, securityInfo)
	return securityInfo
}

func (rac *RamAuthClient) UpdateServerList(serverList []constant.ServerConfig) {
	return
}
