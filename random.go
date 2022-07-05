package cache

import (
	"sync"
	"sync/atomic"
)

// Ensure implements.
var _ Cache[string, string] = (*Random[string, string])(nil)

// Random implements a cache in which items are evicted randomly when space is
// needed.
//
// K is the cache key and must be a comparable. V can be any type, but pointers
// are best for performance.
type Random[K comparable, V any] struct {
	// cache represents the internal cache storage.
	cache map[K]V

	// capacity is the total capacity for the cache.
	capacity int64

	// stopped indicates whether the cache is stopped.
	stopped uint32

	// lock is the internal lock for concurrency.
	lock sync.RWMutex
}

// NewRandom creates a new random replacement cache with the given of the given
// capacity.
func NewRandom[K comparable, V any](capacity int64) *Random[K, V] {
	if capacity <= 0 {
		panic("capacity must be greater than 0")
	}

	return &Random[K, V]{
		cache:    make(map[K]V, capacity),
		capacity: capacity,
	}
}

// Get fetches the cache item at the given key. If the value exists, it is
// returned. If the value does not exist, it returns the zero value for the
// object and the second parameter will be false.
func (l *Random[K, V]) Get(key K) (V, bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.get(key)
}

// get is the internal implementation of Get. It does not lock.
func (l *Random[K, V]) get(key K) (V, bool) {
	if l.isStopped() {
		panic("cache is stopped")
	}

	v, ok := l.cache[key]
	return v, ok
}

// Set inserts the value in the cache. If an entry already exists at the given
// key, it is overwritten. If an entry does not exist, a new entry is created
// (which might trigger eviction of an random entry).
func (l *Random[K, V]) Set(key K, val V) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.set(key, val)
}

// set is the internal implementation for set. It does not lock.
func (l *Random[K, V]) set(key K, val V) {
	if l.isStopped() {
		panic("cache is stopped")
	}

	if int64(len(l.cache)) >= l.capacity {
		// Go's map iteration is random on each invocation, so iterate and delete
		// the first element.
		for k := range l.cache {
			delete(l.cache, k)
			break
		}
	}

	l.cache[key] = val
}

// Fetch retrieves the cached value. If the value does not exist, the FetchFunc
// is called and the result is stored. If the value does exist, the FetchFunc is
// not invoked.
func (l *Random[K, V]) Fetch(key K, fn FetchFunc[V]) (V, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.isStopped() {
		panic("cache is stopped")
	}

	if v, ok := l.get(key); ok {
		return v, nil
	}

	v, err := fn()
	if err != nil {
		var zeroV V
		return zeroV, err
	}

	l.set(key, v)
	return v, nil
}

// Stop clears the cache and prevents new entries from being added and
// retrieved.
func (l *Random[K, V]) Stop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	if !atomic.CompareAndSwapUint32(&l.stopped, 0, 1) {
		return
	}

	for k := range l.cache {
		delete(l.cache, k)
	}
	l.cache = nil
}

// isStopped is a helper for checking if the queue is stopped.
func (l *Random[K, V]) isStopped() bool {
	return atomic.LoadUint32(&l.stopped) == 1
}
