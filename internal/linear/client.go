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
	"sync"
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
	IssueAttachments(ctx context.Context, issueID string, limit int) ([]Attachment, error)
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

	schemaPath  string
	schemaOnce  sync.Once
	schemaCache *schemaCache
	schemaErr   error
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

func NewClient(apiURL, token string, timeout time.Duration) API {
	schemaPath, _ := DefaultSchemaPath()
	return &Client{
		apiURL: apiURL,
		token:  token,
		http: &http.Client{
			Timeout: timeout,
		},
		schemaPath: schemaPath,
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

func (c *Client) schemaArgBaseType(ctx context.Context, fieldName, argName string) (string, bool) {
	c.schemaOnce.Do(func() {
		cache, err := c.loadSchema(ctx)
		if err != nil {
			c.schemaErr = err
			return
		}
		c.schemaCache = cache
	})
	if c.schemaCache == nil {
		return "", false
	}
	return c.schemaCache.argBaseType(fieldName, argName)
}

func (c *Client) schemaHasField(ctx context.Context, typeName, fieldName string) bool {
	c.schemaOnce.Do(func() {
		cache, err := c.loadSchema(ctx)
		if err != nil {
			c.schemaErr = err
			return
		}
		c.schemaCache = cache
	})
	if c.schemaCache == nil {
		return false
	}
	if c.schemaCache.hasField(typeName, fieldName) {
		return true
	}
	info, ok, err := c.loadSchemaType(ctx, typeName)
	if err != nil || !ok {
		return false
	}
	return info.hasField(fieldName)
}

func (info schemaTypeInfo) hasField(fieldName string) bool {
	for _, field := range info.Fields {
		if strings.EqualFold(field.Name, fieldName) {
			return true
		}
	}
	return false
}

func (c *Client) schemaField(ctx context.Context, typeName, fieldName string) (schemaField, bool) {
	c.schemaOnce.Do(func() {
		cache, err := c.loadSchema(ctx)
		if err != nil {
			c.schemaErr = err
			return
		}
		c.schemaCache = cache
	})
	if c.schemaCache == nil {
		return schemaField{}, false
	}
	if field, ok := c.schemaCache.field(typeName, fieldName); ok {
		return field, true
	}
	info, ok, err := c.loadSchemaType(ctx, typeName)
	if err != nil || !ok {
		return schemaField{}, false
	}
	for _, field := range info.Fields {
		if strings.EqualFold(field.Name, fieldName) {
			return field, true
		}
	}
	return schemaField{}, false
}
