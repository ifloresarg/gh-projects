package cache

import (
	"testing"
	"time"
)

func TestCacheTableDrivenSetAndGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		key   string
		value int
	}{
		{name: "first key", key: "alpha", value: 1},
		{name: "second key", key: "beta", value: 2},
		{name: "zero value", key: "gamma", value: 0},
	}

	c := New[int](time.Minute)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c.Set(tt.key, tt.value)
			got, ok := c.Get(tt.key)
			if !ok {
				t.Fatalf("Get(%q) cache miss", tt.key)
			}
			if got != tt.value {
				t.Fatalf("Get(%q) = %d, want %d", tt.key, got, tt.value)
			}
		})
	}
}

func TestCacheZeroTTLExpiresImmediately(t *testing.T) {
	t.Parallel()

	c := New[string](0)
	c.Set("ephemeral", "value")

	if _, ok := c.Get("ephemeral"); ok {
		t.Fatal("Get() cache hit with zero TTL, want immediate expiry")
	}
}

func TestCacheTableDrivenInvalidateOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*Cache[string])
		action func(*Cache[string])
		key    string
	}{
		{
			name: "invalidate single key",
			setup: func(c *Cache[string]) {
				c.Set("a", "1")
				c.Set("b", "2")
			},
			action: func(c *Cache[string]) { c.Invalidate("a") },
			key:    "a",
		},
		{
			name: "invalidate all keys",
			setup: func(c *Cache[string]) {
				c.Set("a", "1")
				c.Set("b", "2")
			},
			action: func(c *Cache[string]) { c.InvalidateAll() },
			key:    "b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New[string](time.Minute)
			tt.setup(c)
			tt.action(c)

			if _, ok := c.Get(tt.key); ok {
				t.Fatalf("Get(%q) cache hit after invalidation", tt.key)
			}
		})
	}
}
