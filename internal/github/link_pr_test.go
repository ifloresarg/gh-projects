package github

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

func newLinkPRTestClient(t *testing.T, existingBody, expectedUpdatedBody string) (*GraphQLClient, *int) {
	t.Helper()

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

			switch {
			case strings.Contains(requestBody, "GetPullRequest"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"repository": {
								"owner": {"login": "octocat"},
								"name": "gh-projects",
								"pullRequest": {
									"id": "PR_1",
									"number": 12,
									"title": "Link issue",
									"url": "https://github.com/octocat/gh-projects/pull/12",
									"body": ` + strconvQuoteJSON(existingBody) + `,
									"state": "OPEN"
								}
							}
						}
					}`)),
					Request: req,
				}, nil
			case strings.Contains(requestBody, "LinkPRToIssue"):
				var payload struct {
					Variables struct {
						Input struct {
							PullRequestID string `json:"pullRequestId"`
							Body          string `json:"body"`
						} `json:"input"`
					} `json:"variables"`
				}

				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("json.Unmarshal(request body) error = %v", err)
				}

				if payload.Variables.Input.PullRequestID != "PR_1" {
					t.Fatalf("pullRequestId = %q, want %q", payload.Variables.Input.PullRequestID, "PR_1")
				}

				if payload.Variables.Input.Body != expectedUpdatedBody {
					t.Fatalf("mutation body = %q, want %q", payload.Variables.Input.Body, expectedUpdatedBody)
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"data":{"updatePullRequest":{"pullRequest":{"id":"PR_1"}}}}`)),
					Request:    req,
				}, nil
			default:
				t.Fatalf("unexpected GraphQL operation request body: %s", requestBody)
				return nil, nil
			}
		}),
	})
	if err != nil {
		t.Fatalf("api.NewGraphQLClient() error = %v", err)
	}

	return &GraphQLClient{client: apiClient}, &callCount
}

func strconvQuoteJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestLinkPRToIssueAppendsClosingKeyword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		existingBody string
		wantBody     string
	}{
		{
			name:         "append to empty body",
			existingBody: "",
			wantBody:     "\n\nCloses octocat/gh-projects#34",
		},
		{
			name:         "append to existing body",
			existingBody: "This PR updates the board rendering.",
			wantBody:     "This PR updates the board rendering.\n\nCloses octocat/gh-projects#34",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, callCount := newLinkPRTestClient(t, tt.existingBody, tt.wantBody)

			if err := client.LinkPRToIssue("octocat", "gh-projects", 12, 34); err != nil {
				t.Fatalf("LinkPRToIssue() error = %v", err)
			}

			if *callCount != 2 {
				t.Fatalf("GraphQL call count = %d, want 2", *callCount)
			}
		})
	}
}

func TestLinkPRToIssueSkipsDuplicate(t *testing.T) {
	t.Parallel()

	existingBody := "Fix rendering.\n\nCloses octocat/gh-projects#34"

	apiClient, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "token",
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll(request body) error = %v", err)
			}

			requestBody := string(body)

			if strings.Contains(requestBody, "GetPullRequest") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"repository": {
								"owner": {"login": "octocat"},
								"name": "gh-projects",
								"pullRequest": {
									"id": "PR_1",
									"number": 12,
									"title": "Link issue",
									"url": "https://github.com/octocat/gh-projects/pull/12",
									"body": ` + strconvQuoteJSON(existingBody) + `,
									"state": "OPEN"
								}
							}
						}
					}`)),
					Request: req,
				}, nil
			}

			t.Fatalf("unexpected mutation call when closing ref already exists: %s", requestBody)
			return nil, nil
		}),
	})
	if err != nil {
		t.Fatalf("api.NewGraphQLClient() error = %v", err)
	}

	client := &GraphQLClient{client: apiClient}

	if err := client.LinkPRToIssue("octocat", "gh-projects", 12, 34); err != nil {
		t.Fatalf("LinkPRToIssue() error = %v", err)
	}
}
