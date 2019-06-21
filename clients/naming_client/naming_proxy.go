package naming_client

import (
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NamingProxy struct {
	sync.RWMutex
	clientConfig        constant.ClientConfig
	httpAgent           http_agent.IHttpAgent
	serverList          []string
	lastSrvRefTime      int64
	vipSrvRefInterMills int64
}

func NewNamingProxy(clientCfg constant.ClientConfig, serverCfgs []constant.ServerConfig, httpAgent http_agent.IHttpAgent) NamingProxy {
	srvProxy := NamingProxy{}
	srvProxy.clientConfig = clientCfg
	srvProxy.httpAgent = httpAgent
	if serverCfgs != nil && len(serverCfgs) != 0 {
		var ss []string
		for _, cfg := range serverCfgs {
			ss = append(ss, cfg.IpAddr+":"+strconv.Itoa(int(cfg.Port)))
		}
		srvProxy.serverList = ss
	}
	srvProxy.initRefreshSrvIfNeed()
	return srvProxy
}

func (proxy *NamingProxy) RegisterService(serviceName string, groupName string, instance model.ServiceInstance) (string, error) {
	log.Printf("[INFO] register service namespaceId:<%s>,serviceName:<%s> with instance:<%s> \n", proxy.clientConfig.NamespaceId, serviceName, utils.ToJsonString(instance))
	params := map[string]string{}
	params["namespaceId"] = proxy.clientConfig.NamespaceId
	params["serviceName"] = serviceName
	params["groupName"] = groupName
	params["clusterName"] = instance.ClusterName
	params["ip"] = instance.Ip
	params["port"] = strconv.Itoa(int(instance.Port))
	params["weight"] = strconv.FormatFloat(instance.Weight, 'f', -1, 64)
	params["enable"] = strconv.FormatBool(instance.Enable)
	params["healthy"] = strconv.FormatBool(instance.Healthy)
	params["metadata"] = utils.ToJsonString(instance.Metadata)
	params["ephemeral"] = strconv.FormatBool(instance.Ephemeral)
	api := constant.WEB_CONTEXT + constant.SERVICE_PATH
	return proxy.reqApi(api, params, http.MethodPost)
}

func (proxy *NamingProxy) DeristerService(serviceName string, ip string, port uint64, clusterName string, ephemeral bool) (string, error) {
	log.Printf("[INFO] deregister service namespaceId:<%s>,serviceName:<%s> with instance:<%s:%d@%s> \n", proxy.clientConfig.NamespaceId, serviceName, ip, port, clusterName)
	params := map[string]string{}
	params["namespaceId"] = proxy.clientConfig.NamespaceId
	params["serviceName"] = serviceName
	params["clusterName"] = clusterName
	params["ip"] = ip
	params["port"] = strconv.Itoa(int(port))
	params["ephemeral"] = strconv.FormatBool(ephemeral)
	api := constant.WEB_CONTEXT + constant.SERVICE_PATH
	return proxy.reqApi(api, params, http.MethodDelete)
}

func (proxy *NamingProxy) SendBeat(info model.BeatInfo) (int64, error) {
	log.Printf("[INFO] namespaceId:<%s> sending beat to server:<%s> \n", proxy.clientConfig.NamespaceId, utils.ToJsonString(info))
	params := map[string]string{}
	params["namespaceId"] = proxy.clientConfig.NamespaceId
	params["serviceName"] = info.ServiceName
	params["beat"] = utils.ToJsonString(info)
	api := constant.WEB_CONTEXT + constant.SERVICE_BASE_PATH + "/instance/beat"
	result, err := proxy.reqApi(api, params, http.MethodPut)
	if err != nil {
		return 0, err
	}
	if result != "" {
		interVal, err := jsonparser.GetInt([]byte(result), "clientBeatInterval")
		if err != nil {
			return 0, errors.New(fmt.Sprintf("[ERROR] namespaceId:<%s> sending beat to server:<%s> get 'clientBeatInterval' from <%s> error:<%s>", proxy.clientConfig.NamespaceId, utils.ToJsonString(info), result, err.Error()))
		} else {
			return interVal, nil
		}
	}
	return 0, nil

}

func (proxy *NamingProxy) GetServiceList(pageNo int, pageSize int, groupName string, selector *model.ExpressionSelector) (*model.ServiceList, error) {
	params := map[string]string{}
	params["namespaceId"] = proxy.clientConfig.NamespaceId
	params["groupName"] = groupName
	params["pageNo"] = strconv.Itoa(pageNo)
	params["pageSize"] = strconv.Itoa(pageSize)

	if selector != nil {
		switch selector.Type {
		case "label":
			params["selector"] = utils.ToJsonString(selector)
			break
		default:
			break

		}
	}

	api := constant.WEB_CONTEXT + constant.SERVICE_BASE_PATH + "/service/list"
	result, err := proxy.reqApi(api, params, http.MethodGet)
	if err != nil {
		return nil, err
	}
	if result == "" {
		return nil, errors.New("request server return empty")
	}

	serviceList := model.ServiceList{}
	count, err := jsonparser.GetInt([]byte(result), "count")
	if err != nil {
		return nil, errors.New(fmt.Sprintf("[ERROR] namespaceId:<%s> get service list pageNo:<%d> pageSize:<%d> selector:<%s> from <%s> get 'count' from <%s> error:<%s>", proxy.clientConfig.NamespaceId, pageNo, pageSize, utils.ToJsonString(selector), groupName, result, err.Error()))
	}
	var doms []string
	_, err = jsonparser.ArrayEach([]byte(result), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		doms = append(doms, string(value))
	}, "doms")
	if err != nil {
		return nil, errors.New(fmt.Sprintf("[ERROR] namespaceId:<%s> get service list pageNo:<%d> pageSize:<%d> selector:<%s> from <%s> get 'doms' from <%s> error:<%s> ", proxy.clientConfig.NamespaceId, pageNo, pageSize, utils.ToJsonString(selector), groupName, result, err.Error()))
	}
	serviceList.Count = count
	serviceList.Doms = doms
	return &serviceList, nil
}

func (proxy *NamingProxy) ServerHealthy() bool {
	api := constant.WEB_CONTEXT + constant.SERVICE_BASE_PATH + "/operator/metrics"
	result, err := proxy.reqApi(api, map[string]string{}, http.MethodGet)
	if err != nil {
		log.Printf("[ERROR]:namespaceId:[%s] sending server healthy failed!,result:%s error:%s", proxy.clientConfig.NamespaceId, result, err.Error())
		return false
	}
	if result != "" {
		status, err := jsonparser.GetString([]byte(result), "status")
		if err != nil {
			log.Printf("[ERROR]:namespaceId:[%s] sending server healthy failed!,result:%s error:%s", proxy.clientConfig.NamespaceId, result, err.Error())
		} else {
			return status == "UP"
		}
	}
	return false
}

func (proxy *NamingProxy) QueryList(serviceName string, clusters string, udpPort int, healthyOnly bool) (string, error) {
	param := make(map[string]string)
	param["namespaceId"] = proxy.clientConfig.NamespaceId
	param["serviceName"] = serviceName
	param["clusters"] = clusters
	param["udpPort"] = strconv.Itoa(udpPort)
	param["healthyOnly"] = strconv.FormatBool(healthyOnly)
	param["clientIp"] = utils.LocalIP()
	api := constant.WEB_CONTEXT + constant.SERVICE_PATH + "/list"
	return proxy.reqApi(api, param, http.MethodGet)
}

func (proxy *NamingProxy) callServer(api string, params map[string]string, method string, curServer string) (result string, err error) {
	url := "http://" + curServer + api
	headers := map[string][]string{}
	headers["Client-Version"] = []string{constant.CLIENT_VERSION}
	headers["User-Agent"] = []string{constant.CLIENT_VERSION}
	headers["Accept-Encoding"] = []string{"gzip,deflate,sdch"}
	headers["Connection"] = []string{"Keep-Alive"}
	//headers["RequestId"] = []string{uuid.NewV4().String()}
	headers["Request-Module"] = []string{"Naming"}
	headers["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	var response *http.Response
	response, err = proxy.httpAgent.Request(method, url, headers, proxy.clientConfig.TimeoutMs, params)
	if err != nil {
		return
	}
	var bytes []byte
	bytes, err = ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return
	}
	result = string(bytes)
	if response.StatusCode == 200 {
		return
	} else {
		err = errors.New(fmt.Sprintf("request return error code %d", response.StatusCode))
		return
	}
}

func (proxy *NamingProxy) reqApi(api string, params map[string]string, method string) (string, error) {
	params["namespaceId"] = proxy.clientConfig.NamespaceId
	srvs := proxy.serverList
	if srvs == nil || len(srvs) == 0 {
		return "", errors.New("server list is empty")
	}
	//only one server,retry request when error
	if len(srvs) == 1 {
		for i := 0; i < constant.REQUEST_DOMAIN_RETRY_TIME; i++ {
			result, err := proxy.callServer(api, params, method, srvs[0])
			if err == nil {
				return result, nil
			}
			log.Printf("[ERROR] api<%s>,method:<%s>, params:<%s>, call domain error:<%s> , result:<%s> \n", api, method, utils.ToJsonString(params), err.Error(), result)
		}
		return "", errors.New("retry " + strconv.Itoa(constant.REQUEST_DOMAIN_RETRY_TIME) + " times request failed!")
	} else {
		index := rand.Intn(len(srvs))
		for i := 1; i <= len(srvs); i++ {
			curServer := srvs[index]
			result, err := proxy.callServer(api, params, method, curServer)
			if err == nil {
				return result, nil
			}
			log.Printf("[ERROR] api<%s>,method:<%s>, params:<%s>, call domain error:<%s> , result:<%s> \n", api, method, utils.ToJsonString(params), err.Error(), result)
			index = (index + i) % len(srvs)
		}
		return "", errors.New("retry " + strconv.Itoa(constant.REQUEST_DOMAIN_RETRY_TIME) + " times request failed!")
	}
}

func (proxy *NamingProxy) initRefreshSrvIfNeed() {
	if proxy.clientConfig.Endpoint == "" {
		return
	}
	proxy.refreshServerSrvIfNeed()
	go func() {
		time.Sleep(time.Duration(1) * time.Second)
		proxy.refreshServerSrvIfNeed()
	}()

}

func (proxy *NamingProxy) refreshServerSrvIfNeed() {
	if utils.CurrentMillis()-proxy.lastSrvRefTime < proxy.vipSrvRefInterMills && len(proxy.serverList) > 0 {
		return
	}

	var list []string
	urlString := "http://" + proxy.clientConfig.Endpoint + "/nacos/serverlist"
	result := proxy.httpAgent.RequestOnlyResult(http.MethodGet, urlString, nil, proxy.clientConfig.TimeoutMs, nil)
	list = strings.Split(result, "\n")
	log.Printf("[info] http nacos server list: <%s> \n", result)

	var servers []string
	for _, line := range list {
		if line != "" {
			servers = append(servers, strings.TrimSpace(line))
		}
	}

	if len(servers) > 0 {
		proxy.Lock()
		if !reflect.DeepEqual(proxy.serverList, servers) {
			log.Printf("[info] server list is updated, old: <%v>,new:<%v> \n", proxy.serverList, servers)
		}
		proxy.serverList = servers
		proxy.lastSrvRefTime = utils.CurrentMillis()
		proxy.Unlock()
	}

	return
}
