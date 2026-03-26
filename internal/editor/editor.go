package editor

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

func ResolveEditor() (string, error) {
	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual, nil
	}

	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, nil
	}

	return "", fmt.Errorf("no editor configured: set $VISUAL or $EDITOR environment variable")
}

func PrepareEdit(content string) (tmpPath string, cleanup func(), err error) {
	tmpFile, err := os.CreateTemp("", "gh-projects-*.md")
	if err != nil {
		return "", nil, err
	}

	tmpPath = tmpFile.Name()

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", nil, err
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", nil, err
	}

	var once sync.Once
	cleanup = func() {
		once.Do(func() {
			_ = os.Remove(tmpPath)
		})
	}

	return tmpPath, cleanup, nil
}

func ReadResult(tmpPath string, originalContent string) (newContent string, changed bool, err error) {
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", false, err
	}

	newContent = string(data)
	changed = strings.TrimRight(newContent, "\n\r") != strings.TrimRight(originalContent, "\n\r")

	return newContent, changed, nil
}
