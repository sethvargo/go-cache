package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

// Ensure implements.
var _ Cache[string, string] = (*TTL[string, string])(nil)

// TTL implements a cache in which items are evicted when they have lived in the
// cached beyond an expiration.
//
// K is the cache key and must be a comparable. V can be any type, but pointers
// are best for performance.
type TTL[K comparable, V any] struct {
	// cache represents the internal cache storage.
	cache map[K]*ttlListItem[K, V]

	// head points to the head of the linked list and tail points to the tail.
	head, tail *ttlListItem[K, V]

	// ttl is the global TTL value.
	ttl time.Duration

	// stopped indicates whether the cache is stopped. stopCh is a channel used to
	// control cancellation.
	stopped uint32
	stopCh  chan struct{}

	// lock is the internal lock to allow for concurrent operations.
	lock sync.RWMutex
}

// NewTTL creates a new TTL cache with the given of the given TTL. The TTL
// applies for all entries in the cache. Items are not guaranteed to be purged
// from the cache at their exact expiration time, but they are guaranteed to not
// be returned past their expiration time. The sweeping operation runs on
// quarterstep intervals of the provided TTL.
func NewTTL[K comparable, V any](ttl time.Duration) *TTL[K, V] {
	if ttl <= 0 {
		panic("ttl must be greater than 0")
	}

	c := &TTL[K, V]{
		cache:  make(map[K]*ttlListItem[K, V], 16),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}

	// Start the sweep!
	sweep := ttl / 4
	if min := 50 * time.Millisecond; sweep < min {
		sweep = min
	}
	go c.start(sweep)

	return c
}

// Get fetches the cache item at the given key. If the value exists, it is
// returned. If the value does not exist, it returns the zero value for the
// object and the second parameter will be false.
func (l *TTL[K, V]) Get(key K) (V, bool) {
	now := time.Now().UTC()
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.get(key, now)
}

// get is the internal implementation of Get. It does not lock.
func (l *TTL[K, V]) get(key K, now time.Time) (V, bool) {
	if l.isStopped() {
		panic("cache is stopped")
	}

	v, ok := l.cache[key]
	if !ok || v.expiresAt.Before(now) {
		var zeroV V
		return zeroV, false
	}
	return v.value, true
}

// Set inserts the value in the cache. If an entry already exists at the given
// key, it is overwritten. If an entry does not exist, a new entry is created.
func (l *TTL[K, V]) Set(key K, val V) {
	now := time.Now().UTC()
	l.lock.Lock()
	defer l.lock.Unlock()
	l.set(key, val, now)
}

// set is the internal implementation for set. It does not lock.
func (l *TTL[K, V]) set(key K, val V, now time.Time) {
	if l.isStopped() {
		panic("cache is stopped")
	}

	node, ok := l.cache[key]
	if !ok {
		node = &ttlListItem[K, V]{
			key: &key,
		}
		l.cache[key] = node
	}
	node.value = val
	node.expiresAt = ptrTo(now.Add(l.ttl))

	// If this is the first entry in the cache, update the head.
	if l.head == nil {
		l.head = node
	}

	// This entry is new, so add it to the end of the list.
	if l.tail != nil {
		l.tail.next = node
	}
	l.tail = node
}

// Fetch retrieves the cached value. If the value does not exist, the FetchFunc
// is called and the result is stored. If the value does exist, the FetchFunc is
// not invoked.
func (l *TTL[K, V]) Fetch(key K, fn FetchFunc[V]) (V, error) {
	now := time.Now().UTC()

	l.lock.Lock()
	defer l.lock.Unlock()

	if l.isStopped() {
		panic("cache is stopped")
	}

	if v, ok := l.get(key, now); ok {
		return v, nil
	}

	v, err := fn()
	if err != nil {
		var zeroV V
		return zeroV, err
	}

	l.set(key, v, now)
	return v, nil
}

// Stop clears the cache and prevents new entries from being added and
// retrieved.
func (l *TTL[K, V]) Stop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	if !atomic.CompareAndSwapUint32(&l.stopped, 0, 1) {
		return
	}
	close(l.stopCh)

	for k, v := range l.cache {
		var zeroV V
		v.key = nil
		v.value = zeroV
		v.expiresAt = nil
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
func (l *TTL[K, V]) isStopped() bool {
	return atomic.LoadUint32(&l.stopped) == 1
}

// start begins the background reaping process for expired entries. It runs
// until stopped via Stop() and is intended to be called as a goroutine.
func (l *TTL[K, V]) start(sweep time.Duration) {
	ticker := time.NewTicker(sweep)
	defer ticker.Stop()

	for {
		// Check if we're stopped first to prevent entering a race between a short
		// time ticker and the stop channel.
		if l.isStopped() {
			return
		}

		select {
		case <-l.stopCh:
			return
		case <-ticker.C:
			func() {
				now := time.Now().UTC()

				l.lock.Lock()
				defer l.lock.Unlock()

				// Walk the LinkedList from the front, since those are the oldest items.
				node := l.head
				for node != nil {
					// If this item isn't a candidate for expiration, then no future items
					// will be a candidate either, since they are in increasing order.
					if node.expiresAt.After(now) {
						break
					}

					delete(l.cache, *node.key)

					var zeroV V
					node.key = nil
					node.value = zeroV
					node.expiresAt = nil
					node, node.next = node.next, nil
				}

				l.head = node
				if node == nil {
					l.tail = nil
				}
			}()
		}
	}
}

// ttlListItem represents an entry in the linked list.
type ttlListItem[K comparable, V any] struct {
	next      *ttlListItem[K, V]
	key       *K
	value     V
	expiresAt *time.Time
}
