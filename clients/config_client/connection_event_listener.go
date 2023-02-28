package config_client

import (
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc"
	"strconv"
)

type ConnectionEventListener struct {
	rpcClient    *rpc.RpcClient
	configClient *ConfigClient
}

func NewConnectionEventListener(rpcClient *rpc.RpcClient, configClient *ConfigClient) *ConnectionEventListener {
	return &ConnectionEventListener{rpcClient: rpcClient, configClient: configClient}
}

// OnConnected notify when  connected to server.
func (e *ConnectionEventListener) OnConnected() {
	logger.Infof("%s Connected,notify listen context...", e.rpcClient.Name())
	e.configClient.notifyListenConfig()
}

// OnDisConnect notify when  disconnected to server.
func (e *ConnectionEventListener) OnDisConnect() {
	logger.Infof("%s DisConnected,clear listen context...", e.rpcClient.Name())

	taskId, exist := e.rpcClient.Labels()["taskId"]
	//not possible, but keep safe
	var setAllSyncWithServerFalse = !exist

	for _, v := range e.configClient.cacheMap.Items() {
		cache, ok := v.(*cacheData)

		if !ok {
			continue
		}
		if setAllSyncWithServerFalse || strconv.Itoa(cache.taskId) == taskId {
			cache.SetIsSyncWithServer(false)
		}
	}
}
