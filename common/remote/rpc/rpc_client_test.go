package rpc

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/nacos_server"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {

}

// mock403Conn rejects every request with ErrorResponse(403), counting calls.
type mock403Conn struct {
	mockConn
	calls int
}

func (m *mock403Conn) request(request rpc_request.IRequest, timeoutMills int64, client *RpcClient) (rpc_response.IResponse, error) {
	m.calls++
	return &rpc_response.ErrorResponse{Response: &rpc_response.Response{
		ErrorCode: constant.NO_RIGHT,
		Message:   "Invalid signature",
	}}, nil
}

// A 403 response must mark re-login, keep the client RUNNING, and fail the current
// request with a non-nil error instead of retrying with the same stale token.
func TestRequest_NoRightErrorResponse(t *testing.T) {
	client := &RpcClient{nacosServer: &nacos_server.NacosServer{}}
	client.rpcClientStatus = RUNNING
	conn := &mock403Conn{}
	client.SetCurrentConnection(conn)

	resp, err := client.Request(rpc_request.NewConfigBatchListenRequest(1), 3000)

	assert.Error(t, err, "403 must fail the current request, a nil error lets callers report silent success")
	assert.NotNil(t, resp)
	assert.Equal(t, constant.NO_RIGHT, resp.GetErrorCode())
	assert.Equal(t, 1, conn.calls, "403 must not be retried with the same stale token")
	assert.True(t, client.IsRunning(), "403 must not mark the client UNHEALTHY")
}

// mockTyped403Conn rejects every request with the handler-specific typed response
// (errorCode=403), matching the Nacos 2.5.1 RemoteRequestAuthFilter path.
type mockTyped403Conn struct {
	mockConn
	calls int
}

func (m *mockTyped403Conn) request(request rpc_request.IRequest, timeoutMills int64, client *RpcClient) (rpc_response.IResponse, error) {
	m.calls++
	return &rpc_response.ConfigQueryResponse{Response: &rpc_response.Response{
		ResultCode: constant.NO_RIGHT,
		ErrorCode:  constant.NO_RIGHT,
		Message:    "Invalid signature",
	}}, nil
}

// The production-shaped 403 is a typed response, not a generic ErrorResponse;
// it must be handled the same way: no retry, non-nil error, client stays RUNNING.
func TestRequest_NoRightTypedResponse(t *testing.T) {
	client := &RpcClient{nacosServer: &nacos_server.NacosServer{}}
	client.rpcClientStatus = RUNNING
	conn := &mockTyped403Conn{}
	client.SetCurrentConnection(conn)

	resp, err := client.Request(rpc_request.NewConfigQueryRequest("group", "dataId", "tenant"), 3000)

	assert.Error(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, constant.NO_RIGHT, resp.GetErrorCode())
	assert.Equal(t, 1, conn.calls)
	assert.True(t, client.IsRunning())
}
