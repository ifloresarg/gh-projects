package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/ifloresarg/gh-projects/internal/github"
)

func TestCardViewForIssueIncludesMetadata(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Implement GraphQL client",
		Type:          "Issue",
		ContentNumber: 101,
		Content: &github.Issue{
			Number:    101,
			Assignees: []github.User{{Login: "ifloresarg"}, {Login: "mona"}},
			Labels:    []github.Label{{Name: "bug", Color: "d73a4a"}},
		},
	}, true, 32, true)

	for _, fragment := range []string{"#101 - Implement", "GraphQL", "@ifloresarg", "@mona", "bug"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("Card() view missing %q in %q", fragment, view)
		}
	}
}

func TestCardViewForPullRequestIncludesPRNumber(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Refine board rendering",
		Type:          "PullRequest",
		ContentNumber: 55,
		Content: &github.PullRequest{
			Number: 55,
		},
	}, false, 32, true)

	for _, fragment := range []string{"Refine board", "PR #55 -"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("Card() view missing %q in %q", fragment, view)
		}
	}
}

func TestCardViewForDraftIssueShowsDraftMarker(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{Title: "Draft migration plan", Type: "DraftIssue"}, false, 28, true)

	for _, fragment := range []string{"Draft migration", "(draft) -"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("Card() view missing %q in %q", fragment, view)
		}
	}
}

func TestCardViewMultiLineTitleWraps(t *testing.T) {
	t.Parallel()

	singleLineView := Card(github.ProjectItem{
		Title:         "Short title",
		Type:          "Issue",
		ContentNumber: 201,
		Content: &github.Issue{
			Number: 201,
		},
	}, false, 32, true)

	view := Card(github.ProjectItem{
		Title:         "Implement the new GraphQL client with full retry logic",
		Type:          "Issue",
		ContentNumber: 202,
		Content: &github.Issue{
			Number: 202,
		},
	}, false, 32, true)

	for _, fragment := range []string{"Implement", "#202"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("Card() view missing %q in %q", fragment, view)
		}
	}

	if lipgloss.Height(view) <= lipgloss.Height(singleLineView) {
		t.Fatalf("Card() height = %d, want > %d for wrapped title\nview:\n%s", lipgloss.Height(view), lipgloss.Height(singleLineView), view)
	}
}

func TestCardViewLongTitleCapsAtThreeLinesWithEllipsis(t *testing.T) {
	t.Parallel()

	shortView := Card(github.ProjectItem{
		Title:         "Short title",
		Type:          "Issue",
		ContentNumber: 500,
		Content: &github.Issue{
			Number: 500,
		},
	}, false, 32, true)

	view := Card(github.ProjectItem{
		Title:         strings.Repeat("wrap me across many words ", 10),
		Type:          "Issue",
		ContentNumber: 501,
		Content: &github.Issue{
			Number: 501,
		},
	}, false, 32, true)

	if got, want := lipgloss.Height(view), lipgloss.Height(shortView)+2; got != want {
		t.Fatalf("Card() height = %d, want %d for max 3 title lines\nview:\n%s", got, want, view)
	}

	lines := strings.Split(view, "\n")
	if len(lines) < 4 {
		t.Fatalf("Card() rendered too few lines: %q", view)
	}

	thirdLine := strings.TrimSpace(strings.Trim(lines[3], "│"))
	if !strings.HasSuffix(thirdLine, "…") {
		t.Fatalf("Card() third title line = %q, want ellipsis suffix in %q", thirdLine, view)
	}
}

func TestCardViewShortTitleRemainsSingleLine(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Short title",
		Type:          "Issue",
		ContentNumber: 600,
		Content: &github.Issue{
			Number: 600,
		},
	}, false, 32, true)

	if got, want := lipgloss.Height(view), 3; got != want {
		t.Fatalf("Card() height = %d, want %d for single-line title\nview:\n%s", got, want, view)
	}

	if !strings.Contains(view, "#600 - Short title") {
		t.Fatalf("Card() view missing single-line title in %q", view)
	}
}

func TestCardViewLabelOverflowShowsPlusN(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Issue with many labels",
		Type:          "Issue",
		ContentNumber: 303,
		Content: &github.Issue{
			Number: 303,
			Labels: []github.Label{
				{Name: "bug", Color: "d73a4a"},
				{Name: "enhancement", Color: "a2eeef"},
				{Name: "help wanted", Color: "008672"},
				{Name: "good first issue", Color: "7057ff"},
				{Name: "wontfix", Color: "ffffff"},
				{Name: "duplicate", Color: "cfd3d7"},
			},
		},
	}, false, 50, true)

	if !strings.Contains(view, "+2") {
		t.Fatalf("Card() view missing +2 overflow indicator in %q", view)
	}
}

func TestCardViewLabelLineTruncation(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Issue with very wide labels",
		Type:          "Issue",
		ContentNumber: 404,
		Content: &github.Issue{
			Number: 404,
			Labels: []github.Label{
				{Name: "very-long-label-one", Color: "d73a4a"},
				{Name: "very-long-label-two", Color: "a2eeef"},
				{Name: "very-long-label-three", Color: "008672"},
				{Name: "very-long-label-four", Color: "7057ff"},
			},
		},
	}, false, 32, true)

	if !strings.Contains(view, "...") {
		t.Fatalf("Card() view missing truncation '...' in %q", view)
	}
}

func TestCardViewTypeBadgeAppearsWhenTypeValueSet(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Some issue",
		Type:          "Issue",
		ContentNumber: 1,
		TypeValue:     "Bug",
		Content:       &github.Issue{Number: 1},
	}, false, 35, true)

	if !strings.Contains(view, "Bug") {
		t.Fatalf("Card() view missing type badge %q in %q", "Bug", view)
	}
}

func TestCardViewNoBadgeWhenTypeValueEmpty(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Some issue",
		Type:          "Issue",
		ContentNumber: 1,
		TypeValue:     "",
		Content:       &github.Issue{Number: 1},
	}, false, 35, true)

	if strings.Contains(view, "[") {
		t.Fatalf("Card() view unexpectedly rendered badge brackets in %q", view)
	}
}

func TestCardViewLabelsShownWhenShowLabelsTrue(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Some issue",
		Type:          "Issue",
		ContentNumber: 1,
		Content: &github.Issue{
			Number: 1,
			Labels: []github.Label{{Name: "hotfix", Color: "e11d48"}},
		},
	}, false, 35, true)

	if !strings.Contains(view, "hotfix") {
		t.Fatalf("Card() view missing label %q in %q", "hotfix", view)
	}
}

func TestCardViewLabelsHiddenWhenShowLabelsFalse(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Some issue",
		Type:          "Issue",
		ContentNumber: 1,
		Content: &github.Issue{
			Number: 1,
			Labels: []github.Label{{Name: "hotfix", Color: "e11d48"}},
		},
	}, false, 35, false)

	if strings.Contains(view, "hotfix") {
		t.Fatalf("Card() view unexpectedly included label %q in %q", "hotfix", view)
	}
}

func TestCardViewPRBadgeAppearsForLinkedPR(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Issue with linked PR",
		Type:          "Issue",
		ContentNumber: 1,
		Content: &github.Issue{
			Number: 1,
			LinkedPRs: []github.LinkedPullRequest{
				{Number: 42, State: "OPEN"},
			},
		},
	}, false, 35, true)

	if !strings.Contains(view, "PR#42") {
		t.Fatalf("Card() view missing PR badge %q in %q", "PR#42", view)
	}
}

func TestCardViewNoPRBadgeWhenNoLinkedPRs(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Issue without linked PRs",
		Type:          "Issue",
		ContentNumber: 2,
		Content: &github.Issue{
			Number:    2,
			LinkedPRs: []github.LinkedPullRequest{},
		},
	}, false, 35, true)

	if strings.Contains(view, "PR#") {
		t.Fatalf("Card() view unexpectedly included PR badge in %q", view)
	}
}

func TestCardViewNoPRBadgeForPullRequestItems(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Some pull request",
		Type:          "PullRequest",
		ContentNumber: 55,
		Content: &github.PullRequest{
			Number: 55,
		},
	}, false, 35, true)

	badgePattern := "PR#55"
	if strings.Contains(view, badgePattern) && !strings.Contains(view, "PR #55 -") {
		t.Fatalf("Card() view unexpectedly included PR badge %q (not title) in %q", badgePattern, view)
	}
}

func TestCardViewMultiplePRBadgesWithOverflow(t *testing.T) {
	t.Parallel()

	view := Card(github.ProjectItem{
		Title:         "Issue with multiple linked PRs",
		Type:          "Issue",
		ContentNumber: 3,
		Content: &github.Issue{
			Number: 3,
			LinkedPRs: []github.LinkedPullRequest{
				{Number: 42, State: "OPEN"},
				{Number: 43, State: "MERGED"},
				{Number: 44, State: "CLOSED"},
			},
		},
	}, false, 35, true)

	if !strings.Contains(view, "PR#42") {
		t.Fatalf("Card() view missing PR#42 in %q", view)
	}
	if !strings.Contains(view, "PR#43") {
		t.Fatalf("Card() view missing PR#43 in %q", view)
	}
	if !strings.Contains(view, "+1") {
		t.Fatalf("Card() view missing +1 overflow indicator in %q", view)
	}
	if strings.Contains(view, "PR#44") {
		t.Fatalf("Card() view unexpectedly included PR#44 in %q", view)
	}
}
