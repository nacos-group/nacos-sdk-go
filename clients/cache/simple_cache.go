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

package cache

import (
	"sync"
)

type ICache[K comparable, V any] interface {
	Load(key K) (V, bool)
	Store(key K, value V)
	LoadOrStore(key K, value V) (V, bool)
	LoadOrStoreFunc(key K, apply func() V) (V, bool)
	LoadAndDelete(key K) (V, bool)
	Delete(key K)
	Range(func(key K, value V) bool)
	Size() int
	Empty() bool
}

type IComputeCache[K comparable, V any] interface {
	ICache[K, V]
	Compute(key K, apply func(value V) V) V
	ComputeIfAbsent(key K, apply func() V) V
	ComputeIfPresent(key K, apply func(value V) V) V
}

// SimpleCache k,v must both be comparable
type SimpleCache[K comparable, V any] struct {
	locker sync.RWMutex
	m      sync.Map
}

func NewCache[K comparable, V any]() *SimpleCache[K, V] {
	return &SimpleCache[K, V]{}
}

func (s *SimpleCache[K, V]) Load(key K) (V, bool) {
	value, ok := s.m.Load(key)
	if ok {
		return value.(V), ok
	}
	var empty V
	return empty, ok
}

func (s *SimpleCache[K, V]) Store(key K, value V) {
	s.locker.RLock()
	defer s.locker.RUnlock()
	s.m.Store(key, value)
}

func (s *SimpleCache[K, V]) LoadOrStore(key K, value V) (V, bool) {
	s.locker.RLock()
	defer s.locker.RUnlock()
	actual, loaded := s.m.LoadOrStore(key, value)
	return actual.(V), loaded

}

func (s *SimpleCache[K, V]) LoadOrStoreFunc(key K, apply func() V) (V, bool) {
	actual, loaded := s.m.Load(key)
	if loaded {
		return actual.(V), loaded
	}
	s.locker.Lock()
	defer s.locker.Unlock()
	actual, loaded = s.m.Load(key)
	if loaded {
		return actual.(V), loaded
	}
	value := apply()
	s.m.Store(key, value)
	return value, loaded

}

func (s *SimpleCache[K, V]) LoadAndDelete(key K) (V, bool) {
	s.locker.RLock()
	defer s.locker.RUnlock()
	value, loaded := s.m.LoadAndDelete(key)
	if loaded {
		return value.(V), loaded
	}
	var empty V
	return empty, loaded
}

func (s *SimpleCache[K, V]) Delete(key K) {
	s.locker.RLock()
	defer s.locker.RUnlock()
	s.m.Delete(key)
}

func (s *SimpleCache[K, V]) Range(f func(key K, value V) bool) {
	s.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func (s *SimpleCache[K, V]) Size() int {
	count := 0
	s.m.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func (s *SimpleCache[K, V]) Empty() bool {
	empty := true
	s.m.Range(func(key, value any) bool {
		empty = false
		return false
	})
	return empty
}
func (s *SimpleCache[K, V]) Compute(key K, apply func(value V) V) V {
	s.locker.Lock()
	defer s.locker.Unlock()
	old, ok := s.m.Load(key)
	var empty, newValue V
	if !ok {
		newValue = apply(empty)
	} else {
		newValue = apply(old.(V))
	}
	s.m.Store(key, newValue)
	return newValue
}

func (s *SimpleCache[K, V]) ComputeIfAbsent(key K, apply func() V) V {
	old, ok := s.m.Load(key)
	if ok {
		return old.(V)
	}
	s.locker.Lock()
	defer s.locker.Unlock()
	old, ok = s.m.Load(key)
	if ok {
		return old.(V)
	}
	newValue := apply()
	s.m.Store(key, newValue)
	return newValue
}

func (s *SimpleCache[K, V]) ComputeIfPresent(key K, apply func(value V) V) V {
	var empty V
	_, ok := s.m.Load(key)
	if !ok {
		return empty
	}
	s.locker.Lock()
	defer s.locker.Unlock()
	old, ok := s.m.Load(key)
	if !ok {
		return empty
	}
	newValue := apply(old.(V))
	s.m.Store(key, newValue)
	return newValue
}
