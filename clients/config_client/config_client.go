package config_client

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/nacos-group/nacos-sdk-go/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/common/logger"
	"github.com/nacos-group/nacos-sdk-go/common/nacos_error"
	"github.com/nacos-group/nacos-sdk-go/common/util"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"github.com/nacos-group/nacos-sdk-go/vo"

	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
)

type ConfigClient struct {
	nacos_client.INacosClient
	kmsClient      *kms.Client
	localConfigs   []vo.ConfigParam
	mutex          sync.Mutex
	configProxy    ConfigProxy
	configCacheDir string
}

const perTaskConfigSize = 3000

var currentTaskCount int

var cacheMap cache.ConcurrentMap

type cacheData struct {
	isInitializing    bool
	dataId            string
	group             string
	content           string
	cacheDataListener *cacheDataListener
	md5               string
	appName           string
}

type cacheDataListener struct {
	listener vo.Listener
	lastMd5  string
}

var listenClient *ConfigClient

func init() {
	cacheMap = cache.NewConcurrentMap()
	go listenConfigActuator()
}

func NewConfigClient(nc nacos_client.INacosClient) (ConfigClient, error) {
	config := ConfigClient{}
	config.INacosClient = nc
	clientConfig, err := nc.GetClientConfig()
	if err != nil {
		return config, err
	}
	serverConfig, err := nc.GetServerConfig()
	if err != nil {
		return config, err
	}
	httpAgent, err := nc.GetHttpAgent()
	if err != nil {
		return config, err
	}
	err = logger.InitLog(clientConfig.LogDir)
	if err != nil {
		return config, err
	}
	config.configCacheDir = clientConfig.CacheDir + string(os.PathSeparator) + "config"
	config.configProxy, err = NewConfigProxy(serverConfig, clientConfig, httpAgent)
	if clientConfig.OpenKMS {
		kmsClient, err := kms.NewClientWithAccessKey(clientConfig.RegionId, clientConfig.AccessKey, clientConfig.SecretKey)
		if err != nil {
			return config, err
		}
		config.kmsClient = kmsClient
	}

	return config, err
}

func (client *ConfigClient) sync() (clientConfig constant.ClientConfig,
	serverConfigs []constant.ServerConfig, agent http_agent.IHttpAgent, err error) {
	clientConfig, err = client.GetClientConfig()
	if err != nil {
		log.Println(err, ";do you call client.SetClientConfig()?")
	}
	if err == nil {
		serverConfigs, err = client.GetServerConfig()
		if err != nil {
			log.Println(err, ";do you call client.SetServerConfig()?")
		}
	}
	if err == nil {
		agent, err = client.GetHttpAgent()
		if err != nil {
			log.Println(err, ";do you call client.SetHttpAgent()?")
		}
	}
	return
}

func (client *ConfigClient) GetConfig(param vo.ConfigParam) (content string, err error) {
	content, err = client.getConfigInner(param)

	if err != nil {
		return "", err
	}

	return client.decrypt(param.DataId, content)
}

func (client *ConfigClient) decrypt(dataId, content string) (string, error) {
	if strings.HasPrefix(dataId, "cipher-") && client.kmsClient != nil {
		request := kms.CreateDecryptRequest()
		request.Method = "POST"
		request.Scheme = "https"
		request.AcceptFormat = "json"
		request.CiphertextBlob = content
		response, err := client.kmsClient.Decrypt(request)
		if err != nil {
			return "", errors.New("kms decrypt failed")
		}
		content = response.Plaintext
	}

	return content, nil
}

func (client *ConfigClient) getConfigInner(param vo.ConfigParam) (content string, err error) {
	if len(param.DataId) <= 0 {
		err = errors.New("[client.GetConfig] param.dataId can not be empty")
	}
	if len(param.Group) <= 0 {
		err = errors.New("[client.GetConfig] param.group can not be empty")
	}
	clientConfig, _ := client.GetClientConfig()
	cacheKey := utils.GetConfigCacheKey(param.DataId, param.Group, clientConfig.NamespaceId)
	content, err = client.configProxy.GetConfigProxy(param, clientConfig.NamespaceId, clientConfig.AccessKey, clientConfig.SecretKey)

	if err != nil {
		log.Printf("[ERROR] get config from server error:%s ", err.Error())
		if _, ok := err.(*nacos_error.NacosError); ok {
			nacosErr := err.(*nacos_error.NacosError)
			if nacosErr.ErrorCode() == "404" {
				cache.WriteConfigToFile(cacheKey, client.configCacheDir, "")
				return "", errors.New("config not found")
			}
			if nacosErr.ErrorCode() == "403" {
				return "", errors.New("get config forbidden")
			}
		}
		content, err = cache.ReadConfigFromFile(cacheKey, client.configCacheDir)
		if err != nil {
			log.Printf("[ERROR] get config from cache  error:%s ", err.Error())
			return "", errors.New("read config from both server and cache fail")
		}

	} else {
		cache.WriteConfigToFile(cacheKey, client.configCacheDir, content)
	}
	return content, nil
}

func (client *ConfigClient) PublishConfig(param vo.ConfigParam) (published bool,
	err error) {
	if len(param.DataId) <= 0 {
		err = errors.New("[client.PublishConfig] param.dataId can not be empty")
	}
	if len(param.Group) <= 0 {
		err = errors.New("[client.PublishConfig] param.group can not be empty")
	}
	if len(param.Content) <= 0 {
		err = errors.New("[client.PublishConfig] param.content can not be empty")
	}
	clientConfig, _ := client.GetClientConfig()
	return client.configProxy.PublishConfigProxy(param, clientConfig.NamespaceId, clientConfig.AccessKey, clientConfig.SecretKey)
}

func (client *ConfigClient) DeleteConfig(param vo.ConfigParam) (deleted bool,
	err error) {
	if len(param.DataId) <= 0 {
		err = errors.New("[client.DeleteConfig] param.dataId can not be empty")
	}
	if len(param.Group) <= 0 {
		err = errors.New("[client.DeleteConfig] param.group can not be empty")
	}

	clientConfig, _ := client.GetClientConfig()
	return client.configProxy.DeleteConfigProxy(param, clientConfig.NamespaceId, clientConfig.AccessKey, clientConfig.SecretKey)
}

func (client *ConfigClient) AddConfigToListen(params []vo.ConfigParam) (err error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	var newParams []vo.ConfigParam
	// 去掉重复添加的
	for _, newParam := range params {
		flag := true
		for _, param := range client.localConfigs {
			if param.Group == newParam.Group && param.DataId == newParam.DataId {
				flag = false
				break
			}
		}
		if flag {
			newParams = append(newParams, newParam)
		}
	}
	client.localConfigs = append(client.localConfigs, newParams...)
	return
}

//Cancel Listen Config
func (client *ConfigClient) CancelListenConfig(param *vo.ConfigParam) error {
	param.ListenCloseChan <- struct{}{}
	log.Printf("Cancel listen config DataId:%s Group:%s", param.DataId, param.Group)
	return nil
}

func (client *ConfigClient) ListenConfig(param vo.ConfigParam) (err error) {
	go func() {
		for {
			select {
			case <-param.ListenCloseChan:
				return
			default:
				clientConfig, _ := client.GetClientConfig()
				client.listenConfigTask(clientConfig, param)
			}

		}
	}()

	return nil
}

func (client *ConfigClient) ListenConfig1(param vo.ConfigParam) (err error) {
	if len(param.DataId) <= 0 {
		log.Fatalf("[client.ListenConfig] DataId can not be empty")
		return
	}
	if len(param.Group) <= 0 {
		log.Fatalf("[client.ListenConfig] Group can not be empty")
		return
	}
	clientConfig, err := client.GetClientConfig()
	if err != nil {
		log.Fatalf("[checkConfigInfo.GetClientConfig] failed.")
		return
	}
	//todo 1: 未读取本地目录，是否有必要？ 2: 同步远程配置 3：监听onChange fun只支持一个
	key := utils.GetConfigCacheKey(param.DataId, param.Group, clientConfig.NamespaceId)
	var cData cacheData
	if mResult, ok := cacheMap.Get(key); ok {
		cData = mResult.(cacheData)
		cData.isInitializing = true
	} else {
		md5Str := util.Md5("")
		cDListener := cacheDataListener{
			listener: param.OnChange,
			lastMd5:  md5Str,
		}
		cData = cacheData{
			isInitializing:    true,
			appName:           param.AppName,
			dataId:            param.DataId,
			group:             param.Group,
			content:           "",
			md5:               md5Str,
			cacheDataListener: &cDListener,
		}
	}
	listenClient = client
	cacheMap.Set(key, cData)
	return nil
}

func listenConfigActuator() {
	t := time.NewTimer(time.Millisecond * 1)
	defer t.Stop()

	for {
		<-t.C
		listenerSize := len(cacheMap.Keys())
		taskCount := int(math.Ceil(float64(listenerSize) / float64(perTaskConfigSize)))
		if taskCount > currentTaskCount {
			for i := 0; i < taskCount; i++ {
				go checkConfigInfo()
			}
			currentTaskCount = taskCount
		}
		// reset
		t.Reset(time.Millisecond * 10)
	}

}

func checkConfigInfo() {
	var listeningConfigs string
	clientConfig, err := listenClient.GetClientConfig()
	if err != nil {
		log.Fatalf("[checkConfigInfo.GetClientConfig] failed.")
		return
	}
	tenant := clientConfig.NamespaceId
	for _, key := range cacheMap.Keys() {
		if value, ok := cacheMap.Get(key); ok {
			cData := value.(cacheData)
			if len(tenant) > 0 {
				listeningConfigs += cData.dataId + constant.SPLIT_CONFIG_INNER + cData.group + constant.SPLIT_CONFIG_INNER +
					cData.md5 + constant.SPLIT_CONFIG_INNER + tenant + constant.SPLIT_CONFIG
			} else {
				listeningConfigs += cData.dataId + constant.SPLIT_CONFIG_INNER + cData.group + constant.SPLIT_CONFIG_INNER +
					cData.md5 + constant.SPLIT_CONFIG
			}
		}
	}
	// http 请求
	params := make(map[string]string)
	params[constant.KEY_LISTEN_CONFIGS] = listeningConfigs
	var changed string
	//todo 有
	changedTmp, err := listenClient.configProxy.ListenConfig(params, tenant, clientConfig.AccessKey, clientConfig.SecretKey)
	if err == nil {
		changed = changedTmp
	} else {
		if _, ok := err.(*nacos_error.NacosError); ok {
			changed = changedTmp
		} else {
			log.Println("[client.ListenConfig] listen config error:", err.Error())
		}
	}
	if strings.ToLower(strings.Trim(changed, " ")) == "" {
		log.Println("[client.ListenConfig] no change")
	} else {
		log.Print("[client.ListenConfig] config changed:" + changed)
		listenClient.updateLocalConfig1(changed)
	}
}

func (client *ConfigClient) updateLocalConfig1(changed string) {
	changedConfigs := strings.Split(changed, "%01")
	for _, config := range changedConfigs {
		attrs := strings.Split(config, "%02")
		if len(attrs) == 2 {
			if value, ok := cacheMap.Get(utils.GetConfigCacheKey(attrs[0], attrs[1], "")); ok {
				cData := value.(cacheData)
				cData.cacheDataListener.listener("", attrs[1], attrs[0], "")
			}
			//todo 对比md5
			content, err := client.getConfigInner(vo.ConfigParam{
				DataId: attrs[0],
				Group:  attrs[1],
			})
			fmt.Println(content, err)
		} else if len(attrs) == 3 {
			content, err := client.getConfigInner(vo.ConfigParam{
				DataId: attrs[0],
				Group:  attrs[1],
			})
			fmt.Println(content, err)
			if value, ok := cacheMap.Get(utils.GetConfigCacheKey(attrs[0], attrs[1], attrs[2])); ok {
				cData := value.(cacheData)
				cData.cacheDataListener.listener(attrs[2], attrs[1], attrs[0], "")
			}
		}
	}
}

func (client *ConfigClient) listenConfigTask(clientConfig constant.ClientConfig, param vo.ConfigParam) {
	var listeningConfigs string
	// 检查&拼接监听参数
	client.mutex.Lock()
	if len(param.DataId) <= 0 {
		log.Fatalf("[client.ListenConfig] DataId can not be empty")
		return
	}
	if len(param.Group) <= 0 {
		log.Fatalf("[client.ListenConfig] Group can not be empty")
		return
	}
	var tenant string
	if len(clientConfig.NamespaceId) > 0 {
		tenant = clientConfig.NamespaceId
	}

	for _, config := range client.localConfigs {
		if config.Group == param.Group && config.DataId == param.DataId {
			param.Content = config.Content
			break
		}
	}

	var md5 string
	if len(param.Content) > 0 {
		md5 = util.Md5(param.Content)
	}
	if len(tenant) > 0 {
		listeningConfigs += param.DataId + constant.SPLIT_CONFIG_INNER + param.Group + constant.SPLIT_CONFIG_INNER +
			md5 + constant.SPLIT_CONFIG_INNER + tenant + constant.SPLIT_CONFIG
	} else {
		listeningConfigs += param.DataId + constant.SPLIT_CONFIG_INNER + param.Group + constant.SPLIT_CONFIG_INNER +
			md5 + constant.SPLIT_CONFIG
	}

	client.mutex.Unlock()
	// http 请求
	params := make(map[string]string)
	params[constant.KEY_LISTEN_CONFIGS] = listeningConfigs
	var changed string
	changedTmp, err := client.configProxy.ListenConfig(params, tenant, clientConfig.AccessKey, clientConfig.SecretKey)
	if err == nil {
		changed = changedTmp
	} else {
		if _, ok := err.(*nacos_error.NacosError); ok {
			changed = changedTmp
		} else {
			log.Println("[client.ListenConfig] listen config error:", err.Error())
		}
	}
	if strings.ToLower(strings.Trim(changed, " ")) == "" {
		log.Println("[client.ListenConfig] no change")
	} else {
		log.Print("[client.ListenConfig] config changed:" + changed)
		client.updateLocalConfig(changed, param)
	}
}

func (client *ConfigClient) updateLocalConfig(changed string, param vo.ConfigParam) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	changedConfigs := strings.Split(changed, "%01")
	for _, config := range changedConfigs {
		attrs := strings.Split(config, "%02")
		if len(attrs) == 2 {
			content, err := client.getConfigInner(vo.ConfigParam{
				DataId: attrs[0],
				Group:  attrs[1],
			})
			if err != nil {
				log.Println("[client.updateLocalConfig] update config failed:", err.Error())
			} else {
				client.putLocalConfig(vo.ConfigParam{
					DataId:  attrs[0],
					Group:   attrs[1],
					Content: content,
				})

				// call listener:
				decrept, _ := client.decrypt(attrs[0], content)
				param.OnChange("", attrs[1], attrs[0], decrept)

			}
		} else if len(attrs) == 3 {
			content, err := client.getConfigInner(vo.ConfigParam{
				DataId: attrs[0],
				Group:  attrs[1],
			})
			if err != nil {
				log.Println("[client.updateLocalConfig] update config failed:", err.Error())
			} else {
				client.putLocalConfig(vo.ConfigParam{
					DataId:  attrs[0],
					Group:   attrs[1],
					Content: content,
				})

				// call listener:
				decrept, _ := client.decrypt(attrs[0], content)
				param.OnChange(attrs[2], attrs[1], attrs[0], decrept)
			}
		}
	}
	log.Println("[client.updateLocalConfig] update config complete")
	log.Println("[client.localConfig] ", client.localConfigs)
}

func (client *ConfigClient) putLocalConfig(config vo.ConfigParam) {
	if len(config.DataId) > 0 && len(config.Group) > 0 {
		exist := false
		for i := 0; i < len(client.localConfigs); i++ {
			if len(client.localConfigs[i].DataId) > 0 && len(client.localConfigs[i].Group) > 0 &&
				config.DataId == client.localConfigs[i].DataId && config.Group == client.localConfigs[i].Group {
				// 本地存在 则更新
				client.localConfigs[i] = config
				exist = true
				break
			}
		}
		if !exist {
			// 本地不存在 放入
			client.localConfigs = append(client.localConfigs, config)
		}
	}
	log.Println("[client.putLocalConfig] putLocalConfig success")
}

func (client *ConfigClient) buildBasePath(serverConfig constant.ServerConfig) (basePath string) {
	basePath = "http://" + serverConfig.IpAddr + ":" +
		strconv.FormatUint(serverConfig.Port, 10) + serverConfig.ContextPath + constant.CONFIG_PATH
	return
}

func (client *ConfigClient) SearchConfig(param vo.SearchConfigParm) (*model.ConfigPage, error) {
	return client.searchConfigInnter(param)
}

func (client *ConfigClient) searchConfigInnter(param vo.SearchConfigParm) (*model.ConfigPage, error) {
	if param.Search != "accurate" && param.Search != "blur" {
		return nil, errors.New("[client.searchConfigInnter] param.search must be accurate or blur")
	}
	if param.PageNo <= 0 {
		param.PageNo = 1
	}
	if param.PageSize <= 0 {
		param.PageSize = 10
	}
	clientConfig, _ := client.GetClientConfig()
	configItems, err := client.configProxy.SearchConfigProxy(param, clientConfig.NamespaceId, clientConfig.AccessKey, clientConfig.SecretKey)
	if err != nil {
		log.Printf("[ERROR] search config from server error:%s ", err.Error())
		if _, ok := err.(*nacos_error.NacosError); ok {
			nacosErr := err.(*nacos_error.NacosError)
			if nacosErr.ErrorCode() == "404" {
				return nil, errors.New("config not found")
			}
			if nacosErr.ErrorCode() == "403" {
				return nil, errors.New("get config forbidden")
			}
		}
		return nil, err
	}
	return configItems, nil
}
