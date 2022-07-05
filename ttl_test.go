package cache

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestNewTTL(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		cache := NewTTL[string, string](5 * time.Minute)
		defer cache.Stop()

		if got, want := cache.ttl, 5*time.Minute; got != want {
			t.Errorf("expected %d to be %d", got, want)
		}
		if got, want := cache.cache, make(map[string]*ttlItem[string, string], 10); !reflect.DeepEqual(got, want) {
			t.Errorf("expected %#v to be %#v", got, want)
		}
	})

	t.Run("panic_on_negative", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if got, want := fmt.Sprintf("%s", recover()), "ttl must be greater than 0"; got != want {
				t.Errorf("expected %q to contain %q", got, want)
			}
		}()

		cache := NewTTL[string, string](0)
		defer cache.Stop()
		t.Errorf("did not panic")
	})
}

func TestTTL_Get(t *testing.T) {
	t.Parallel()

	t.Run("not_exist", func(t *testing.T) {
		t.Parallel()

		cache := NewTTL[string, int](5 * time.Minute)
		defer cache.Stop()

		if v, ok := cache.Get("foo"); ok {
			t.Errorf("expected not found, got %#v", v)
		}
		if v, ok := cache.Get("bar"); ok {
			t.Errorf("expected not found, got %#v", v)
		}

		if got, want := len(cache.cache), 0; got != want {
			t.Errorf("expected %#v to be empty", cache.cache)
		}
	})

	t.Run("exists", func(t *testing.T) {
		t.Parallel()

		cache := NewTTL[string, int](5 * time.Minute)
		defer cache.Stop()

		cache.Set("foo", 5)

		if v, _ := cache.Get("foo"); v != 5 {
			t.Errorf("expected %#v, got %#v", 5, v)
		}
		if v, ok := cache.Get("bar"); ok {
			t.Errorf("expected not found, got %#v", v)
		}

		if got, want := len(cache.cache), 1; got != want {
			t.Errorf("expected %#v to be empty", cache.cache)
		}
	})
}

func TestTTL_Set(t *testing.T) {
	t.Parallel()

	t.Run("sets", func(t *testing.T) {
		t.Parallel()

		cache := NewTTL[string, int](5 * time.Minute)
		defer cache.Stop()

		cache.Set("foo", 5)

		if v, _ := cache.Get("foo"); v != 5 {
			t.Errorf("expected %#v, got %#v", 5, v)
		}
	})

	t.Run("evicts", func(t *testing.T) {
		t.Parallel()

		cache := NewTTL[string, int](50 * time.Millisecond)
		defer cache.Stop()

		cache.Set("foo", 5)
		cache.Set("bar", 4)

		if v, _ := cache.Get("foo"); v != 5 {
			t.Errorf("expected %#v, got %#v", 5, v)
		}
		if v, _ := cache.Get("bar"); v != 4 {
			t.Errorf("expected %#v, got %#v", 5, v)
		}

		time.Sleep(50 * time.Millisecond)

		if v, ok := cache.Get("foo"); ok {
			t.Errorf("expected %#v to be evicted", v)
		}
		if v, ok := cache.Get("bar"); ok {
			t.Errorf("expected %#v to be evicted", v)
		}
	})
}

func TestTTL_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("saves", func(t *testing.T) {
		t.Parallel()

		cache := NewTTL[string, string](50 * time.Millisecond)
		defer cache.Stop()

		v, err := cache.Fetch("foo", func() (string, error) {
			return "bar", nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if got, want := v, "bar"; got != want {
			t.Errorf("expected %q to eb %q", got, want)
		}

		v, ok := cache.Get("foo")
		if !ok {
			t.Errorf("expected item to be cached")
		}
		if got, want := v, "bar"; got != want {
			t.Errorf("expected %q to eb %q", got, want)
		}
	})

	t.Run("returns_cached", func(t *testing.T) {
		t.Parallel()

		cache := NewTTL[string, string](50 * time.Millisecond)
		defer cache.Stop()

		cache.Set("foo", "bar")

		cache.Fetch("foo", func() (string, error) {
			t.Errorf("function was called")
			return "", nil
		})
	})

	t.Run("returns_error", func(t *testing.T) {
		t.Parallel()

		cache := NewTTL[string, string](50 * time.Millisecond)
		defer cache.Stop()

		if _, err := cache.Fetch("foo", func() (string, error) {
			return "", fmt.Errorf("error")
		}); err == nil {
			t.Error("expected error")
		}
	})
}

func TestTTL_Stop(t *testing.T) {
	t.Parallel()

	t.Run("deletes_all_entries", func(t *testing.T) {
		t.Parallel()

		cache := NewTTL[string, int](5 * time.Minute)
		cache.Set("foo", 5)
		cache.Set("bar", 10)
		cache.Set("baz", 15)

		cache.Stop()

		if cache.cache != nil {
			t.Errorf("expected %#v to be nil", cache.cache)
		}
	})

	t.Run("panics_get", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if got, want := fmt.Sprintf("%s", recover()), "cache is stopped"; got != want {
				t.Errorf("expected %q to contain %q", got, want)
			}
		}()

		cache := NewTTL[string, int](5 * time.Minute)
		cache.Stop()
		cache.Get("foo")
		t.Errorf("did not panic")
	})

	t.Run("panics_set", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if got, want := fmt.Sprintf("%s", recover()), "cache is stopped"; got != want {
				t.Errorf("expected %q to contain %q", got, want)
			}
		}()

		cache := NewTTL[string, int](5 * time.Minute)
		cache.Stop()
		cache.Set("foo", 5)
		t.Errorf("did not panic")
	})
}
