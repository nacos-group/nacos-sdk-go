/*
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
	"path/filepath"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v2/common/monitor"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/prometheus/client_golang/prometheus"
)

func TestServiceInfoMapSizeMetric(t *testing.T) {
	newMetricReader := func(t *testing.T) func() float64 {
		t.Helper()
		gauge := monitor.GetServiceInfoMapSizeMonitor()
		gauge.Set(0)
		t.Cleanup(func() { gauge.Set(0) })
		registry := prometheus.NewRegistry()
		registry.MustRegister(gauge)
		return func() float64 {
			t.Helper()
			metrics, err := registry.Gather()
			if err != nil {
				t.Fatal(err)
			}
			if len(metrics) != 1 || len(metrics[0].Metric) != 1 {
				t.Fatalf("unexpected metrics: %v", metrics)
			}
			return metrics[0].Metric[0].GetGauge().GetValue()
		}
	}
	service := func(name string, timestamp uint64) *model.Service {
		return &model.Service{
			Name:        name,
			GroupName:   "DEFAULT_GROUP",
			LastRefTime: timestamp,
			Hosts:       []model.Instance{{Ip: "127.0.0.1", Port: 8848, Healthy: true, Enable: true, Weight: 1}},
		}
	}

	t.Run("remove the last cached service", func(t *testing.T) {
		metric := newMetricReader(t)
		holder := NewServiceInfoHolder("public", t.TempDir(), true, true)
		defer holder.Close()
		holder.ProcessService(service("one", 1))
		if got := metric(); got != 1 {
			t.Fatalf("before removal: got %v, want 1", got)
		}
		holder.StopUpdateIfContain(util.GetGroupName("one", "DEFAULT_GROUP"), "")
		if got := metric(); got != 0 {
			t.Fatalf("after removal: got %v, want 0", got)
		}
	})

	t.Run("remove one of two cached services repeatedly", func(t *testing.T) {
		metric := newMetricReader(t)
		holder := NewServiceInfoHolder("public", t.TempDir(), true, true)
		defer holder.Close()
		holder.ProcessService(service("one", 1))
		holder.ProcessService(service("two", 1))
		for i := 0; i < 2; i++ {
			holder.StopUpdateIfContain(util.GetGroupName("one", "DEFAULT_GROUP"), "")
			if got := metric(); got != 1 {
				t.Fatalf("removal %d: got %v, want 1", i, got)
			}
		}
	})

	t.Run("load cached services from disk", func(t *testing.T) {
		metric := newMetricReader(t)
		directory := t.TempDir()
		for _, name := range []string{"one", "two"} {
			value := service(name, 1)
			key := util.GetServiceCacheKey(util.GetGroupName(value.Name, value.GroupName), "")
			if !cache.WriteServicesToFileWithResult(value, key, filepath.Join(directory, "naming", "public")) {
				t.Fatal("could not create disk-cache fixture")
			}
		}
		holder := NewServiceInfoHolder("public", directory, true, false)
		defer holder.Close()
		if _, ok := holder.GetServiceInfo("one", "DEFAULT_GROUP", ""); !ok {
			t.Fatal("disk cache was not loaded")
		}
		if got := metric(); got != 2 {
			t.Fatalf("after disk load: got %v, want 2", got)
		}
	})

	t.Run("updates count entries rather than notifications", func(t *testing.T) {
		metric := newMetricReader(t)
		holder := NewServiceInfoHolder("public", t.TempDir(), true, true)
		defer holder.Close()
		holder.ProcessService(service("one", 1))
		holder.ProcessService(service("one", 2))
		if got := metric(); got != 1 {
			t.Fatalf("after repeated updates with no callbacks: got %v, want 1", got)
		}
	})

	t.Run("retain the shared gauge last-writer semantics", func(t *testing.T) {
		metric := newMetricReader(t)
		first := NewServiceInfoHolder("first", t.TempDir(), true, true)
		defer first.Close()
		second := NewServiceInfoHolder("second", t.TempDir(), true, true)
		defer second.Close()
		first.ProcessService(service("one", 1))
		first.ProcessService(service("two", 1))
		second.ProcessService(service("three", 1))
		if got := metric(); got != 1 {
			t.Fatalf("after second holder update: got %v, want 1", got)
		}
		first.ProcessService(service("one", 2))
		if got := metric(); got != 2 {
			t.Fatalf("after first holder update: got %v, want 2", got)
		}
	})

	t.Run("a missing service does not replace another holder metric", func(t *testing.T) {
		metric := newMetricReader(t)
		first := NewServiceInfoHolder("first", t.TempDir(), true, true)
		defer first.Close()
		second := NewServiceInfoHolder("second", t.TempDir(), true, true)
		defer second.Close()
		first.ProcessService(service("one", 1))
		first.ProcessService(service("two", 1))
		second.ProcessService(service("three", 1))
		first.StopUpdateIfContain(util.GetGroupName("missing", "DEFAULT_GROUP"), "")
		if got := metric(); got != 1 {
			t.Fatalf("after no-op removal: got %v, want 1", got)
		}
	})
}
