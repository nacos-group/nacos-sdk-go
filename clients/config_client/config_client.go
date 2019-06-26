package config_client

import (
	"errors"
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/common/nacos_error"
	"github.com/nacos-group/nacos-sdk-go/common/util"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ConfigClient struct {
	nacos_client.INacosClient
	localConfigs []vo.ConfigParam
	mutex        sync.Mutex
	listening    bool
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

func (client *ConfigClient) GetConfigContent(dataId string, group string) (content string,
	err error) {
	if len(dataId) <= 0 {
		err = errors.New("[client.GetConfigContent] dataId param can not be empty")
	}
	if err == nil && len(group) <= 0 {
		err = errors.New("[client.GetConfigContent] group param can not be empty")
	}
	if err == nil {
		client.mutex.Lock()
		defer client.mutex.Unlock()
		exist := false
		for _, config := range client.localConfigs {
			if config.Group == group && config.DataId == dataId {
				content = config.Content
				exist = true
				break
			}
		}
		if !exist || len(content) <= 0 {
			content, err = client.GetConfig(vo.ConfigParam{
				DataId: dataId,
				Group:  group,
			})
		}
	}
	return
}

func (client *ConfigClient) GetConfig(param vo.ConfigParam) (content string, err error) {
	if len(param.DataId) <= 0 {
		err = errors.New("[client.GetConfig] param.dataId can not be empty")
	}
	if len(param.Group) <= 0 {
		err = errors.New("[client.GetConfig] param.group can not be empty")
	}
	var clientConfig constant.ClientConfig
	var serverConfigs []constant.ServerConfig
	var agent http_agent.IHttpAgent
	if err == nil {
		clientConfig, serverConfigs, agent, err = client.sync()
	}
	if err == nil {
		params := util.TransformObject2Param(param)
		for _, serverConfig := range serverConfigs {
			path := client.buildBasePath(serverConfig)
			content, err = getConfig(agent, path, clientConfig.TimeoutMs, params)
			if err == nil {
				break
			} else {
				if _, ok := err.(*nacos_error.NacosError); ok {
					break
				} else {
					log.Println("[client.GetConfig] get config failed:", err.Error())
				}
			}
		}
	}
	return
}

func getConfig(agent http_agent.IHttpAgent, path string, timeoutMs uint64,
	params map[string]string) (content string, err error) {
	var response *http.Response
	log.Println("[client.GetConfig] request url :", path, ",params:", params)
	response, err = agent.Get(path, nil, timeoutMs, params)
	if err == nil {
		bytes, errRead := ioutil.ReadAll(response.Body)
		defer response.Body.Close()
		if errRead != nil {
			err = errRead
		} else {
			if response.StatusCode == 200 {
				content = string(bytes)
			} else {
				err = &nacos_error.NacosError{
					ErrMsg: "[client.GetConfig] [" + strconv.Itoa(response.StatusCode) + "]" + string(bytes),
				}
			}
		}
	}
	return
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
	var clientConfig constant.ClientConfig
	var serverConfigs []constant.ServerConfig
	var agent http_agent.IHttpAgent
	if err == nil {
		clientConfig, serverConfigs, agent, err = client.sync()
	}
	if err == nil {
		params := util.TransformObject2Param(param)
		for _, serverConfig := range serverConfigs {
			path := client.buildBasePath(serverConfig)
			published, err = publishConfig(agent, path, clientConfig.TimeoutMs, params)
			if err == nil {
				break
			} else {
				if _, ok := err.(*nacos_error.NacosError); ok {
					break
				} else {
					log.Println("[client.PublishConfig] publish config failed:" + err.Error())
				}
			}
		}
	}
	return
}

func publishConfig(agent http_agent.IHttpAgent, path string,
	timeoutMs uint64, params map[string]string) (published bool, err error) {
	header := map[string][]string{
		"Content-Type": {"application/x-www-form-urlencoded"},
	}
	log.Println("[client.PublishConfig] request url:", path, " ;params:", params, " ;header:", header)
	var response *http.Response
	response, err = agent.Post(path, header, timeoutMs, params)
	if err == nil {
		bytes, errRead := ioutil.ReadAll(response.Body)
		defer response.Body.Close()
		if errRead != nil {
			err = errRead
		} else {
			if response.StatusCode == 200 {
				if strings.ToLower(strings.Trim(string(bytes), " ")) == "true" {
					published = true
				} else {
					published = false
					err = errors.New("[client.PublishConfig] " + string(bytes))
				}
			} else {
				published = false
				err = &nacos_error.NacosError{
					ErrMsg: "[client.PublishConfig] [" + strconv.Itoa(response.StatusCode) + "]" + string(bytes),
				}
			}
		}
	}
	return
}

func (client *ConfigClient) DeleteConfig(param vo.ConfigParam) (deleted bool,
	err error) {
	if len(param.DataId) <= 0 {
		err = errors.New("[client.DeleteConfig] param.dataId can not be empty")
	}
	if len(param.Group) <= 0 {
		err = errors.New("[client.DeleteConfig] param.group can not be empty")
	}
	var clientConfig constant.ClientConfig
	var serverConfigs []constant.ServerConfig
	var agent http_agent.IHttpAgent
	if err == nil {
		clientConfig, serverConfigs, agent, err = client.sync()
	}
	if err == nil {
		params := util.TransformObject2Param(param)
		for _, serverConfig := range serverConfigs {
			path := client.buildBasePath(serverConfig)
			deleted, err = deleteConfig(agent, path, clientConfig.TimeoutMs, params)
			if err == nil {
				break
			} else {
				if _, ok := err.(*nacos_error.NacosError); ok {
					break
				} else {
					log.Println("[client.DeleteConfig] deleted config failed:", err.Error())
				}
			}
		}
	}
	return
}

func deleteConfig(agent http_agent.IHttpAgent, path string, timeoutMs uint64,
	params map[string]string) (deleted bool, err error) {
	var response *http.Response
	log.Println("[client.DeleteConfig] request url:", path, ",params:", params)
	response, err = agent.Delete(path, nil, timeoutMs, params)
	if err == nil {
		bytes, errRead := ioutil.ReadAll(response.Body)
		defer response.Body.Close()
		if errRead != nil {
			err = errRead
		} else {
			if response.StatusCode == 200 {
				if strings.ToLower(strings.Trim(string(bytes), " ")) == "true" {
					deleted = true
				} else {
					deleted = false
					err = errors.New("[client.DeleteConfig] " + string(bytes))
				}
			} else {
				deleted = false
				err = &nacos_error.NacosError{
					ErrMsg: "[client.DeleteConfig] [" + strconv.Itoa(response.StatusCode) + "]" + string(bytes),
				}
			}
		}
	}
	return
}

func (client *ConfigClient) AddConfigToListen(params []vo.ConfigParam) (err error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	if !client.listening {
		err = errors.New("[client.ListenConfig] client is not listening")
	} else {
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
	}
	return
}

func (client *ConfigClient) ListenConfig(params []vo.ConfigParam) (err error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	if client.listening {
		err = errors.New("[client.ListenConfig] client is listening,do not operator repeat")
	}
	// 监听
	if err == nil {
		client.listening = true
		client.localConfigs = params
		client.listenConfig()
	}
	return
}

func (client *ConfigClient) listenConfig() {
	go func() {
		for {
			clientConfig, serverConfigs, agent, err := client.sync()
			// 创建计时器
			var timer *time.Timer
			if err == nil {
				timer = time.NewTimer(time.Duration(clientConfig.ListenInterval) * time.Millisecond)
			}
			client.listenConfigTask(clientConfig, serverConfigs, agent)
			if !client.listening {
				break
			}
			<-timer.C
		}
	}()
}

func (client *ConfigClient) listenConfigTask(clientConfig constant.ClientConfig,
	serverConfigs []constant.ServerConfig, agent http_agent.IHttpAgent) {
	var listeningConfigs string
	var err error
	if len(client.localConfigs) <= 0 {
		err = errors.New("[client.ListenConfig] listen configs can not be empty")
	}
	// 检查&拼接监听参数
	if err == nil {
		client.mutex.Lock()
		for index, param := range client.localConfigs {
			if len(param.DataId) <= 0 {
				err = errors.New("[client.ListenConfig] params[" + strconv.Itoa(index) + "].DataId can not be empty")
				break
			}
			if len(param.Group) <= 0 {
				err = errors.New("[client.ListenConfig] params[" + strconv.Itoa(index) + "].Group can not be empty")
				break
			}
			var tenant string
			if len(param.Tenant) > 0 {
				tenant = param.Tenant
			}
			var md5 string
			if len(param.Content) > 0 {
				md5 = util.Md5(param.Content)
			}
			listeningConfigs += param.DataId + constant.SPLIT_CONFIG_INNER + param.Group + constant.SPLIT_CONFIG_INNER +
				md5 + constant.SPLIT_CONFIG_INNER + tenant + constant.SPLIT_CONFIG
		}
		client.mutex.Unlock()
	}
	if err != nil {
		client.mutex.Lock()
		client.listening = false
		client.mutex.Unlock()
		log.Println(err)
		log.Println("client.ListenConfig failed")
	}
	// http 请求
	if err == nil {
		params := make(map[string]string)
		params[constant.KEY_LISTEN_CONFIGS] = listeningConfigs
		var changed string
		for _, serverConfig := range serverConfigs {
			path := client.buildBasePath(serverConfig) + "/listener"
			changedTmp, err := listen(agent, path, clientConfig.TimeoutMs, clientConfig.ListenInterval, params)
			if err == nil {
				changed = changedTmp
				break
			} else {
				if _, ok := err.(*nacos_error.NacosError); ok {
					changed = changedTmp
					break
				} else {
					log.Println("[client.ListenConfig] listen config error:", err.Error())
				}
			}
		}
		if strings.ToLower(strings.Trim(changed, " ")) == "" {
			log.Println("[client.ListenConfig] no change")
		} else {
			log.Print("[client.ListenConfig] config changed:" + changed)
			client.updateLocalConfig(changed)
		}
	}
}

func listen(agent http_agent.IHttpAgent, path string,
	timeoutMs uint64, listenInterval uint64,
	params map[string]string) (changed string, err error) {
	header := map[string][]string{
		"Content-Type":         {"application/x-www-form-urlencoded"},
		"Long-Pulling-Timeout": {strconv.FormatUint(listenInterval, 10)},
	}
	log.Println("[client.ListenConfig] request url:", path, " ;params:", params, " ;header:", header)
	var response *http.Response
	response, err = agent.Post(path, header, timeoutMs, params)
	if err == nil {
		bytes, errRead := ioutil.ReadAll(response.Body)
		defer response.Body.Close()
		if errRead != nil {
			err = errRead
		} else {
			if response.StatusCode == 200 {
				changed = string(bytes)
			} else {
				err = &nacos_error.NacosError{
					ErrMsg: "[" + strconv.Itoa(response.StatusCode) + "]" + string(bytes),
				}
			}
		}
	}
	return
}

func (client *ConfigClient) StopListenConfig() {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	if client.listening {
		client.listening = false
	}
	log.Println("[client.StopListenConfig] stop listen config success")
}

// ListenConfig 发现配置变化时候，修改本地配置
func (client *ConfigClient) updateLocalConfig(changed string) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	changedConfigs := strings.Split(changed, "%01")
	for _, config := range changedConfigs {
		attrs := strings.Split(config, "%02")
		if len(attrs) == 2 {
			content, err := client.GetConfig(vo.ConfigParam{
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
			}
		} else if len(attrs) == 3 {
			content, err := client.GetConfig(vo.ConfigParam{
				DataId: attrs[0],
				Group:  attrs[1],
				Tenant: attrs[2],
			})
			if err != nil {
				log.Println("[client.updateLocalConfig] update config failed:", err.Error())
			} else {
				client.putLocalConfig(vo.ConfigParam{
					DataId:  attrs[0],
					Group:   attrs[1],
					Tenant:  attrs[2],
					Content: content,
				})
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
