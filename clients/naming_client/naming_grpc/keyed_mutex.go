package naming_grpc

import "sync"

// keyedMutex serializes the operations that must not overlap for the same key
// while keeping different keys independent. The zero value is ready to use.
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*keyedMutexEntry
}

type keyedMutexEntry struct {
	mu   sync.Mutex
	refs int
}

// Lock acquires the lock of key and returns the function that releases it.
func (k *keyedMutex) Lock(key string) func() {
	k.mu.Lock()
	entry := k.locks[key]
	if entry == nil {
		if k.locks == nil {
			k.locks = make(map[string]*keyedMutexEntry)
		}
		entry = &keyedMutexEntry{}
		k.locks[key] = entry
	}
	entry.refs++
	k.mu.Unlock()

	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()
		k.mu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}
