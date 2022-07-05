// Package cache implements a collection of caching algorithms in Go which
// support generics for strong typing. The implementations are the purely
// "academic" definitions of the cache algorithms, and more finely-tuned
// libraries might be a better fit for high-throughput or high-storage use
// cases.
//
// In addition to the standard Get and Set functions, there is also a
// package-level Fetch function which acts as a write-through operation:
//
//     lru := cache.NewLRU[string, string](15)
//
//     v, err := cache.Fetch(lru, "foo", func() (string, error)) {
//       return "bar", nil
//     }
//     if err != nil {
//       // TODO: handle error
//     }
//     fmt.Println(v) // Output: bar
//
// By default, none of the implementations are safe for concurrent use. To make
// a cache as safe for concurrent use, wrap it in the sync cache:
//
//     lruSync := cache.NewSync[string, string](cache.NewLRU[string, string](15))
//
// Unfortunately Go does not currently infer the type constraint from the input,
// so you must declare it twice.
package cache

// Cache is a generic interface for various cache implementations.
type Cache[K comparable, V any] interface {
	// Get retrives the given key from the cache. If the item exists, it is
	// returned. If it does not exist, the second argument will be false.
	Get(K) (V, bool)

	// Set inserts the given key into the cache. If the key already exists, it
	// will be overwritten.
	Set(K, V)

	// Fetch retrieves the cached value. If the value does not exist, the
	// FetchFunc is called and the result is stored. If the value does exist, the
	// FetchFunc is not invoked.
	Fetch(K, FetchFunc[V]) (V, error)

	// Stop terminates the cache, deleting any cached entries. Once invoked, any
	// future calls to Get or Set will panic.
	Stop()
}

// FetchFunc is a function that is invoked when a cached value is not found.
type FetchFunc[V any] func() (V, error)

// ptrTo is a helper for returning the pointer to a type.
func ptrTo[V any](v V) *V {
	return &v
}
