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

func newUpdateIssueBodyTestClient(t *testing.T, wantIssueID, wantBody string, response string) *GraphQLClient {
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

			requestBody := string(body)
			if !strings.Contains(requestBody, "UpdateIssueBody") {
				t.Fatalf("request body missing UpdateIssueBody operation: %s", requestBody)
			}

			var payload struct {
				Variables struct {
					Input struct {
						IssueID string `json:"id"`
						Body    string `json:"body"`
					} `json:"input"`
				} `json:"variables"`
			}

			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("json.Unmarshal(request body) error = %v", err)
			}

			if payload.Variables.Input.IssueID != wantIssueID {
				t.Fatalf("input.id = %q, want %q", payload.Variables.Input.IssueID, wantIssueID)
			}

			if payload.Variables.Input.Body != wantBody {
				t.Fatalf("input.body = %q, want %q", payload.Variables.Input.Body, wantBody)
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

func TestUpdateIssueBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		issueID string
		body    string
		resp    string
		wantErr string
	}{
		{
			name:    "updates issue body with updateIssue mutation",
			issueID: "I_123",
			body:    "Updated issue body",
			resp:    `{"data":{"updateIssue":{"issue":{"id":"I_123"}}}}`,
		},
		{
			name:    "wraps graphql mutation errors",
			issueID: "I_456",
			body:    "Body that fails",
			resp:    `{"errors":[{"message":"boom"}]}`,
			wantErr: "update issue body I_456:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := newUpdateIssueBodyTestClient(t, tt.issueID, tt.body, tt.resp)

			err := client.UpdateIssueBody(tt.issueID, tt.body)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("UpdateIssueBody() error = %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("UpdateIssueBody() error = nil, want error containing %q", tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("UpdateIssueBody() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUpdateIssueBodyInputJSONSerializationUsesCorrectTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		issueID graphql.ID
		body    graphql.String
	}{
		{
			name:    "serializes with id field, not issueId",
			issueID: graphql.ID("I_123"),
			body:    graphql.String("Body one"),
		},
		{
			name:    "serializes both id and body",
			issueID: graphql.ID("I_456"),
			body:    graphql.String("Body two"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			type UpdateIssueInput struct {
				IssueID graphql.ID     `json:"id"`
				Body    graphql.String `json:"body"`
			}

			input := UpdateIssueInput{
				IssueID: tt.issueID,
				Body:    tt.body,
			}

			data, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			jsonStr := string(data)

			if !strings.Contains(jsonStr, `"id"`) {
				t.Fatalf("JSON output should contain %q, got: %s", `"id"`, jsonStr)
			}

			if strings.Contains(jsonStr, `"issueId"`) {
				t.Fatalf("JSON output should NOT contain %q, got: %s", `"issueId"`, jsonStr)
			}

			if !strings.Contains(jsonStr, `"body"`) {
				t.Fatalf("JSON output should contain %q, got: %s", `"body"`, jsonStr)
			}
		})
	}
}
