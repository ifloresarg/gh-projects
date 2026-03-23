package github

import (
	"strings"
	"testing"
)

func TestMissingProjectScope(t *testing.T) {
	t.Parallel()

	output := `✓ Logged in to github.com as alice (...)
✓ Token: gho_...
✓ Token scopes: repo, read:org, workflow`

	err := CheckProjectScope(output)
	if err == nil {
		t.Fatal("expected error for missing project scope")
	}

	if !strings.Contains(err.Error(), "gh auth refresh -s project") {
		t.Fatalf("expected refresh instruction in error, got %q", err.Error())
	}
}

func TestValidProjectScope(t *testing.T) {
	t.Parallel()

	output := `✓ Logged in to github.com as alice (...)
✓ Token: gho_...
✓ Token scopes: repo, read:org, project, workflow`

	err := CheckProjectScope(output)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
