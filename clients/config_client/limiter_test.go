package config_client

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimiter(t *testing.T) {
	key := "abc"
	count := 0
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			limited := IsLimited(key)
			if !limited {
				count++
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, 5, count)
}
