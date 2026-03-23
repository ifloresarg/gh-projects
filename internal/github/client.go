package github

type GitHubClient interface {
	ListProjects(owner string) ([]Project, error)
	GetProject(owner string, number int) (*Project, error)
	GetProjectItems(projectID string) ([]ProjectItem, error)
	GetProjectFields(projectID string) ([]ProjectField, error)
	GetProjectViews(projectID string) ([]ProjectView, error)

	GetIssue(owner, repo string, number int) (*Issue, error)
	GetIssueComments(owner, repo string, number int) ([]Comment, error)

	ListRepositoryLabels(owner, repo string) ([]Label, error)
	ListRepositoryPullRequests(owner, repo string, limit int) ([]PullRequest, error)
	ListAssignableUsers(owner, repo string) ([]User, error)
	ListIssueTypes(owner, repo string) ([]IssueType, error)
	GetLinkedPullRequests(owner, repo string, issueNumber int) ([]PullRequest, error)
	GetPullRequest(owner, repo string, number int) (*PullRequest, error)
	GetPullRequestNodeID(owner, repo string, number int) (string, error)

	MoveItem(projectID, itemID, fieldID, optionID string) error
	AddComment(owner, repo string, number int, body string) error
	AssignUser(owner, repo string, number int, login string) error
	UnassignUser(owner, repo string, number int, login string) error
	AddLabel(owner, repo string, number int, labelName string) error
	RemoveLabel(owner, repo string, number int, labelID string) error
	UpdateIssueType(issueID string, typeID *string) error
	CloseIssue(owner, repo string, number int) error
	ReopenIssue(owner, repo string, number int) error
	LinkPRToIssue(owner, repo string, prNumber, issueNumber int) error
	AddItemToProject(projectID string, contentID string) error
}
