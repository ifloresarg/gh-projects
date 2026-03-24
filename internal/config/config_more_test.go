package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTableDrivenScenarios(t *testing.T) {
	tests := []struct {
		name         string
		contents     string
		writeFile    bool
		wantErr      bool
		wantOwner    string
		wantProject  int
		wantCacheTTL int
	}{
		{
			name:         "missing file returns defaults",
			writeFile:    false,
			wantOwner:    "",
			wantProject:  0,
			wantCacheTTL: 300,
		},
		{
			name: "partial yaml keeps defaults",
			contents: `default_owner: octocat
cache_ttl: 90
`,
			writeFile:    true,
			wantOwner:    "octocat",
			wantProject:  0,
			wantCacheTTL: 90,
		},
		{
			name:      "malformed yaml returns error",
			contents:  "default_owner: [broken",
			writeFile: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			path := filepath.Join(tempDir, "config.yaml")

			origOverride := configPathOverride
			configPathOverride = path
			defer func() { configPathOverride = origOverride }()

			if tt.writeFile {
				if err := os.WriteFile(path, []byte(tt.contents), 0644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
			}

			cfg, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Fatal("Load() error = nil, want non-nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.DefaultOwner != tt.wantOwner || cfg.DefaultProject != tt.wantProject || cfg.CacheTTL != tt.wantCacheTTL {
				t.Fatalf("Load() = %#v, want owner=%q project=%d cacheTTL=%d", cfg, tt.wantOwner, tt.wantProject, tt.wantCacheTTL)
			}
		})
	}
}

func TestConfigPathUsesOverride(t *testing.T) {
	tempDir := t.TempDir()
	wantPath := filepath.Join(tempDir, "custom", "config.yaml")

	origOverride := configPathOverride
	configPathOverride = wantPath
	defer func() { configPathOverride = origOverride }()

	gotPath, err := configPath()
	if err != nil {
		t.Fatalf("configPath() error = %v", err)
	}
	if gotPath != wantPath {
		t.Fatalf("configPath() = %q, want %q", gotPath, wantPath)
	}
}

func TestSaveCreatesNestedConfigDirectory(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "nested", "gh-projects", "config.yaml")

	origOverride := configPathOverride
	configPathOverride = path
	defer func() { configPathOverride = origOverride }()

	if err := Save(Config{DefaultOwner: "octocat", DefaultProject: 7, CacheTTL: 45}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved config path stat error = %v", err)
	}
}
