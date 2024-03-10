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

package naming_grpc

import (
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"go.uber.org/atomic"
	"sync"
)

const (
	none Type = iota
	register
	unregister
	remove
)

type (
	Type      int32
	IRedoData interface {
		SetRegistered(bool)
		SetUnRegistering(bool)
		Registered()
		Unregistered()
		SetExpectRegistered(bool)
		IsNeedRedo() bool
		GetRedoType() Type
	}
	Data struct {
		locker           *sync.Mutex
		ServiceName      string
		GroupName        string
		registered       *atomic.Bool
		unregistering    *atomic.Bool
		expectRegistered *atomic.Bool
	}

	InstanceRedoData struct {
		*Data
		value model.Instance
	}

	BatchInstancesRedoData struct {
		*Data
		value []model.Instance
	}

	SubscribeRedoData struct {
		*Data
		cluster string
	}
)

func (r *Data) IsNeedRedo() bool {
	return none != r.GetRedoType()
}

func (r *Data) GetRedoType() Type {
	r.locker.Lock()
	defer r.locker.Unlock()
	if r.registered.Load() && !r.unregistering.Load() {
		if r.expectRegistered.Load() {
			return none
		}
		return unregister
	} else if r.registered.Load() && r.unregistering.Load() {
		return unregister
	} else if !r.registered.Load() && !r.unregistering.Load() {
		return register
	} else if r.expectRegistered.Load() {
		return register
	} else {
		return remove
	}
}

func (r *Data) SetRegistered(b bool) {
	r.registered.Store(b)
}

func (r *Data) SetUnRegistering(b bool) {
	r.unregistering.Store(b)
}

func (r *Data) SetExpectRegistered(b bool) {
	r.expectRegistered.Store(b)
}

func (r *Data) Registered() {
	r.locker.Lock()
	defer r.locker.Unlock()
	r.registered.Store(true)
	r.unregistering.Store(false)
}

func (r *Data) Unregistered() {
	r.locker.Lock()
	defer r.locker.Unlock()
	r.registered.Store(false)
	r.unregistering.Store(true)
}

func (i *InstanceRedoData) Get() model.Instance {
	return i.value
}

func (i *BatchInstancesRedoData) Get() []model.Instance {
	return i.value
}

func (i *SubscribeRedoData) Get() string {
	return i.cluster
}

func NewInstanceRedoData(service, group string, ins model.Instance) *InstanceRedoData {
	n := &InstanceRedoData{value: ins}
	d := &Data{
		locker:           &sync.Mutex{},
		ServiceName:      service,
		GroupName:        group,
		registered:       &atomic.Bool{},
		unregistering:    &atomic.Bool{},
		expectRegistered: &atomic.Bool{},
	}
	d.expectRegistered.Store(true)
	n.Data = d
	return n

}

func NewBatchInstancesRedoData(service, group string, ins []model.Instance) *BatchInstancesRedoData {
	d := &Data{
		locker:           &sync.Mutex{},
		ServiceName:      service,
		GroupName:        group,
		registered:       &atomic.Bool{},
		unregistering:    &atomic.Bool{},
		expectRegistered: &atomic.Bool{},
	}
	d.expectRegistered.Store(true)
	n := &BatchInstancesRedoData{value: ins, Data: d}
	return n
}

func NewSubscribeRedoData(service, group, cluster string) *SubscribeRedoData {
	d := &Data{
		locker:           &sync.Mutex{},
		ServiceName:      service,
		GroupName:        group,
		registered:       &atomic.Bool{},
		unregistering:    &atomic.Bool{},
		expectRegistered: &atomic.Bool{},
	}
	d.expectRegistered.Store(true)
	n := &SubscribeRedoData{cluster: cluster, Data: d}

	return n
}
