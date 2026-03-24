package github

import (
	"errors"
	"testing"
	"time"
)

func TestCachedClientListProjectsCachesReadOperations(t *testing.T) {
	t.Parallel()

	callCount := 0
	client := NewCachedClient(&MockClient{
		ListProjectsFn: func(owner string) ([]Project, error) {
			callCount++
			return []Project{{ID: "p1", Title: "Roadmap", Owner: owner}}, nil
		},
	}, time.Minute)

	first, err := client.ListProjects("octocat")
	if err != nil {
		t.Fatalf("ListProjects() first call error = %v", err)
	}

	second, err := client.ListProjects("octocat")
	if err != nil {
		t.Fatalf("ListProjects() second call error = %v", err)
	}

	if callCount != 1 {
		t.Fatalf("ListProjects() inner call count = %d, want 1", callCount)
	}
	if len(first) != 1 || len(second) != 1 || first[0].ID != second[0].ID {
		t.Fatalf("cached results mismatch: first=%v second=%v", first, second)
	}
}

func TestCachedClientGetProjectItemsCachesReadOperations(t *testing.T) {
	t.Parallel()

	callCount := 0
	client := NewCachedClient(&MockClient{
		GetProjectItemsFn: func(projectID string) ([]ProjectItem, error) {
			callCount++
			return []ProjectItem{{ID: "item-1", Title: "Cache me", StatusID: "todo"}}, nil
		},
	}, time.Minute)

	_, err := client.GetProjectItems("PVT_1")
	if err != nil {
		t.Fatalf("GetProjectItems() first call error = %v", err)
	}

	items, err := client.GetProjectItems("PVT_1")
	if err != nil {
		t.Fatalf("GetProjectItems() second call error = %v", err)
	}

	if callCount != 1 {
		t.Fatalf("GetProjectItems() inner call count = %d, want 1", callCount)
	}
	if len(items) != 1 || items[0].Title != "Cache me" {
		t.Fatalf("unexpected cached items = %#v", items)
	}
}

func TestCachedClientMutationInvalidatesReadCaches(t *testing.T) {
	t.Parallel()

	listCalls := 0
	moveCalls := 0
	client := NewCachedClient(&MockClient{
		ListProjectsFn: func(owner string) ([]Project, error) {
			listCalls++
			return []Project{{ID: "p1", Title: "Roadmap", Owner: owner}}, nil
		},
		MoveItemFn: func(projectID, itemID, fieldID, optionID string) error {
			moveCalls++
			return nil
		},
	}, time.Minute)

	if _, err := client.ListProjects("octocat"); err != nil {
		t.Fatalf("ListProjects() initial call error = %v", err)
	}
	if _, err := client.ListProjects("octocat"); err != nil {
		t.Fatalf("ListProjects() cached call error = %v", err)
	}
	if listCalls != 1 {
		t.Fatalf("ListProjects() inner call count before mutation = %d, want 1", listCalls)
	}

	if err := client.MoveItem("PVT_1", "item-1", "field-1", "done"); err != nil {
		t.Fatalf("MoveItem() error = %v", err)
	}
	if moveCalls != 1 {
		t.Fatalf("MoveItem() inner call count = %d, want 1", moveCalls)
	}

	if _, err := client.ListProjects("octocat"); err != nil {
		t.Fatalf("ListProjects() after invalidation error = %v", err)
	}
	if listCalls != 2 {
		t.Fatalf("ListProjects() inner call count after invalidation = %d, want 2", listCalls)
	}
}

func TestCachedClientReadErrorsAreNotCached(t *testing.T) {
	t.Parallel()

	callCount := 0
	wantErr := errors.New("network error")
	client := NewCachedClient(&MockClient{
		ListRepositoryLabelsFn: func(owner, repo string) ([]Label, error) {
			callCount++
			return nil, wantErr
		},
	}, time.Minute)

	for range 2 {
		_, err := client.ListRepositoryLabels("octocat", "gh-projects")
		if !errors.Is(err, wantErr) {
			t.Fatalf("ListRepositoryLabels() error = %v, want %v", err, wantErr)
		}
	}

	if callCount != 2 {
		t.Fatalf("ListRepositoryLabels() inner call count = %d, want 2", callCount)
	}
}

func TestCachedClientListRepositoryPullRequests(t *testing.T) {
	t.Parallel()

	callCount := 0
	client := NewCachedClient(&MockClient{
		ListRepositoryPullRequestsFn: func(owner, repo string, limit int) ([]PullRequest, error) {
			callCount++
			return []PullRequest{
				{
					ID:        "PR_1",
					Number:    42,
					Title:     "Fix login bug",
					State:     "OPEN",
					Author:    User{Login: "octocat"},
					URL:       "https://github.com/octocat/gh-projects/pull/42",
					CreatedAt: time.Now(),
					RepoOwner: owner,
					RepoName:  repo,
				},
			}, nil
		},
	}, time.Minute)

	first, err := client.ListRepositoryPullRequests("octocat", "gh-projects", 200)
	if err != nil {
		t.Fatalf("ListRepositoryPullRequests() first call error = %v", err)
	}

	second, err := client.ListRepositoryPullRequests("octocat", "gh-projects", 200)
	if err != nil {
		t.Fatalf("ListRepositoryPullRequests() second call error = %v", err)
	}

	if callCount != 1 {
		t.Fatalf("ListRepositoryPullRequests() inner call count = %d, want 1", callCount)
	}
	if len(first) != 1 || len(second) != 1 || first[0].ID != second[0].ID {
		t.Fatalf("cached results mismatch: first=%v second=%v", first, second)
	}
}

func TestCachedClientListAssignableUsersCachesReadOperations(t *testing.T) {
	t.Parallel()

	callCount := 0
	client := NewCachedClient(&MockClient{
		ListAssignableUsersFn: func(owner, repo string) ([]User, error) {
			callCount++
			return []User{{Login: "octocat", Name: "The Octocat"}}, nil
		},
	}, time.Minute)

	first, err := client.ListAssignableUsers("octocat", "gh-projects")
	if err != nil {
		t.Fatalf("ListAssignableUsers() first call error = %v", err)
	}

	second, err := client.ListAssignableUsers("octocat", "gh-projects")
	if err != nil {
		t.Fatalf("ListAssignableUsers() second call error = %v", err)
	}

	if callCount != 1 {
		t.Fatalf("ListAssignableUsers() inner call count = %d, want 1", callCount)
	}
	if len(first) != 1 || len(second) != 1 || first[0].Login != second[0].Login {
		t.Fatalf("cached results mismatch: first=%v second=%v", first, second)
	}
}

func TestCachedClientListAssignableUsersErrorsAreNotCached(t *testing.T) {
	t.Parallel()

	callCount := 0
	wantErr := errors.New("network error")
	client := NewCachedClient(&MockClient{
		ListAssignableUsersFn: func(owner, repo string) ([]User, error) {
			callCount++
			return nil, wantErr
		},
	}, time.Minute)

	for range 2 {
		_, err := client.ListAssignableUsers("octocat", "gh-projects")
		if !errors.Is(err, wantErr) {
			t.Fatalf("ListAssignableUsers() error = %v, want %v", err, wantErr)
		}
	}

	if callCount != 2 {
		t.Fatalf("ListAssignableUsers() inner call count = %d, want 2", callCount)
	}
}

func TestCachedClientListAssignableUsersMutationInvalidatesCache(t *testing.T) {
	t.Parallel()

	listCalls := 0
	assignCalls := 0
	client := NewCachedClient(&MockClient{
		ListAssignableUsersFn: func(owner, repo string) ([]User, error) {
			listCalls++
			return []User{{Login: "octocat", Name: "The Octocat"}}, nil
		},
		AssignUserFn: func(owner, repo string, number int, login string) error {
			assignCalls++
			return nil
		},
	}, time.Minute)

	if _, err := client.ListAssignableUsers("octocat", "gh-projects"); err != nil {
		t.Fatalf("ListAssignableUsers() initial call error = %v", err)
	}
	if _, err := client.ListAssignableUsers("octocat", "gh-projects"); err != nil {
		t.Fatalf("ListAssignableUsers() cached call error = %v", err)
	}
	if listCalls != 1 {
		t.Fatalf("ListAssignableUsers() inner call count before mutation = %d, want 1", listCalls)
	}

	if err := client.AssignUser("octocat", "gh-projects", 1, "octocat"); err != nil {
		t.Fatalf("AssignUser() error = %v", err)
	}
	if assignCalls != 1 {
		t.Fatalf("AssignUser() inner call count = %d, want 1", assignCalls)
	}

	if _, err := client.ListAssignableUsers("octocat", "gh-projects"); err != nil {
		t.Fatalf("ListAssignableUsers() after invalidation error = %v", err)
	}
	if listCalls != 2 {
		t.Fatalf("ListAssignableUsers() inner call count after invalidation = %d, want 2", listCalls)
	}
}

func TestCachedClientListViewerOrganizationsCachesResult(t *testing.T) {
	t.Parallel()

	callCount := 0
	client := NewCachedClient(&MockClient{
		ListViewerOrganizationsFn: func() ([]string, error) {
			callCount++
			return []string{"org1", "org2"}, nil
		},
	}, time.Minute)

	first, err := client.ListViewerOrganizations()
	if err != nil {
		t.Fatalf("ListViewerOrganizations() first call error = %v", err)
	}

	second, err := client.ListViewerOrganizations()
	if err != nil {
		t.Fatalf("ListViewerOrganizations() second call error = %v", err)
	}

	if callCount != 1 {
		t.Fatalf("ListViewerOrganizations() inner call count = %d, want 1", callCount)
	}
	if len(first) != 2 || len(second) != 2 || first[0] != second[0] {
		t.Fatalf("cached results mismatch: first=%v second=%v", first, second)
	}
}

func TestCachedClientListViewerLoginCachesResult(t *testing.T) {
	t.Parallel()

	callCount := 0
	client := NewCachedClient(&MockClient{
		ListViewerLoginFn: func() (string, error) {
			callCount++
			return "octocat", nil
		},
	}, time.Minute)

	first, err := client.ListViewerLogin()
	if err != nil {
		t.Fatalf("ListViewerLogin() first call error = %v", err)
	}

	second, err := client.ListViewerLogin()
	if err != nil {
		t.Fatalf("ListViewerLogin() second call error = %v", err)
	}

	if callCount != 1 {
		t.Fatalf("ListViewerLogin() inner call count = %d, want 1", callCount)
	}
	if first != "octocat" || second != "octocat" || first != second {
		t.Fatalf("cached results mismatch: first=%s second=%s", first, second)
	}
}

func TestCachedClientListAllAccessibleProjectsCachesResult(t *testing.T) {
	t.Parallel()

	callCount := 0
	client := NewCachedClient(&MockClient{
		ListAllAccessibleProjectsFn: func() ([]Project, error) {
			callCount++
			return []Project{{ID: "p1", Title: "Project 1"}, {ID: "p2", Title: "Project 2"}}, nil
		},
	}, time.Minute)

	first, err := client.ListAllAccessibleProjects()
	if err != nil {
		t.Fatalf("ListAllAccessibleProjects() first call error = %v", err)
	}

	second, err := client.ListAllAccessibleProjects()
	if err != nil {
		t.Fatalf("ListAllAccessibleProjects() second call error = %v", err)
	}

	if callCount != 1 {
		t.Fatalf("ListAllAccessibleProjects() inner call count = %d, want 1", callCount)
	}
	if len(first) != 2 || len(second) != 2 || first[0].ID != second[0].ID {
		t.Fatalf("cached results mismatch: first=%v second=%v", first, second)
	}
}

func TestCachedClientListAllAccessibleProjectsPassesThroughPartialScopeError(t *testing.T) {
	t.Parallel()

	client := NewCachedClient(&MockClient{
		ListAllAccessibleProjectsFn: func() ([]Project, error) {
			return []Project{{ID: "p1", Title: "Personal Project", Owner: "octocat"}}, ErrMissingScopeReadOrg
		},
	}, time.Minute)

	projects, err := client.ListAllAccessibleProjects()
	if !errors.Is(err, ErrMissingScopeReadOrg) {
		t.Fatalf("ListAllAccessibleProjects() error = %v, want ErrMissingScopeReadOrg", err)
	}
	if len(projects) != 1 || projects[0].ID != "p1" {
		t.Fatalf("ListAllAccessibleProjects() projects = %v, want 1 partial result", projects)
	}
}
