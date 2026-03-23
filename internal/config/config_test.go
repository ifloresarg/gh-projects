package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var configTestMu sync.Mutex

func TestLoadDefaults(t *testing.T) {
	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "config.yaml")

	// Override the config path for this test
	origOverride := configPathOverride
	configPathOverride = tempConfigPath
	defer func() { configPathOverride = origOverride }()

	// Load should return defaults when file doesn't exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify defaults
	if cfg.DefaultOwner != "" {
		t.Errorf("DefaultOwner = %q, want empty string", cfg.DefaultOwner)
	}
	if cfg.DefaultProject != 0 {
		t.Errorf("DefaultProject = %d, want 0", cfg.DefaultProject)
	}
	if cfg.CacheTTL != 300 {
		t.Errorf("CacheTTL = %d, want 300", cfg.CacheTTL)
	}
}

func TestRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "config.yaml")

	// Override the config path for this test
	origOverride := configPathOverride
	configPathOverride = tempConfigPath
	defer func() { configPathOverride = origOverride }()

	// Create a config with non-default values
	original := Config{
		DefaultOwner:   "test-org",
		DefaultProject: 3,
		CacheTTL:       60,
	}

	// Save the config
	err := Save(original)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tempConfigPath); err != nil {
		t.Fatalf("Config file not created: %v", err)
	}

	// Load it back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify all fields match
	if loaded.DefaultOwner != original.DefaultOwner {
		t.Errorf("DefaultOwner = %q, want %q", loaded.DefaultOwner, original.DefaultOwner)
	}
	if loaded.DefaultProject != original.DefaultProject {
		t.Errorf("DefaultProject = %d, want %d", loaded.DefaultProject, original.DefaultProject)
	}
	if loaded.CacheTTL != original.CacheTTL {
		t.Errorf("CacheTTL = %d, want %d", loaded.CacheTTL, original.CacheTTL)
	}
}

func TestShowLabelsAndShowClosedItemsRoundTrip(t *testing.T) {
	t.Parallel()

	configTestMu.Lock()
	defer configTestMu.Unlock()

	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "config.yaml")

	origOverride := configPathOverride
	configPathOverride = tempConfigPath
	defer func() { configPathOverride = origOverride }()

	original := Config{
		DefaultOwner:    "test-org",
		DefaultProject:  3,
		CacheTTL:        60,
		ShowLabels:      false,
		ShowClosedItems: true,
	}

	err := Save(original)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.ShowLabels != original.ShowLabels {
		t.Errorf("ShowLabels = %t, want %t", loaded.ShowLabels, original.ShowLabels)
	}
	if loaded.ShowClosedItems != original.ShowClosedItems {
		t.Errorf("ShowClosedItems = %t, want %t", loaded.ShowClosedItems, original.ShowClosedItems)
	}
}

func TestShowLabelsDefaultsToTrue(t *testing.T) {
	t.Parallel()

	configTestMu.Lock()
	defer configTestMu.Unlock()

	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "config.yaml")

	origOverride := configPathOverride
	configPathOverride = tempConfigPath
	defer func() { configPathOverride = origOverride }()

	err := os.WriteFile(tempConfigPath, []byte("default_owner: \"test\"\n"), 0o644)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !loaded.ShowLabels {
		t.Errorf("ShowLabels = %t, want true", loaded.ShowLabels)
	}
	if loaded.ShowClosedItems {
		t.Errorf("ShowClosedItems = %t, want false", loaded.ShowClosedItems)
	}
}

func TestMergedPRWindowDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	if cfg.MergedPRWindow != 12 {
		t.Errorf("MergedPRWindow = %d, want 12", cfg.MergedPRWindow)
	}
}

func TestMergedPRWindowFromYAML(t *testing.T) {
	t.Parallel()

	configTestMu.Lock()
	defer configTestMu.Unlock()

	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "config.yaml")

	origOverride := configPathOverride
	configPathOverride = tempConfigPath
	defer func() { configPathOverride = origOverride }()

	err := os.WriteFile(tempConfigPath, []byte("merged_pr_window: 24\n"), 0o644)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.MergedPRWindow != 24 {
		t.Errorf("MergedPRWindow = %d, want 24", loaded.MergedPRWindow)
	}
}

func TestMergedPRWindowDefaultsWhenMissing(t *testing.T) {
	t.Parallel()

	configTestMu.Lock()
	defer configTestMu.Unlock()

	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "config.yaml")

	origOverride := configPathOverride
	configPathOverride = tempConfigPath
	defer func() { configPathOverride = origOverride }()

	err := os.WriteFile(tempConfigPath, []byte("default_owner: \"test\"\n"), 0o644)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.MergedPRWindow != 12 {
		t.Errorf("MergedPRWindow = %d, want 12", loaded.MergedPRWindow)
	}
}

func TestPRFetchLimitDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	if cfg.PRFetchLimit != 200 {
		t.Errorf("PRFetchLimit = %d, want 200", cfg.PRFetchLimit)
	}
}

func TestPRFetchLimitFromYAML(t *testing.T) {
	t.Parallel()

	configTestMu.Lock()
	defer configTestMu.Unlock()

	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "config.yaml")

	origOverride := configPathOverride
	configPathOverride = tempConfigPath
	defer func() { configPathOverride = origOverride }()

	err := os.WriteFile(tempConfigPath, []byte("pr_fetch_limit: 50\n"), 0o644)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.PRFetchLimit != 50 {
		t.Errorf("PRFetchLimit = %d, want 50", loaded.PRFetchLimit)
	}
}
