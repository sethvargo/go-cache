package cache_test

import (
	"fmt"
	"time"

	"github.com/sethvargo/go-cache"
)

func ExampleNewFIFO() {
	fifo := cache.NewFIFO[string, string](15)
	defer fifo.Stop()

	fifo.Set("foo", "bar")
	v, _ := fifo.Get("foo")
	fmt.Println(v) // Output: bar
}

func ExampleNewLIFO() {
	lifo := cache.NewLIFO[string, string](15)
	defer lifo.Stop()

	lifo.Set("foo", "bar")
	v, _ := lifo.Get("foo")
	fmt.Println(v) // Output: bar
}

func ExampleNewLRU() {
	lru := cache.NewLRU[string, string](15)
	defer lru.Stop()

	lru.Set("foo", "bar")
	v, _ := lru.Get("foo")
	fmt.Println(v) // Output: bar
}

func ExampleNewRandom() {
	random := cache.NewRandom[string, string](15)
	defer random.Stop()

	random.Set("foo", "bar")
	v, _ := random.Get("foo")
	fmt.Println(v) // Output: bar
}

func ExampleNewTTL() {
	ttl := cache.NewTTL[string, string](5 * time.Minute)
	defer ttl.Stop()

	ttl.Set("foo", "bar")
	v, _ := ttl.Get("foo")
	fmt.Println(v) // Output: bar
}
