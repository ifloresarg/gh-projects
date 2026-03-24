package github

import (
	"strconv"
	"time"
)

type MockClient struct {
	ListProjectsFn               func(owner string) ([]Project, error)
	GetProjectFn                 func(owner string, number int) (*Project, error)
	GetProjectItemsFn            func(projectID string) ([]ProjectItem, error)
	GetProjectFieldsFn           func(projectID string) ([]ProjectField, error)
	GetProjectViewsFn            func(projectID string) ([]ProjectView, error)
	GetIssueFn                   func(owner, repo string, number int) (*Issue, error)
	GetIssueCommentsFn           func(owner, repo string, number int) ([]Comment, error)
	ListRepositoryLabelsFn       func(owner, repo string) ([]Label, error)
	ListRepositoryPullRequestsFn func(owner, repo string, limit int) ([]PullRequest, error)
	ListAssignableUsersFn        func(owner, repo string) ([]User, error)
	ListIssueTypesFn             func(owner, repo string) ([]IssueType, error)
	GetLinkedPullRequestsFn      func(owner, repo string, issueNumber int) ([]PullRequest, error)
	GetPullRequestFunc           func(owner, repo string, number int) (*PullRequest, error)
	GetPullRequestNodeIDFn       func(owner, repo string, number int) (string, error)
	MoveItemFn                   func(projectID, itemID, fieldID, optionID string) error
	AddCommentFn                 func(owner, repo string, number int, body string) error
	AssignUserFn                 func(owner, repo string, number int, login string) error
	UnassignUserFn               func(owner, repo string, number int, login string) error
	AddLabelFn                   func(owner, repo string, number int, labelName string) error
	RemoveLabelFn                func(owner, repo string, number int, labelID string) error
	UpdateIssueTypeFn            func(issueID string, typeID *string) error
	CloseIssueFn                 func(owner, repo string, number int) error
	ReopenIssueFn                func(owner, repo string, number int) error
	LinkPRToIssueFunc            func(owner, repo string, prNumber, issueNumber int) error
	AddItemToProjectFn           func(projectID string, contentID string) error
	ListViewerOrganizationsFn    func() ([]string, error)
	ListViewerLoginFn            func() (string, error)
	ListAllAccessibleProjectsFn  func() ([]Project, error)
}

var _ GitHubClient = &MockClient{}

func (m *MockClient) ListProjects(owner string) ([]Project, error) {
	if m.ListProjectsFn != nil {
		return m.ListProjectsFn(owner)
	}

	return []Project{
		{ID: "PVT_kwHOA1-project-1", Title: "Platform Roadmap", Number: 1, Owner: owner, ItemCount: 12},
		{ID: "PVT_kwHOA1-project-2", Title: "Engineering Backlog", Number: 2, Owner: owner, ItemCount: 27},
	}, nil
}

func (m *MockClient) GetProject(owner string, number int) (*Project, error) {
	if m.GetProjectFn != nil {
		return m.GetProjectFn(owner, number)
	}

	return &Project{
		ID:        "PVT_kwHOA1-project-detail",
		Title:     "Platform Roadmap",
		Number:    number,
		Owner:     owner,
		ItemCount: 12,
	}, nil
}

func (m *MockClient) GetProjectItems(projectID string) ([]ProjectItem, error) {
	if m.GetProjectItemsFn != nil {
		return m.GetProjectItemsFn(projectID)
	}

	return []ProjectItem{
		{
			ID:       "PVTI_issue_todo",
			Title:    "Implement GraphQL client",
			Type:     "Issue",
			Status:   "Todo",
			StatusID: "status_todo",
			Content: &Issue{
				ID:        "I_kwDOA1-101",
				Number:    101,
				Title:     "Implement GraphQL client",
				Body:      "Build the core GitHub GraphQL client for Projects v2.",
				State:     "OPEN",
				Author:    User{Login: "octocat", Name: "The Octocat"},
				Assignees: []User{{Login: "ifloresarg", Name: "Ignacio Flores"}},
				Labels:    []Label{{ID: "LA_bug", Name: "bug", Color: "d73a4a"}},
				CreatedAt: time.Date(2026, time.March, 18, 9, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, time.March, 19, 15, 30, 0, 0, time.UTC),
				RepoOwner: "ifloresarg",
				RepoName:  "gh-projects",
			},
			RepoOwner:     "ifloresarg",
			RepoName:      "gh-projects",
			ContentNumber: 101,
		},
		{
			ID:       "PVTI_issue_progress",
			Title:    "Add cache invalidation",
			Type:     "Issue",
			Status:   "In Progress",
			StatusID: "status_in_progress",
			Content: &Issue{
				ID:        "I_kwDOA1-102",
				Number:    102,
				Title:     "Add cache invalidation",
				Body:      "Cache entries should refresh when project items move columns.",
				State:     "OPEN",
				Author:    User{Login: "monalisa", Name: "Mona Lisa"},
				Assignees: []User{{Login: "ifloresarg", Name: "Ignacio Flores"}},
				Labels:    []Label{{ID: "LA_enhancement", Name: "enhancement", Color: "a2eeef"}},
				CreatedAt: time.Date(2026, time.March, 17, 11, 15, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, time.March, 20, 8, 45, 0, 0, time.UTC),
				RepoOwner: "ifloresarg",
				RepoName:  "gh-projects",
			},
			RepoOwner:     "ifloresarg",
			RepoName:      "gh-projects",
			ContentNumber: 102,
		},
		{
			ID:       "PVTI_pr_done",
			Title:    "Refine board rendering",
			Type:     "PullRequest",
			Status:   "Done",
			StatusID: "status_done",
			Content: &PullRequest{
				ID:        "PR_kwDOA1-55",
				Number:    55,
				Title:     "Refine board rendering",
				State:     "MERGED",
				Author:    User{Login: "hubot", Name: "Hubot"},
				URL:       "https://github.com/ifloresarg/gh-projects/pull/55",
				RepoOwner: "ifloresarg",
				RepoName:  "gh-projects",
			},
			RepoOwner:     "ifloresarg",
			RepoName:      "gh-projects",
			ContentNumber: 55,
		},
	}, nil
}

func (m *MockClient) GetProjectFields(projectID string) ([]ProjectField, error) {
	if m.GetProjectFieldsFn != nil {
		return m.GetProjectFieldsFn(projectID)
	}

	return []ProjectField{
		{
			ID:       "PVTSSF_status",
			Name:     "Status",
			DataType: "SINGLE_SELECT",
			Options: []FieldOption{
				{ID: "status_todo", Name: "Todo"},
				{ID: "status_in_progress", Name: "In Progress"},
				{ID: "status_done", Name: "Done"},
			},
		},
	}, nil
}

func (m *MockClient) GetProjectViews(projectID string) ([]ProjectView, error) {
	if m.GetProjectViewsFn != nil {
		return m.GetProjectViewsFn(projectID)
	}

	return []ProjectView{
		{ID: "PVT_view_1", Name: "All Items", Number: 1, Layout: "BOARD_LAYOUT", Filter: ""},
		{ID: "PVT_view_2", Name: "Development", Number: 2, Layout: "BOARD_LAYOUT", Filter: `-status:Backlog,"To Design"`},
		{ID: "PVT_view_3", Name: "Design", Number: 3, Layout: "TABLE_LAYOUT", Filter: ""},
	}, nil
}

func (m *MockClient) GetIssue(owner, repo string, number int) (*Issue, error) {
	if m.GetIssueFn != nil {
		return m.GetIssueFn(owner, repo, number)
	}

	return &Issue{
		ID:     "I_kwDOA1-issue-detail",
		Number: number,
		Title:  "Keyboard navigation skips hidden items",
		Body:   "When filtering the board, keyboard navigation should ignore hidden cards and preserve selection.",
		State:  "OPEN",
		Author: User{Login: "octocat", Name: "The Octocat"},
		Assignees: []User{
			{Login: "ifloresarg", Name: "Ignacio Flores"},
			{Login: "ifloresarg", Name: "Ignacio Flores"},
		},
		Labels: []Label{
			{ID: "LA_bug", Name: "bug", Color: "d73a4a"},
			{ID: "LA_docs", Name: "documentation", Color: "0075ca"},
		},
		CreatedAt: time.Date(2026, time.March, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.March, 20, 9, 20, 0, 0, time.UTC),
		RepoOwner: owner,
		RepoName:  repo,
	}, nil
}

func (m *MockClient) GetIssueComments(owner, repo string, number int) ([]Comment, error) {
	if m.GetIssueCommentsFn != nil {
		return m.GetIssueCommentsFn(owner, repo, number)
	}

	return []Comment{
		{
			ID:        "IC_kwDOA1-1",
			Body:      "I can reproduce this when switching filters quickly.",
			Author:    User{Login: "monalisa", Name: "Mona Lisa"},
			CreatedAt: time.Date(2026, time.March, 16, 13, 5, 0, 0, time.UTC),
		},
		{
			ID:        "IC_kwDOA1-2",
			Body:      "Fix should keep the active index aligned with visible cards only.",
			Author:    User{Login: "hubot", Name: "Hubot"},
			CreatedAt: time.Date(2026, time.March, 17, 8, 40, 0, 0, time.UTC),
		},
	}, nil
}

func (m *MockClient) ListRepositoryLabels(owner, repo string) ([]Label, error) {
	if m.ListRepositoryLabelsFn != nil {
		return m.ListRepositoryLabelsFn(owner, repo)
	}

	return []Label{
		{ID: "LA_bug", Name: "bug", Color: "d73a4a"},
		{ID: "LA_enhancement", Name: "enhancement", Color: "a2eeef"},
		{ID: "LA_documentation", Name: "documentation", Color: "0075ca"},
	}, nil
}

func (m *MockClient) ListRepositoryPullRequests(owner, repo string, limit int) ([]PullRequest, error) {
	if m.ListRepositoryPullRequestsFn != nil {
		return m.ListRepositoryPullRequestsFn(owner, repo, limit)
	}

	return nil, nil
}

func (m *MockClient) ListAssignableUsers(owner, repo string) ([]User, error) {
	if m.ListAssignableUsersFn != nil {
		return m.ListAssignableUsersFn(owner, repo)
	}

	return []User{
		{Login: "octocat", Name: "The Octocat"},
		{Login: "monalisa", Name: "Mona Lisa"},
		{Login: "hubot", Name: "Hubot"},
	}, nil
}

func (m *MockClient) ListIssueTypes(owner, repo string) ([]IssueType, error) {
	if m.ListIssueTypesFn != nil {
		return m.ListIssueTypesFn(owner, repo)
	}

	return []IssueType{
		{ID: "IT_1", Name: "Bug"},
		{ID: "IT_2", Name: "Feature"},
		{ID: "IT_3", Name: "Enhancement"},
	}, nil
}

func (m *MockClient) GetLinkedPullRequests(owner, repo string, issueNumber int) ([]PullRequest, error) {
	if m.GetLinkedPullRequestsFn != nil {
		return m.GetLinkedPullRequestsFn(owner, repo, issueNumber)
	}

	return []PullRequest{
		{
			ID:        "PR_kwDOA1-linked-1",
			Number:    56,
			Title:     "Fix keyboard navigation for filtered boards",
			State:     "OPEN",
			Author:    User{Login: "ifloresarg", Name: "Ignacio Flores"},
			URL:       "https://github.com/" + owner + "/" + repo + "/pull/56",
			RepoOwner: owner,
			RepoName:  repo,
		},
	}, nil
}

func (m *MockClient) GetPullRequestNodeID(owner, repo string, number int) (string, error) {
	if m.GetPullRequestNodeIDFn != nil {
		return m.GetPullRequestNodeIDFn(owner, repo, number)
	}

	return "PR_kwDOA1_node_" + owner + "_" + repo + "_" + strconv.Itoa(number), nil
}

func (m *MockClient) GetPullRequest(owner, repo string, number int) (*PullRequest, error) {
	if m.GetPullRequestFunc != nil {
		return m.GetPullRequestFunc(owner, repo, number)
	}

	return nil, nil
}

func (m *MockClient) MoveItem(projectID, itemID, fieldID, optionID string) error {
	if m.MoveItemFn != nil {
		return m.MoveItemFn(projectID, itemID, fieldID, optionID)
	}

	return nil
}

func (m *MockClient) AddComment(owner, repo string, number int, body string) error {
	if m.AddCommentFn != nil {
		return m.AddCommentFn(owner, repo, number, body)
	}

	return nil
}

func (m *MockClient) AssignUser(owner, repo string, number int, login string) error {
	if m.AssignUserFn != nil {
		return m.AssignUserFn(owner, repo, number, login)
	}

	return nil
}

func (m *MockClient) UnassignUser(owner, repo string, number int, login string) error {
	if m.UnassignUserFn != nil {
		return m.UnassignUserFn(owner, repo, number, login)
	}

	return nil
}

func (m *MockClient) AddLabel(owner, repo string, number int, labelName string) error {
	if m.AddLabelFn != nil {
		return m.AddLabelFn(owner, repo, number, labelName)
	}

	return nil
}

func (m *MockClient) RemoveLabel(owner, repo string, number int, labelID string) error {
	if m.RemoveLabelFn != nil {
		return m.RemoveLabelFn(owner, repo, number, labelID)
	}

	return nil
}

func (m *MockClient) UpdateIssueType(issueID string, typeID *string) error {
	if m.UpdateIssueTypeFn != nil {
		return m.UpdateIssueTypeFn(issueID, typeID)
	}

	return nil
}

func (m *MockClient) CloseIssue(owner, repo string, number int) error {
	if m.CloseIssueFn != nil {
		return m.CloseIssueFn(owner, repo, number)
	}

	return nil
}

func (m *MockClient) ReopenIssue(owner, repo string, number int) error {
	if m.ReopenIssueFn != nil {
		return m.ReopenIssueFn(owner, repo, number)
	}

	return nil
}

func (m *MockClient) LinkPRToIssue(owner, repo string, prNumber, issueNumber int) error {
	if m.LinkPRToIssueFunc != nil {
		return m.LinkPRToIssueFunc(owner, repo, prNumber, issueNumber)
	}

	return nil
}

func (m *MockClient) AddItemToProject(projectID string, contentID string) error {
	if m.AddItemToProjectFn != nil {
		return m.AddItemToProjectFn(projectID, contentID)
	}

	return nil
}

func (m *MockClient) ListViewerOrganizations() ([]string, error) {
	if m.ListViewerOrganizationsFn != nil {
		return m.ListViewerOrganizationsFn()
	}

	return nil, nil
}

func (m *MockClient) ListViewerLogin() (string, error) {
	if m.ListViewerLoginFn != nil {
		return m.ListViewerLoginFn()
	}

	return "", nil
}

func (m *MockClient) ListAllAccessibleProjects() ([]Project, error) {
	if m.ListAllAccessibleProjectsFn != nil {
		return m.ListAllAccessibleProjectsFn()
	}

	return nil, nil
}
