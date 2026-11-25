package rpc

import (
	"sync"
	"testing"

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

func TestConnectionAbandonConcurrent(t *testing.T) {
	connection := &Connection{}
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for i := 0; i < 1000; i++ {
			connection.setAbandon(true)
			connection.setAbandon(false)
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 1000; i++ {
			_ = connection.getAbandon()
		}
	}()
	workers.Wait()
}
