package security

import (
	"encoding/json"
	"errors"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type AuthClient struct {
	username           string
	password           string
	accessToken        *atomic.Value
	tokenTtl           int64
	lastRefreshTime    int64
	tokenRefreshWindow int64
	agent              http_agent.IHttpAgent
	clientCfg          constant.ClientConfig
	serverCfgs         []constant.ServerConfig
}

func NewAuthClient(clientCfg constant.ClientConfig, serverCfgs []constant.ServerConfig, agent http_agent.IHttpAgent) AuthClient {
	client := AuthClient{
		username:    clientCfg.Username,
		password:    clientCfg.Password,
		serverCfgs:  serverCfgs,
		clientCfg:   clientCfg,
		agent:       agent,
		accessToken: &atomic.Value{},
	}

	return client
}

func (ac *AuthClient) GetAccessToken() string {
	v := ac.accessToken.Load()
	if v == nil {
		return ""
	}
	return v.(string)
}

func (ac *AuthClient) AutoRefresh() {

	// If the username is not set, the automatic refresh Token is not enabled

	if ac.username == "" {
		return
	}

	go func() {
		ticker := time.NewTicker(time.Millisecond * 5)

		for range ticker.C {
			_, err := ac.Login()
			if err != nil {
				log.Printf("[ERROR]: login has error %s", err)
			}
		}
	}()
}

func (ac *AuthClient) Login() (bool, error) {
	var throwable error = nil
	for i := 0; i < len(ac.serverCfgs); i++ {
		result, err := ac.login(ac.serverCfgs[i])
		throwable = err
		if result {
			return true, nil
		}
	}
	return false, throwable
}

func (ac *AuthClient) login(server constant.ServerConfig) (bool, error) {
	if ac.username != "" {
		query := map[string]string{"username": ac.username, "password": ac.password}

		contextPath := server.ContextPath

		if !strings.HasPrefix(contextPath, "/") {
			contextPath = "/" + contextPath
		}

		if strings.HasSuffix(contextPath, "/") {
			contextPath = contextPath[0 : len(contextPath)-1]
		}

		reqUrl := "http://" + server.IpAddr + ":" + strconv.FormatInt(int64(server.Port), 10) + contextPath + "/v1/auth/users/login"

		queryInfo := ""

		for key, value := range query {
			if len(value) > 0 {
				queryInfo += key + "=" + value + "&"
			}
		}
		if strings.HasSuffix(queryInfo, "&") {
			queryInfo = queryInfo[:len(queryInfo)-1]
		}

		reqUrl += "?" + queryInfo

		header := http.Header{}
		resp, err := ac.agent.Post(reqUrl, header, ac.clientCfg.TimeoutMs, map[string]string{})

		if err != nil {
			return false, err
		}

		var bytes []byte
		bytes, err = ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return false, err
		}

		if resp.StatusCode != 200 {
			errMsg := string(bytes)
			return false, errors.New(errMsg)
		}

		var result map[string]interface{}

		err = json.Unmarshal(bytes, &result)

		if err != nil {
			return false, err
		}

		if val, ok := result[constant.KEY_ACCESS_TOKEN]; ok {
			ac.accessToken.Store(val)
			ac.tokenTtl = int64(result[constant.KEY_TOKEN_TTL].(float64))
			ac.lastRefreshTime = ac.tokenTtl / 10
		}
	}
	return true, nil

}
