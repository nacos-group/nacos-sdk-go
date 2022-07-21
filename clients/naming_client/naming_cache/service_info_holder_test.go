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
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
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

// create random ip addr
func createRandomIp() string {
	ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
	return ip
}

func creatRandomPort() uint64 {
	return rand.Uint64()
}
