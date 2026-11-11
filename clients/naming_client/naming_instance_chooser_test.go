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

package naming_client

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/stretchr/testify/assert"
)

func healthyInstance(id string, weight float64) model.Instance {
	return model.Instance{
		InstanceId:  id,
		Ip:          id,
		Port:        80,
		Weight:      weight,
		ClusterName: "a",
		ServiceName: "DEMO",
		Enable:      true,
		Healthy:     true,
	}
}

func TestNamingClient_SelectOneHealthyInstance_FractionalWeight(t *testing.T) {
	services := model.Service{
		Name:     "DEFAULT_GROUP@@DEMO",
		Clusters: "a",
		Hosts: []model.Instance{
			healthyInstance("10.10.10.10", 0.5),
		},
	}
	assert.NotPanics(t, func() {
		instance, err := NewTestNamingClient().selectOneHealthyInstances(services)
		assert.Nil(t, err)
		assert.Equal(t, "10.10.10.10", instance.InstanceId)
	})
}

func TestChooser_Pick_AllWeightsBelowOne(t *testing.T) {
	chooser := newChooser([]model.Instance{
		healthyInstance("10.10.10.10", 0.2),
		healthyInstance("10.10.10.11", 0.8),
	})
	counts := pickTimes(chooser, 10000)
	assert.Equal(t, 10000, counts["10.10.10.10"]+counts["10.10.10.11"])
	assert.InDelta(t, 2000, counts["10.10.10.10"], 400)
}

func TestChooser_Pick_FractionalWeightKeepsProportion(t *testing.T) {
	// with int truncation the 0.5 instance would get weight 0 and never be picked
	chooser := newChooser([]model.Instance{
		healthyInstance("10.10.10.10", 0.5),
		healthyInstance("10.10.10.11", 1.5),
	})
	counts := pickTimes(chooser, 10000)
	assert.InDelta(t, 2500, counts["10.10.10.10"], 400)
}

func TestChooser_Pick_ZeroWeightNeverPicked(t *testing.T) {
	chooser := newChooser([]model.Instance{
		healthyInstance("10.10.10.10", 0),
		healthyInstance("10.10.10.11", 1),
	})
	counts := pickTimes(chooser, 1000)
	assert.Equal(t, 0, counts["10.10.10.10"])
}

func pickTimes(chooser Chooser, n int) map[string]int {
	counts := make(map[string]int)
	for i := 0; i < n; i++ {
		counts[chooser.pick().InstanceId]++
	}
	return counts
}
