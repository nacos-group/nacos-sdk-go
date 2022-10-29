package config_client

import (
	"context"

	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type IConfigProxy interface {
	queryConfig(dataId, group, tenant string, timeout uint64, notify bool, client *ConfigClient) (*rpc_response.ConfigQueryResponse, error)
	searchConfigProxy(param vo.SearchConfigParam, tenant, accessKey, secretKey string) (*model.ConfigPage, error)
	requestProxy(rpcClient *rpc.RpcClient, request rpc_request.IRequest, timeoutMills uint64) (rpc_response.IResponse, error)
	createRpcClient(ctx context.Context, taskId string, client *ConfigClient) *rpc.RpcClient
	getRpcClient(client *ConfigClient) *rpc.RpcClient
}
