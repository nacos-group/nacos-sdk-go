package cache

import (
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
)

type valueDemo struct {
	name string
}

var cacheNotPoint IComputeCache[string, string]
var cachePoint IComputeCache[string, *valueDemo]

func TestMain(t *testing.M) {
	cacheNotPoint = NewCache[string, string]()
	cachePoint = NewCache[string, *valueDemo]()
	t.Run()
}

func TestComputeNotPoint(t *testing.T) {
	defer cacheNotPoint.Delete("TestComputeNotPoint")
	v := "not empty"
	count := atomic.Uint32{}
	computeFunc := func(value string) string {
		count.Add(1)
		if len(value) == 0 {
			return v
		}
		return value
	}
	result1 := cacheNotPoint.Compute("TestComputeNotPoint", computeFunc)
	result2 := cacheNotPoint.Compute("TestComputeNotPoint", computeFunc)
	at := assert.New(t)
	at.EqualValues(2, count.Load())
	at.EqualValues(result1, result2)
	at.EqualValues(result1, v)
}

func TestComputePoint(t *testing.T) {
	defer cachePoint.Delete("TestComputePoint")
	v := &valueDemo{name: "init"}
	count := atomic.Uint32{}
	computeFunc := func(value *valueDemo) *valueDemo {
		count.Add(1)
		if value == nil {
			return v
		}
		return value
	}
	result1 := cachePoint.Compute("TestComputePoint", computeFunc)
	result2 := cachePoint.Compute("TestComputePoint", computeFunc)
	at := assert.New(t)
	at.EqualValues(2, count.Load())
	at.Equal(result1, result2)
	at.Equal(result1, v)
	v.name = "finish"
	at.Equal(v, result1)
}

func TestComputeIfAbsentNotPoint(t *testing.T) {
	defer cacheNotPoint.Delete("TestComputeIfAbsentNotPoint1")
	defer cacheNotPoint.Delete("TestComputeIfAbsentNotPoint2")
	init := "init"
	v := "computed"
	count := atomic.Uint32{}
	computeFunc := func() string {
		count.Add(1)
		return v
	}
	cacheNotPoint.Store("TestComputeIfAbsentNotPoint1", init)
	result1 := cacheNotPoint.ComputeIfAbsent("TestComputeIfAbsentNotPoint1", computeFunc)
	result2 := cacheNotPoint.ComputeIfAbsent("TestComputeIfAbsentNotPoint2", computeFunc)
	at := assert.New(t)
	at.EqualValues(1, count.Load())
	at.EqualValues(init, result1)
	at.NotEqualValues(result1, v)
	at.EqualValues(v, result2)
}

func TestComputeIfAbsentPoint(t *testing.T) {
	defer cachePoint.Delete("TestComputeIfAbsentPoint1")
	defer cachePoint.Delete("TestComputeIfAbsentPoint2")
	init := &valueDemo{name: "init"}
	cachePoint.Store("TestComputeIfAbsentPoint1", init)
	compute := &valueDemo{name: "compute"}
	count := atomic.Uint32{}
	computeFunc := func() *valueDemo {
		count.Add(1)
		return compute
	}
	result1 := cachePoint.ComputeIfAbsent("TestComputeIfAbsentPoint1", computeFunc)
	result2 := cachePoint.ComputeIfAbsent("TestComputeIfAbsentPoint2", computeFunc)
	at := assert.New(t)
	at.EqualValues(1, count.Load())
	at.Equal(result1, init)
	at.Equal(result2, compute)
	compute.name = "finish"
	at.Equal(compute, result2)
}

func TestComputeIfPresentNotPoint(t *testing.T) {
	defer cacheNotPoint.Delete("TestComputeIfPresentNotPoint1")
	defer cacheNotPoint.Delete("TestComputeIfPresentNotPoint2")
	init := "init"
	v := "computed"
	count := atomic.Uint32{}
	computeFunc := func(value string) string {
		count.Add(1)
		return value + v
	}
	cacheNotPoint.Store("TestComputeIfPresentNotPoint1", init)
	result1 := cacheNotPoint.ComputeIfPresent("TestComputeIfPresentNotPoint1", computeFunc)
	result2 := cacheNotPoint.ComputeIfPresent("TestComputeIfPresentNotPoint2", computeFunc)
	at := assert.New(t)
	at.EqualValues(1, count.Load())
	at.EqualValues(init+v, result1)
	at.Empty(result2)
}

func TestComputeIfPresentPoint(t *testing.T) {
	defer cachePoint.Delete("TestComputeIfPresentPoint1")
	defer cachePoint.Delete("TestComputeIfPresentPoint2")
	init := &valueDemo{name: "init"}
	cachePoint.Store("TestComputeIfPresentPoint1", init)
	count := atomic.Uint32{}
	computeFunc := func(v *valueDemo) *valueDemo {
		count.Add(1)
		v.name = "compute"
		return v
	}
	result1 := cachePoint.ComputeIfPresent("TestComputeIfPresentPoint1", computeFunc)
	result2 := cachePoint.ComputeIfPresent("TestComputeIfPresentPoint2", computeFunc)
	at := assert.New(t)
	at.EqualValues(1, count.Load())
	at.Equal(init, result1)
	at.Nil(result2)
	init.name = "finish"
	at.Equal(init, result1)
}

func TestCRUDNotPoint(t *testing.T) {
	var empty string
	load, ok := cacheNotPoint.Load("TestCRUDNotPoint1")
	at := assert.New(t)
	at.EqualValues(empty, load)
	at.False(ok)
	at.Equal(0, cacheNotPoint.Size())
	at.True(cacheNotPoint.Empty())

	cacheNotPoint.Store("TestCRUDNotPoint1", "1")
	load, ok = cacheNotPoint.Load("TestCRUDNotPoint1")
	at.True(ok)
	at.Equal("1", load)
	at.Equal(1, cacheNotPoint.Size())
	at.False(cacheNotPoint.Empty())
	cacheNotPoint.Delete("TestCRUDNotPoint1")
	at.Equal(0, cacheNotPoint.Size())
	at.True(cacheNotPoint.Empty())

	load, ok = cacheNotPoint.Load("TestCRUDNotPoint1")
	at.EqualValues(empty, load)
	at.False(ok)

	at.NotPanics(func() { cacheNotPoint.Delete("TestCRUDNotPoint2") })
	at.Equal(0, cacheNotPoint.Size())
	at.True(cacheNotPoint.Empty())
}

func TestCRUDPoint(t *testing.T) {
	load, ok := cachePoint.Load("TestCRUDPoint1")
	at := assert.New(t)
	at.Nil(load)
	at.False(ok)
	at.Equal(0, cachePoint.Size())
	at.True(cachePoint.Empty())

	v := &valueDemo{name: "init"}
	cachePoint.Store("TestCRUDPoint1", v)
	load, ok = cachePoint.Load("TestCRUDPoint1")
	at.True(ok)
	at.Equal(v, load)
	at.Equal(1, cachePoint.Size())
	at.False(cachePoint.Empty())

	cachePoint.Delete("TestCRUDPoint1")
	at.Equal(0, cachePoint.Size())
	at.True(cachePoint.Empty())

	load, ok = cachePoint.Load("TestCRUDPoint1")
	at.Nil(load)
	at.False(ok)

	at.NotPanics(func() { cachePoint.Delete("TestCRUDPoint2") })
	at.Equal(0, cachePoint.Size())
	at.True(cachePoint.Empty())
}

func TestLoadAndOpsNotPoint(t *testing.T) {
	at := assert.New(t)
	value, loaded := cacheNotPoint.LoadOrStore("TestLoadAndOpsNotPoint1", "1")
	at.False(loaded)
	at.Equal("1", value)

	value, loaded = cacheNotPoint.LoadOrStore("TestLoadAndOpsNotPoint1", "2")
	at.True(loaded)
	at.Equal("1", value)

	value, deleted := cacheNotPoint.LoadAndDelete("TestLoadAndOpsNotPoint1")
	at.True(deleted)
	at.Equal("1", value)

	value, deleted = cacheNotPoint.LoadAndDelete("TestLoadAndOpsNotPoint1")
	at.False(deleted)
	at.Empty(value)

	value, deleted = cacheNotPoint.LoadAndDelete("TestLoadAndOpsNotPoint2")
	at.False(deleted)
	at.Empty(value)

	value, loaded = cacheNotPoint.LoadOrStoreFunc("TestLoadAndOpsNotPoint1", func() string {
		return "2"
	})
	at.False(loaded)
	at.Equal("2", value)

	value, loaded = cacheNotPoint.LoadOrStoreFunc("TestLoadAndOpsNotPoint1", func() string {
		return "3"
	})
	at.True(loaded)
	at.Equal("2", value)
	cacheNotPoint.Delete("TestLoadAndOpsNotPoint1")
}

func TestLoadAndOpsPoint(t *testing.T) {
	at := assert.New(t)
	init := &valueDemo{name: "init"}
	changed := &valueDemo{name: "changed"}
	value, loaded := cachePoint.LoadOrStore("TestLoadAndOpsPoint1", init)
	at.False(loaded)
	at.Equal(init, value)

	value, loaded = cachePoint.LoadOrStore("TestLoadAndOpsPoint1", changed)
	at.True(loaded)
	at.Equal(init, value)

	value, deleted := cachePoint.LoadAndDelete("TestLoadAndOpsPoint1")
	at.True(deleted)
	at.Equal(init, value)

	value, deleted = cachePoint.LoadAndDelete("TestLoadAndOpsPoint1")
	at.False(deleted)
	at.Nil(value)

	value, deleted = cachePoint.LoadAndDelete("TestLoadAndOpsPoint2")
	at.False(deleted)
	at.Nil(value)

	value, loaded = cachePoint.LoadOrStoreFunc("TestLoadAndOpsPoint1", func() *valueDemo {
		return init
	})
	at.False(loaded)
	at.Equal(init, value)

	value, loaded = cachePoint.LoadOrStoreFunc("TestLoadAndOpsPoint1", func() *valueDemo {
		return changed
	})
	at.True(loaded)
	at.Equal(init, value)
	cachePoint.Delete("TestLoadAndOpsPoint1")
}

func TestCompareAndOpsNotPoint(t *testing.T) {
	at := assert.New(t)
	cacheNotPoint.Store("TestCompareAndOpsNotPoint1", "1")
	at.True(cacheNotPoint.CompareAndSwap("TestCompareAndOpsNotPoint1", "1", "2"))

	at.False(cacheNotPoint.CompareAndSwap("TestCompareAndOpsNotPoint1", "1", "3"))
	value, ok := cacheNotPoint.Load("TestCompareAndOpsNotPoint1")
	at.Equal("2", value)
	at.True(ok)

	called := 0
	at.True(cacheNotPoint.CompareAndSwapFunc("TestCompareAndOpsNotPoint1", "2", func() string {
		called++
		return "3"
	}))
	at.Equal(1, called)
	at.False(cacheNotPoint.CompareAndSwapFunc("TestCompareAndOpsNotPoint1", "2", func() string {
		called++
		return "4"
	}))
	at.Equal(1, called)

	value, ok = cacheNotPoint.Load("TestCompareAndOpsNotPoint1")
	at.Equal("3", value)
	at.True(ok)

	at.False(cacheNotPoint.CompareAndDelete("TestCompareAndOpsNotPoint1", "4"))
	_, ok = cacheNotPoint.Load("TestCompareAndOpsNotPoint1")
	at.True(ok)

	at.True(cacheNotPoint.CompareAndDelete("TestCompareAndOpsNotPoint1", "3"))
	_, ok = cacheNotPoint.Load("TestCompareAndOpsNotPoint1")
	at.False(ok)

	cacheNotPoint.Delete("TestCompareAndOpsNotPoint1")
}

func TestCompareAndOpsPoint(t *testing.T) {
	at := assert.New(t)
	init := &valueDemo{name: "init"}
	changed1 := &valueDemo{name: "changed1"}
	changed2 := &valueDemo{name: "changed2"}
	cachePoint.Store("TestCompareAndOpsPoint1", init)
	at.True(cachePoint.CompareAndSwap("TestCompareAndOpsPoint1", init, changed1))

	at.False(cachePoint.CompareAndSwap("TestCompareAndOpsPoint1", init, changed2))
	value, ok := cachePoint.Load("TestCompareAndOpsPoint1")
	at.Equal(changed1, value)
	at.True(ok)

	called := 0
	at.True(cachePoint.CompareAndSwapFunc("TestCompareAndOpsPoint1", changed1, func() *valueDemo {
		called++
		return changed2
	}))
	at.Equal(1, called)
	changed3 := &valueDemo{name: "changed3"}
	at.False(cachePoint.CompareAndSwapFunc("TestCompareAndOpsPoint1", changed1, func() *valueDemo {
		called++
		return changed3
	}))
	at.Equal(1, called)

	value, ok = cachePoint.Load("TestCompareAndOpsPoint1")
	at.Equal(changed2, value)
	at.True(ok)

	at.False(cachePoint.CompareAndDelete("TestCompareAndOpsPoint1", changed3))
	_, ok = cachePoint.Load("TestCompareAndOpsPoint1")
	at.True(ok)

	at.True(cachePoint.CompareAndDelete("TestCompareAndOpsPoint1", changed2))
	_, ok = cachePoint.Load("TestCompareAndOpsPoint1")
	at.False(ok)

	cachePoint.Delete("TestCompareAndOpsPoint1")
}

func TestSwapNotPoint(t *testing.T) {
	at := assert.New(t)
	value, ok := cacheNotPoint.Swap("TestSwapNotPoint1", "1")

	at.Empty(value)
	at.False(ok)

	value, ok = cacheNotPoint.Swap("TestSwapNotPoint1", "2")
	at.Equal("1", value)
	at.True(ok)

	cacheNotPoint.Delete("TestSwapNotPoint1")
}

func TestSwapPoint(t *testing.T) {
	at := assert.New(t)
	init := &valueDemo{name: "init"}
	value, ok := cachePoint.Swap("TestSwapPoint1", init)

	at.Nil(value)
	at.False(ok)

	changed := &valueDemo{name: "changed"}
	value, ok = cachePoint.Swap("TestSwapPoint1", changed)
	at.Equal(init, value)
	at.True(ok)

	cachePoint.Delete("TestSwapPoint1")
}

func TestRangeNotPoint(t *testing.T) {
	at := assert.New(t)
	called := 0
	rangeFunc := func(key string, value string) bool {
		called++
		return true
	}
	cacheNotPoint.Range(rangeFunc)

	at.Zero(called)

	cacheNotPoint.Store("TestRangeNotPoint1", "1")
	cacheNotPoint.Range(rangeFunc)
	at.Equal(1, called)

	cacheNotPoint.Store("TestRangeNotPoint2", "2")
	at.Equal(2, cacheNotPoint.Size())
	called = 0
	cacheNotPoint.Range(func(key string, value string) bool {
		called++
		return false
	})
	at.Equal(1, called)
}

func TestRangePoint(t *testing.T) {
	at := assert.New(t)
	called := 0
	rangeFunc := func(key string, value *valueDemo) bool {
		called++
		return true
	}
	cachePoint.Range(rangeFunc)

	at.Zero(called)

	cachePoint.Store("TestRangePoint1", &valueDemo{})
	cachePoint.Range(rangeFunc)
	at.Equal(1, called)

	cachePoint.Store("TestRangePoint2", &valueDemo{})
	at.Equal(2, cachePoint.Size())
	called = 0
	cachePoint.Range(func(key string, value *valueDemo) bool {
		called++
		return false
	})
	at.Equal(1, called)
}
