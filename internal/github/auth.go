package github

import (
	"fmt"
	"os/exec"
	"strings"
)

func CheckProjectScope(output string) error {
	for line := range strings.SplitSeq(output, "\n") {
		if !strings.Contains(line, "Token scopes:") {
			continue
		}

		_, scopesText, found := strings.Cut(line, "Token scopes:")
		if !found {
			continue
		}
		scopesText = strings.TrimSpace(scopesText)

		for scope := range strings.SplitSeq(scopesText, ",") {
			normalized := strings.Trim(strings.TrimSpace(scope), "'\"")
			if normalized == "project" {
				return nil
			}
		}

		return fmt.Errorf("missing 'project' scope. Run: gh auth refresh -s project")
	}

	return fmt.Errorf("could not determine GitHub CLI token scopes. Run: gh auth refresh -s project")
}

func CheckAuth() error {
	cmd := exec.Command("gh", "auth", "status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := strings.TrimSpace(string(out))
		if output == "" {
			return fmt.Errorf("GitHub CLI authentication check failed: %w\nRun: gh auth login", err)
		}

		return fmt.Errorf("GitHub CLI authentication check failed: %w\n%s\nRun: gh auth login", err, output)
	}

	return CheckProjectScope(string(out))
}
