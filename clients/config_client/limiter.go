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
