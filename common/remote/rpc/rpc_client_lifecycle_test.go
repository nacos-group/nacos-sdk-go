package rpc

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	nacos_grpc_service "github.com/nacos-group/nacos-sdk-go/v2/api/grpc"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"google.golang.org/grpc"
)

func TestLifecycleShutdownRejectsLateReconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := NewGrpcClient(ctx, t.Name(), nil, nil)
	atomic.StoreInt32((*int32)(&client.rpcClientStatus), int32(UNHEALTHY))
	candidate := &lifecycleConnection{}
	connector := &lifecycleConnector{GrpcClient: client, entered: make(chan struct{}), release: make(chan struct{}), candidate: candidate}
	client.executeClient = connector
	done := make(chan struct{})
	go func() {
		defer close(done)
		client.reconnect(ServerInfo{serverIp: "test", serverPort: 8848}, false)
	}()
	waitLifecycle(t, connector.entered)
	client.Shutdown()
	close(connector.release)
	waitLifecycle(t, done)
	if !client.isShutdown() {
		t.Errorf("late reconnect restored status %v after shutdown", client.getStatus())
	}
	if !candidate.closed.Load() {
		t.Error("late reconnect candidate was not closed")
	}
}

func TestLifecycleShutdownRejectsLateHealthCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := NewGrpcClient(ctx, t.Name(), nil, nil)
	connection := &lifecycleConnection{entered: make(chan struct{}), release: make(chan struct{})}
	client.SetCurrentConnection(connection)
	atomic.StoreInt32((*int32)(&client.rpcClientStatus), int32(RUNNING))
	done := make(chan struct{})
	go func() {
		defer close(done)
		client.reconnect(ServerInfo{}, true)
	}()
	waitLifecycle(t, connection.entered)
	client.Shutdown()
	close(connection.release)
	waitLifecycle(t, done)
	if !client.isShutdown() {
		t.Errorf("late health-check restored status %v after shutdown", client.getStatus())
	}
}

func TestLifecycleConcurrentShutdown(t *testing.T) {
	client := NewGrpcClient(context.Background(), t.Name(), nil, nil)
	client.SetCurrentConnection(&lifecycleConnection{})
	const callers = 8
	var workers sync.WaitGroup
	workers.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer workers.Done()
			client.Shutdown()
		}()
	}
	done := make(chan struct{})
	go func() {
		workers.Wait()
		close(done)
	}()
	waitLifecycle(t, done)
}

func TestLifecycleReconnectPreservesConnectionEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := NewGrpcClient(ctx, t.Name(), nil, nil)
	client.eventChan = make(chan ConnectionEvent, 2)
	previous := &lifecycleConnection{}
	candidate := &lifecycleConnection{}
	release := make(chan struct{})
	close(release)
	client.executeClient = &lifecycleConnector{
		GrpcClient: client,
		entered:    make(chan struct{}),
		release:    release,
		candidate:  candidate,
	}
	client.SetCurrentConnection(previous)
	client.reconnect(ServerInfo{serverIp: "test", serverPort: 8848}, false)
	if !previous.closed.Load() || !previous.abandoned.Load() {
		t.Error("reconnect did not abandon and close the previous connection")
	}
	if got := client.GetCurrentConnection(); got != candidate {
		t.Errorf("current connection = %v, want candidate", got)
	}
	for _, want := range []ConnectionStatus{DISCONNECTED, CONNECTED} {
		select {
		case event := <-client.eventChan:
			if event.eventType != want {
				t.Errorf("connection event = %v, want %v", event.eventType, want)
			}
		default:
			t.Fatalf("missing connection event %v", want)
		}
	}
}

func TestLifecycleShutdownCancelsRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := NewGrpcClient(ctx, t.Name(), nil, nil)
	requestClient := &lifecycleRequestClient{entered: make(chan struct{})}
	connection := &GrpcConnection{
		Connection: &Connection{connectionId: "lifecycle-test"},
		client:     requestClient,
	}
	done := make(chan struct{})
	var requestErr error
	go func() {
		defer close(done)
		_, requestErr = connection.request(rpc_request.NewHealthCheckRequest(), 60000, client.RpcClient)
	}()
	waitLifecycle(t, requestClient.entered)
	client.Shutdown()
	waitLifecycle(t, done)
	if !errors.Is(requestErr, context.Canceled) {
		t.Errorf("request error = %v, want context cancellation", requestErr)
	}
}

func TestLifecycleCanceledNotificationDoesNotBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := NewGrpcClient(ctx, t.Name(), nil, nil)
	client.eventChan <- ConnectionEvent{eventType: CONNECTED}
	cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		client.notifyConnectionChange(DISCONNECTED)
	}()
	t.Cleanup(func() {
		select {
		case <-client.eventChan:
		default:
		}
		waitLifecycle(t, done)
	})
	waitLifecycle(t, done)
}

type lifecycleConnector struct {
	*GrpcClient
	entered   chan struct{}
	release   chan struct{}
	candidate IConnection
}

func (c *lifecycleConnector) connectToServer(ServerInfo) (IConnection, error) {
	close(c.entered)
	<-c.release
	return c.candidate, nil
}

type lifecycleRequestClient struct {
	entered chan struct{}
}

func (c *lifecycleRequestClient) Request(ctx context.Context, _ *nacos_grpc_service.Payload, _ ...grpc.CallOption) (*nacos_grpc_service.Payload, error) {
	close(c.entered)
	<-ctx.Done()
	return nil, ctx.Err()
}

type lifecycleConnection struct {
	closed    atomic.Bool
	abandoned atomic.Bool
	entered   chan struct{}
	release   chan struct{}
}

func (c *lifecycleConnection) request(rpc_request.IRequest, int64, *RpcClient) (rpc_response.IResponse, error) {
	if c.entered != nil {
		close(c.entered)
		<-c.release
	}
	return &rpc_response.HealthCheckResponse{Response: &rpc_response.Response{ResultCode: 200, Success: true}}, nil
}

func (c *lifecycleConnection) close()                  { c.closed.Store(true) }
func (*lifecycleConnection) getConnectionId() string   { return "lifecycle-test" }
func (*lifecycleConnection) getServerInfo() ServerInfo { return ServerInfo{serverIp: "test"} }
func (c *lifecycleConnection) setAbandon(flag bool)    { c.abandoned.Store(flag) }
func (c *lifecycleConnection) getAbandon() bool        { return c.abandoned.Load() }

func waitLifecycle(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("lifecycle operation did not finish")
	}
}
