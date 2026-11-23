package naming_grpc

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client/naming_cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/stretchr/testify/assert"
)

func okSubscribeResponse() *rpc_response.SubscribeServiceResponse {
	return &rpc_response.SubscribeServiceResponse{
		Response:    &rpc_response.Response{Success: true, ResultCode: 200},
		ServiceInfo: model.Service{Name: "DEMO", GroupName: "DEFAULT_GROUP"},
	}
}

func newRedoSchedulerProxy(t *testing.T, stub func(request rpc_request.IRequest) (rpc_response.IResponse, error)) *NamingGrpcProxy {
	proxy := newRedoSchedulerProxyWithoutStopCleanup(t, stub)
	t.Cleanup(proxy.eventListener.Stop)
	return proxy
}

// newRedoSchedulerProxyWithoutStopCleanup is for the tests that stop the
// scheduler themselves, so a failing Stop cannot block the test cleanup.
func newRedoSchedulerProxyWithoutStopCleanup(t *testing.T,
	stub func(request rpc_request.IRequest) (rpc_response.IResponse, error)) *NamingGrpcProxy {
	proxy := newProxyWithStub(stub)
	proxy.eventListener.redoRunHook = func() {}
	holder := naming_cache.NewServiceInfoHolder("", t.TempDir(), false, true)
	t.Cleanup(holder.Close)
	proxy.serviceInfoHolder = holder
	return proxy
}

// watchRedoRuns installs a redo run hook, so a test can wait for the redo runs
// it triggers to finish instead of observing them while they are in flight.
func watchRedoRuns(proxy *NamingGrpcProxy) <-chan struct{} {
	done := make(chan struct{}, 16)
	proxy.eventListener.redoRunHook = func() {
		done <- struct{}{}
	}
	return done
}

func waitRedoRuns(t *testing.T, done <-chan struct{}, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("redo run %d did not finish", i+1)
		}
	}
}

// TestConnectionEventListener_RedoDoesNotBlockConnectionEvents covers the design
// decision of the issue: the redo rpc must not run on the connection event
// consumer goroutine, so a slow redo request cannot delay the connection events
// that arrive after it.
func TestConnectionEventListener_RedoDoesNotBlockConnectionEvents(t *testing.T) {
	redoStarted := make(chan struct{})
	releaseRedo := make(chan struct{})
	var startOnce, releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(releaseRedo) })
	}
	defer release()

	proxy := newRedoSchedulerProxy(t, func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		startOnce.Do(func() { close(redoStarted) })
		<-releaseRedo
		return okSubscribeResponse(), nil
	})
	proxy.eventListener.CacheSubscriberForRedo(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")

	connected := make(chan struct{})
	go func() {
		proxy.eventListener.OnConnected()
		close(connected)
	}()

	select {
	case <-redoStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("the connection event did not reach the redo rpc")
	}
	select {
	case <-connected:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("OnConnected blocked on the redo rpc instead of handing it to a scheduler")
	}
	release()
}

// TestConnectionEventListener_RedoSignalsCoalesce covers the duplicate task
// merging of the issue: connection events that arrive while a redo run is in
// flight must merge into at most one pending run.
func TestConnectionEventListener_RedoSignalsCoalesce(t *testing.T) {
	redoRuns := make(chan struct{}, 16)
	releaseRedo := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(releaseRedo) })
	}
	defer release()

	var runs int32
	proxy := newRedoSchedulerProxy(t, func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		atomic.AddInt32(&runs, 1)
		redoRuns <- struct{}{}
		<-releaseRedo
		return okSubscribeResponse(), nil
	})
	proxy.eventListener.CacheSubscriberForRedo(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")

	go proxy.eventListener.OnConnected()
	select {
	case <-redoRuns:
	case <-time.After(5 * time.Second):
		t.Fatal("the first redo run did not start")
	}

	// repeated connection events while the first run is still in flight
	signalled := make(chan struct{})
	go func() {
		for i := 0; i < 5; i++ {
			proxy.eventListener.OnConnected()
		}
		close(signalled)
	}()
	select {
	case <-signalled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("OnConnected blocked while a redo run was in flight")
	}

	release()
	select {
	case <-redoRuns:
	case <-time.After(5 * time.Second):
		t.Fatal("the merged redo run did not start")
	}
	select {
	case <-redoRuns:
		t.Fatal("the repeated connection events must merge into at most one pending redo run")
	case <-time.After(300 * time.Millisecond):
	}
	assert.EqualValues(t, 2, atomic.LoadInt32(&runs),
		"five connection events during one redo run must schedule exactly one more run")
}

// TestConnectionEventListener_StopInsideRedoCallbackReturns covers the shutdown
// deadlock: the reconnect redo delivers the subscriber callbacks on the scheduler
// goroutine, so a callback that closes the client must not wait for the scheduler
// goroutine to exit first.
func TestConnectionEventListener_StopInsideRedoCallbackReturns(t *testing.T) {
	proxy := newRedoSchedulerProxyWithoutStopCleanup(t, func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		response := okSubscribeResponse()
		response.ServiceInfo.Hosts = []model.Instance{{Ip: "1.1.1.1", Port: 8848, Weight: 1}}
		return response, nil
	})
	fullServiceName := util.GetGroupName("DEMO", "DEFAULT_GROUP")
	proxy.eventListener.CacheSubscriberForRedo(fullServiceName, "")

	closeReturned := make(chan struct{})
	callback := func(services []model.Instance, err error) {
		proxy.eventListener.Stop()
		close(closeReturned)
	}
	proxy.serviceInfoHolder.RegisterCallback(fullServiceName, "",
		naming_cache.NewSubscribeCallbackFuncWrapper(naming_cache.NewClusterSelector(nil), &callback))

	proxy.eventListener.OnConnected()

	select {
	case <-closeReturned:
	case <-time.After(5 * time.Second):
		t.Fatal("closing the client inside the redo callback deadlocked on the scheduler goroutine")
	}
}

// TestConnectionEventListener_StopSkipsRemainingRedo covers the shutdown
// requirement of the issue: stopping the scheduler stops a run in flight, so the
// cached services after the current one are not replayed.
func TestConnectionEventListener_StopSkipsRemainingRedo(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var startOnce, releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(releaseFirst) })
	}
	defer release()
	var mu sync.Mutex
	var requests int

	proxy := newRedoSchedulerProxyWithoutStopCleanup(t, func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		mu.Lock()
		requests++
		mu.Unlock()
		startOnce.Do(func() { close(firstStarted) })
		<-releaseFirst
		return okSubscribeResponse(), nil
	})
	proxy.eventListener.CacheSubscriberForRedo(util.GetGroupName("DEMO-1", "DEFAULT_GROUP"), "")
	proxy.eventListener.CacheSubscriberForRedo(util.GetGroupName("DEMO-2", "DEFAULT_GROUP"), "")
	redoDone := watchRedoRuns(proxy)

	proxy.eventListener.OnConnected()
	select {
	case <-firstStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("the redo run did not start")
	}

	stopped := make(chan struct{})
	go func() {
		proxy.eventListener.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop blocked on the redo rpc in flight")
	}

	release()
	select {
	case <-redoDone:
	case <-time.After(5 * time.Second):
		t.Fatal("the redo run did not finish after the rpc was released")
	}
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, requests, "a stopped scheduler must not replay the remaining cached services")
}

// TestConnectionEventListener_StopEndsRedo covers the shutdown requirement of
// the issue: once the client is closed, a later connection event must not start
// a redo run, and stopping the scheduler again is safe.
func TestConnectionEventListener_StopEndsRedo(t *testing.T) {
	var runs int32
	proxy := newRedoSchedulerProxy(t, func(request rpc_request.IRequest) (rpc_response.IResponse, error) {
		atomic.AddInt32(&runs, 1)
		return okSubscribeResponse(), nil
	})
	proxy.eventListener.CacheSubscriberForRedo(util.GetGroupName("DEMO", "DEFAULT_GROUP"), "")

	proxy.eventListener.Stop()

	stopped := make(chan struct{})
	go func() {
		proxy.eventListener.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop must be safe to call more than once")
	}

	proxy.eventListener.OnConnected()
	time.Sleep(300 * time.Millisecond)
	assert.EqualValues(t, 0, atomic.LoadInt32(&runs),
		"a connection event after close must not start a redo run")
}
