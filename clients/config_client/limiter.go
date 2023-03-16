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

package config_client

import (
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/cache"
	"golang.org/x/time/rate"
)

type rateLimiterCheck struct {
	rateLimiterCache cache.ConcurrentMap // cache
	mux              sync.Mutex
}

var checker rateLimiterCheck

func init() {
	checker = rateLimiterCheck{
		rateLimiterCache: cache.NewConcurrentMap(),
		mux:              sync.Mutex{},
	}
}

// IsLimited return true when request is limited
func IsLimited(checkKey string) bool {
	checker.mux.Lock()
	defer checker.mux.Unlock()
	var limiter *rate.Limiter
	lm, exist := checker.rateLimiterCache.Get(checkKey)
	if !exist {
		// define a new limiter,allow 5 times per second,and reserve stock is 5.
		limiter = rate.NewLimiter(rate.Limit(5), 5)
		checker.rateLimiterCache.Set(checkKey, limiter)
	} else {
		limiter = lm.(*rate.Limiter)
	}
	add := time.Now().Add(time.Second)
	return !limiter.AllowN(add, 1)
}
