package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ErrCacheMiss = errors.New("cache miss")

const diskCacheVersion = 1

type cacheEnvelope struct {
	Version   int             `json:"version"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

type DiskCache struct {
	dir string
	mu  sync.Mutex
}

func NewDiskCache(dir string) (*DiskCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	return &DiskCache{dir: dir}, nil
}

func (d *DiskCache) Save(key string, data any) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal cache data: %w", err)
	}

	envelopeBytes, err := json.Marshal(cacheEnvelope{
		Version:   diskCacheVersion,
		Timestamp: time.Now().UTC(),
		Data:      raw,
	})
	if err != nil {
		return fmt.Errorf("marshal cache envelope: %w", err)
	}

	finalPath := d.pathForKey(key)
	finalDir := filepath.Dir(finalPath)
	if err := os.MkdirAll(finalDir, 0o755); err != nil {
		return fmt.Errorf("create cache key directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(finalDir, "*.tmp")
	if err != nil {
		return fmt.Errorf("create temp cache file: %w", err)
	}

	tmpName := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(envelopeBytes); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp cache file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp cache file: %w", err)
	}

	if err := os.Rename(tmpName, finalPath); err != nil {
		return fmt.Errorf("rename temp cache file: %w", err)
	}

	cleanup = false
	return nil
}

func (d *DiskCache) Load(key string, target any) error {
	payload, err := os.ReadFile(d.pathForKey(key))
	if err != nil {
		if os.IsNotExist(err) {
			return ErrCacheMiss
		}
		return fmt.Errorf("read cache file: %w", err)
	}

	var envelope cacheEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("unmarshal cache envelope: %w", err)
	}

	if envelope.Version != diskCacheVersion {
		if err := os.Remove(d.pathForKey(key)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove stale cache file: %w", err)
		}
		return ErrCacheMiss
	}

	if err := json.Unmarshal(envelope.Data, target); err != nil {
		return fmt.Errorf("unmarshal cache data: %w", err)
	}

	return nil
}

func (d *DiskCache) InvalidateAll() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err := filepath.WalkDir(d.dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("invalidate disk cache: %w", err)
	}

	return nil
}

func (d *DiskCache) CacheDir() string {
	return d.dir
}

func (d *DiskCache) pathForKey(key string) string {
	return filepath.Join(d.dir, key+".json")
}
