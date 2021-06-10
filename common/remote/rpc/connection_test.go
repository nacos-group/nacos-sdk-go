package rpc

import (
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
)

type MockConnection struct {
}

func (m *MockConnection) request(request rpc_request.IRequest, timeoutMills int64, client *RpcClient) (rpc_response.IResponse, error) {
	return nil, nil
}
func (m *MockConnection) close() {

}
func (m *MockConnection) getConnectionId() string {
	return ""
}
func (m *MockConnection) getServerInfo() ServerInfo {
	return ServerInfo{}
}
func (m *MockConnection) setAbandon(flag bool) {

}
