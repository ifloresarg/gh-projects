package github

import (
	"errors"
	"fmt"
	"time"

	"github.com/ifloresarg/gh-projects/internal/cache"
)

// CachedClient wraps a GitHubClient with in-memory caching for read operations.
type CachedClient struct {
	inner       GitHubClient
	diskCache   *cache.DiskCache
	projects    *cache.Cache[[]Project]
	items       *cache.Cache[[]ProjectItem]
	fields      *cache.Cache[[]ProjectField]
	views       *cache.Cache[[]ProjectView]
	issues      *cache.Cache[*Issue]
	comments    *cache.Cache[[]Comment]
	labels      *cache.Cache[[]Label]
	prs         *cache.Cache[[]PullRequest]
	users       *cache.Cache[[]User]
	issueTypes  *cache.Cache[[]IssueType]
	viewerOrgs  *cache.Cache[[]string]
	viewerLogin *cache.Cache[string]
	allProjects *cache.Cache[[]Project]
}

func loadBoardCache[T any](diskCache *cache.DiskCache, key string, target *T) bool {
	if diskCache == nil {
		return false
	}

	err := diskCache.Load(key, target)
	if err == nil {
		return true
	}
	if errors.Is(err, cache.ErrCacheMiss) {
		return false
	}

	return false
}

func saveBoardCache(diskCache *cache.DiskCache, key string, value any) {
	if diskCache != nil {
		_ = diskCache.Save(key, value)
	}
}

// NewCachedClient creates a CachedClient wrapping inner with the given TTL.
func NewCachedClient(inner GitHubClient, ttl time.Duration, diskCache *cache.DiskCache) *CachedClient {
	return &CachedClient{
		inner:       inner,
		diskCache:   diskCache,
		projects:    cache.New[[]Project](ttl),
		items:       cache.New[[]ProjectItem](ttl),
		fields:      cache.New[[]ProjectField](ttl),
		views:       cache.New[[]ProjectView](ttl),
		issues:      cache.New[*Issue](ttl),
		comments:    cache.New[[]Comment](ttl),
		labels:      cache.New[[]Label](ttl),
		prs:         cache.New[[]PullRequest](ttl),
		users:       cache.New[[]User](ttl),
		issueTypes:  cache.New[[]IssueType](ttl),
		viewerOrgs:  cache.New[[]string](ttl),
		viewerLogin: cache.New[string](ttl),
		allProjects: cache.New[[]Project](ttl),
	}
}

// InvalidateAll clears all caches (called on R key refresh).
func (c *CachedClient) InvalidateAll() {
	c.projects.InvalidateAll()
	c.items.InvalidateAll()
	c.fields.InvalidateAll()
	c.views.InvalidateAll()
	c.issues.InvalidateAll()
	c.comments.InvalidateAll()
	c.labels.InvalidateAll()
	c.prs.InvalidateAll()
	c.users.InvalidateAll()
	c.issueTypes.InvalidateAll()
	c.viewerOrgs.InvalidateAll()
	c.viewerLogin.InvalidateAll()
	c.allProjects.InvalidateAll()
	if c.diskCache != nil {
		_ = c.diskCache.InvalidateAll()
	}
}

// InvalidateMemory clears all in-memory caches without touching disk.
// Used before background SWR refresh so API is actually called.
func (c *CachedClient) InvalidateMemory() {
	c.projects.InvalidateAll()
	c.items.InvalidateAll()
	c.fields.InvalidateAll()
	c.views.InvalidateAll()
	c.issues.InvalidateAll()
	c.comments.InvalidateAll()
	c.labels.InvalidateAll()
	c.prs.InvalidateAll()
	c.users.InvalidateAll()
	c.issueTypes.InvalidateAll()
	c.viewerOrgs.InvalidateAll()
	c.viewerLogin.InvalidateAll()
	c.allProjects.InvalidateAll()
	// NOTE: intentionally does NOT call c.diskCache.InvalidateAll()
}

func (c *CachedClient) ListProjects(owner string) ([]Project, error) {
	key := "projects:" + owner
	if val, ok := c.projects.Get(key); ok {
		return val, nil
	}
	result, err := c.inner.ListProjects(owner)
	if err != nil {
		return nil, err
	}
	c.projects.Set(key, result)
	return result, nil
}

func (c *CachedClient) GetProject(owner string, number int) (*Project, error) {
	key := "project:" + owner + ":" + fmt.Sprint(number)
	if val, ok := c.projects.Get(key); ok && len(val) > 0 {
		project := val[0]
		return &project, nil
	}
	var cached []Project
	if loadBoardCache(c.diskCache, key, &cached) {
		c.projects.Set(key, cached)
		if len(cached) > 0 {
			project := cached[0]
			return &project, nil
		}
	}
	result, err := c.inner.GetProject(owner, number)
	if err != nil {
		return nil, err
	}
	if result != nil {
		cached := []Project{*result}
		c.projects.Set(key, cached)
		saveBoardCache(c.diskCache, key, cached)
	}
	return result, nil
}

func (c *CachedClient) GetProjectItems(projectID string) ([]ProjectItem, error) {
	key := "items:" + projectID
	if val, ok := c.items.Get(key); ok {
		return val, nil
	}
	var cached []ProjectItem
	if loadBoardCache(c.diskCache, key, &cached) {
		c.items.Set(key, cached)
		return cached, nil
	}
	result, err := c.inner.GetProjectItems(projectID)
	if err != nil {
		return nil, err
	}
	c.items.Set(key, result)
	saveBoardCache(c.diskCache, key, result)
	return result, nil
}

func (c *CachedClient) GetProjectFields(projectID string) ([]ProjectField, error) {
	key := "fields:" + projectID
	if val, ok := c.fields.Get(key); ok {
		return val, nil
	}
	var cached []ProjectField
	if loadBoardCache(c.diskCache, key, &cached) {
		c.fields.Set(key, cached)
		return cached, nil
	}
	result, err := c.inner.GetProjectFields(projectID)
	if err != nil {
		return nil, err
	}
	c.fields.Set(key, result)
	saveBoardCache(c.diskCache, key, result)
	return result, nil
}

func (c *CachedClient) GetProjectViews(projectID string) ([]ProjectView, error) {
	key := "views:" + projectID
	if val, ok := c.views.Get(key); ok {
		return val, nil
	}
	var cached []ProjectView
	if loadBoardCache(c.diskCache, key, &cached) {
		c.views.Set(key, cached)
		return cached, nil
	}
	result, err := c.inner.GetProjectViews(projectID)
	if err != nil {
		return nil, err
	}
	c.views.Set(key, result)
	saveBoardCache(c.diskCache, key, result)
	return result, nil
}

func (c *CachedClient) GetIssue(owner, repo string, number int) (*Issue, error) {
	key := "issue:" + owner + "/" + repo + "#" + fmt.Sprint(number)
	result, err := c.inner.GetIssue(owner, repo, number)
	if err != nil {
		return nil, err
	}
	if result != nil {
		c.issues.Set(key, result)
	}
	return result, nil
}

func (c *CachedClient) GetIssueComments(owner, repo string, number int) ([]Comment, error) {
	key := "comments:" + owner + "/" + repo + "#" + fmt.Sprint(number)
	if val, ok := c.comments.Get(key); ok {
		return val, nil
	}
	result, err := c.inner.GetIssueComments(owner, repo, number)
	if err != nil {
		return nil, err
	}
	c.comments.Set(key, result)
	return result, nil
}

func (c *CachedClient) ListRepositoryLabels(owner, repo string) ([]Label, error) {
	key := "labels:" + owner + "/" + repo
	if val, ok := c.labels.Get(key); ok {
		return val, nil
	}
	result, err := c.inner.ListRepositoryLabels(owner, repo)
	if err != nil {
		return nil, err
	}
	c.labels.Set(key, result)
	return result, nil
}

func (c *CachedClient) ListRepositoryPullRequests(owner, repo string, limit int) ([]PullRequest, error) {
	key := "repoPRs:" + owner + ":" + repo + ":" + fmt.Sprint(limit)
	if val, ok := c.prs.Get(key); ok {
		return val, nil
	}
	result, err := c.inner.ListRepositoryPullRequests(owner, repo, limit)
	if err != nil {
		return nil, err
	}
	c.prs.Set(key, result)
	return result, nil
}

func (c *CachedClient) ListAssignableUsers(owner, repo string) ([]User, error) {
	key := "users:" + owner + "/" + repo
	if val, ok := c.users.Get(key); ok {
		return val, nil
	}
	result, err := c.inner.ListAssignableUsers(owner, repo)
	if err != nil {
		return nil, err
	}
	c.users.Set(key, result)
	return result, nil
}

func (c *CachedClient) ListIssueTypes(owner, repo string) ([]IssueType, error) {
	key := "issueTypes:" + owner + "/" + repo
	if cached, ok := c.issueTypes.Get(key); ok {
		return cached, nil
	}
	result, err := c.inner.ListIssueTypes(owner, repo)
	if err != nil {
		return nil, err
	}
	c.issueTypes.Set(key, result)
	return result, nil
}

func (c *CachedClient) GetLinkedPullRequests(owner, repo string, issueNumber int) ([]PullRequest, error) {
	key := "linked-prs:" + owner + "/" + repo + "#" + fmt.Sprint(issueNumber)
	if val, ok := c.prs.Get(key); ok {
		return val, nil
	}
	result, err := c.inner.GetLinkedPullRequests(owner, repo, issueNumber)
	if err != nil {
		return nil, err
	}
	c.prs.Set(key, result)
	return result, nil
}

func (c *CachedClient) GetPullRequestNodeID(owner, repo string, number int) (string, error) {
	return c.inner.GetPullRequestNodeID(owner, repo, number)
}

func (c *CachedClient) GetPullRequest(owner, repo string, number int) (*PullRequest, error) {
	return c.inner.GetPullRequest(owner, repo, number)
}

func (c *CachedClient) MoveItem(projectID, itemID, fieldID, optionID string) error {
	c.InvalidateAll()
	return c.inner.MoveItem(projectID, itemID, fieldID, optionID)
}

func (c *CachedClient) AddComment(owner, repo string, number int, body string) error {
	c.InvalidateAll()
	return c.inner.AddComment(owner, repo, number, body)
}

func (c *CachedClient) AssignUser(owner, repo string, number int, login string) error {
	c.InvalidateAll()
	return c.inner.AssignUser(owner, repo, number, login)
}

func (c *CachedClient) UnassignUser(owner, repo string, number int, login string) error {
	c.InvalidateAll()
	return c.inner.UnassignUser(owner, repo, number, login)
}

func (c *CachedClient) AddLabel(owner, repo string, number int, labelName string) error {
	c.InvalidateAll()
	return c.inner.AddLabel(owner, repo, number, labelName)
}

func (c *CachedClient) RemoveLabel(owner, repo string, number int, labelID string) error {
	c.InvalidateAll()
	return c.inner.RemoveLabel(owner, repo, number, labelID)
}

func (c *CachedClient) UpdateIssueType(issueID string, typeID *string) error {
	if err := c.inner.UpdateIssueType(issueID, typeID); err != nil {
		return err
	}
	c.InvalidateAll()
	return nil
}

func (c *CachedClient) CloseIssue(owner, repo string, number int) error {
	c.InvalidateAll()
	return c.inner.CloseIssue(owner, repo, number)
}

func (c *CachedClient) ReopenIssue(owner, repo string, number int) error {
	c.InvalidateAll()
	return c.inner.ReopenIssue(owner, repo, number)
}

func (c *CachedClient) LinkPRToIssue(owner, repo string, prNumber, issueNumber int) error {
	c.InvalidateAll()
	return c.inner.LinkPRToIssue(owner, repo, prNumber, issueNumber)
}

func (c *CachedClient) AddItemToProject(projectID string, contentID string) error {
	c.InvalidateAll()
	return c.inner.AddItemToProject(projectID, contentID)
}

func (c *CachedClient) ListViewerOrganizations() ([]string, error) {
	key := "viewer-orgs"
	if val, ok := c.viewerOrgs.Get(key); ok {
		return val, nil
	}
	result, err := c.inner.ListViewerOrganizations()
	if err != nil {
		return nil, err
	}
	c.viewerOrgs.Set(key, result)
	return result, nil
}

func (c *CachedClient) ListViewerLogin() (string, error) {
	key := "viewer-login"
	if val, ok := c.viewerLogin.Get(key); ok {
		return val, nil
	}
	result, err := c.inner.ListViewerLogin()
	if err != nil {
		return "", err
	}
	c.viewerLogin.Set(key, result)
	return result, nil
}

func (c *CachedClient) ListAllAccessibleProjects() ([]Project, error) {
	key := "all-projects"
	if val, ok := c.allProjects.Get(key); ok {
		return val, nil
	}
	result, err := c.inner.ListAllAccessibleProjects()
	if err != nil && !errors.Is(err, ErrMissingScopeReadOrg) {
		return nil, err
	}
	if result != nil {
		c.allProjects.Set(key, result)
	}
	return result, err
}

var _ GitHubClient = &CachedClient{}
