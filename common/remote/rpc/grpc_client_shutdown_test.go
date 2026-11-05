package rpc

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	nacos_grpc_service "github.com/nacos-group/nacos-sdk-go/v2/api/grpc"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestReceiveBiRequestStreamShutdown(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"connection closing", grpc.ErrClientConnClosing},
		{"grpc cancellation", status.Error(codes.Canceled, "context canceled")},
		{"wrapped grpc cancellation", fmt.Errorf("receive: %w", status.Error(codes.Canceled, "context canceled"))},
		{"context cancellation", context.Canceled},
		{"wrapped context cancellation", fmt.Errorf("receive: %w", context.Canceled)},
		{"EOF", io.EOF},
		{"wrapped EOF", fmt.Errorf("receive: %w", io.EOF)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, stream, logs, done := startStreamReceiver(t, RUNNING, tc.err, false)
			waitForStreamReceiver(t, stream.receiving, "Recv to block")

			// Shutdown must happen while Recv is blocked, as it does when the
			// client closes its connection while waiting for server messages.
			client.Shutdown()
			stream.release()
			waitForStreamReceiver(t, done, "receiver to exit")

			assertStreamLog(t, logs, zapcore.InfoLevel, "stream client closed after shutdown")
			assertStreamState(t, client, SHUTDOWN, false)
			if got := atomic.LoadInt32(&stream.recvCalls); got != 1 {
				t.Fatalf("Recv called %d times, want 1", got)
			}
		})
	}
}

func TestReceiveBiRequestStreamPreservesFailures(t *testing.T) {
	for _, tc := range []struct {
		name          string
		initialStatus RpcClientStatus
		err           error
		shutdown      bool
		wantLevel     zapcore.Level
		wantMessage   string
		wantStatus    RpcClientStatus
		wantReconnect bool
	}{
		{
			name: "running cancellation", initialStatus: RUNNING,
			err:       grpc.ErrClientConnClosing,
			wantLevel: zapcore.ErrorLevel, wantMessage: "request stream error, switch server",
			wantStatus: UNHEALTHY, wantReconnect: true,
		},
		{
			name: "running unavailable", initialStatus: RUNNING,
			err:       status.Error(codes.Unavailable, "connection lost"),
			wantLevel: zapcore.ErrorLevel, wantMessage: "request stream error, switch server",
			wantStatus: UNHEALTHY, wantReconnect: true,
		},
		{
			name: "running EOF", initialStatus: RUNNING, err: io.EOF,
			wantLevel: zapcore.InfoLevel, wantMessage: "request stream onCompleted, switch server",
			wantStatus: UNHEALTHY, wantReconnect: true,
		},
		{
			name: "shutdown unavailable", initialStatus: RUNNING, shutdown: true,
			err:       status.Error(codes.Unavailable, "connection lost"),
			wantLevel: zapcore.ErrorLevel, wantMessage: "received error event",
			wantStatus: SHUTDOWN,
		},
		{
			name: "unhealthy cancellation", initialStatus: UNHEALTHY,
			err:       grpc.ErrClientConnClosing,
			wantLevel: zapcore.ErrorLevel, wantMessage: "received error event",
			wantStatus: UNHEALTHY,
		},
		{
			name: "initialized cancellation", initialStatus: INITIALIZED,
			err:       grpc.ErrClientConnClosing,
			wantLevel: zapcore.ErrorLevel, wantMessage: "received error event",
			wantStatus: INITIALIZED,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, stream, logs, done := startStreamReceiver(t, tc.initialStatus, tc.err, false)
			waitForStreamReceiver(t, stream.receiving, "Recv to block")
			if tc.shutdown {
				client.Shutdown()
			}
			stream.release()
			waitForStreamReceiver(t, done, "receiver to exit")

			assertStreamLog(t, logs, tc.wantLevel, tc.wantMessage)
			if tc.wantLevel == zapcore.ErrorLevel && !strings.Contains(logs.All()[0].Message, tc.err.Error()) {
				t.Errorf("error log lost the original failure: %s", logs.All()[0].Message)
			}
			assertStreamState(t, client, tc.wantStatus, tc.wantReconnect)
		})
	}
}

func TestReceiveBiRequestStreamCanceledContext(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status RpcClientStatus
		level  zapcore.Level
	}{
		{"shutdown", SHUTDOWN, zapcore.InfoLevel},
		{"running", RUNNING, zapcore.WarnLevel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, stream, logs, done := startStreamReceiver(t, tc.status, nil, true)
			waitForStreamReceiver(t, done, "receiver to exit")

			assertStreamLog(t, logs, tc.level, "stream client close")
			assertStreamState(t, client, tc.status, false)
			if got := atomic.LoadInt32(&stream.recvCalls); got != 0 {
				t.Fatalf("Recv called %d times with an already canceled context", got)
			}
		})
	}
}

type shutdownTestStream struct {
	nacos_grpc_service.BiRequestStream_RequestBiStreamClient
	ctx         context.Context
	err         error
	receiving   chan struct{}
	released    chan struct{}
	receiveOnce sync.Once
	releaseOnce sync.Once
	recvCalls   int32
}

func (s *shutdownTestStream) Context() context.Context { return s.ctx }

func (s *shutdownTestStream) Recv() (*nacos_grpc_service.Payload, error) {
	atomic.AddInt32(&s.recvCalls, 1)
	s.receiveOnce.Do(func() { close(s.receiving) })
	<-s.released
	return nil, s.err
}

func (s *shutdownTestStream) release() {
	s.releaseOnce.Do(func() { close(s.released) })
}

func startStreamReceiver(t *testing.T, initialStatus RpcClientStatus, recvErr error, canceled bool) (*GrpcClient, *shutdownTestStream, *observer.ObservedLogs, <-chan struct{}) {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	previousLogger := logger.GetLogger()
	logger.SetLogger(&logger.NacosLogger{SugaredLogger: zap.New(core).Sugar()})

	ctx, cancel := context.WithCancel(context.Background())
	if canceled {
		cancel()
	}
	stream := &shutdownTestStream{
		ctx: ctx, err: recvErr,
		receiving: make(chan struct{}), released: make(chan struct{}),
	}
	client := &GrpcClient{RpcClient: &RpcClient{
		name:             t.Name(),
		rpcClientStatus:  initialStatus,
		reconnectionChan: make(chan ReconnectContext, 2),
	}}
	connection := &GrpcConnection{Connection: &Connection{connectionId: "shutdown-test-connection"}}
	done := make(chan struct{})
	go func() {
		defer close(done)
		client.receiveBiRequestStream(stream, connection)
	}()
	t.Cleanup(func() {
		cancel()
		stream.release()
		// Joining the receiver before restoring the global logger ensures
		// no log or reconnect operation can spill into the next test.
		waitForStreamReceiver(t, done, "receiver cleanup")
		logger.SetLogger(previousLogger)
	})
	return client, stream, logs, done
}

func waitForStreamReceiver(t *testing.T, done <-chan struct{}, operation string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", operation)
	}
}

func assertStreamLog(t *testing.T, logs *observer.ObservedLogs, level zapcore.Level, message string) {
	t.Helper()
	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("got %d log entries, want 1: %+v", len(entries), entries)
	}
	entry := entries[0]
	if entry.Level != level || !strings.Contains(entry.Message, message) || !strings.Contains(entry.Message, "shutdown-test-connection") {
		t.Fatalf("got log %+v, want %s containing %q and connection ID", entry, level, message)
	}
}

func assertStreamState(t *testing.T, client *GrpcClient, wantStatus RpcClientStatus, wantReconnect bool) {
	t.Helper()
	if got := client.getStatus(); got != wantStatus {
		t.Errorf("client status = %s, want %s", got.getDesc(), wantStatus.getDesc())
	}
	wantRequests := 0
	if wantReconnect {
		wantRequests = 1
	}
	if got := len(client.reconnectionChan); got != wantRequests {
		t.Fatalf("reconnection requests = %d, want %d", got, wantRequests)
	}
	if wantReconnect {
		request := <-client.reconnectionChan
		if request.serverInfo != (ServerInfo{}) || request.onRequestFail {
			t.Fatalf("unexpected reconnect request: %+v", request)
		}
	}
}
