package rpc

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- Unit Tests for getStatus ---

func TestGetStatus_InitiallyInitialized(t *testing.T) {
	client := &RpcClient{}
	assert.Equal(t, INITIALIZED, client.getStatus())
	assert.True(t, client.IsInitialized())
}

func TestGetStatus_ReflectsStore(t *testing.T) {
	client := &RpcClient{}

	atomic.StoreInt32((*int32)(&client.rpcClientStatus), (int32)(RUNNING))
	assert.Equal(t, RUNNING, client.getStatus())
	assert.Equal(t, "RUNNING", client.getStatus().getDesc())
	assert.True(t, client.IsRunning())

	atomic.StoreInt32((*int32)(&client.rpcClientStatus), (int32)(SHUTDOWN))
	assert.Equal(t, SHUTDOWN, client.getStatus())
	assert.Equal(t, "SHUTDOWN", client.getStatus().getDesc())
	assert.True(t, client.isShutdown())
}

// --- Concurrent Data Race Test ---
// Designed to trigger a data race if rpcClientStatus is read without an
// atomic load. Run with: go test -race -count=1

func TestClientStatus_ConcurrentReadWrite(t *testing.T) {
	// Both racing accesses live in Request(): the retry loop formats the
	// "client not connected, current status:%s" message while the tail of a
	// concurrent Request() CASes rpcClientStatus to UNHEALTHY.
	client := &RpcClient{}
	atomic.StoreInt32((*int32)(&client.rpcClientStatus), (int32)(RUNNING))

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writer: simulates the CAS at the tail of Request() and switchServerAsync.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				atomic.CompareAndSwapInt32((*int32)(&client.rpcClientStatus), (int32)(RUNNING), (int32)(UNHEALTHY))
				atomic.StoreInt32((*int32)(&client.rpcClientStatus), (int32)(RUNNING))
			}
		}
	}()

	// Reader 1: simulates the error message built by Request() when the
	// connection is not usable.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = client.getStatus().getDesc()
			}
		}
	}()

	// Reader 2: simulates IsRunning/isShutdown checks from health check and
	// the reconnect loop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = client.IsRunning()
				_ = client.isShutdown()
			}
		}
	}()

	// Let them race for 200ms
	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
	// If we get here without -race reporting, the fix works.
}
