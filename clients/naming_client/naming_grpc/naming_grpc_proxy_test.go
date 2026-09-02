package naming_grpc

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/stretchr/testify/assert"
)

type MockNamingGrpc struct {
}

func (m *MockNamingGrpc) RegisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error) {
	return true, nil
}

func (m *MockNamingGrpc) BatchRegisterInstance(serviceName string, groupName string, instances []model.Instance) (bool, error) {
	return true, nil
}

func (m *MockNamingGrpc) DeregisterInstance(serviceName string, groupName string, instance model.Instance) (bool, error) {
	return true, nil
}

func (m *MockNamingGrpc) GetServiceList(pageNo uint32, pageSize uint32, groupName string, selector *model.ExpressionSelector) (model.ServiceList, error) {
	return model.ServiceList{Doms: []string{""}}, nil
}

func (m *MockNamingGrpc) ServerHealthy() bool {
	return true
}

func (m *MockNamingGrpc) QueryInstancesOfService(serviceName, groupName, clusters string, udpPort int, healthyOnly bool) (*model.Service, error) {
	return &model.Service{}, nil
}

func (m *MockNamingGrpc) Subscribe(serviceName, groupName, clusters string) (model.Service, error) {
	return model.Service{}, nil
}

func (m *MockNamingGrpc) Unsubscribe(serviceName, groupName, clusters string) error {
	return nil
}

func (m *MockNamingGrpc) CloseClient() {}

// newSubscribeProxyWithStub builds a NamingGrpcProxy whose rpc round-trip is
// replaced by stub, so the subscribe response handling can be exercised
// without a live server.
func newSubscribeProxyWithStub(stub func(request rpc_request.IRequest) (rpc_response.IResponse, error)) *NamingGrpcProxy {
	proxy := &NamingGrpcProxy{}
	proxy.requestToServerFn = stub
	proxy.eventListener = NewConnectionEventListener(proxy)
	return proxy
}

// TestNamingGrpcProxy_Subscribe_FailedResponseNotConfirmed covers the review
// regression: RpcClient.Request returns (failed response, nil) for a
// SubscribeServiceResponse with success=false, so Subscribe must validate the
// response instead of confirming the subscription on a nil error.
func TestNamingGrpcProxy_Subscribe_FailedResponseNotConfirmed(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")

	// the rpc layer returns a failed business response without an error,
	// exactly like RpcClient.Request does for non-ErrorResponse failures
	proxy := newSubscribeProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		return &rpc_response.SubscribeServiceResponse{
			Response: &rpc_response.Response{Success: false, ErrorCode: 500, ResultCode: 500, Message: "server error"},
		}, nil
	})

	_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
	assert.Error(t, err, "a failed subscribe response must be translated into an error")
	assert.False(t, proxy.eventListener.IsSubscriberRegistered(key),
		"a failed response must not confirm the subscription")
	assert.True(t, proxy.eventListener.IsSubscriberCached(key),
		"the redo intent must stay cached so the subscription is retried")

	// the server recovers: the next read retries the subscription and succeeds
	proxy.requestToServerFn = func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		return &rpc_response.SubscribeServiceResponse{
			Response:    &rpc_response.Response{Success: true, ResultCode: 200},
			ServiceInfo: model.Service{Name: "DEMO", GroupName: "DEFAULT_GROUP"},
		}, nil
	}
	service, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
	assert.NoError(t, err)
	assert.Equal(t, "DEMO", service.Name)
	assert.True(t, proxy.eventListener.IsSubscriberRegistered(key),
		"a successful response must confirm the subscription")
}

// TestNamingGrpcProxy_Subscribe_UnexpectedResponseNotConfirmed keeps the
// response type assertion defensive: an unexpected response type must surface
// as an error instead of a panic, and must not confirm the subscription.
func TestNamingGrpcProxy_Subscribe_UnexpectedResponseNotConfirmed(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")

	proxy := newSubscribeProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		return &rpc_response.InstanceResponse{
			Response: &rpc_response.Response{Success: true, ResultCode: 200},
		}, nil
	})

	assert.NotPanics(t, func() {
		_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
		assert.Error(t, err, "an unexpected response type must be translated into an error")
	})
	assert.False(t, proxy.eventListener.IsSubscriberRegistered(key),
		"an unexpected response type must not confirm the subscription")
	assert.True(t, proxy.eventListener.IsSubscriberCached(key),
		"the redo intent must stay cached so the subscription is retried")
}
