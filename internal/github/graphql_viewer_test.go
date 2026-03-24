package github

import (
	"errors"
	"sort"
	"sync"
	"testing"
)

func TestListViewerOrganizationsMockReturnsOrgs(t *testing.T) {
	t.Parallel()

	client := &MockClient{
		ListViewerOrganizationsFn: func() ([]string, error) {
			return []string{"github", "octocat-org", "my-org"}, nil
		},
	}

	result, err := client.ListViewerOrganizations()
	if err != nil {
		t.Fatalf("ListViewerOrganizations() error = %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("ListViewerOrganizations() returned %d orgs, want 3", len(result))
	}
	if result[0] != "github" || result[1] != "octocat-org" || result[2] != "my-org" {
		t.Fatalf("ListViewerOrganizations() = %v, want [github octocat-org my-org]", result)
	}
}

func TestListViewerOrganizationsMockReturnsEmpty(t *testing.T) {
	t.Parallel()

	client := &MockClient{
		ListViewerOrganizationsFn: func() ([]string, error) {
			return []string{}, nil
		},
	}

	result, err := client.ListViewerOrganizations()
	if err != nil {
		t.Fatalf("ListViewerOrganizations() error = %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("ListViewerOrganizations() returned %d orgs, want 0", len(result))
	}
}

func TestListViewerOrganizationsMockReturnsScopeError(t *testing.T) {
	t.Parallel()

	client := &MockClient{
		ListViewerOrganizationsFn: func() ([]string, error) {
			return nil, ErrMissingScopeReadOrg
		},
	}

	result, err := client.ListViewerOrganizations()
	if err == nil {
		t.Fatalf("ListViewerOrganizations() error = nil, want ErrMissingScopeReadOrg")
	}
	if !errors.Is(err, ErrMissingScopeReadOrg) {
		t.Fatalf("ListViewerOrganizations() error = %v, want %v", err, ErrMissingScopeReadOrg)
	}
	if result != nil {
		t.Fatalf("ListViewerOrganizations() result = %v, want nil", result)
	}
}

func TestListViewerLoginMockReturnsLogin(t *testing.T) {
	t.Parallel()

	client := &MockClient{
		ListViewerLoginFn: func() (string, error) {
			return "ifloresarg", nil
		},
	}

	result, err := client.ListViewerLogin()
	if err != nil {
		t.Fatalf("ListViewerLogin() error = %v", err)
	}
	if result != "ifloresarg" {
		t.Fatalf("ListViewerLogin() = %q, want %q", result, "ifloresarg")
	}
}

func TestListViewerLoginMockDefault(t *testing.T) {
	t.Parallel()

	client := &MockClient{
		ListViewerLoginFn: nil,
	}

	result, err := client.ListViewerLogin()
	if err != nil {
		t.Fatalf("ListViewerLogin() error = %v", err)
	}
	if result != "" {
		t.Fatalf("ListViewerLogin() = %q, want %q", result, "")
	}
}

func TestListAllAccessibleProjectsAggregatesFromMultipleOwners(t *testing.T) {
	t.Parallel()

	var (
		mu           sync.Mutex
		calledOwners = make(map[string]bool)
	)

	client := &MockClient{
		ListViewerLoginFn: func() (string, error) {
			return "octocat", nil
		},
		ListViewerOrganizationsFn: func() ([]string, error) {
			return []string{"zeta-org", "alpha-org", "beta-org"}, nil
		},
		ListProjectsFn: func(owner string) ([]Project, error) {
			mu.Lock()
			calledOwners[owner] = true
			mu.Unlock()

			switch owner {
			case "octocat":
				return []Project{{ID: "p-personal", Owner: "octocat", Title: "Personal"}}, nil
			case "alpha-org":
				return []Project{{ID: "p-alpha", Owner: "alpha-org", Title: "Alpha"}}, nil
			case "beta-org":
				return []Project{{ID: "p-beta", Owner: "beta-org", Title: "Beta"}}, nil
			case "zeta-org":
				return []Project{{ID: "p-zeta", Owner: "zeta-org", Title: "Zeta"}}, nil
			default:
				return nil, errors.New("unexpected owner")
			}
		},
	}

	projects, err := listAllAccessibleProjects(client)
	if err != nil {
		t.Fatalf("listAllAccessibleProjects() error = %v", err)
	}

	if len(projects) != 4 {
		t.Fatalf("listAllAccessibleProjects() returned %d projects, want 4", len(projects))
	}

	owners := []string{projects[0].Owner, projects[1].Owner, projects[2].Owner, projects[3].Owner}
	if !sort.StringsAreSorted(owners) {
		t.Fatalf("project owners are not sorted: %v", owners)
	}

	for _, owner := range []string{"octocat", "alpha-org", "beta-org", "zeta-org"} {
		if !calledOwners[owner] {
			t.Fatalf("ListProjects() was not called for owner %q", owner)
		}
	}
}

func TestListAllAccessibleProjectsFallbackOnMissingScope(t *testing.T) {
	t.Parallel()

	var (
		mu           sync.Mutex
		calledOwners []string
	)

	client := &MockClient{
		ListViewerLoginFn: func() (string, error) {
			return "octocat", nil
		},
		ListViewerOrganizationsFn: func() ([]string, error) {
			return nil, ErrMissingScopeReadOrg
		},
		ListProjectsFn: func(owner string) ([]Project, error) {
			mu.Lock()
			calledOwners = append(calledOwners, owner)
			mu.Unlock()

			if owner != "octocat" {
				return nil, errors.New("unexpected owner")
			}

			return []Project{{ID: "p-personal", Owner: "octocat", Title: "Personal"}}, nil
		},
	}

	projects, err := listAllAccessibleProjects(client)
	if !errors.Is(err, ErrMissingScopeReadOrg) {
		t.Fatalf("listAllAccessibleProjects() error = %v, want %v", err, ErrMissingScopeReadOrg)
	}

	if len(projects) != 1 {
		t.Fatalf("listAllAccessibleProjects() returned %d projects, want 1", len(projects))
	}
	if projects[0].Owner != "octocat" {
		t.Fatalf("project owner = %q, want %q", projects[0].Owner, "octocat")
	}

	if len(calledOwners) != 1 || calledOwners[0] != "octocat" {
		t.Fatalf("ListProjects() called owners = %v, want [octocat]", calledOwners)
	}
}

func TestListAllAccessibleProjectsEmptyOrgs(t *testing.T) {
	t.Parallel()

	client := &MockClient{
		ListViewerLoginFn: func() (string, error) {
			return "octocat", nil
		},
		ListViewerOrganizationsFn: func() ([]string, error) {
			return []string{}, nil
		},
		ListProjectsFn: func(owner string) ([]Project, error) {
			if owner != "octocat" {
				return nil, errors.New("unexpected owner")
			}

			return []Project{{ID: "p-personal", Owner: "octocat", Title: "Personal"}}, nil
		},
	}

	projects, err := listAllAccessibleProjects(client)
	if err != nil {
		t.Fatalf("listAllAccessibleProjects() error = %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("listAllAccessibleProjects() returned %d projects, want 1", len(projects))
	}
	if projects[0].Owner != "octocat" {
		t.Fatalf("project owner = %q, want %q", projects[0].Owner, "octocat")
	}
}
