/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package naming_cache

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestServiceInfoHolder_isServiceInstanceChanged(t *testing.T) {
	rand.Seed(time.Now().Unix())
	defaultIp := createRandomIp()
	defaultPort := creatRandomPort()
	serviceA := model.Service{
		LastRefTime: 1000,
		Hosts: []model.Instance{
			{
				Ip:   defaultIp,
				Port: defaultPort,
			},
			{
				Ip:   defaultIp,
				Port: defaultPort + 1,
			},
			{
				Ip:   defaultIp,
				Port: defaultPort + 2,
			},
		},
	}
	serviceB := model.Service{
		LastRefTime: 1001,
		Hosts: []model.Instance{
			{
				Ip:   defaultIp,
				Port: defaultPort,
			},
			{
				Ip:   defaultIp,
				Port: defaultPort + 3,
			},
			{
				Ip:   defaultIp,
				Port: defaultPort + 4,
			},
		},
	}
	ip := createRandomIp()
	serviceC := model.Service{
		LastRefTime: 1001,
		Hosts: []model.Instance{
			{
				Ip:   ip,
				Port: defaultPort,
			},
			{
				Ip:   ip,
				Port: defaultPort + 3,
			},
			{
				Ip:   ip,
				Port: defaultPort + 4,
			},
		},
	}

	t.Run("compareWithSelf", func(t *testing.T) {
		changed := isServiceInstanceChanged(serviceA, serviceA)
		assert.Equal(t, false, changed)
	})
	// compareWithIp
	t.Run("compareWithIp", func(t *testing.T) {
		changed := isServiceInstanceChanged(serviceA, serviceC)
		assert.Equal(t, true, changed)
	})
	// compareWithPort
	t.Run("compareWithPort", func(t *testing.T) {
		changed := isServiceInstanceChanged(serviceA, serviceB)
		assert.Equal(t, true, changed)
	})
}

func TestHostReactor_isServiceInstanceChangedWithUnOrdered(t *testing.T) {
	rand.Seed(time.Now().Unix())
	serviceA := model.Service{
		LastRefTime: 1001,
		Hosts: []model.Instance{
			{
				Ip:   createRandomIp(),
				Port: creatRandomPort(),
			},
			{
				Ip:   createRandomIp(),
				Port: creatRandomPort(),
			},
			{
				Ip:   createRandomIp(),
				Port: creatRandomPort(),
			},
		},
	}

	serviceB := model.Service{
		LastRefTime: 1001,
		Hosts: []model.Instance{
			{
				Ip:   createRandomIp(),
				Port: creatRandomPort(),
			},
			{
				Ip:   createRandomIp(),
				Port: creatRandomPort(),
			},
			{
				Ip:   createRandomIp(),
				Port: creatRandomPort(),
			},
		},
	}
	logger.Info("serviceA:%s and serviceB:%s are comparing", serviceA.Hosts, serviceB.Hosts)
	changed := isServiceInstanceChanged(serviceA, serviceB)
	assert.True(t, changed)
}

func TestServiceInfoDiskCacheRefresherKeepsLatestSnapshot(t *testing.T) {
	writeCount := 0
	writtenServices := map[string]*model.Service{}
	refresher := newServiceInfoDiskCacheRefresher(time.Hour, func(service *model.Service, cacheKey, cacheDir string) bool {
		writeCount++
		writtenServices[cacheKey] = service
		return true
	})
	defer refresher.close()

	cacheKey := "DEFAULT_GROUP@@svc"
	first := newTestService("svc", "1.1.1.1", 1)
	second := newTestService("svc", "1.1.1.2", 2)
	refresher.publish(&serviceInfoDiskCacheRefreshEvent{service: first, cacheKey: cacheKey, cacheDir: "cache"})
	refresher.publish(&serviceInfoDiskCacheRefreshEvent{service: second, cacheKey: cacheKey, cacheDir: "cache"})

	refresher.flushPendingEvents()

	assert.Equal(t, 1, writeCount)
	assert.Equal(t, "1.1.1.2", writtenServices[cacheKey].Hosts[0].Ip)
	assert.Equal(t, 0, refresher.pendingEventSize())
}

func TestServiceInfoDiskCacheRefresherRetriesFailedWrite(t *testing.T) {
	attempts := 0
	refresher := newServiceInfoDiskCacheRefresher(time.Hour, func(service *model.Service, cacheKey, cacheDir string) bool {
		attempts++
		return attempts > 1
	})
	defer refresher.close()

	cacheKey := "DEFAULT_GROUP@@svc"
	refresher.publish(&serviceInfoDiskCacheRefreshEvent{service: newTestService("svc", "1.1.1.1", 1), cacheKey: cacheKey, cacheDir: "cache"})

	refresher.flushPendingEvents()

	assert.Equal(t, 1, attempts)
	assert.Equal(t, 1, refresher.pendingEventSize())

	refresher.flushPendingEvents()

	assert.Equal(t, 2, attempts)
	assert.Equal(t, 0, refresher.pendingEventSize())
}

func TestServiceInfoDiskCacheRefresherCloseTimeout(t *testing.T) {
	writerStarted := make(chan struct{})
	releaseWriter := make(chan struct{})
	refresher := newServiceInfoDiskCacheRefresher(time.Hour, func(service *model.Service, cacheKey, cacheDir string) bool {
		close(writerStarted)
		<-releaseWriter
		return true
	})
	refresher.shutdownTimeout = 10 * time.Millisecond
	refresher.publish(&serviceInfoDiskCacheRefreshEvent{
		service:  newTestService("svc", "1.1.1.1", 1),
		cacheKey: "DEFAULT_GROUP@@svc",
		cacheDir: "cache",
	})

	closeReturned := make(chan struct{})
	go func() {
		refresher.close()
		close(closeReturned)
	}()

	select {
	case <-writerStarted:
	case <-time.After(time.Second):
		t.Fatal("writer was not started")
	}
	select {
	case <-closeReturned:
	case <-time.After(time.Second):
		t.Fatal("close did not return after shutdown timeout")
	}

	close(releaseWriter)
	select {
	case <-refresher.doneCh:
	case <-time.After(time.Second):
		t.Fatal("refresher did not stop after writer was released")
	}
}

func TestServiceInfoHolderProcessServiceQueuesDiskCacheBeforeCallback(t *testing.T) {
	var writeCount int32
	writtenIp := ""
	holder := NewServiceInfoHolder("public", t.TempDir(), true, true)
	holder.diskCacheRefresher.close()
	holder.diskCacheRefresher = newServiceInfoDiskCacheRefresher(time.Hour, func(service *model.Service, cacheKey, cacheDir string) bool {
		atomic.AddInt32(&writeCount, 1)
		writtenIp = service.Hosts[0].Ip
		return true
	})

	service := newTestService("svc", "1.1.1.1", 1)
	callbackCalled := false
	callback := func(services []model.Instance, err error) {
		callbackCalled = true
		assert.Equal(t, int32(0), atomic.LoadInt32(&writeCount))
		services[0].Ip = "callback-mutated"
	}
	callbackWrapper := NewSubscribeCallbackFuncWrapper(NewClusterSelector(nil), &callback)
	holder.RegisterCallback(util.GetGroupName(service.Name, service.GroupName), "", callbackWrapper)

	holder.ProcessService(service)

	assert.True(t, callbackCalled)
	assert.Equal(t, int32(0), atomic.LoadInt32(&writeCount))
	assert.Equal(t, 1, holder.diskCacheRefresher.pendingEventSize())

	holder.Close()

	assert.Equal(t, int32(1), atomic.LoadInt32(&writeCount))
	assert.Equal(t, "1.1.1.1", writtenIp)
}

// create random ip addr
func createRandomIp() string {
	ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
	return ip
}

func creatRandomPort() uint64 {
	return rand.Uint64()
}

func newTestService(serviceName, ip string, lastRefTime uint64) *model.Service {
	return &model.Service{
		Name:        serviceName,
		GroupName:   "DEFAULT_GROUP",
		LastRefTime: lastRefTime,
		Hosts: []model.Instance{
			{
				Ip:       ip,
				Port:     8848,
				Metadata: map[string]string{"k": "v"},
			},
		},
	}
}

// TestIsServiceInstanceChanged_NoDataRace 验证 isServiceInstanceChanged 不会在排序时修改原始 Hosts 切片
// 回归测试 https://github.com/nacos-group/nacos-sdk-go/issues/818
func TestIsServiceInstanceChanged_NoDataRace(t *testing.T) {
	rand.Seed(time.Now().Unix())
	// 构造一个包含多个实例的 service，Hosts 顺序是乱序的
	ip := createRandomIp()
	basePort := creatRandomPort()
	oldService := model.Service{
		LastRefTime: 1000,
		Hosts: []model.Instance{
			{Ip: ip, Port: basePort + 5},
			{Ip: ip, Port: basePort + 1},
			{Ip: ip, Port: basePort + 3},
			{Ip: ip, Port: basePort + 2},
			{Ip: ip, Port: basePort + 4},
		},
	}
	// newService 与 oldService 实例相同但顺序不同
	newService := model.Service{
		LastRefTime: 1001,
		Hosts: []model.Instance{
			{Ip: ip, Port: basePort + 1},
			{Ip: ip, Port: basePort + 2},
			{Ip: ip, Port: basePort + 3},
			{Ip: ip, Port: basePort + 4},
			{Ip: ip, Port: basePort + 5},
		},
	}
	// 保存原始顺序用于后续验证
	originalHosts := make([]model.Instance, len(oldService.Hosts))
	copy(originalHosts, oldService.Hosts)

	// 并发场景：一个 goroutine 调用 isServiceInstanceChanged（内部排序），
	// 另一个 goroutine 读取 oldService.Hosts（模拟 selectInstances 的遍历）
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			isServiceInstanceChanged(oldService, newService)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			// 模拟 selectInstances 中遍历 Hosts
			for _, host := range oldService.Hosts {
				_ = host.Ip
				_ = host.Port
			}
		}
	}()
	wg.Wait()
	// 验证 isServiceInstanceChanged 没有修改原始 Hosts 切片
	assert.Equal(t, originalHosts, oldService.Hosts, "isServiceInstanceChanged 不应修改原始 Hosts 切片")
}
