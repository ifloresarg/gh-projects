package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

type diskCachePayload struct {
	Writer    int    `json:"writer"`
	Iteration int    `json:"iteration"`
	Name      string `json:"name"`
}

func TestDiskCacheSaveLoad(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "cache")
	diskCache, err := NewDiskCache(dir)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	if diskCache.CacheDir() != dir {
		t.Fatalf("CacheDir() = %q, want %q", diskCache.CacheDir(), dir)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("Stat(%q) error = %v", dir, err)
	}

	want := diskCachePayload{Writer: 1, Iteration: 2, Name: "board"}
	if err := diskCache.Save("project", want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var got diskCachePayload
	if err := diskCache.Load("project", &got); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}

func TestDiskCacheMiss(t *testing.T) {
	t.Parallel()

	diskCache, err := NewDiskCache(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	var got diskCachePayload
	err = diskCache.Load("missing", &got)
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("Load() error = %v, want %v", err, ErrCacheMiss)
	}
}

func TestDiskCacheVersionMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	diskCache, err := NewDiskCache(dir)
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	payload, err := json.Marshal(cacheEnvelope{
		Version: diskCacheVersion + 1,
		Data:    json.RawMessage(`{"name":"stale"}`),
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	stalePath := filepath.Join(dir, "stale.json")
	if err := os.WriteFile(stalePath, payload, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var got diskCachePayload
	err = diskCache.Load("stale", &got)
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("Load() error = %v, want %v", err, ErrCacheMiss)
	}

	if _, err := os.Stat(stalePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(%q) error = %v, want not exist", stalePath, err)
	}
}

func TestDiskCacheInvalidateAll(t *testing.T) {
	t.Parallel()

	diskCache, err := NewDiskCache(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	keys := []string{"project", "view", "items/nested"}
	for i, key := range keys {
		if err := diskCache.Save(key, diskCachePayload{Writer: i, Name: key}); err != nil {
			t.Fatalf("Save(%q) error = %v", key, err)
		}
	}

	if err := diskCache.InvalidateAll(); err != nil {
		t.Fatalf("InvalidateAll() error = %v", err)
	}

	for _, key := range keys {
		var got diskCachePayload
		err := diskCache.Load(key, &got)
		if !errors.Is(err, ErrCacheMiss) {
			t.Fatalf("Load(%q) error = %v, want %v", key, err, ErrCacheMiss)
		}
	}
}

func TestDiskCacheAtomicWrite(t *testing.T) {
	t.Parallel()

	diskCache, err := NewDiskCache(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	const (
		writers    = 10
		iterations = 50
	)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for writer := range writers {
		wg.Add(1)
		go func(writer int) {
			defer wg.Done()
			<-start
			for iteration := range iterations {
				if err := diskCache.Save("shared", diskCachePayload{
					Writer:    writer,
					Iteration: iteration,
					Name:      "shared",
				}); err != nil {
					t.Errorf("Save() error = %v", err)
					return
				}
			}
		}(writer)
	}

	close(start)
	wg.Wait()

	var got diskCachePayload
	if err := diskCache.Load("shared", &got); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Name != "shared" {
		t.Fatalf("Load() name = %q, want %q", got.Name, "shared")
	}
	if got.Writer < 0 || got.Writer >= writers {
		t.Fatalf("Load() writer = %d, want [0,%d)", got.Writer, writers)
	}
	if got.Iteration < 0 || got.Iteration >= iterations {
		t.Fatalf("Load() iteration = %d, want [0,%d)", got.Iteration, iterations)
	}
}

func TestDiskCacheDifferentTypes(t *testing.T) {
	t.Parallel()

	diskCache, err := NewDiskCache(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	wantNames := []string{"todo", "in-progress", "done"}
	wantCounts := map[string]int{"todo": 1, "done": 2}

	if err := diskCache.Save("names", wantNames); err != nil {
		t.Fatalf("Save(names) error = %v", err)
	}
	if err := diskCache.Save("counts", wantCounts); err != nil {
		t.Fatalf("Save(counts) error = %v", err)
	}

	var gotNames []string
	if err := diskCache.Load("names", &gotNames); err != nil {
		t.Fatalf("Load(names) error = %v", err)
	}

	var gotCounts map[string]int
	if err := diskCache.Load("counts", &gotCounts); err != nil {
		t.Fatalf("Load(counts) error = %v", err)
	}

	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("Load(names) = %#v, want %#v", gotNames, wantNames)
	}
	if !reflect.DeepEqual(gotCounts, wantCounts) {
		t.Fatalf("Load(counts) = %#v, want %#v", gotCounts, wantCounts)
	}
}
