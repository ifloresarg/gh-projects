package github

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ifloresarg/gh-projects/internal/cache"
)

func newTestDiskCache(t *testing.T) *cache.DiskCache {
	t.Helper()

	diskCache, err := cache.NewDiskCache(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskCache() error = %v", err)
	}

	return diskCache
}

func TestCachedClientListProjectsCachesReadOperations(t *testing.T) {
	t.Parallel()

	callCount := 0
	client := NewCachedClient(&MockClient{
		ListProjectsFn: func(owner string) ([]Project, error) {
			callCount++
			return []Project{{ID: "p1", Title: "Roadmap", Owner: owner}}, nil
		},
	}, time.Minute, nil)

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
	}, time.Minute, nil)

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
	}, time.Minute, nil)

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
	}, time.Minute, nil)

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
	}, time.Minute, nil)

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
	}, time.Minute, nil)

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
	}, time.Minute, nil)

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
	}, time.Minute, nil)

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

func TestCachedClientUpdateIssueBodyInvalidatesReadCachesAfterSuccessfulMutation(t *testing.T) {
	t.Parallel()

	listCalls := 0
	updateCalls := 0
	client := NewCachedClient(&MockClient{
		ListProjectsFn: func(owner string) ([]Project, error) {
			listCalls++
			return []Project{{ID: "p1", Title: "Roadmap", Owner: owner}}, nil
		},
		UpdateIssueBodyFn: func(issueID string, body string) error {
			updateCalls++
			return nil
		},
	}, time.Minute, nil)

	if _, err := client.ListProjects("octocat"); err != nil {
		t.Fatalf("ListProjects() initial call error = %v", err)
	}
	if _, err := client.ListProjects("octocat"); err != nil {
		t.Fatalf("ListProjects() cached call error = %v", err)
	}
	if listCalls != 1 {
		t.Fatalf("ListProjects() inner call count before mutation = %d, want 1", listCalls)
	}

	if err := client.UpdateIssueBody("I_123", "new body"); err != nil {
		t.Fatalf("UpdateIssueBody() error = %v", err)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateIssueBody() inner call count = %d, want 1", updateCalls)
	}

	if _, err := client.ListProjects("octocat"); err != nil {
		t.Fatalf("ListProjects() after invalidation error = %v", err)
	}
	if listCalls != 2 {
		t.Fatalf("ListProjects() inner call count after invalidation = %d, want 2", listCalls)
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
	}, time.Minute, nil)

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
	}, time.Minute, nil)

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
	}, time.Minute, nil)

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
	}, time.Minute, nil)

	projects, err := client.ListAllAccessibleProjects()
	if !errors.Is(err, ErrMissingScopeReadOrg) {
		t.Fatalf("ListAllAccessibleProjects() error = %v, want ErrMissingScopeReadOrg", err)
	}
	if len(projects) != 1 || projects[0].ID != "p1" {
		t.Fatalf("ListAllAccessibleProjects() projects = %v, want 1 partial result", projects)
	}
}

func TestCachedClientDiskPersist(t *testing.T) {
	t.Parallel()

	t.Run("GetProject", func(t *testing.T) {
		t.Parallel()

		diskCache := newTestDiskCache(t)
		calls := 0
		want := &Project{ID: "p1", Title: "Roadmap", Owner: "octocat", Number: 7, ItemCount: 3}

		client1 := NewCachedClient(&MockClient{
			GetProjectFn: func(owner string, number int) (*Project, error) {
				calls++
				return want, nil
			},
		}, time.Minute, diskCache)

		got, err := client1.GetProject("octocat", 7)
		if err != nil {
			t.Fatalf("GetProject() first client error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GetProject() first client = %#v, want %#v", got, want)
		}

		client2 := NewCachedClient(&MockClient{
			GetProjectFn: func(owner string, number int) (*Project, error) {
				calls++
				return nil, errors.New("api should not be called")
			},
		}, time.Minute, diskCache)

		got, err = client2.GetProject("octocat", 7)
		if err != nil {
			t.Fatalf("GetProject() second client error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GetProject() second client = %#v, want %#v", got, want)
		}
		if calls != 1 {
			t.Fatalf("GetProject() API calls = %d, want 1", calls)
		}
	})

	t.Run("GetProjectItems", func(t *testing.T) {
		t.Parallel()

		diskCache := newTestDiskCache(t)
		calls := 0
		want := []ProjectItem{{ID: "item-1", Title: "Cache me", Type: "Issue", Status: "Todo", StatusID: "todo", Content: &Issue{ID: "issue-1", Number: 11, Title: "Cache me"}}}

		client1 := NewCachedClient(&MockClient{
			GetProjectItemsFn: func(projectID string) ([]ProjectItem, error) {
				calls++
				return want, nil
			},
		}, time.Minute, diskCache)

		got, err := client1.GetProjectItems("PVT_items")
		if err != nil {
			t.Fatalf("GetProjectItems() first client error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GetProjectItems() first client = %#v, want %#v", got, want)
		}

		client2 := NewCachedClient(&MockClient{
			GetProjectItemsFn: func(projectID string) ([]ProjectItem, error) {
				calls++
				return nil, errors.New("api should not be called")
			},
		}, time.Minute, diskCache)

		got, err = client2.GetProjectItems("PVT_items")
		if err != nil {
			t.Fatalf("GetProjectItems() second client error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GetProjectItems() second client = %#v, want %#v", got, want)
		}
		if calls != 1 {
			t.Fatalf("GetProjectItems() API calls = %d, want 1", calls)
		}
	})

	t.Run("GetProjectFields", func(t *testing.T) {
		t.Parallel()

		diskCache := newTestDiskCache(t)
		calls := 0
		want := []ProjectField{{ID: "field-1", Name: "Status", DataType: "SINGLE_SELECT", Options: []FieldOption{{ID: "todo", Name: "Todo"}}}}

		client1 := NewCachedClient(&MockClient{
			GetProjectFieldsFn: func(projectID string) ([]ProjectField, error) {
				calls++
				return want, nil
			},
		}, time.Minute, diskCache)

		got, err := client1.GetProjectFields("PVT_fields")
		if err != nil {
			t.Fatalf("GetProjectFields() first client error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GetProjectFields() first client = %#v, want %#v", got, want)
		}

		client2 := NewCachedClient(&MockClient{
			GetProjectFieldsFn: func(projectID string) ([]ProjectField, error) {
				calls++
				return nil, errors.New("api should not be called")
			},
		}, time.Minute, diskCache)

		got, err = client2.GetProjectFields("PVT_fields")
		if err != nil {
			t.Fatalf("GetProjectFields() second client error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GetProjectFields() second client = %#v, want %#v", got, want)
		}
		if calls != 1 {
			t.Fatalf("GetProjectFields() API calls = %d, want 1", calls)
		}
	})

	t.Run("GetProjectViews", func(t *testing.T) {
		t.Parallel()

		diskCache := newTestDiskCache(t)
		calls := 0
		want := []ProjectView{{ID: "view-1", Name: "Board", Number: 1, Layout: "BOARD_LAYOUT", Filter: ""}}

		client1 := NewCachedClient(&MockClient{
			GetProjectViewsFn: func(projectID string) ([]ProjectView, error) {
				calls++
				return want, nil
			},
		}, time.Minute, diskCache)

		got, err := client1.GetProjectViews("PVT_views")
		if err != nil {
			t.Fatalf("GetProjectViews() first client error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GetProjectViews() first client = %#v, want %#v", got, want)
		}

		client2 := NewCachedClient(&MockClient{
			GetProjectViewsFn: func(projectID string) ([]ProjectView, error) {
				calls++
				return nil, errors.New("api should not be called")
			},
		}, time.Minute, diskCache)

		got, err = client2.GetProjectViews("PVT_views")
		if err != nil {
			t.Fatalf("GetProjectViews() second client error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GetProjectViews() second client = %#v, want %#v", got, want)
		}
		if calls != 1 {
			t.Fatalf("GetProjectViews() API calls = %d, want 1", calls)
		}
	})
}

func TestCachedClientInvalidateBoth(t *testing.T) {
	t.Parallel()

	diskCache := newTestDiskCache(t)
	projectCalls := 0
	itemsCalls := 0

	client := NewCachedClient(&MockClient{
		GetProjectFn: func(owner string, number int) (*Project, error) {
			projectCalls++
			return &Project{ID: "p1", Title: "Roadmap", Owner: owner, Number: number}, nil
		},
		GetProjectItemsFn: func(projectID string) ([]ProjectItem, error) {
			itemsCalls++
			return []ProjectItem{{ID: "item-1", Title: "Cache me"}}, nil
		},
	}, time.Minute, diskCache)

	if _, err := client.GetProject("octocat", 7); err != nil {
		t.Fatalf("GetProject() initial error = %v", err)
	}
	if _, err := client.GetProjectItems("PVT_1"); err != nil {
		t.Fatalf("GetProjectItems() initial error = %v", err)
	}

	client.InvalidateAll()

	var projectFromDisk []Project
	err := diskCache.Load("project:octocat:7", &projectFromDisk)
	if !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("Load(project) error = %v, want %v", err, cache.ErrCacheMiss)
	}

	var itemsFromDisk []ProjectItem
	err = diskCache.Load("items:PVT_1", &itemsFromDisk)
	if !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("Load(items) error = %v, want %v", err, cache.ErrCacheMiss)
	}

	if _, err := client.GetProject("octocat", 7); err != nil {
		t.Fatalf("GetProject() after invalidate error = %v", err)
	}
	if _, err := client.GetProjectItems("PVT_1"); err != nil {
		t.Fatalf("GetProjectItems() after invalidate error = %v", err)
	}

	if projectCalls != 2 {
		t.Fatalf("GetProject() API calls = %d, want 2", projectCalls)
	}
	if itemsCalls != 2 {
		t.Fatalf("GetProjectItems() API calls = %d, want 2", itemsCalls)
	}
}

func TestCachedClientNilDisk(t *testing.T) {
	t.Parallel()

	projectCalls := 0
	itemsCalls := 0
	client := NewCachedClient(&MockClient{
		GetProjectFn: func(owner string, number int) (*Project, error) {
			projectCalls++
			return &Project{ID: "p1", Title: "Roadmap", Owner: owner, Number: number}, nil
		},
		GetProjectItemsFn: func(projectID string) ([]ProjectItem, error) {
			itemsCalls++
			return []ProjectItem{{ID: "item-1", Title: "Cache me"}}, nil
		},
	}, time.Minute, nil)

	firstProject, err := client.GetProject("octocat", 7)
	if err != nil {
		t.Fatalf("GetProject() first call error = %v", err)
	}
	secondProject, err := client.GetProject("octocat", 7)
	if err != nil {
		t.Fatalf("GetProject() second call error = %v", err)
	}
	if !reflect.DeepEqual(firstProject, secondProject) {
		t.Fatalf("GetProject() cached results mismatch: first=%#v second=%#v", firstProject, secondProject)
	}

	firstItems, err := client.GetProjectItems("PVT_1")
	if err != nil {
		t.Fatalf("GetProjectItems() first call error = %v", err)
	}
	secondItems, err := client.GetProjectItems("PVT_1")
	if err != nil {
		t.Fatalf("GetProjectItems() second call error = %v", err)
	}
	if !reflect.DeepEqual(firstItems, secondItems) {
		t.Fatalf("GetProjectItems() cached results mismatch: first=%#v second=%#v", firstItems, secondItems)
	}

	client.InvalidateAll()

	if projectCalls != 1 {
		t.Fatalf("GetProject() API calls = %d, want 1", projectCalls)
	}
	if itemsCalls != 1 {
		t.Fatalf("GetProjectItems() API calls = %d, want 1", itemsCalls)
	}
}

func TestCachedClientDiskFallthrough(t *testing.T) {
	t.Parallel()

	diskCache := newTestDiskCache(t)
	cacheFile := filepath.Join(diskCache.CacheDir(), "items:PVT_broken.json")
	if err := os.WriteFile(cacheFile, []byte("not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	apiCalls := 0
	client := NewCachedClient(&MockClient{
		GetProjectItemsFn: func(projectID string) ([]ProjectItem, error) {
			apiCalls++
			return []ProjectItem{{ID: "item-1", Title: "Recovered"}}, nil
		},
	}, time.Minute, diskCache)

	items, err := client.GetProjectItems("PVT_broken")
	if err != nil {
		t.Fatalf("GetProjectItems() error = %v", err)
	}
	if apiCalls != 1 {
		t.Fatalf("GetProjectItems() API calls = %d, want 1", apiCalls)
	}
	if len(items) != 1 || items[0].Title != "Recovered" {
		t.Fatalf("GetProjectItems() = %#v, want recovered API data", items)
	}

	var persisted []ProjectItem
	if err := diskCache.Load("items:PVT_broken", &persisted); err != nil {
		t.Fatalf("Load() persisted cache error = %v", err)
	}
	if !reflect.DeepEqual(persisted, items) {
		t.Fatalf("Load() persisted cache = %#v, want %#v", persisted, items)
	}
}
