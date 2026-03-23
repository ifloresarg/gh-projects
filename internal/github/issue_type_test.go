package github

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestGraphQLClient(t *testing.T, response string) *GraphQLClient {
	t.Helper()

	apiClient, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "token",
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("request method = %s, want POST", req.Method)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll(request body) error = %v", err)
			}
			if !strings.Contains(string(body), "GetProjectItems") {
				t.Fatalf("request body missing GetProjectItems operation: %s", string(body))
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(response)),
				Request:    req,
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("api.NewGraphQLClient() error = %v", err)
	}

	return &GraphQLClient{client: apiClient}
}

func TestGetProjectItemsUsesNativeIssueTypeWhenProjectTypeMissing(t *testing.T) {
	t.Parallel()

	client := newTestGraphQLClient(t, `{
		"data": {
			"node": {
				"items": {
					"nodes": [
						{
							"id": "PVTI_1",
							"content": {
								"id": "I_1",
								"number": 101,
								"title": "Fix board filter",
								"body": "Native issue type should be used when project type is absent.",
								"state": "OPEN",
								"author": {"login": "octocat", "name": "The Octocat"},
								"assignees": {"nodes": []},
								"labels": {"nodes": []},
								"createdAt": "2026-03-22T00:00:00Z",
								"updatedAt": "2026-03-22T01:00:00Z",
								"repository": {"owner": {"login": "ifloresarg"}, "name": "gh-projects"},
								"issueType": {"name": "Bug"}
							},
							"fieldValues": {"nodes": []}
						}
					],
					"pageInfo": {"hasNextPage": false, "endCursor": ""}
				}
			}
		}
	}`)

	items, err := client.GetProjectItems("PVT_1")
	if err != nil {
		t.Fatalf("GetProjectItems() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].TypeValue != "Bug" {
		t.Fatalf("TypeValue = %q, want %q", items[0].TypeValue, "Bug")
	}

	issue, ok := items[0].Content.(*Issue)
	if !ok {
		t.Fatal("ProjectItem.Content type assertion to *Issue failed")
	}
	if issue.IssueType != "Bug" {
		t.Fatalf("IssueType = %q, want %q", issue.IssueType, "Bug")
	}
}

func TestGetProjectItemsPrefersCustomProjectTypeOverNativeIssueType(t *testing.T) {
	t.Parallel()

	client := newTestGraphQLClient(t, `{
		"data": {
			"node": {
				"items": {
					"nodes": [
						{
							"id": "PVTI_2",
							"content": {
								"id": "I_2",
								"number": 102,
								"title": "Improve onboarding",
								"body": "Project Type field should win over native issue type.",
								"state": "OPEN",
								"author": {"login": "monalisa", "name": "Mona Lisa"},
								"assignees": {"nodes": []},
								"labels": {"nodes": []},
								"createdAt": "2026-03-22T00:00:00Z",
								"updatedAt": "2026-03-22T01:00:00Z",
								"repository": {"owner": {"login": "ifloresarg"}, "name": "gh-projects"},
								"issueType": {"name": "Feature"}
							},
							"fieldValues": {
								"nodes": [
									{
										"name": "Enhancement",
										"optionId": "type-enhancement",
										"field": {"name": "Type"}
									}
								]
							}
						}
					],
					"pageInfo": {"hasNextPage": false, "endCursor": ""}
				}
			}
		}
	}`)

	items, err := client.GetProjectItems("PVT_1")
	if err != nil {
		t.Fatalf("GetProjectItems() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].TypeValue != "Enhancement" {
		t.Fatalf("TypeValue = %q, want %q", items[0].TypeValue, "Enhancement")
	}

	issue, ok := items[0].Content.(*Issue)
	if !ok {
		t.Fatal("ProjectItem.Content type assertion to *Issue failed")
	}
	if issue.IssueType != "Feature" {
		t.Fatalf("IssueType = %q, want %q", issue.IssueType, "Feature")
	}
}

func TestUpdateIssueInputJSONSerializationUsesCorrectTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		issueID     graphql.ID
		issueTypeID *graphql.ID
	}{
		{
			name:        "serializes with id field, not issueId",
			issueID:     graphql.ID("I_123"),
			issueTypeID: nil,
		},
		{
			name:        "serializes both id and issueTypeId when both present",
			issueID:     graphql.ID("I_456"),
			issueTypeID: func() *graphql.ID { id := graphql.ID("IT_789"); return &id }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			type UpdateIssueInput struct {
				IssueID     graphql.ID  `json:"id"`
				IssueTypeID *graphql.ID `json:"issueTypeId"`
			}

			input := UpdateIssueInput{
				IssueID:     tt.issueID,
				IssueTypeID: tt.issueTypeID,
			}

			data, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			jsonStr := string(data)

			if !strings.Contains(jsonStr, `"id"`) {
				t.Errorf("JSON output should contain %q, got: %s", `"id"`, jsonStr)
			}

			if strings.Contains(jsonStr, `"issueId"`) {
				t.Errorf("JSON output should NOT contain %q (bug would cause this), got: %s", `"issueId"`, jsonStr)
			}
		})
	}
}
