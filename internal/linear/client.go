package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("not found")
	ErrRateLimited  = errors.New("rate limited")
)

type API interface {
	Me(ctx context.Context) (User, error)
	Teams(ctx context.Context) ([]Team, error)
	ResolveTeamID(ctx context.Context, keyOrID string) (string, error)
	ResolveUserID(ctx context.Context, value string) (string, error)
	ResolveStateID(ctx context.Context, teamID, value string) (string, error)
	ResolveLabelIDs(ctx context.Context, labels []string) ([]string, error)
	ResolveProjectID(ctx context.Context, value string) (string, error)
	ResolveCycleID(ctx context.Context, teamID, value string) (string, error)
	ResolveIssueID(ctx context.Context, value string) (string, error)
	Issue(ctx context.Context, value string) (IssueDetail, error)
	IssueComments(ctx context.Context, issueID string, limit int) ([]Comment, error)
	IssueUploads(ctx context.Context, issueID string, limit int) ([]Attachment, error)
	IssueRelations(ctx context.Context, issueID string, limit int) (IssueRelationSet, error)
	Issues(ctx context.Context, filter IssueFilter, limit int, after string) (IssuePage, error)
	IssueCreate(ctx context.Context, input map[string]any) (IssueSummary, error)
	IssueUpdate(ctx context.Context, input map[string]any) (IssueSummary, error)
	IssueComment(ctx context.Context, issueID, body string) (string, error)
	IssueRelationCreate(ctx context.Context, issueID, relatedIssueID, relationType string) (IssueRelation, error)
	IssueRelationDelete(ctx context.Context, relationID string) error
	Cycles(ctx context.Context, teamID string, current bool, limit int, after string) (CyclePage, error)
	Cycle(ctx context.Context, id string) (Cycle, error)
	WorkflowStates(ctx context.Context, teamID string) ([]WorkflowState, error)
}

type Client struct {
	apiURL string
	token  string
	http   *http.Client
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlError struct {
	Message string `json:"message"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []gqlError      `json:"errors"`
}

type gqlErrors struct {
	Errors []gqlError
}

func (e gqlErrors) Error() string {
	messages := make([]string, 0, len(e.Errors))
	for _, err := range e.Errors {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, "; ")
}

func (e gqlErrors) hasUnknownField(field string) bool {
	needle := fmt.Sprintf("Cannot query field \"%s\"", field)
	for _, err := range e.Errors {
		if strings.Contains(err.Message, needle) {
			return true
		}
	}
	return false
}

const defaultAPIURL = "https://api.linear.app/graphql"

func NewClient(token string, timeout time.Duration) API {
	return &Client{
		apiURL: defaultAPIURL,
		token:  token,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) do(ctx context.Context, query string, variables map[string]any, out any) error {
	payload, err := json.Marshal(gqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if c.token != "" {
		token := normalizeToken(c.token)
		if token != "" {
			req.Header.Set("Authorization", token)
		}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return ErrRateLimited
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var gqlResp gqlResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return gqlErrors{Errors: gqlResp.Errors}
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(gqlResp.Data, out); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}
	return nil
}

func normalizeToken(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "bearer ") {
		return strings.TrimSpace(trimmed[7:])
	}
	return trimmed
}
