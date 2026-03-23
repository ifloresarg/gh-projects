package github

import "time"

type Project struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Number    int    `json:"number"`
	Owner     string `json:"owner"`
	ItemCount int    `json:"itemCount"`
}

type ProjectItem struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	StatusID      string `json:"statusId"`
	TypeValue     string `json:"typeValue"`
	TypeID        string `json:"typeId"`
	Content       any    `json:"content"`
	RepoOwner     string `json:"repoOwner"`
	RepoName      string `json:"repoName"`
	ContentNumber int    `json:"contentNumber"`
}

type ProjectView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Number int    `json:"number"`
	Layout string `json:"layout"`
	Filter string `json:"filter"`
}

type ProjectField struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	DataType string        `json:"dataType"`
	Options  []FieldOption `json:"options"`
}

type FieldOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IssueType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Issue struct {
	ID        string              `json:"id"`
	Number    int                 `json:"number"`
	Title     string              `json:"title"`
	Body      string              `json:"body"`
	State     string              `json:"state"`
	IssueType string              `json:"issueType"`
	Author    User                `json:"author"`
	Assignees []User              `json:"assignees"`
	Labels    []Label             `json:"labels"`
	LinkedPRs []LinkedPullRequest `json:"linkedPRs"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
	RepoOwner string              `json:"repoOwner"`
	RepoName  string              `json:"repoName"`
}

// LinkedPullRequest is a minimal representation of a PR linked to an issue,
// used for badge display on kanban cards.
type LinkedPullRequest struct {
	Number int    `json:"number"`
	State  string `json:"state"`
}

type PullRequest struct {
	ID        string    `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	Author    User      `json:"author"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	MergedAt  time.Time `json:"mergedAt"`
	RepoOwner string    `json:"repoOwner"`
	RepoName  string    `json:"repoName"`
}

type Comment struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	Author    User      `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
}

type User struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

type Label struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}
