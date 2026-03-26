package github

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
)

var ErrMissingScopeReadOrg = errors.New("missing required scope: read:org")

type GraphQLClient struct {
	client *api.GraphQLClient
}

var _ GitHubClient = &GraphQLClient{}

func NewGraphQLClient() (*GraphQLClient, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, err
	}
	return &GraphQLClient{client: client}, nil
}

func (g *GraphQLClient) ListViewerOrganizations() ([]string, error) {
	organizations := make([]string, 0, 32)
	var after *graphql.String

	for {
		var query struct {
			Viewer struct {
				Organizations struct {
					Nodes []struct {
						Login string
					} `graphql:"nodes"`
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
				} `graphql:"organizations(first: 100, after: $after)"`
			}
		}

		vars := map[string]interface{}{"after": after}
		if err := g.client.Query("ListViewerOrganizations", &query, vars); err != nil {
			if strings.Contains(err.Error(), "insufficient scopes") || strings.Contains(err.Error(), "read:org") {
				return nil, ErrMissingScopeReadOrg
			}
			return nil, fmt.Errorf("query viewer organizations: %w", err)
		}

		for _, n := range query.Viewer.Organizations.Nodes {
			organizations = append(organizations, n.Login)
		}

		if !query.Viewer.Organizations.PageInfo.HasNextPage {
			break
		}

		end := graphql.String(query.Viewer.Organizations.PageInfo.EndCursor)
		after = &end
	}

	return organizations, nil
}

func (g *GraphQLClient) ListViewerLogin() (string, error) {
	var query struct {
		Viewer struct {
			Login string
		}
	}

	if err := g.client.Query("ListViewerLogin", &query, nil); err != nil {
		return "", fmt.Errorf("query viewer login: %w", err)
	}

	return query.Viewer.Login, nil
}

func (g *GraphQLClient) ListAllAccessibleProjects() ([]Project, error) {
	return listAllAccessibleProjects(g)
}

type accessibleProjectsLister interface {
	ListViewerLogin() (string, error)
	ListViewerOrganizations() ([]string, error)
	ListProjects(owner string) ([]Project, error)
}

func listAllAccessibleProjects(client accessibleProjectsLister) ([]Project, error) {
	viewerLogin, err := client.ListViewerLogin()
	if err != nil {
		return nil, err
	}

	var partialScopeErr error
	organizations, err := client.ListViewerOrganizations()
	if err != nil {
		if !errors.Is(err, ErrMissingScopeReadOrg) {
			return nil, err
		}
		organizations = nil
		partialScopeErr = ErrMissingScopeReadOrg
	}

	owners := make([]string, 0, 1+len(organizations))
	owners = append(owners, viewerLogin)
	owners = append(owners, organizations...)

	allProjects := make([]Project, 0, len(owners))
	errCh := make(chan error, len(owners))

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	for _, owner := range owners {
		wg.Add(1)
		go func(o string) {
			defer wg.Done()

			projects, listErr := client.ListProjects(o)
			if listErr != nil {
				errCh <- fmt.Errorf("list projects for owner %q: %w", o, listErr)
				return
			}

			mu.Lock()
			allProjects = append(allProjects, projects...)
			mu.Unlock()
		}(owner)
	}

	wg.Wait()
	close(errCh)

	for fetchErr := range errCh {
		if fetchErr != nil {
			return nil, fetchErr
		}
	}

	sort.Slice(allProjects, func(i, j int) bool {
		return allProjects[i].Owner < allProjects[j].Owner
	})

	return allProjects, partialScopeErr
}

func (g *GraphQLClient) ListProjects(owner string) ([]Project, error) {
	projects, err := g.listOwnerProjects(owner, true)
	if err == nil {
		return projects, nil
	}

	fallback, fallbackErr := g.listOwnerProjects(owner, false)
	if fallbackErr != nil {
		return nil, fmt.Errorf("list projects for owner %q (org then user): %w | fallback: %v", owner, err, fallbackErr)
	}

	return fallback, nil
}

func (g *GraphQLClient) GetProject(owner string, number int) (*Project, error) {
	project, err := g.getOwnerProject(owner, number, true)
	if err == nil {
		return project, nil
	}

	fallback, fallbackErr := g.getOwnerProject(owner, number, false)
	if fallbackErr != nil {
		return nil, fmt.Errorf("get project %d for owner %q (org then user): %w | fallback: %v", number, owner, err, fallbackErr)
	}

	return fallback, nil
}

func (g *GraphQLClient) GetProjectItems(projectID string) ([]ProjectItem, error) {
	items := make([]ProjectItem, 0, 128)
	var after *graphql.String

	for {
		var query struct {
			Node struct {
				ProjectV2 struct {
					Items struct {
						Nodes []struct {
							ID      string
							Content struct {
								Issue struct {
									ID     string
									Number int
									Title  string
									Body   string
									State  string
									Author struct {
										Login string
										User  struct {
											Name string
										} `graphql:"... on User"`
									}
									Assignees struct {
										Nodes []struct {
											Login string
											Name  string
										}
									} `graphql:"assignees(first: 10)"`
									Labels struct {
										Nodes []struct {
											ID    string
											Name  string
											Color string
										}
									} `graphql:"labels(first: 10)"`
									Comments struct {
										TotalCount int `graphql:"totalCount"`
									} `graphql:"comments"`
									CreatedAt  time.Time
									UpdatedAt  time.Time
									Repository struct {
										Owner struct {
											Login string
										}
										Name string
									}
									IssueType struct {
										Name string
									}
									ClosedByPullRequestsReferences struct {
										Nodes []struct {
											Number int
											State  string
										}
									} `graphql:"closedByPullRequestsReferences(first: 3)"`
								} `graphql:"... on Issue"`
								PullRequest struct {
									ID     string
									Number int
									Title  string
									State  string
									Author struct {
										Login string
										User  struct {
											Name string
										} `graphql:"... on User"`
									}
									Comments struct {
										TotalCount int `graphql:"totalCount"`
									} `graphql:"comments"`
									URL        string
									CreatedAt  time.Time
									Repository struct {
										Owner struct {
											Login string
										}
										Name string
									}
								} `graphql:"... on PullRequest"`
								DraftIssue struct {
									ID    string
									Title string
									Body  string
								} `graphql:"... on DraftIssue"`
							}
							FieldValues struct {
								Nodes []struct {
									SingleSelectValue struct {
										Name     string
										OptionID string
										Field    struct {
											SingleSelectField struct {
												Name string
											} `graphql:"... on ProjectV2SingleSelectField"`
										}
									} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
								} `graphql:"nodes"`
							} `graphql:"fieldValues(first: 20)"`
						} `graphql:"nodes"`
						PageInfo struct {
							HasNextPage bool
							EndCursor   string
						}
					} `graphql:"items(first: 100, after: $after)"`
				} `graphql:"... on ProjectV2"`
			} `graphql:"node(id: $projectID)"`
		}

		vars := map[string]interface{}{
			"projectID": graphql.ID(projectID),
			"after":     after,
		}

		if err := g.client.Query("GetProjectItems", &query, vars); err != nil {
			return nil, fmt.Errorf("query project items: %w", err)
		}

		page := query.Node.ProjectV2.Items
		for _, n := range page.Nodes {
			item := ProjectItem{ID: n.ID}

			for _, fv := range n.FieldValues.Nodes {
				if fv.SingleSelectValue.Field.SingleSelectField.Name == "Status" {
					item.Status = fv.SingleSelectValue.Name
					item.StatusID = fv.SingleSelectValue.OptionID
				}

				if fv.SingleSelectValue.Field.SingleSelectField.Name == "Type" {
					item.TypeValue = fv.SingleSelectValue.Name
					item.TypeID = fv.SingleSelectValue.OptionID
				}
			}

			if n.Content.Issue.ID != "" {
				assignees := make([]User, 0, len(n.Content.Issue.Assignees.Nodes))
				for _, a := range n.Content.Issue.Assignees.Nodes {
					assignees = append(assignees, User{Login: a.Login, Name: a.Name})
				}

				labels := make([]Label, 0, len(n.Content.Issue.Labels.Nodes))
				for _, l := range n.Content.Issue.Labels.Nodes {
					labels = append(labels, Label{ID: l.ID, Name: l.Name, Color: l.Color})
				}

				linkedPRs := make([]LinkedPullRequest, 0, len(n.Content.Issue.ClosedByPullRequestsReferences.Nodes))
				for _, pr := range n.Content.Issue.ClosedByPullRequestsReferences.Nodes {
					if pr.Number != 0 {
						linkedPRs = append(linkedPRs, LinkedPullRequest{
							Number: pr.Number,
							State:  pr.State,
						})
					}
				}

				issue := &Issue{
					ID:            n.Content.Issue.ID,
					Number:        n.Content.Issue.Number,
					Title:         n.Content.Issue.Title,
					Body:          n.Content.Issue.Body,
					State:         n.Content.Issue.State,
					IssueType:     n.Content.Issue.IssueType.Name,
					CommentsCount: n.Content.Issue.Comments.TotalCount,
					Author:        User{Login: n.Content.Issue.Author.Login, Name: n.Content.Issue.Author.User.Name},
					Assignees:     assignees,
					Labels:        labels,
					CreatedAt:     n.Content.Issue.CreatedAt,
					UpdatedAt:     n.Content.Issue.UpdatedAt,
					RepoOwner:     n.Content.Issue.Repository.Owner.Login,
					RepoName:      n.Content.Issue.Repository.Name,
				}
				issue.LinkedPRs = linkedPRs

				item.Title = issue.Title
				item.Type = "Issue"
				item.Content = issue
				if item.TypeValue == "" && n.Content.Issue.IssueType.Name != "" {
					item.TypeValue = n.Content.Issue.IssueType.Name
				}
				item.RepoOwner = issue.RepoOwner
				item.RepoName = issue.RepoName
				item.ContentNumber = issue.Number
			} else if n.Content.PullRequest.ID != "" {
				pr := &PullRequest{
					ID:            n.Content.PullRequest.ID,
					Number:        n.Content.PullRequest.Number,
					Title:         n.Content.PullRequest.Title,
					State:         n.Content.PullRequest.State,
					CommentsCount: n.Content.PullRequest.Comments.TotalCount,
					Author:        User{Login: n.Content.PullRequest.Author.Login, Name: n.Content.PullRequest.Author.User.Name},
					URL:           n.Content.PullRequest.URL,
					CreatedAt:     n.Content.PullRequest.CreatedAt,
					RepoOwner:     n.Content.PullRequest.Repository.Owner.Login,
					RepoName:      n.Content.PullRequest.Repository.Name,
				}

				item.Title = pr.Title
				item.Type = "PullRequest"
				item.Content = pr
				item.RepoOwner = pr.RepoOwner
				item.RepoName = pr.RepoName
				item.ContentNumber = pr.Number
			} else {
				item.Title = n.Content.DraftIssue.Title
				item.Type = "DraftIssue"
				item.Content = nil
			}

			items = append(items, item)
		}

		if !page.PageInfo.HasNextPage {
			break
		}

		end := graphql.String(page.PageInfo.EndCursor)
		after = &end
	}

	return items, nil
}

func (g *GraphQLClient) GetProjectFields(projectID string) ([]ProjectField, error) {
	var query struct {
		Node struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						CommonField struct {
							ID       string
							Name     string
							DataType string
						} `graphql:"... on ProjectV2FieldCommon"`
						SingleSelectField struct {
							ID       string
							Name     string
							DataType string
							Options  []struct {
								ID   string
								Name string
							} `graphql:"options"`
						} `graphql:"... on ProjectV2SingleSelectField"`
					}
				} `graphql:"fields(first: 100)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $projectID)"`
	}

	vars := map[string]interface{}{"projectID": graphql.ID(projectID)}
	if err := g.client.Query("GetProjectFields", &query, vars); err != nil {
		return nil, fmt.Errorf("query project fields: %w", err)
	}

	fields := make([]ProjectField, 0, len(query.Node.ProjectV2.Fields.Nodes))
	for _, n := range query.Node.ProjectV2.Fields.Nodes {
		id := n.CommonField.ID
		if n.SingleSelectField.ID != "" {
			id = n.SingleSelectField.ID
		}

		name := n.CommonField.Name
		if n.SingleSelectField.Name != "" {
			name = n.SingleSelectField.Name
		}

		dataType := n.CommonField.DataType
		if n.SingleSelectField.DataType != "" {
			dataType = n.SingleSelectField.DataType
		}

		field := ProjectField{ID: id, Name: name, DataType: dataType}
		if len(n.SingleSelectField.Options) > 0 {
			field.Options = make([]FieldOption, 0, len(n.SingleSelectField.Options))
			for _, o := range n.SingleSelectField.Options {
				field.Options = append(field.Options, FieldOption{ID: o.ID, Name: o.Name})
			}
		}

		fields = append(fields, field)
	}

	return fields, nil
}

func (g *GraphQLClient) GetProjectViews(projectID string) ([]ProjectView, error) {
	views := make([]ProjectView, 0, 16)
	var after *graphql.String

	for {
		var query struct {
			Node struct {
				ProjectV2 struct {
					Views struct {
						Nodes []struct {
							ID     string
							Name   string
							Number int
							Layout string
							Filter string
						} `graphql:"nodes"`
						PageInfo struct {
							HasNextPage bool
							EndCursor   string
						}
					} `graphql:"views(first: 50, after: $after)"`
				} `graphql:"... on ProjectV2"`
			} `graphql:"node(id: $projectID)"`
		}

		vars := map[string]interface{}{
			"projectID": graphql.ID(projectID),
			"after":     after,
		}

		if err := g.client.Query("GetProjectViews", &query, vars); err != nil {
			return nil, fmt.Errorf("query project views: %w", err)
		}

		page := query.Node.ProjectV2.Views
		for _, n := range page.Nodes {
			views = append(views, ProjectView{
				ID:     n.ID,
				Name:   n.Name,
				Number: n.Number,
				Layout: n.Layout,
				Filter: n.Filter,
			})
		}

		if !page.PageInfo.HasNextPage {
			break
		}

		end := graphql.String(page.PageInfo.EndCursor)
		after = &end
	}

	return views, nil
}

func (g *GraphQLClient) GetIssue(owner, repo string, number int) (*Issue, error) {
	var query struct {
		Repository struct {
			Issue struct {
				ID     string
				Number int
				Title  string
				Body   string
				State  string
				Author struct {
					Login string
					User  struct {
						Name string
					} `graphql:"... on User"`
				}
				Assignees struct {
					Nodes []struct{ Login, Name string }
				} `graphql:"assignees(first: 10)"`
				Labels struct {
					Nodes []struct {
						ID    string
						Name  string
						Color string
					}
				} `graphql:"labels(first: 20)"`
				CreatedAt  time.Time
				UpdatedAt  time.Time
				Repository struct {
					Owner struct{ Login string }
					Name  string
				}
				IssueType struct {
					Name string
				}
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	vars := map[string]interface{}{"owner": graphql.String(owner), "repo": graphql.String(repo), "number": graphql.Int(number)}
	if err := g.client.Query("GetIssue", &query, vars); err != nil {
		return nil, fmt.Errorf("query issue %s/%s#%d: %w", owner, repo, number, err)
	}

	qIssue := query.Repository.Issue
	if qIssue.ID == "" {
		return nil, fmt.Errorf("issue %s/%s#%d not found", owner, repo, number)
	}

	assignees := make([]User, 0, len(qIssue.Assignees.Nodes))
	for _, a := range qIssue.Assignees.Nodes {
		assignees = append(assignees, User{Login: a.Login, Name: a.Name})
	}

	labels := make([]Label, 0, len(qIssue.Labels.Nodes))
	for _, l := range qIssue.Labels.Nodes {
		labels = append(labels, Label{ID: l.ID, Name: l.Name, Color: l.Color})
	}

	return &Issue{
		ID:        qIssue.ID,
		Number:    qIssue.Number,
		Title:     qIssue.Title,
		Body:      qIssue.Body,
		State:     qIssue.State,
		IssueType: qIssue.IssueType.Name,
		Author:    User{Login: qIssue.Author.Login, Name: qIssue.Author.User.Name},
		Assignees: assignees,
		Labels:    labels,
		CreatedAt: qIssue.CreatedAt,
		UpdatedAt: qIssue.UpdatedAt,
		RepoOwner: qIssue.Repository.Owner.Login,
		RepoName:  qIssue.Repository.Name,
	}, nil
}
func (g *GraphQLClient) GetIssueComments(owner, repo string, number int) ([]Comment, error) {
	comments := make([]Comment, 0, 64)
	var after *graphql.String

	for {
		var query struct {
			Repository struct {
				Issue struct {
					Comments struct {
						Nodes []struct {
							ID     string
							Body   string
							Author struct {
								Login string
								User  struct {
									Name string
								} `graphql:"... on User"`
							}
							CreatedAt time.Time
						}
						PageInfo struct {
							HasNextPage bool
							EndCursor   string
						}
					} `graphql:"comments(first: 100, after: $after)"`
				} `graphql:"issue(number: $number)"`
			} `graphql:"repository(owner: $owner, name: $repo)"`
		}

		vars := map[string]interface{}{
			"owner":  graphql.String(owner),
			"repo":   graphql.String(repo),
			"number": graphql.Int(number),
			"after":  after,
		}

		if err := g.client.Query("GetIssueComments", &query, vars); err != nil {
			return nil, fmt.Errorf("query issue comments %s/%s#%d: %w", owner, repo, number, err)
		}

		for _, n := range query.Repository.Issue.Comments.Nodes {
			comments = append(comments, Comment{
				ID:        n.ID,
				Body:      n.Body,
				Author:    User{Login: n.Author.Login, Name: n.Author.User.Name},
				CreatedAt: n.CreatedAt,
			})
		}

		if !query.Repository.Issue.Comments.PageInfo.HasNextPage {
			break
		}

		end := graphql.String(query.Repository.Issue.Comments.PageInfo.EndCursor)
		after = &end
	}

	return comments, nil
}

func (g *GraphQLClient) ListRepositoryLabels(owner, repo string) ([]Label, error) {
	labels := make([]Label, 0, 128)
	var after *graphql.String

	for {
		var query struct {
			Repository struct {
				Labels struct {
					Nodes []struct {
						ID    string
						Name  string
						Color string
					}
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
				} `graphql:"labels(first: 100, after: $after)"`
			} `graphql:"repository(owner: $owner, name: $repo)"`
		}

		vars := map[string]interface{}{"owner": graphql.String(owner), "repo": graphql.String(repo), "after": after}
		if err := g.client.Query("ListRepositoryLabels", &query, vars); err != nil {
			return nil, fmt.Errorf("query repository labels %s/%s: %w", owner, repo, err)
		}

		for _, n := range query.Repository.Labels.Nodes {
			labels = append(labels, Label{ID: n.ID, Name: n.Name, Color: n.Color})
		}

		if !query.Repository.Labels.PageInfo.HasNextPage {
			break
		}

		end := graphql.String(query.Repository.Labels.PageInfo.EndCursor)
		after = &end
	}

	return labels, nil
}

func (g *GraphQLClient) ListAssignableUsers(owner, repo string) ([]User, error) {
	users := make([]User, 0, 128)
	var after *graphql.String

	for {
		var query struct {
			Repository struct {
				AssignableUsers struct {
					Nodes []struct {
						Login string
						Name  string
					}
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
				} `graphql:"assignableUsers(first: 100, after: $after)"`
			} `graphql:"repository(owner: $owner, name: $repo)"`
		}

		vars := map[string]interface{}{"owner": graphql.String(owner), "repo": graphql.String(repo), "after": after}
		if err := g.client.Query("ListAssignableUsers", &query, vars); err != nil {
			return nil, fmt.Errorf("query assignable users %s/%s: %w", owner, repo, err)
		}

		for _, n := range query.Repository.AssignableUsers.Nodes {
			users = append(users, User{Login: n.Login, Name: n.Name})
		}

		if !query.Repository.AssignableUsers.PageInfo.HasNextPage {
			break
		}

		end := graphql.String(query.Repository.AssignableUsers.PageInfo.EndCursor)
		after = &end
	}

	return users, nil
}

func (g *GraphQLClient) ListIssueTypes(owner, repo string) ([]IssueType, error) {
	var query struct {
		Repository struct {
			IssueTypes struct {
				Nodes []struct {
					ID   string
					Name string
				}
			} `graphql:"issueTypes(first: 25)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	vars := map[string]interface{}{"owner": graphql.String(owner), "repo": graphql.String(repo)}
	if err := g.client.Query("ListIssueTypes", &query, vars); err != nil {
		return nil, fmt.Errorf("query issue types %s/%s: %w", owner, repo, err)
	}

	types := make([]IssueType, 0, len(query.Repository.IssueTypes.Nodes))
	for _, n := range query.Repository.IssueTypes.Nodes {
		types = append(types, IssueType{ID: n.ID, Name: n.Name})
	}

	return types, nil
}

func (g *GraphQLClient) ListRepositoryPullRequests(owner, repo string, limit int) ([]PullRequest, error) {
	prs := make([]PullRequest, 0, 128)
	var after *graphql.String

	for {
		var query struct {
			Repository struct {
				Owner struct {
					Login string
				}
				Name         string
				PullRequests struct {
					Nodes []struct {
						ID        string
						Number    int
						Title     string
						State     string
						CreatedAt time.Time
						MergedAt  time.Time
						Author    struct {
							Login string
							User  struct {
								Name string
							} `graphql:"... on User"`
						}
						URL string
					}
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
				} `graphql:"pullRequests(first: 100, after: $after, states: [OPEN, MERGED], orderBy: {field: CREATED_AT, direction: DESC})"`
			} `graphql:"repository(owner: $owner, name: $repo)"`
		}

		vars := map[string]interface{}{"owner": graphql.String(owner), "repo": graphql.String(repo), "after": after}
		if err := g.client.Query("ListRepositoryPullRequests", &query, vars); err != nil {
			return nil, fmt.Errorf("query repository pull requests %s/%s: %w", owner, repo, err)
		}

		for _, n := range query.Repository.PullRequests.Nodes {
			prs = append(prs, PullRequest{
				ID:        n.ID,
				Number:    n.Number,
				Title:     n.Title,
				State:     n.State,
				Author:    User{Login: n.Author.Login, Name: n.Author.User.Name},
				URL:       n.URL,
				CreatedAt: n.CreatedAt,
				MergedAt:  n.MergedAt,
				RepoOwner: query.Repository.Owner.Login,
				RepoName:  query.Repository.Name,
			})
		}

		if limit > 0 && len(prs) >= limit {
			break
		}

		if !query.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}

		end := graphql.String(query.Repository.PullRequests.PageInfo.EndCursor)
		after = &end
	}

	if limit > 0 && len(prs) > limit {
		prs = prs[:limit]
	}

	return prs, nil
}

func (g *GraphQLClient) GetLinkedPullRequests(owner, repo string, issueNumber int) ([]PullRequest, error) {
	prs := make([]PullRequest, 0, 16)
	var after *graphql.String

	for {
		var query struct {
			Repository struct {
				Issue struct {
					TimelineItems struct {
						Nodes []struct {
							ConnectedEvent struct {
								Subject struct {
									PullRequest struct {
										ID        string
										Number    int
										Title     string
										State     string
										URL       string
										CreatedAt time.Time
										Author    struct {
											Login string
											User  struct {
												Name string
											} `graphql:"... on User"`
										}
										Repository struct {
											Owner struct{ Login string }
											Name  string
										}
									} `graphql:"... on PullRequest"`
								}
							} `graphql:"... on ConnectedEvent"`
						}
						PageInfo struct {
							HasNextPage bool
							EndCursor   string
						}
					} `graphql:"timelineItems(first: 100, after: $after, itemTypes: [CONNECTED_EVENT])"`
				} `graphql:"issue(number: $number)"`
			} `graphql:"repository(owner: $owner, name: $repo)"`
		}

		vars := map[string]interface{}{
			"owner":  graphql.String(owner),
			"repo":   graphql.String(repo),
			"number": graphql.Int(issueNumber),
			"after":  after,
		}

		if err := g.client.Query("GetLinkedPullRequests", &query, vars); err != nil {
			return nil, fmt.Errorf("query linked pull requests for %s/%s#%d: %w", owner, repo, issueNumber, err)
		}

		for _, n := range query.Repository.Issue.TimelineItems.Nodes {
			prNode := n.ConnectedEvent.Subject.PullRequest
			if prNode.ID == "" {
				continue
			}

			prs = append(prs, PullRequest{
				ID:        prNode.ID,
				Number:    prNode.Number,
				Title:     prNode.Title,
				State:     prNode.State,
				Author:    User{Login: prNode.Author.Login, Name: prNode.Author.User.Name},
				URL:       prNode.URL,
				CreatedAt: prNode.CreatedAt,
				RepoOwner: prNode.Repository.Owner.Login,
				RepoName:  prNode.Repository.Name,
			})
		}

		if !query.Repository.Issue.TimelineItems.PageInfo.HasNextPage {
			break
		}

		end := graphql.String(query.Repository.Issue.TimelineItems.PageInfo.EndCursor)
		after = &end
	}

	return prs, nil
}

func (g *GraphQLClient) GetPullRequestNodeID(owner, repo string, number int) (string, error) {
	var query struct {
		Repository struct {
			PullRequest struct {
				ID string
			} `graphql:"pullRequest(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	vars := map[string]interface{}{"owner": graphql.String(owner), "repo": graphql.String(repo), "number": graphql.Int(number)}
	if err := g.client.Query("GetPullRequestNodeID", &query, vars); err != nil {
		return "", fmt.Errorf("query pull request node id for %s/%s#%d: %w", owner, repo, number, err)
	}

	if query.Repository.PullRequest.ID == "" {
		return "", fmt.Errorf("pull request %s/%s#%d not found", owner, repo, number)
	}

	return query.Repository.PullRequest.ID, nil
}

func (g *GraphQLClient) GetPullRequest(owner, repo string, number int) (*PullRequest, error) {
	var query struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name        string
			PullRequest struct {
				ID     string
				Number int
				Title  string
				URL    string
				Body   string
				State  string
			} `graphql:"pullRequest(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	vars := map[string]interface{}{"owner": graphql.String(owner), "repo": graphql.String(repo), "number": graphql.Int(number)}
	if err := g.client.Query("GetPullRequest", &query, vars); err != nil {
		return nil, fmt.Errorf("query pull request %s/%s#%d: %w", owner, repo, number, err)
	}

	if query.Repository.PullRequest.ID == "" {
		return nil, fmt.Errorf("pull request %s/%s#%d not found", owner, repo, number)
	}

	return &PullRequest{
		ID:        query.Repository.PullRequest.ID,
		Number:    query.Repository.PullRequest.Number,
		Title:     query.Repository.PullRequest.Title,
		Body:      query.Repository.PullRequest.Body,
		State:     query.Repository.PullRequest.State,
		URL:       query.Repository.PullRequest.URL,
		RepoOwner: query.Repository.Owner.Login,
		RepoName:  query.Repository.Name,
	}, nil
}

func (g *GraphQLClient) LinkPRToIssue(owner, repo string, prNumber, issueNumber int) error {
	pr, err := g.GetPullRequest(owner, repo, prNumber)
	if err != nil {
		return err
	}

	newBody := pr.Body + fmt.Sprintf("\n\nCloses %s/%s#%d", owner, repo, issueNumber)

	var mutation struct {
		UpdatePullRequest struct {
			PullRequest struct{ ID string }
		} `graphql:"updatePullRequest(input: $input)"`
	}

	type UpdatePullRequestInput struct {
		PullRequestID graphql.ID     `json:"pullRequestId"`
		Body          graphql.String `json:"body"`
	}

	input := UpdatePullRequestInput{
		PullRequestID: graphql.ID(pr.ID),
		Body:          graphql.String(newBody),
	}

	if err := g.client.Mutate("LinkPRToIssue", &mutation, map[string]interface{}{"input": input}); err != nil {
		return fmt.Errorf("link pull request %s/%s#%d to issue #%d: %w", owner, repo, prNumber, issueNumber, err)
	}

	return nil
}

func (g *GraphQLClient) MoveItem(projectID, itemID, fieldID, optionID string) error {
	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ClientMutationId string
		} `graphql:"updateProjectV2ItemFieldValue(input: {projectId: $projectID, itemId: $itemID, fieldId: $fieldID, value: {singleSelectOptionId: $optionID}})"`
	}

	vars := map[string]interface{}{
		"projectID": graphql.ID(projectID),
		"itemID":    graphql.ID(itemID),
		"fieldID":   graphql.ID(fieldID),
		"optionID":  graphql.String(optionID),
	}

	if err := g.client.Mutate("MoveItem", &mutation, vars); err != nil {
		return fmt.Errorf("move project item: %w", err)
	}

	return nil
}

func (g *GraphQLClient) AddComment(owner, repo string, number int, body string) error {
	issueID, err := g.getIssueNodeID(owner, repo, number)
	if err != nil {
		return err
	}

	var mutation struct {
		AddComment struct {
			CommentEdge struct {
				Node struct{ ID string }
			}
		} `graphql:"addComment(input: {subjectId: $subjectID, body: $body})"`
	}

	vars := map[string]interface{}{"subjectID": graphql.ID(issueID), "body": graphql.String(body)}
	if err := g.client.Mutate("AddComment", &mutation, vars); err != nil {
		return fmt.Errorf("add comment to %s/%s#%d: %w", owner, repo, number, err)
	}

	return nil
}

func (g *GraphQLClient) AssignUser(owner, repo string, number int, login string) error {
	issueID, err := g.getIssueNodeID(owner, repo, number)
	if err != nil {
		return err
	}

	userID, err := g.getUserNodeID(login)
	if err != nil {
		return err
	}

	var mutation struct {
		AddAssigneesToAssignable struct {
			ClientMutationID string
		} `graphql:"addAssigneesToAssignable(input: {assignableId: $assignableID, assigneeIds: [$assigneeID]})"`
	}

	vars := map[string]interface{}{
		"assignableID": graphql.ID(issueID),
		"assigneeID":   graphql.ID(userID),
	}

	if err := g.client.Mutate("AssignUser", &mutation, vars); err != nil {
		return fmt.Errorf("assign user %q to %s/%s#%d: %w", login, owner, repo, number, err)
	}

	return nil
}

func (g *GraphQLClient) UnassignUser(owner, repo string, number int, login string) error {
	issueID, err := g.getIssueNodeID(owner, repo, number)
	if err != nil {
		return err
	}

	userID, err := g.getUserNodeID(login)
	if err != nil {
		return err
	}

	var mutation struct {
		RemoveAssigneesFromAssignable struct {
			ClientMutationID string
		} `graphql:"removeAssigneesFromAssignable(input: {assignableId: $assignableID, assigneeIds: [$assigneeID]})"`
	}

	vars := map[string]interface{}{
		"assignableID": graphql.ID(issueID),
		"assigneeID":   graphql.ID(userID),
	}

	if err := g.client.Mutate("UnassignUser", &mutation, vars); err != nil {
		return fmt.Errorf("unassign user %q from %s/%s#%d: %w", login, owner, repo, number, err)
	}

	return nil
}

func (g *GraphQLClient) AddLabel(owner, repo string, number int, labelName string) error {
	issueID, err := g.getIssueNodeID(owner, repo, number)
	if err != nil {
		return err
	}

	labelID, err := g.getRepositoryLabelNodeID(owner, repo, labelName)
	if err != nil {
		return err
	}

	var mutation struct {
		AddLabelsToLabelable struct {
			Labelable struct{ ID string }
		} `graphql:"addLabelsToLabelable(input: {labelableId: $labelableID, labelIds: [$labelID]})"`
	}

	vars := map[string]interface{}{"labelableID": graphql.ID(issueID), "labelID": graphql.ID(labelID)}
	if err := g.client.Mutate("AddLabel", &mutation, vars); err != nil {
		return fmt.Errorf("add label %q to %s/%s#%d: %w", labelName, owner, repo, number, err)
	}

	return nil
}

func (g *GraphQLClient) RemoveLabel(owner, repo string, number int, labelID string) error {
	issueID, err := g.getIssueNodeID(owner, repo, number)
	if err != nil {
		return err
	}

	var mutation struct {
		RemoveLabelsFromLabelable struct {
			Labelable struct{ ID string }
		} `graphql:"removeLabelsFromLabelable(input: {labelableId: $labelableID, labelIds: [$labelID]})"`
	}

	vars := map[string]interface{}{"labelableID": graphql.ID(issueID), "labelID": graphql.ID(labelID)}
	if err := g.client.Mutate("RemoveLabel", &mutation, vars); err != nil {
		return fmt.Errorf("remove label %q from %s/%s#%d: %w", labelID, owner, repo, number, err)
	}

	return nil
}

func (g *GraphQLClient) UpdateIssueType(issueID string, typeID *string) error {
	var mutation struct {
		UpdateIssue struct {
			Issue struct{ ID string }
		} `graphql:"updateIssue(input: $input)"`
	}

	type UpdateIssueInput struct {
		IssueID     graphql.ID  `json:"id"`
		IssueTypeID *graphql.ID `json:"issueTypeId"`
	}

	var gqlTypeID *graphql.ID
	if typeID != nil {
		id := graphql.ID(*typeID)
		gqlTypeID = &id
	}

	input := UpdateIssueInput{
		IssueID:     graphql.ID(issueID),
		IssueTypeID: gqlTypeID,
	}

	if err := g.client.Mutate("UpdateIssueType", &mutation, map[string]interface{}{"input": input}); err != nil {
		return fmt.Errorf("update issue type %s: %w", issueID, err)
	}

	return nil
}

func (g *GraphQLClient) UpdateIssueBody(issueID string, body string) error {
	var mutation struct {
		UpdateIssue struct {
			Issue struct{ ID string }
		} `graphql:"updateIssue(input: $input)"`
	}

	type UpdateIssueInput struct {
		IssueID graphql.ID     `json:"id"`
		Body    graphql.String `json:"body"`
	}

	input := UpdateIssueInput{
		IssueID: graphql.ID(issueID),
		Body:    graphql.String(body),
	}

	if err := g.client.Mutate("UpdateIssueBody", &mutation, map[string]interface{}{"input": input}); err != nil {
		return fmt.Errorf("update issue body %s: %w", issueID, err)
	}

	return nil
}

func (g *GraphQLClient) CloseIssue(owner, repo string, number int) error {
	issueID, err := g.getIssueNodeID(owner, repo, number)
	if err != nil {
		return err
	}

	var mutation struct {
		CloseIssue struct {
			Issue struct{ ID string }
		} `graphql:"closeIssue(input: {issueId: $issueID})"`
	}

	if err := g.client.Mutate("CloseIssue", &mutation, map[string]interface{}{"issueID": graphql.ID(issueID)}); err != nil {
		return fmt.Errorf("close issue %s/%s#%d: %w", owner, repo, number, err)
	}

	return nil
}

func (g *GraphQLClient) ReopenIssue(owner, repo string, number int) error {
	issueID, err := g.getIssueNodeID(owner, repo, number)
	if err != nil {
		return err
	}

	var mutation struct {
		ReopenIssue struct {
			Issue struct{ ID string }
		} `graphql:"reopenIssue(input: {issueId: $issueID})"`
	}

	if err := g.client.Mutate("ReopenIssue", &mutation, map[string]interface{}{"issueID": graphql.ID(issueID)}); err != nil {
		return fmt.Errorf("reopen issue %s/%s#%d: %w", owner, repo, number, err)
	}

	return nil
}

func (g *GraphQLClient) AddItemToProject(projectID string, contentID string) error {
	var mutation struct {
		AddProjectV2ItemById struct {
			Item struct{ ID string }
		} `graphql:"addProjectV2ItemById(input: {projectId: $projectID, contentId: $contentID})"`
	}

	vars := map[string]interface{}{"projectID": graphql.ID(projectID), "contentID": graphql.ID(contentID)}
	if err := g.client.Mutate("AddItemToProject", &mutation, vars); err != nil {
		return fmt.Errorf("add item %q to project %q: %w", contentID, projectID, err)
	}

	return nil
}

func (g *GraphQLClient) listOwnerProjects(owner string, asOrg bool) ([]Project, error) {
	projects := make([]Project, 0, 64)
	var after *graphql.String

	for {
		if asOrg {
			var query struct {
				Organization struct {
					ProjectsV2 struct {
						Nodes []struct {
							ID     string
							Title  string
							Number int
							Items  struct {
								TotalCount int
							}
						}
						PageInfo struct {
							HasNextPage bool
							EndCursor   string
						}
					} `graphql:"projectsV2(first: 100, after: $after)"`
				} `graphql:"organization(login: $owner)"`
			}

			vars := map[string]interface{}{"owner": graphql.String(owner), "after": after}
			if err := g.client.Query("ListOrgProjects", &query, vars); err != nil {
				return nil, err
			}
			if query.Organization.ProjectsV2.Nodes == nil {
				return nil, fmt.Errorf("organization %q not found", owner)
			}

			for _, n := range query.Organization.ProjectsV2.Nodes {
				projects = append(projects, Project{ID: n.ID, Title: n.Title, Number: n.Number, Owner: owner, ItemCount: n.Items.TotalCount})
			}

			if !query.Organization.ProjectsV2.PageInfo.HasNextPage {
				break
			}

			end := graphql.String(query.Organization.ProjectsV2.PageInfo.EndCursor)
			after = &end
			continue
		}

		var query struct {
			User struct {
				ProjectsV2 struct {
					Nodes []struct {
						ID     string
						Title  string
						Number int
						Items  struct {
							TotalCount int
						}
					}
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
				} `graphql:"projectsV2(first: 100, after: $after)"`
			} `graphql:"user(login: $owner)"`
		}

		vars := map[string]interface{}{"owner": graphql.String(owner), "after": after}
		if err := g.client.Query("ListUserProjects", &query, vars); err != nil {
			return nil, err
		}
		if query.User.ProjectsV2.Nodes == nil {
			return nil, fmt.Errorf("user %q not found", owner)
		}

		for _, n := range query.User.ProjectsV2.Nodes {
			projects = append(projects, Project{ID: n.ID, Title: n.Title, Number: n.Number, Owner: owner, ItemCount: n.Items.TotalCount})
		}

		if !query.User.ProjectsV2.PageInfo.HasNextPage {
			break
		}

		end := graphql.String(query.User.ProjectsV2.PageInfo.EndCursor)
		after = &end
	}

	return projects, nil
}

func (g *GraphQLClient) getOwnerProject(owner string, number int, asOrg bool) (*Project, error) {
	if asOrg {
		var query struct {
			Organization struct {
				ProjectV2 struct {
					ID     string
					Title  string
					Number int
					Items  struct {
						TotalCount int
					}
				} `graphql:"projectV2(number: $number)"`
			} `graphql:"organization(login: $owner)"`
		}

		vars := map[string]interface{}{"owner": graphql.String(owner), "number": graphql.Int(number)}
		if err := g.client.Query("GetOrgProject", &query, vars); err != nil {
			return nil, err
		}
		if query.Organization.ProjectV2.ID == "" {
			return nil, fmt.Errorf("organization project %q/%d not found", owner, number)
		}

		p := query.Organization.ProjectV2
		return &Project{ID: p.ID, Title: p.Title, Number: p.Number, Owner: owner, ItemCount: p.Items.TotalCount}, nil
	}

	var query struct {
		User struct {
			ProjectV2 struct {
				ID     string
				Title  string
				Number int
				Items  struct {
					TotalCount int
				}
			} `graphql:"projectV2(number: $number)"`
		} `graphql:"user(login: $owner)"`
	}

	vars := map[string]interface{}{"owner": graphql.String(owner), "number": graphql.Int(number)}
	if err := g.client.Query("GetUserProject", &query, vars); err != nil {
		return nil, err
	}
	if query.User.ProjectV2.ID == "" {
		return nil, fmt.Errorf("user project %q/%d not found", owner, number)
	}

	p := query.User.ProjectV2
	return &Project{ID: p.ID, Title: p.Title, Number: p.Number, Owner: owner, ItemCount: p.Items.TotalCount}, nil
}

func (g *GraphQLClient) getIssueNodeID(owner, repo string, number int) (string, error) {
	var query struct {
		Repository struct {
			Issue struct {
				ID string
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	vars := map[string]interface{}{"owner": graphql.String(owner), "repo": graphql.String(repo), "number": graphql.Int(number)}
	if err := g.client.Query("GetIssueNodeID", &query, vars); err != nil {
		return "", fmt.Errorf("query issue node id for %s/%s#%d: %w", owner, repo, number, err)
	}

	if query.Repository.Issue.ID == "" {
		return "", fmt.Errorf("issue %s/%s#%d not found", owner, repo, number)
	}

	return query.Repository.Issue.ID, nil
}

func (g *GraphQLClient) getUserNodeID(login string) (string, error) {
	var query struct {
		User struct {
			ID string
		} `graphql:"user(login: $login)"`
	}

	if err := g.client.Query("GetUserNodeID", &query, map[string]interface{}{"login": graphql.String(login)}); err != nil {
		return "", fmt.Errorf("query user node id for %q: %w", login, err)
	}
	if query.User.ID == "" {
		return "", fmt.Errorf("user %q not found", login)
	}

	return query.User.ID, nil
}

func (g *GraphQLClient) getRepositoryLabelNodeID(owner, repo, labelName string) (string, error) {
	var after *graphql.String

	for {
		var query struct {
			Repository struct {
				Labels struct {
					Nodes []struct {
						ID   string
						Name string
					}
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
				} `graphql:"labels(first: 100, after: $after, query: $labelName)"`
			} `graphql:"repository(owner: $owner, name: $repo)"`
		}

		vars := map[string]interface{}{
			"owner":     graphql.String(owner),
			"repo":      graphql.String(repo),
			"labelName": graphql.String(labelName),
			"after":     after,
		}
		if err := g.client.Query("GetRepositoryLabelNodeID", &query, vars); err != nil {
			return "", fmt.Errorf("query label id for %s/%s label %q: %w", owner, repo, labelName, err)
		}

		for _, n := range query.Repository.Labels.Nodes {
			if strings.EqualFold(n.Name, labelName) {
				return n.ID, nil
			}
		}

		if !query.Repository.Labels.PageInfo.HasNextPage {
			break
		}

		end := graphql.String(query.Repository.Labels.PageInfo.EndCursor)
		after = &end
	}

	return "", fmt.Errorf("label %q not found in repository %s/%s", labelName, owner, repo)
}
