package cache

import (
	"sync"
	"sync/atomic"
)

// Ensure implements.
var _ Cache[string, string] = (*FIFO[string, string])(nil)

// FIFO implements the first-in-first-out cache algorithm, evicting the cache
// elements in the order in which they were inserted with the cache is at
// capacity.
//
// K is the cache key and must be a comparable. V can be any type, but pointers
// are best for performance.
type FIFO[K comparable, V any] struct {
	// cache represents the internal cache storage. It has a comparable key and
	// points to an entry in the singly-linked list. The node in the linked list
	// contains the actual cached data.
	cache map[K]*fifoListItem[K, V]

	// head points to the head of the linked list and tail points to the tail.
	head, tail *fifoListItem[K, V]

	// capacity is the total capacity for the cache.
	capacity int64

	// stopped indicates whether the cache is stopped.
	stopped uint32

	// lock is the internal lock for concurrency.
	lock sync.RWMutex
}

// NewFIFO creates a new FIFO cache with the given of the given capacity.
func NewFIFO[K comparable, V any](capacity int64) *FIFO[K, V] {
	if capacity <= 0 {
		panic("capacity must be greater than 0")
	}

	return &FIFO[K, V]{
		cache:    make(map[K]*fifoListItem[K, V], capacity),
		capacity: capacity,
	}
}

// Get fetches the cache item at the given key. If the value exists, it is
// returned. If the value does not exist, it returns the zero value for the
// object and the second parameter will be false.
func (l *FIFO[K, V]) Get(key K) (V, bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.get(key)
}

// get is the internal implementation of Get. It does not lock.
func (l *FIFO[K, V]) get(key K) (V, bool) {
	if l.isStopped() {
		panic("cache is stopped")
	}

	node, ok := l.cache[key]
	if !ok {
		var v V
		return v, false
	}
	return node.value, true
}

// Set inserts the value in the cache. If an entry already exists at the given
// key, it is overwritten. If an entry does not exist, a new entry is created
// (which might trigger eviction of an older entry).
func (l *FIFO[K, V]) Set(key K, val V) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.set(key, val)
}

// set is the internal implementation for set. It does not lock.
func (l *FIFO[K, V]) set(key K, val V) {
	if l.isStopped() {
		panic("cache is stopped")
	}

	if int64(len(l.cache)) >= l.capacity {
		head := l.head
		next := head.next

		delete(l.cache, *head.key)

		// Zero out the old node to improve gc sweeps.
		var zeroK *K
		var zeroV V
		head.key = zeroK
		head.value = zeroV
		head.next = nil

		l.head = next
	}

	node, ok := l.cache[key]
	if !ok {
		node = &fifoListItem[K, V]{
			key: &key,
		}
		l.cache[key] = node

		// This entry is new, so add it to the end of the list.
		if l.tail != nil {
			l.tail.next = node
		}
		l.tail = node

		// If this is the first entry in the cache, update the head.
		if l.head == nil {
			l.head = node
		}
	}
	node.value = val
}

// Fetch retrieves the cached value. If the value does not exist, the FetchFunc
// is called and the result is stored. If the value does exist, the FetchFunc is
// not invoked.
func (l *FIFO[K, V]) Fetch(key K, fn FetchFunc[V]) (V, error) {
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
func (l *FIFO[K, V]) Stop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	if !atomic.CompareAndSwapUint32(&l.stopped, 0, 1) {
		return
	}

	for k := range l.cache {
		delete(l.cache, k)
	}
	l.cache = nil

	var zeroK *K
	var zeroV V

	node := l.head
	for node != nil {
		node.key = zeroK
		node.value = zeroV
		node, node.next = node.next, nil
	}

	l.head = nil
	l.tail = nil
}

// isStopped is a helper for checking if the queue is stopped.
func (l *FIFO[K, V]) isStopped() bool {
	return atomic.LoadUint32(&l.stopped) == 1
}

// fifoListItem represents an entry in the linked list.
type fifoListItem[K comparable, V any] struct {
	next  *fifoListItem[K, V]
	key   *K
	value V
}
