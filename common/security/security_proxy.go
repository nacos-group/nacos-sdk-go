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
	"time"
)

type AuthClient struct {
	username           string
	password           string
	accessToken        string
	tokenTtl           int64
	lastRefreshTime    int64
	tokenRefreshWindow int64
	contextPath        string
	agent              http_agent.IHttpAgent
	config             constant.ClientConfig
}

func NewAuthClient(properties map[string]string, config constant.ClientConfig) *AuthClient {
	username := properties[constant.KEY_USERNAME]
	password := properties[constant.KEY_PASSWORD]
	contextPath := properties[constant.KEY_CONTEXT_PATH]

	client := &AuthClient{}

	client.username = username
	client.password = password
	client.contextPath = contextPath
	client.config = config

	return client
}

func (ac *AuthClient) GetAccessToken() string {
	return ac.accessToken
}

func (ac *AuthClient) AutoRefresh(servers []string)  {
	go func() {
		ticker := time.NewTicker(time.Millisecond * 5)

		for range ticker.C {
			_, err := ac.Login(servers)
			if  err != nil {
				log.Printf("[ERROR]: login has error %s", err)
			}
		}
	}()
}

func (ac *AuthClient) Login(servers []string) (bool, error) {
	var throwable error = nil
	for i := 0; i < len(servers); i++ {
		result, err := ac.login(servers[i])
		throwable = err
		if result {
			return true, nil
		}
	}
	return false, throwable
}

func (ac *AuthClient) SetHttpAgent(agent http_agent.IHttpAgent) (err error) {
	if agent == nil {
		err = errors.New("[client.SetHttpAgent] http agent can not be nil")
	} else {
		ac.agent = agent
	}
	return
}

func (ac *AuthClient) login(server string) (bool, error) {
	if ac.username != "" {

		params := map[string]string{"username": ac.username, "password": ac.password}
		reqUrl := "http://" + server + ac.contextPath + "/v1/auth/users/login"
		header := http.Header{}
		resp, err := ac.agent.Post(reqUrl, header, ac.config.TimeoutMs, params)

		if err != nil {
			return false, err
		}

		if resp.StatusCode != 200 {
			return false, nil
		}

		var bytes []byte
		bytes, err = ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return false, err
		}

		var result map[string]string

		err = json.Unmarshal(bytes, &result)

		if err != nil {
			return false, err
		}

		if val, ok := result[constant.KEY_ACCESS_TOKEN]; ok {
			ac.accessToken = val
			ac.tokenTtl, _ = strconv.ParseInt(result[constant.KEY_TOKEN_TTL], 10, 64)
			ac.lastRefreshTime = ac.tokenTtl / 10
		}
	}
	return true, nil

}
