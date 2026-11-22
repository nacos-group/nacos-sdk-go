package naming_grpc

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/stretchr/testify/assert"
)

// appliedOps records the subscriber operations in the order the fake server
// applied them. The server decides the final subscription state by processing
// order, so this order is what the tests assert on.
type appliedOps struct {
	mu  sync.Mutex
	ops []string
}

func (o *appliedOps) apply(op string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.ops = append(o.ops, op)
}

func (o *appliedOps) list() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.ops...)
}

func okSubscribeResponse() *rpc_response.SubscribeServiceResponse {
	return &rpc_response.SubscribeServiceResponse{
		Response:    &rpc_response.Response{Success: true, ResultCode: 200},
		ServiceInfo: model.Service{Name: "DEMO", GroupName: "DEFAULT_GROUP"},
	}
}

func waitResult(t *testing.T, result <-chan error, what string) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return nil
	}
}

// TestNamingGrpcProxy_SubscribeWaitsForInFlightUnsubscribe covers the deferred
// race of issue #934: a subscribe issued while an unsubscribe for the same
// service is still in flight must wait for it. Otherwise the server may apply
// the stale cancel after the new subscription and stop pushing while the client
// reports the service as subscribed.
func TestNamingGrpcProxy_SubscribeWaitsForInFlightUnsubscribe(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")
	server := &appliedOps{}
	unsubscribeSent := make(chan struct{})
	releaseUnsubscribe := make(chan struct{})
	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		if request.(*rpc_request.SubscribeServiceRequest).Subscribe {
			server.apply("subscribe")
			return okSubscribeResponse(), nil
		}
		close(unsubscribeSent)
		<-releaseUnsubscribe
		server.apply("unsubscribe")
		return okSubscribeResponse(), nil
	})

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

	subscribeErr := make(chan error, 1)
	go func() {
		_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
		subscribeErr <- err
	}()
	select {
	case <-subscribeErr:
		t.Fatal("the subscribe must wait for the in-flight unsubscribe of the same service")
	case <-time.After(200 * time.Millisecond):
	}

	close(releaseUnsubscribe)
	assert.NoError(t, waitResult(t, unsubscribeErr, "the unsubscribe response"))
	assert.NoError(t, waitResult(t, subscribeErr, "the subscribe response"))

	assert.Equal(t, []string{"subscribe", "unsubscribe", "subscribe"}, server.list(),
		"the server must apply the operations in the order the client issued them")
	assert.True(t, proxy.eventListener.IsSubscriberRegistered(key))
}

// TestNamingGrpcProxy_UnsubscribeWaitsForInFlightSubscribe is the symmetric
// half: a late subscribe must not re-create a subscription the user cancelled
// while that subscribe was still in flight.
func TestNamingGrpcProxy_UnsubscribeWaitsForInFlightSubscribe(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")
	server := &appliedOps{}
	subscribeSent := make(chan struct{})
	releaseSubscribe := make(chan struct{})
	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		if request.(*rpc_request.SubscribeServiceRequest).Subscribe {
			close(subscribeSent)
			<-releaseSubscribe
			server.apply("subscribe")
			return okSubscribeResponse(), nil
		}
		server.apply("unsubscribe")
		return okSubscribeResponse(), nil
	})

	subscribeErr := make(chan error, 1)
	go func() {
		_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
		subscribeErr <- err
	}()
	select {
	case <-subscribeSent:
	case <-time.After(5 * time.Second):
		t.Fatal("the subscribe request was not sent")
	}

	unsubscribeErr := make(chan error, 1)
	go func() {
		unsubscribeErr <- proxy.Unsubscribe("DEMO", "DEFAULT_GROUP", "")
	}()
	select {
	case <-unsubscribeErr:
		t.Fatal("the unsubscribe must wait for the in-flight subscribe of the same service")
	case <-time.After(200 * time.Millisecond):
	}

	close(releaseSubscribe)
	assert.NoError(t, waitResult(t, subscribeErr, "the subscribe response"))
	assert.NoError(t, waitResult(t, unsubscribeErr, "the unsubscribe response"))

	assert.Equal(t, []string{"subscribe", "unsubscribe"}, server.list(),
		"the server must apply the operations in the order the client issued them")
	assert.False(t, proxy.eventListener.IsSubscriberCached(key))
	assert.False(t, proxy.eventListener.IsSubscriberRegistered(key))
}

// TestNamingGrpcProxy_SubscribeOfOtherServiceNotBlocked keeps the serialization
// per service: a slow unsubscribe of one service must not stall subscriptions of
// another.
func TestNamingGrpcProxy_SubscribeOfOtherServiceNotBlocked(t *testing.T) {
	server := &appliedOps{}
	unsubscribeSent := make(chan struct{})
	releaseUnsubscribe := make(chan struct{})
	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		subscribeRequest := request.(*rpc_request.SubscribeServiceRequest)
		switch {
		case subscribeRequest.Subscribe:
			server.apply("subscribe:" + subscribeRequest.ServiceName)
		case subscribeRequest.ServiceName == "DEMO":
			close(unsubscribeSent)
			<-releaseUnsubscribe
			server.apply("unsubscribe:DEMO")
		}
		return okSubscribeResponse(), nil
	})
	defer close(releaseUnsubscribe)

	_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
	assert.NoError(t, err)

	go func() {
		_ = proxy.Unsubscribe("DEMO", "DEFAULT_GROUP", "")
	}()
	<-unsubscribeSent

	otherErr := make(chan error, 1)
	go func() {
		_, err := proxy.Subscribe("OTHER", "DEFAULT_GROUP", "")
		otherErr <- err
	}()
	select {
	case err := <-otherErr:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("a subscription of another service must not wait for the unsubscribe of DEMO")
	}
}

// TestNamingGrpcProxy_SubscribeWaitsForReplayUnsubscribe covers the background
// half of issue #934: the reconnect replay may still send the cancel of an
// unsubscribe that failed earlier. A subscribe issued meanwhile must wait,
// otherwise the replayed cancel lands after it on the server.
func TestNamingGrpcProxy_SubscribeWaitsForReplayUnsubscribe(t *testing.T) {
	key := util.GetServiceCacheKey(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")
	server := &appliedOps{}
	var mu sync.Mutex
	unsubscribeCalls := 0
	replaySent := make(chan struct{})
	releaseReplay := make(chan struct{})
	proxy := newProxyWithStub(func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		if request.(*rpc_request.SubscribeServiceRequest).Subscribe {
			server.apply("subscribe")
			return okSubscribeResponse(), nil
		}
		mu.Lock()
		unsubscribeCalls++
		call := unsubscribeCalls
		mu.Unlock()
		if call == 1 {
			// the user unsubscribe fails on the live connection, so the intent
			// stays cached and the reconnect replay continues it
			return nil, errors.New("connection is not established")
		}
		close(replaySent)
		<-releaseReplay
		server.apply("unsubscribe")
		return okSubscribeResponse(), nil
	})

	_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
	assert.NoError(t, err)
	err = proxy.Unsubscribe("DEMO", "DEFAULT_GROUP", "")
	assert.Error(t, err, "the unsubscribe request failure must be surfaced")
	assert.True(t, proxy.eventListener.IsSubscriberRegistered(key),
		"the server still holds the subscription on the connection")

	replayDone := make(chan struct{})
	go func() {
		proxy.eventListener.OnConnected()
		close(replayDone)
	}()
	select {
	case <-replaySent:
	case <-time.After(5 * time.Second):
		t.Fatal("the reconnect replay did not send the unsubscribe")
	}

	subscribeErr := make(chan error, 1)
	go func() {
		_, err := proxy.Subscribe("DEMO", "DEFAULT_GROUP", "")
		subscribeErr <- err
	}()
	select {
	case <-subscribeErr:
		t.Fatal("the subscribe must wait for the replayed unsubscribe of the same service")
	case <-time.After(200 * time.Millisecond):
	}

	close(releaseReplay)
	select {
	case <-replayDone:
	case <-time.After(5 * time.Second):
		t.Fatal("the reconnect replay did not complete after the replayed unsubscribe finished")
	}
	assert.NoError(t, waitResult(t, subscribeErr, "the subscribe response"))

	assert.Equal(t, []string{"subscribe", "unsubscribe", "subscribe"}, server.list(),
		"the server must apply the operations in the order the client issued them")
	assert.True(t, proxy.eventListener.IsSubscriberRegistered(key))
}
