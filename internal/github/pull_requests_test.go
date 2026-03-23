package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestListRepositoryPullRequestsHonorsLimit(t *testing.T) {
	t.Parallel()

	callCount := 0

	apiClient, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "token",
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("request method = %s, want POST", req.Method)
			}

			callCount++

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll(request body) error = %v", err)
			}

			requestBody := string(body)
			if !strings.Contains(requestBody, "ListRepositoryPullRequests") {
				t.Fatalf("request body missing ListRepositoryPullRequests operation: %s", requestBody)
			}

			hasNext := callCount < 5
			responseJSON := buildPullRequestsPageResponse(callCount, hasNext)

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(responseJSON)),
				Request:    req,
			}, nil
		}),
	})
	if err != nil {
		t.Fatalf("api.NewGraphQLClient() error = %v", err)
	}

	client := &GraphQLClient{client: apiClient}
	prs, err := client.ListRepositoryPullRequests("ifloresarg", "gh-projects", 25)
	if err != nil {
		t.Fatalf("ListRepositoryPullRequests() error = %v", err)
	}

	if len(prs) != 25 {
		t.Fatalf("len(prs) = %d, want 25", len(prs))
	}

	if callCount != 3 {
		t.Fatalf("GraphQL call count = %d, want 3 (early exit after reaching limit)", callCount)
	}

	if prs[0].Number != 1 {
		t.Fatalf("first PR number = %d, want 1", prs[0].Number)
	}

	if prs[len(prs)-1].Number != 25 {
		t.Fatalf("last PR number = %d, want 25", prs[len(prs)-1].Number)
	}
}

func buildPullRequestsPageResponse(page int, hasNext bool) string {
	nodes := make([]map[string]any, 0, 10)

	for i := range 10 {
		number := (page-1)*10 + i + 1
		nodes = append(nodes, map[string]any{
			"id":        fmt.Sprintf("PR_%d", number),
			"number":    number,
			"title":     fmt.Sprintf("PR %d", number),
			"state":     "OPEN",
			"createdAt": "2026-03-23T00:00:00Z",
			"mergedAt":  "2026-03-23T00:00:00Z",
			"author": map[string]any{
				"login": "octocat",
				"name":  "The Octocat",
			},
			"url": fmt.Sprintf("https://github.com/ifloresarg/gh-projects/pull/%d", number),
		})
	}

	response := map[string]any{
		"data": map[string]any{
			"repository": map[string]any{
				"owner": map[string]any{
					"login": "ifloresarg",
				},
				"name": "gh-projects",
				"pullRequests": map[string]any{
					"nodes": nodes,
					"pageInfo": map[string]any{
						"hasNextPage": hasNext,
						"endCursor":   fmt.Sprintf("cursor-%d", page),
					},
				},
			},
		},
	}

	b, _ := json.Marshal(response)

	return string(b)
}
