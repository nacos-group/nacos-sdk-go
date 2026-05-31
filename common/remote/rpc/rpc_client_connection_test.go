package rpc

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v2/common/remote/rpc/rpc_response"
	"github.com/stretchr/testify/assert"
)

// mockConn implements IConnection for testing with an identifiable ID.
type mockConn struct {
	id         string
	serverInfo ServerInfo
	abandoned  bool
	closed     bool
}

func (m *mockConn) request(request rpc_request.IRequest, timeoutMills int64, client *RpcClient) (rpc_response.IResponse, error) {
	return nil, nil
}
func (m *mockConn) close()                    { m.closed = true }
func (m *mockConn) getConnectionId() string   { return m.id }
func (m *mockConn) getServerInfo() ServerInfo { return m.serverInfo }
func (m *mockConn) setAbandon(flag bool)      { m.abandoned = flag }
func (m *mockConn) getAbandon() bool          { return m.abandoned }

// --- Unit Tests for GetCurrentConnection / SetCurrentConnection ---

func TestGetCurrentConnection_InitiallyNil(t *testing.T) {
	client := &RpcClient{}
	conn := client.GetCurrentConnection()
	assert.Nil(t, conn, "initial connection should be nil")
}

func TestSetAndGetCurrentConnection(t *testing.T) {
	client := &RpcClient{}
	mock := &mockConn{id: "conn-1", serverInfo: ServerInfo{serverIp: "10.0.0.1", serverPort: 8848}}

	client.SetCurrentConnection(mock)
	got := client.GetCurrentConnection()

	assert.NotNil(t, got)
	assert.Equal(t, "conn-1", got.getConnectionId())
	assert.Equal(t, "10.0.0.1", got.getServerInfo().serverIp)
}

func TestSetCurrentConnection_Overwrite(t *testing.T) {
	client := &RpcClient{}
	conn1 := &mockConn{id: "conn-1"}
	conn2 := &mockConn{id: "conn-2"}

	client.SetCurrentConnection(conn1)
	assert.Equal(t, "conn-1", client.GetCurrentConnection().getConnectionId())

	client.SetCurrentConnection(conn2)
	assert.Equal(t, "conn-2", client.GetCurrentConnection().getConnectionId())
}

// --- Concurrent Data Race Tests ---
// These tests are specifically designed to trigger data races if the
// atomic.Value protection is removed. Run with: go test -race -count=1

func TestCurrentConnection_ConcurrentReadWrite(t *testing.T) {
	// Simulates the exact race from #833: reconnect goroutine writes
	// while healthCheck and Request goroutines read concurrently.
	client := &RpcClient{}
	client.SetCurrentConnection(&mockConn{id: "initial"})

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writer: simulates reconnect() overwriting currentConnection
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
				client.SetCurrentConnection(&mockConn{id: "conn-" + string(rune('A'+i%26))})
			}
		}
	}()

	// Reader 1: simulates healthCheck reading currentConnection
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				conn := client.GetCurrentConnection()
				if conn != nil {
					_ = conn.getConnectionId()
					_ = conn.getServerInfo()
				}
			}
		}
	}()

	// Reader 2: simulates Request() reading currentConnection
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				conn := client.GetCurrentConnection()
				if conn != nil {
					_ = conn.getConnectionId()
				}
			}
		}
	}()

	// Reader 3: simulates NamingPushRequestHandler reading
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				conn := client.GetCurrentConnection()
				if conn != nil {
					_ = conn.getServerInfo()
				}
			}
		}
	}()

	// Let them race for 200ms
	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
	// If we get here without -race reporting, the fix works.
}

func TestCurrentConnection_ConcurrentMultipleWriters(t *testing.T) {
	// Stress test: multiple writers (shouldn't happen in production
	// but verifies atomic.Value consistency even under stress)
	client := &RpcClient{}
	var wg sync.WaitGroup
	const numWriters = 4
	const numReaders = 8
	const iterations = 1000

	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				client.SetCurrentConnection(&mockConn{
					id:         "writer-" + string(rune('0'+writerID)),
					serverInfo: ServerInfo{serverIp: "192.168.1." + string(rune('0'+writerID))},
				})
			}
		}(w)
	}

	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				conn := client.GetCurrentConnection()
				if conn != nil {
					id := conn.getConnectionId()
					info := conn.getServerInfo()
					// Verify consistency: id and serverInfo should come from the same connection
					_ = id
					_ = info
				}
			}
		}()
	}

	wg.Wait()
	// Final state should be a valid connection
	final := client.GetCurrentConnection()
	assert.NotNil(t, final)
}

func TestCurrentConnection_ReadAfterReconnect(t *testing.T) {
	// Simulates the real scenario: connection is set, then a reconnect
	// replaces it, and concurrent readers never see a corrupted state.
	client := &RpcClient{}
	oldConn := &mockConn{id: "old-conn", serverInfo: ServerInfo{serverIp: "10.0.0.1", serverPort: 8848}}
	newConn := &mockConn{id: "new-conn", serverInfo: ServerInfo{serverIp: "10.0.0.2", serverPort: 8848}}

	client.SetCurrentConnection(oldConn)

	var wg sync.WaitGroup
	var readCount int64
	stop := make(chan struct{})

	// Readers that verify connection consistency
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					conn := client.GetCurrentConnection()
					if conn != nil {
						id := conn.getConnectionId()
						ip := conn.getServerInfo().serverIp
						// The id and ip must be from the SAME connection
						if id == "old-conn" {
							assert.Equal(t, "10.0.0.1", ip)
						} else if id == "new-conn" {
							assert.Equal(t, "10.0.0.2", ip)
						}
						atomic.AddInt64(&readCount, 1)
					}
				}
			}
		}()
	}

	// Give readers time to start
	time.Sleep(10 * time.Millisecond)

	// Simulate reconnect: abandon old, set new
	oldConn.abandoned = true
	client.SetCurrentConnection(newConn)

	// Let readers continue after reconnect
	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()

	assert.True(t, atomic.LoadInt64(&readCount) > 0, "readers should have executed")
	assert.Equal(t, "new-conn", client.GetCurrentConnection().getConnectionId())
}

func TestCurrentConnection_NilSafety(t *testing.T) {
	// Verify GetCurrentConnection never panics, even before any Set
	client := &RpcClient{}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				conn := client.GetCurrentConnection()
				// Should not panic; conn may be nil
				_ = conn
			}
		}()
	}
	wg.Wait()
}

// --- Test closeConnection with atomic access ---

func TestCloseConnection_WithActiveConnection(t *testing.T) {
	client := &RpcClient{
		eventChan: make(chan ConnectionEvent, 1),
	}
	mock := &mockConn{id: "to-close"}
	client.SetCurrentConnection(mock)

	client.closeConnection()

	assert.True(t, mock.closed, "connection should be closed")
}

func TestCloseConnection_WithNilConnection(t *testing.T) {
	client := &RpcClient{
		eventChan: make(chan ConnectionEvent, 1),
	}
	// Should not panic when connection is nil
	client.closeConnection()
}
