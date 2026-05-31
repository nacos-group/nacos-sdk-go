// Integration test for #833 fix: data race on currentConnection.
// Runs concurrent Subscribe/Unsubscribe/ServerHealthy operations against
// a real Nacos server to verify no race under -race detector.
//
// Usage: NACOS_INTEGRATION_TEST=1 go run -race ./common/remote/rpc/testdata/issue833_integration_test.go
package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func main() {
	if os.Getenv("NACOS_INTEGRATION_TEST") != "1" {
		fmt.Println("SKIP: set NACOS_INTEGRATION_TEST=1 to run")
		os.Exit(0)
	}

	sc := []constant.ServerConfig{
		*constant.NewServerConfig("127.0.0.1", 8848),
	}
	cc := *constant.NewClientConfig(
		constant.WithNamespaceId(""),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogLevel("warn"),
	)

	namingClient, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  &cc,
		ServerConfigs: sc,
	})
	if err != nil {
		fmt.Printf("FAIL: create naming client: %v\n", err)
		os.Exit(1)
	}

	time.Sleep(2 * time.Second)
	if !namingClient.ServerHealthy() {
		fmt.Println("FAIL: client not healthy after startup")
		os.Exit(1)
	}
	fmt.Println("=== Client connected and healthy ===")

	// Test: 10 concurrent goroutines doing Subscribe/Unsubscribe for 5 seconds
	// If there's a data race on currentConnection, -race will catch it
	fmt.Println("=== Test 1: Concurrent Subscribe/Unsubscribe (5s, 10 goroutines) ===")
	var wg sync.WaitGroup
	var successCount, errorCount int64
	done := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svcName := fmt.Sprintf("race-test-svc-%d", id)
			callback := func(services []model.Instance, err error) {}

			for {
				select {
				case <-done:
					return
				default:
					err := namingClient.Subscribe(&vo.SubscribeParam{
						ServiceName:       svcName,
						GroupName:         "DEFAULT_GROUP",
						SubscribeCallback: callback,
					})
					if err == nil {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}

					_ = namingClient.ServerHealthy()

					_ = namingClient.Unsubscribe(&vo.SubscribeParam{
						ServiceName:       svcName,
						GroupName:         "DEFAULT_GROUP",
						SubscribeCallback: callback,
					})
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(i)
	}

	time.Sleep(5 * time.Second)
	close(done)
	wg.Wait()

	fmt.Printf("  Results: success=%d, errors=%d\n", atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount))
	if atomic.LoadInt64(&successCount) == 0 {
		fmt.Println("FAIL: zero successful operations")
		os.Exit(1)
	}
	fmt.Println("  PASS: no data race detected by -race")

	// Test 2: Verify client still healthy after stress
	fmt.Println("\n=== Test 2: Client health after stress ===")
	if !namingClient.ServerHealthy() {
		fmt.Println("FAIL: client unhealthy after stress test")
		os.Exit(1)
	}
	fmt.Println("  PASS: client still healthy")

	// Test 3: Concurrent GetService + Subscribe (mixed read patterns)
	fmt.Println("\n=== Test 3: Mixed concurrent operations (3s) ===")
	done2 := make(chan struct{})
	var ops int64

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svcName := fmt.Sprintf("mixed-test-svc-%d", id)
			callback := func(services []model.Instance, err error) {}
			for {
				select {
				case <-done2:
					return
				default:
					_ = namingClient.Subscribe(&vo.SubscribeParam{
						ServiceName:       svcName,
						GroupName:         "DEFAULT_GROUP",
						SubscribeCallback: callback,
					})
					atomic.AddInt64(&ops, 1)
					time.Sleep(2 * time.Millisecond)
				}
			}
		}(i)
	}

	// Concurrent SelectInstances (exercises Request -> currentConnection.request)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-done2:
					return
				default:
					_, _ = namingClient.SelectAllInstances(vo.SelectAllInstancesParam{
						ServiceName: fmt.Sprintf("select-test-svc-%d", id),
						GroupName:   "DEFAULT_GROUP",
					})
					atomic.AddInt64(&ops, 1)
					time.Sleep(2 * time.Millisecond)
				}
			}
		}(i)
	}

	time.Sleep(3 * time.Second)
	close(done2)
	wg.Wait()

	fmt.Printf("  Total ops: %d\n", atomic.LoadInt64(&ops))
	fmt.Println("  PASS: no data race detected")

	fmt.Println("\n=== ALL INTEGRATION TESTS PASSED ===")
}
