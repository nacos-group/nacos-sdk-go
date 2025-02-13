package security

import (
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/pkg/errors"
)

type RamContext struct {
	SignatrueRegionId    string
	AccessKey            string
	SecretKey            string
	RamRoleName          string
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
		&StsTokenCredentialProvider{
			clientConfig: clientCfg,
		},
		&AccessKeyCredentialProvider{
			clientConfig: clientCfg,
		},
	}
	injectors := map[string]ResourceInjector{
		"naming": &NamingResourceInjector{},
		"config": &ConfigResourceInjector{},
	}
	return &RamAuthClient{
		clientConfig:           clientCfg,
		ramCredentialProviders: providers,
		resourceInjector:       injectors,
	}
}

func (rac *RamAuthClient) Login() (bool, error) {
	for _, provider := range rac.ramCredentialProviders {
		if provider.matchProvider() {
			rac.matchedProvider = provider
			break
		}
	}

	if rac.matchedProvider == nil {
		return false, errors.Errorf("no matched provider")
	}
	rac.matchedProvider.init()
	return true, nil
}

func (rac *RamAuthClient) GetSecurityInfo(resource RequestResource) map[string]string {
	var securityInfo = make(map[string]string)
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
