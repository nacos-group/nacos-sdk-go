package naming_grpc

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_cache"
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

// newProxyWithStub builds a NamingGrpcProxy whose rpc round-trip is
// replaced by stub, so the request response handling can be exercised
// without a live server.
func newProxyWithStub(stub func(request rpc_request.IRequest) (rpc_response.IResponse, error)) *NamingGrpcProxy {
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
	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
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

	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
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

// newConfirmedSubscribedProxy builds a proxy with one confirmed subscription
// for DEMO in the default group, established through the real Subscribe path.
func newConfirmedSubscribedProxy() *NamingGrpcProxy {
	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		return &rpc_response.SubscribeServiceResponse{
			Response:    &rpc_response.Response{Success: true, ResultCode: 200},
			ServiceInfo: model.Service{Name: "DEMO", GroupName: "DEFAULT_GROUP"},
		}, nil
	})
	_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
	if err != nil {
		panic(err)
	}
	return proxy
}

// TestNamingGrpcProxy_Unsubscribe_FailedResponseKeepsRedoEntry covers the
// unsubscribe half of the confirmation semantics: RpcClient.Request returns
// (failed response, nil) for a business failure, so Unsubscribe must validate
// the response before removing the redo entry, and a failed unsubscribe must
// leave the entry cached and confirmed, matching the server-side state.
func TestNamingGrpcProxy_Unsubscribe_FailedResponseKeepsRedoEntry(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")
	proxy := newConfirmedSubscribedProxy()

	// the rpc layer returns a failed business response without an error,
	// exactly like RpcClient.Request does for non-ErrorResponse failures
	proxy.requestToServerFn = func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		return &rpc_response.SubscribeServiceResponse{
			Response: &rpc_response.Response{Success: false, ErrorCode: 403, ResultCode: 403, Message: "access denied"},
		}, nil
	}

	err := proxy.Unsubscribe("DEMO", "DEFAULT_GROUP", "")
	assert.Error(t, err, "a failed unsubscribe response must be translated into an error")
	assert.True(t, proxy.eventListener.IsSubscriberCached(key),
		"a failed response must keep the redo entry, the server still has the subscription")
	assert.True(t, proxy.eventListener.IsSubscriberRegistered(key),
		"a failed response must keep the confirmed state, the server still pushes for it")

	// the server recovers: the next unsubscribe succeeds and cleans the entry
	proxy.requestToServerFn = func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		return &rpc_response.SubscribeServiceResponse{
			Response: &rpc_response.Response{Success: true, ResultCode: 200},
		}, nil
	}
	err = proxy.Unsubscribe("DEMO", "DEFAULT_GROUP", "")
	assert.NoError(t, err)
	assert.False(t, proxy.eventListener.IsSubscriberCached(key),
		"a successful unsubscribe must remove the redo entry")
	assert.False(t, proxy.eventListener.IsSubscriberRegistered(key),
		"a successful unsubscribe must clear the confirmed state")
}

// TestNamingGrpcProxy_Unsubscribe_StaleResponseKeepsNewSubscription covers the
// concurrency review: a subscription started while an unsubscribe response is in
// flight waits for that response, so the server applies the unsubscribe first and
// the newer subscription second. The newer subscription must survive and still be
// restored on reconnect.
func TestNamingGrpcProxy_Unsubscribe_StaleResponseKeepsNewSubscription(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")
	unsubscribeSent := make(chan struct{})
	releaseUnsubscribe := make(chan struct{})

	var mu sync.Mutex
	subscribeRequests := 0
	var applied []string
	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		subscribeRequest := request.(*rpc_request.SubscribeServiceRequest)
		if !subscribeRequest.Subscribe {
			close(unsubscribeSent)
			<-releaseUnsubscribe
			mu.Lock()
			applied = append(applied, "unsubscribe")
			mu.Unlock()
		} else {
			mu.Lock()
			subscribeRequests++
			applied = append(applied, "subscribe")
			mu.Unlock()
		}
		return &rpc_response.SubscribeServiceResponse{
			Response:    &rpc_response.Response{Success: true, ResultCode: 200},
			ServiceInfo: model.Service{Name: "DEMO", GroupName: "DEFAULT_GROUP"},
		}, nil
	})
	holder := naming_cache.NewServiceInfoHolder("", t.TempDir(), false, true)
	defer holder.Close()
	proxy.serviceInfoHolder = holder

	_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
	assert.NoError(t, err)

	// the server already executed the unsubscribe, but its response is delayed
	unsubscribeErr := make(chan error, 1)
	go func() {
		unsubscribeErr <- proxy.Unsubscribe("DEMO", "DEFAULT_GROUP", "")
	}()
	select {
	case <-unsubscribeSent:
	case <-time.After(5 * time.Second):
		t.Fatal("the unsubscribe request was not sent")
	}

	// a subscription created in the meantime must wait for the delayed response
	subscribeErr := make(chan error, 1)
	go func() {
		_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
		subscribeErr <- err
	}()
	select {
	case <-subscribeErr:
		t.Fatal("the newer subscription must wait for the in-flight unsubscribe of the same service")
	case <-time.After(200 * time.Millisecond):
	}

	close(releaseUnsubscribe)
	assert.NoError(t, waitResult(t, unsubscribeErr, "the delayed unsubscribe response"))
	assert.NoError(t, waitResult(t, subscribeErr, "the newer subscription"))

	mu.Lock()
	assert.Equal(t, []string{"subscribe", "unsubscribe", "subscribe"}, applied,
		"the server must apply the unsubscribe before the newer subscription")
	mu.Unlock()

	assert.True(t, proxy.eventListener.IsSubscriberCached(key),
		"the newer subscription must have a redo entry")
	assert.True(t, proxy.eventListener.IsSubscriberRegistered(key))

	// reconnect: the newer subscription must be restored through redo
	proxy.eventListener.OnDisConnect()
	proxy.eventListener.OnConnected()

	mu.Lock()
	requests := subscribeRequests
	mu.Unlock()
	assert.Equal(t, 3, requests, "reconnect redo must re-subscribe the newer subscription")
	assert.True(t, proxy.eventListener.IsSubscriberRegistered(key),
		"the newer subscription must be confirmed again after the reconnect")
}

// TestNamingGrpcProxy_Unsubscribe_UnexpectedResponseKeepsRedoEntry keeps the
// response type assertion defensive on the unsubscribe path: an unexpected
// response type must surface as an error instead of a panic, and must not
// remove the redo entry.
func TestNamingGrpcProxy_Unsubscribe_UnexpectedResponseKeepsRedoEntry(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")
	proxy := newConfirmedSubscribedProxy()

	proxy.requestToServerFn = func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		return &rpc_response.InstanceResponse{
			Response: &rpc_response.Response{Success: true, ResultCode: 200},
		}, nil
	}

	assert.NotPanics(t, func() {
		err := proxy.Unsubscribe("DEMO", "DEFAULT_GROUP", "")
		assert.Error(t, err, "an unexpected response type must be translated into an error")
	})
	assert.True(t, proxy.eventListener.IsSubscriberCached(key),
		"an unexpected response type must keep the redo entry")
	assert.True(t, proxy.eventListener.IsSubscriberRegistered(key),
		"an unexpected response type must keep the confirmed state")
}

// TestNamingGrpcProxy_Unsubscribe_ReconnectWithDelayedResponseKeepsIntent covers
// the third review: the reconnect replay went through Subscribe, so it took the
// redo entry back as a subscription while the user was unsubscribing from it.
// The cancelled subscription then survived both the delayed unsubscribe
// response and the next reconnect.
func TestNamingGrpcProxy_Unsubscribe_ReconnectWithDelayedResponseKeepsIntent(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")
	unsubscribeSent := make(chan struct{})
	releaseUnsubscribe := make(chan struct{})

	var mu sync.Mutex
	var subscribeRequests, unsubscribeRequests int
	var applied []string
	var unsubscribeSentOnce sync.Once
	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		subscribeRequest := request.(*rpc_request.SubscribeServiceRequest)
		mu.Lock()
		if subscribeRequest.Subscribe {
			subscribeRequests++
			applied = append(applied, "subscribe")
		} else {
			unsubscribeRequests++
			applied = append(applied, "unsubscribe")
		}
		mu.Unlock()
		if !subscribeRequest.Subscribe {
			// the server executed the unsubscribe, but the response is delayed
			unsubscribeSentOnce.Do(func() { close(unsubscribeSent) })
			<-releaseUnsubscribe
		}
		return &rpc_response.SubscribeServiceResponse{
			Response:    &rpc_response.Response{Success: true, ResultCode: 200},
			ServiceInfo: model.Service{Name: "DEMO", GroupName: "DEFAULT_GROUP"},
		}, nil
	})
	holder := naming_cache.NewServiceInfoHolder("", t.TempDir(), false, true)
	defer holder.Close()
	proxy.serviceInfoHolder = holder

	_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
	assert.NoError(t, err)

	unsubscribeErr := make(chan error, 1)
	go func() {
		unsubscribeErr <- proxy.Unsubscribe("DEMO", "DEFAULT_GROUP", "")
	}()
	select {
	case <-unsubscribeSent:
	case <-time.After(5 * time.Second):
		t.Fatal("the unsubscribe request was not sent")
	}

	// the connection is lost and re-established while the unsubscribe is in flight;
	// the replay must wait for the response instead of racing it
	proxy.eventListener.OnDisConnect()
	replayDone := make(chan struct{})
	go func() {
		proxy.eventListener.OnConnected()
		close(replayDone)
	}()
	select {
	case <-replayDone:
		t.Fatal("the reconnect replay must wait for the in-flight unsubscribe of the same service")
	case <-time.After(200 * time.Millisecond):
	}

	// the unsubscribe response arrives and is a success
	close(releaseUnsubscribe)
	assert.NoError(t, waitResult(t, unsubscribeErr, "the delayed unsubscribe response"))
	select {
	case <-replayDone:
	case <-time.After(5 * time.Second):
		t.Fatal("the reconnect replay did not complete after the unsubscribe finished")
	}

	mu.Lock()
	assert.Equal(t, []string{"subscribe", "unsubscribe"}, applied,
		"the replay must not send a second unsubscribe after the delayed response")
	assert.Equal(t, 1, unsubscribeRequests,
		"the replay must not send the unsubscribe a second time")
	mu.Unlock()

	assert.False(t, proxy.eventListener.IsSubscriberCached(key),
		"a successful unsubscribe must remove the redo entry even if a reconnect happened in between")
	assert.False(t, proxy.eventListener.IsSubscriberRegistered(key))

	// a later reconnect must not restore the subscription the user cancelled
	proxy.eventListener.OnDisConnect()
	proxy.eventListener.OnConnected()

	mu.Lock()
	requests := subscribeRequests
	mu.Unlock()
	assert.Equal(t, 1, requests, "the reconnect replay must not subscribe a service the user is unsubscribing from")
}

// TestNamingGrpcProxy_Unsubscribe_FailedRpcReconnectKeepsIntent covers the same
// intent loss when the unsubscribe never gets a response: the request failed
// with the lost connection, so the replay must not turn the cancelled
// subscription back into an active one.
func TestNamingGrpcProxy_Unsubscribe_FailedRpcReconnectKeepsIntent(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")

	var subscribeRequests int
	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		subscribeRequest := request.(*rpc_request.SubscribeServiceRequest)
		if !subscribeRequest.Subscribe {
			return nil, errors.New("connection is not established")
		}
		subscribeRequests++
		return &rpc_response.SubscribeServiceResponse{
			Response:    &rpc_response.Response{Success: true, ResultCode: 200},
			ServiceInfo: model.Service{Name: "DEMO", GroupName: "DEFAULT_GROUP"},
		}, nil
	})
	holder := naming_cache.NewServiceInfoHolder("", t.TempDir(), false, true)
	defer holder.Close()
	proxy.serviceInfoHolder = holder

	_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
	assert.NoError(t, err)

	err = proxy.Unsubscribe("DEMO", "DEFAULT_GROUP", "")
	assert.Error(t, err, "an unsubscribe request failure must be surfaced")
	assert.True(t, proxy.eventListener.IsSubscriberCached(key),
		"the unsubscribe intent must survive for the reconnect redo")

	// reconnect: the cancelled subscription must not be replayed as a subscribe
	proxy.eventListener.OnDisConnect()
	proxy.eventListener.OnConnected()

	assert.Equal(t, 1, subscribeRequests, "the reconnect replay must not subscribe a service the user is unsubscribing from")
	assert.False(t, proxy.eventListener.IsSubscriberCached(key),
		"the redo entry of a cancelled subscription is cleared once the lost connection dropped it")
	assert.False(t, proxy.eventListener.IsSubscriberRegistered(key))
}

// TestNamingGrpcProxy_Unsubscribe_ReplayContinuesCancel covers the replay branch
// reached when the client reports a new connection without having seen the
// disconnect (RpcClient.reconnect with onRequestFail), so the entry is still
// confirmed while the unsubscribe intent is pending. The server still holds the
// subscription on that connection, so the replay must send the unsubscribe
// instead of subscribing again.
func TestNamingGrpcProxy_Unsubscribe_ReplayContinuesCancel(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")

	var subscribeRequests, unsubscribeRequests int
	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		subscribeRequest := request.(*rpc_request.SubscribeServiceRequest)
		if subscribeRequest.Subscribe {
			subscribeRequests++
		} else {
			unsubscribeRequests++
			if unsubscribeRequests == 1 {
				return nil, errors.New("connection is not established")
			}
		}
		return &rpc_response.SubscribeServiceResponse{
			Response:    &rpc_response.Response{Success: true, ResultCode: 200},
			ServiceInfo: model.Service{Name: "DEMO", GroupName: "DEFAULT_GROUP"},
		}, nil
	})
	holder := naming_cache.NewServiceInfoHolder("", t.TempDir(), false, true)
	defer holder.Close()
	proxy.serviceInfoHolder = holder

	_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
	assert.NoError(t, err)

	err = proxy.Unsubscribe("DEMO", "DEFAULT_GROUP", "")
	assert.Error(t, err, "an unsubscribe request failure must be surfaced")
	assert.True(t, proxy.eventListener.IsSubscriberRegistered(key),
		"the server still holds the subscription on the connection")

	proxy.eventListener.OnConnected()

	assert.Equal(t, 1, subscribeRequests, "the replay must not subscribe a service the user is unsubscribing from")
	assert.Equal(t, 2, unsubscribeRequests, "the replay must continue the pending unsubscribe")
	assert.False(t, proxy.eventListener.IsSubscriberCached(key),
		"a successful unsubscribe must remove the redo entry")
}
