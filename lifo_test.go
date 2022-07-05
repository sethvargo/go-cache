package cache

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNewLIFO(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		cache := NewLIFO[string, string](10)
		defer cache.Stop()

		if got, want := cache.capacity, int64(10); got != want {
			t.Errorf("expected %d to be %d", got, want)
		}
		if got, want := cache.cache, make(map[string]*lifoListItem[string, string], 10); !reflect.DeepEqual(got, want) {
			t.Errorf("expected %#v to be %#v", got, want)
		}
		if got, want := cache.head, (*lifoListItem[string, string])(nil); got != want {
			t.Errorf("expected %#v to be %#v", got, want)
		}
	})

	t.Run("panic_on_negative", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if got, want := fmt.Sprintf("%s", recover()), "capacity must be greater than 0"; got != want {
				t.Errorf("expected %q to contain %q", got, want)
			}
		}()

		cache := NewLIFO[string, string](0)
		defer cache.Stop()

		t.Errorf("did not panic")
	})
}

func TestLIFO_Get(t *testing.T) {
	t.Parallel()

	t.Run("not_exist", func(t *testing.T) {
		t.Parallel()

		cache := NewLIFO[string, int](1)
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

		cache := NewLIFO[string, int](1)
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

func TestLIFO_Set(t *testing.T) {
	t.Parallel()

	t.Run("sets", func(t *testing.T) {
		t.Parallel()

		cache := NewLIFO[string, int](1)
		defer cache.Stop()

		cache.Set("foo", 5)

		if v, _ := cache.Get("foo"); v != 5 {
			t.Errorf("expected %#v, got %#v", 5, v)
		}
	})

	t.Run("evicts", func(t *testing.T) {
		t.Parallel()

		cache := NewLIFO[string, int](2)
		defer cache.Stop()

		cache.Set("foo", 5)
		cache.Set("bar", 4)

		if v, _ := cache.Get("foo"); v != 5 {
			t.Errorf("expected %#v, got %#v", 5, v)
		}
		if v, _ := cache.Get("bar"); v != 4 {
			t.Errorf("expected %#v, got %#v", 5, v)
		}

		cache.Set("baz", 3)

		if v, _ := cache.Get("baz"); v != 3 {
			t.Errorf("expected %#v, got %#v", 3, v)
		}
		if v, ok := cache.Get("bar"); ok {
			t.Errorf("expected %#v to be evicted", v)
		}
		if v, _ := cache.Get("foo"); v != 5 {
			t.Errorf("expected %#v, got %#v", 5, v)
		}
	})
}

func TestLIFO_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("saves", func(t *testing.T) {
		t.Parallel()

		cache := NewLIFO[string, string](3)
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

		cache := NewLIFO[string, string](3)
		defer cache.Stop()

		cache.Set("foo", "bar")

		cache.Fetch("foo", func() (string, error) {
			t.Errorf("function was called")
			return "", nil
		})
	})

	t.Run("returns_error", func(t *testing.T) {
		t.Parallel()

		cache := NewLIFO[string, string](3)
		defer cache.Stop()

		if _, err := cache.Fetch("foo", func() (string, error) {
			return "", fmt.Errorf("error")
		}); err == nil {
			t.Error("expected error")
		}
	})
}

func TestLIFO_Stop(t *testing.T) {
	t.Parallel()

	t.Run("deletes_all_entries", func(t *testing.T) {
		t.Parallel()

		cache := NewLIFO[string, int](1)

		cache.Set("foo", 5)
		cache.Set("bar", 10)
		cache.Set("baz", 15)

		cache.Stop()

		if cache.cache != nil {
			t.Errorf("expected %#v to be nil", cache.cache)
		}
		if cache.head != nil {
			t.Errorf("expected %#v to be nil", cache.head)
		}
	})

	t.Run("panics_get", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if got, want := fmt.Sprintf("%s", recover()), "cache is stopped"; got != want {
				t.Errorf("expected %q to contain %q", got, want)
			}
		}()

		cache := NewLIFO[string, int](10)
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

		cache := NewLIFO[string, int](10)
		cache.Stop()
		cache.Set("foo", 5)
		t.Errorf("did not panic")
	})
}
