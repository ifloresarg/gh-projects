package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCacheHit(t *testing.T) {
	c := New[string](1 * time.Minute)
	c.Set("key1", "value1")

	val, ok := c.Get("key1")
	if !ok {
		t.Fatalf("expected cache hit, got miss")
	}
	if val != "value1" {
		t.Fatalf("expected 'value1', got %q", val)
	}
}

func TestCacheExpiry(t *testing.T) {
	c := New[string](100 * time.Millisecond)
	c.Set("key1", "value1")

	// Immediate get should hit
	val, ok := c.Get("key1")
	if !ok {
		t.Fatalf("expected immediate cache hit, got miss")
	}
	if val != "value1" {
		t.Fatalf("expected 'value1', got %q", val)
	}

	// Wait for expiry
	time.Sleep(200 * time.Millisecond)

	// Get should miss after expiry
	_, ok = c.Get("key1")
	if ok {
		t.Fatalf("expected cache miss after expiry, got hit")
	}
}

func TestCacheInvalidate(t *testing.T) {
	c := New[string](1 * time.Minute)
	c.Set("key1", "value1")

	_, ok := c.Get("key1")
	if !ok {
		t.Fatalf("expected cache hit before invalidate, got miss")
	}

	c.Invalidate("key1")

	_, ok = c.Get("key1")
	if ok {
		t.Fatalf("expected cache miss after invalidate, got hit")
	}
}

func TestCacheInvalidateAll(t *testing.T) {
	c := New[string](1 * time.Minute)
	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	c.InvalidateAll()

	_, ok := c.Get("key1")
	if ok {
		t.Fatalf("expected cache miss for key1 after InvalidateAll, got hit")
	}
	_, ok = c.Get("key2")
	if ok {
		t.Fatalf("expected cache miss for key2 after InvalidateAll, got hit")
	}
	_, ok = c.Get("key3")
	if ok {
		t.Fatalf("expected cache miss for key3 after InvalidateAll, got hit")
	}
}

func TestCacheConcurrent(t *testing.T) {
	c := New[int](10 * time.Second)
	var wg sync.WaitGroup
	wg.Add(100)

	for i := 0; i < 100; i++ {
		go func(idx int) {
			defer wg.Done()
			key := "key" + string(rune(idx%10))
			for j := 0; j < 10; j++ {
				c.Set(key, idx*10+j)
				_, _ = c.Get(key)
			}
		}(i)
	}

	wg.Wait()
}
