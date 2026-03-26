package editor

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var envMu sync.Mutex

func TestResolveEditor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		visual      string
		editor      string
		wantEditor  string
		wantErrPart string
	}{
		{
			name:       "VISUAL set returns VISUAL",
			visual:     "nvim",
			editor:     "vim",
			wantEditor: "nvim",
		},
		{
			name:       "EDITOR set and VISUAL empty returns EDITOR",
			visual:     "",
			editor:     "nano",
			wantEditor: "nano",
		},
		{
			name:       "both set VISUAL takes priority",
			visual:     "/usr/bin/micro",
			editor:     "/usr/bin/vim",
			wantEditor: "/usr/bin/micro",
		},
		{
			name:        "both empty returns error",
			visual:      "",
			editor:      "",
			wantErrPart: "no editor configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			envMu.Lock()
			t.Cleanup(func() {
				envMu.Unlock()
			})

			oldVisual, hadVisual := os.LookupEnv("VISUAL")
			oldEditor, hadEditor := os.LookupEnv("EDITOR")

			if err := os.Setenv("VISUAL", tt.visual); err != nil {
				t.Fatalf("os.Setenv(VISUAL) error = %v", err)
			}
			if err := os.Setenv("EDITOR", tt.editor); err != nil {
				t.Fatalf("os.Setenv(EDITOR) error = %v", err)
			}

			t.Cleanup(func() {
				if hadVisual {
					if err := os.Setenv("VISUAL", oldVisual); err != nil {
						t.Fatalf("restore VISUAL error = %v", err)
					}
				} else {
					if err := os.Unsetenv("VISUAL"); err != nil {
						t.Fatalf("unset VISUAL error = %v", err)
					}
				}

				if hadEditor {
					if err := os.Setenv("EDITOR", oldEditor); err != nil {
						t.Fatalf("restore EDITOR error = %v", err)
					}
				} else {
					if err := os.Unsetenv("EDITOR"); err != nil {
						t.Fatalf("unset EDITOR error = %v", err)
					}
				}
			})

			got, err := ResolveEditor()
			if tt.wantErrPart != "" {
				if err == nil {
					t.Fatalf("ResolveEditor() error = nil, want error containing %q", tt.wantErrPart)
				}
				if !strings.Contains(err.Error(), tt.wantErrPart) {
					t.Fatalf("ResolveEditor() error = %q, want to contain %q", err.Error(), tt.wantErrPart)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveEditor() error = %v", err)
			}
			if got != tt.wantEditor {
				t.Fatalf("ResolveEditor() = %q, want %q", got, tt.wantEditor)
			}
		})
	}
}

func TestPrepareEdit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "normal content",
			content: "hello from gh-projects",
		},
		{
			name:    "empty content",
			content: "",
		},
		{
			name:    "unicode and emoji content",
			content: "hola 👋 café 漢字",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpPath, cleanup, err := PrepareEdit(tt.content)
			if err != nil {
				t.Fatalf("PrepareEdit() error = %v", err)
			}
			if cleanup == nil {
				t.Fatalf("PrepareEdit() cleanup = nil, want non-nil cleanup function")
			}

			if filepath.Ext(tmpPath) != ".md" {
				t.Fatalf("PrepareEdit() path extension = %q, want %q", filepath.Ext(tmpPath), ".md")
			}

			data, err := os.ReadFile(tmpPath)
			if err != nil {
				t.Fatalf("os.ReadFile(%q) error = %v", tmpPath, err)
			}
			if string(data) != tt.content {
				t.Fatalf("file content = %q, want %q", string(data), tt.content)
			}

			cleanup()

			_, err = os.Stat(tmpPath)
			if err == nil {
				t.Fatalf("os.Stat(%q) error = nil, want not-exist after cleanup", tmpPath)
			}
			if !os.IsNotExist(err) {
				t.Fatalf("os.Stat(%q) error = %v, want IsNotExist", tmpPath, err)
			}
		})
	}
}

func TestReadResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		original    string
		fileContent string
		createFile  bool
		wantChanged bool
		wantErr     bool
	}{
		{
			name:        "same content reports unchanged",
			original:    "same",
			fileContent: "same",
			createFile:  true,
			wantChanged: false,
			wantErr:     false,
		},
		{
			name:        "different content reports changed",
			original:    "old",
			fileContent: "new",
			createFile:  true,
			wantChanged: true,
			wantErr:     false,
		},
		{
			name:        "trailing newline only reports unchanged",
			original:    "content",
			fileContent: "content\n",
			createFile:  true,
			wantChanged: false,
			wantErr:     false,
		},
		{
			name:       "missing file returns error",
			original:   "anything",
			createFile: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpPath := filepath.Join(t.TempDir(), "result.md")
			if tt.createFile {
				err := os.WriteFile(tmpPath, []byte(tt.fileContent), 0o600)
				if err != nil {
					t.Fatalf("os.WriteFile(%q) error = %v", tmpPath, err)
				}
			}

			newContent, changed, err := ReadResult(tmpPath, tt.original)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ReadResult(%q) error = nil, want non-nil", tmpPath)
				}
				return
			}

			if err != nil {
				t.Fatalf("ReadResult(%q) error = %v", tmpPath, err)
			}
			if newContent != tt.fileContent {
				t.Fatalf("ReadResult(%q) newContent = %q, want %q", tmpPath, newContent, tt.fileContent)
			}
			if changed != tt.wantChanged {
				t.Fatalf("ReadResult(%q) changed = %v, want %v", tmpPath, changed, tt.wantChanged)
			}
		})
	}
}

func TestCleanup(t *testing.T) {
	t.Parallel()

	tmpPath, cleanup, err := PrepareEdit("cleanup test")
	if err != nil {
		t.Fatalf("PrepareEdit() error = %v", err)
	}
	if cleanup == nil {
		t.Fatalf("PrepareEdit() cleanup = nil, want non-nil")
	}

	_, err = os.Stat(tmpPath)
	if err != nil {
		t.Fatalf("os.Stat(%q) before cleanup error = %v", tmpPath, err)
	}

	cleanup()

	_, err = os.Stat(tmpPath)
	if err == nil {
		t.Fatalf("os.Stat(%q) after cleanup error = nil, want not-exist", tmpPath)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) after cleanup error = %v, want IsNotExist", tmpPath, err)
	}
}
