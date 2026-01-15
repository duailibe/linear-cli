package linear

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

const (
	viewerQuery = `query {
  viewer {
    id
    name
    email
  }
}`
	meQuery = `query {
  me {
    id
    name
    email
  }
}`
)

type cycleNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Number   int    `json:"number"`
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
	IsActive bool   `json:"isActive"`
}

func (c *Client) Me(ctx context.Context) (User, error) {
	var resp struct {
		Viewer *User `json:"viewer"`
	}
	err := c.do(ctx, viewerQuery, nil, &resp)
	if err == nil && resp.Viewer != nil {
		return *resp.Viewer, nil
	}
	var gqlErr gqlErrors
	if errors.As(err, &gqlErr) && gqlErr.hasUnknownField("viewer") {
		var resp2 struct {
			Me *User `json:"me"`
		}
		err = c.do(ctx, meQuery, nil, &resp2)
		if err != nil {
			return User{}, err
		}
		if resp2.Me == nil {
			return User{}, ErrNotFound
		}
		return *resp2.Me, nil
	}
	if err != nil {
		return User{}, err
	}
	return User{}, ErrNotFound
}

func (c *Client) Teams(ctx context.Context) ([]Team, error) {
	query := `query {
  teams {
    nodes { id key name }
  }
}`
	var resp struct {
		Teams struct {
			Nodes []Team `json:"nodes"`
		} `json:"teams"`
	}
	if err := c.do(ctx, query, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Teams.Nodes, nil
}

func (c *Client) ResolveTeamID(ctx context.Context, keyOrID string) (string, error) {
	if isLikelyID(keyOrID) {
		team, err := c.teamByID(ctx, keyOrID)
		if err == nil {
			return team.ID, nil
		}
	}
	team, err := c.teamByKey(ctx, keyOrID)
	if err != nil {
		return "", err
	}
	return team.ID, nil
}

func (c *Client) teamByID(ctx context.Context, id string) (Team, error) {
	query := `query($id: ID!) {
  team(id: $id) { id key name }
}`
	var resp struct {
		Team *Team `json:"team"`
	}
	if err := c.do(ctx, query, map[string]any{"id": id}, &resp); err != nil {
		return Team{}, err
	}
	if resp.Team == nil {
		return Team{}, ErrNotFound
	}
	return *resp.Team, nil
}

func (c *Client) teamByKey(ctx context.Context, key string) (Team, error) {
	query := `query($key: String!) {
  teams(filter: { key: { eq: $key } }) {
    nodes { id key name }
  }
}`
	var resp struct {
		Teams struct {
			Nodes []Team `json:"nodes"`
		} `json:"teams"`
	}
	if err := c.do(ctx, query, map[string]any{"key": key}, &resp); err != nil {
		return Team{}, err
	}
	if len(resp.Teams.Nodes) == 0 {
		return Team{}, ErrNotFound
	}
	return resp.Teams.Nodes[0], nil
}

func (c *Client) ResolveUserID(ctx context.Context, value string) (string, error) {
	if value == "me" {
		user, err := c.Me(ctx)
		if err != nil {
			return "", err
		}
		return user.ID, nil
	}
	if isLikelyID(value) {
		return value, nil
	}
	if strings.Contains(value, "@") {
		query := `query($email: String!) {
  users(filter: { email: { eq: $email } }) {
    nodes { id }
  }
}`
		var resp struct {
			Users struct {
				Nodes []struct {
					ID string `json:"id"`
				} `json:"nodes"`
			} `json:"users"`
		}
		if err := c.do(ctx, query, map[string]any{"email": value}, &resp); err != nil {
			return "", err
		}
		if len(resp.Users.Nodes) == 0 {
			return "", ErrNotFound
		}
		return resp.Users.Nodes[0].ID, nil
	}
	return "", fmt.Errorf("assignee must be 'me', an id, or an email")
}

func (c *Client) WorkflowStates(ctx context.Context, teamID string) ([]WorkflowState, error) {
	query := `query($id: String!) {
  team(id: $id) {
    states {
      nodes { id name type }
    }
  }
}`
	var resp struct {
		Team *struct {
			States struct {
				Nodes []WorkflowState `json:"nodes"`
			} `json:"states"`
		} `json:"team"`
	}
	if err := c.do(ctx, query, map[string]any{"id": teamID}, &resp); err != nil {
		return nil, err
	}
	if resp.Team == nil {
		return nil, ErrNotFound
	}
	return resp.Team.States.Nodes, nil
}

func (c *Client) ResolveStateID(ctx context.Context, teamID, value string) (string, error) {
	if isLikelyID(value) {
		return value, nil
	}
	states, err := c.WorkflowStates(ctx, teamID)
	if err != nil {
		return "", err
	}
	for _, state := range states {
		if strings.EqualFold(state.Name, value) {
			return state.ID, nil
		}
	}
	return "", ErrNotFound
}

func (c *Client) ResolveLabelIDs(ctx context.Context, labels []string) ([]string, error) {
	if len(labels) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(labels))
	for _, label := range labels {
		if label == "" {
			continue
		}
		if isLikelyID(label) {
			ids = append(ids, label)
			continue
		}
		query := `query($name: String!) {
  issueLabels(filter: { name: { eq: $name } }) {
    nodes { id }
  }
}`
		var resp struct {
			IssueLabels struct {
				Nodes []struct {
					ID string `json:"id"`
				} `json:"nodes"`
			} `json:"issueLabels"`
		}
		if err := c.do(ctx, query, map[string]any{"name": label}, &resp); err != nil {
			return nil, err
		}
		if len(resp.IssueLabels.Nodes) == 0 {
			return nil, ErrNotFound
		}
		ids = append(ids, resp.IssueLabels.Nodes[0].ID)
	}
	return ids, nil
}

func (c *Client) ResolveProjectID(ctx context.Context, value string) (string, error) {
	if isLikelyID(value) {
		return value, nil
	}
	query := `query($name: String!) {
  projects(filter: { name: { eq: $name } }) {
    nodes { id }
  }
}`
	var resp struct {
		Projects struct {
			Nodes []struct {
				ID string `json:"id"`
			} `json:"nodes"`
		} `json:"projects"`
	}
	if err := c.do(ctx, query, map[string]any{"name": value}, &resp); err != nil {
		return "", err
	}
	if len(resp.Projects.Nodes) == 0 {
		return "", ErrNotFound
	}
	return resp.Projects.Nodes[0].ID, nil
}

func (c *Client) ResolveCycleID(ctx context.Context, teamID, value string) (string, error) {
	if value == "current" {
		page, err := c.Cycles(ctx, teamID, true, 1, "")
		if err != nil {
			return "", err
		}
		if len(page.Nodes) == 0 {
			return "", ErrNotFound
		}
		return page.Nodes[0].ID, nil
	}
	if isLikelyID(value) {
		return value, nil
	}
	return "", fmt.Errorf("cycle must be an id or 'current'")
}

func (c *Client) ResolveIssueID(ctx context.Context, value string) (string, error) {
	query := `query($id: String!) {
  issue(id: $id) { id }
}`
	var resp struct {
		Issue *struct {
			ID string `json:"id"`
		} `json:"issue"`
	}
	if err := c.do(ctx, query, map[string]any{"id": value}, &resp); err != nil {
		return "", err
	}
	if resp.Issue == nil {
		return "", ErrNotFound
	}
	return resp.Issue.ID, nil
}

func (c *Client) Issue(ctx context.Context, value string) (IssueDetail, error) {
	query := `query($id: String!) {
  issue(id: $id) {
    id
    identifier
    title
    url
    description
    priority
    createdAt
    updatedAt
    team { id key }
    state { name }
    assignee { name }
    cycle { name }
    project { name }
    labels { nodes { name } }
  }
}`
	var resp struct {
		Issue *struct {
			ID          string `json:"id"`
			Identifier  string `json:"identifier"`
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Priority    int    `json:"priority"`
			CreatedAt   string `json:"createdAt"`
			UpdatedAt   string `json:"updatedAt"`
			Team        struct {
				ID  string `json:"id"`
				Key string `json:"key"`
			} `json:"team"`
			State struct {
				Name string `json:"name"`
			} `json:"state"`
			Assignee *struct {
				Name string `json:"name"`
			} `json:"assignee"`
			Cycle *struct {
				Name string `json:"name"`
			} `json:"cycle"`
			Project *struct {
				Name string `json:"name"`
			} `json:"project"`
			Labels struct {
				Nodes []struct {
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"labels"`
		} `json:"issue"`
	}
	if err := c.do(ctx, query, map[string]any{"id": value}, &resp); err != nil {
		return IssueDetail{}, err
	}
	if resp.Issue == nil {
		return IssueDetail{}, ErrNotFound
	}

	labels := make([]string, 0, len(resp.Issue.Labels.Nodes))
	for _, label := range resp.Issue.Labels.Nodes {
		labels = append(labels, label.Name)
	}

	var assignee string
	if resp.Issue.Assignee != nil {
		assignee = resp.Issue.Assignee.Name
	}

	var cycle string
	if resp.Issue.Cycle != nil {
		cycle = resp.Issue.Cycle.Name
	}
	var project string
	if resp.Issue.Project != nil {
		project = resp.Issue.Project.Name
	}

	return IssueDetail{
		ID:          resp.Issue.ID,
		Identifier:  resp.Issue.Identifier,
		Title:       resp.Issue.Title,
		URL:         resp.Issue.URL,
		Description: resp.Issue.Description,
		Priority:    resp.Issue.Priority,
		State:       resp.Issue.State.Name,
		Assignee:    assignee,
		TeamID:      resp.Issue.Team.ID,
		TeamKey:     resp.Issue.Team.Key,
		Cycle:       cycle,
		Project:     project,
		Labels:      labels,
		CreatedAt:   resp.Issue.CreatedAt,
		UpdatedAt:   resp.Issue.UpdatedAt,
	}, nil
}

func (c *Client) IssueComments(ctx context.Context, issueID string, limit int) ([]Comment, error) {
	query := `query($id: String!, $first: Int) {
  issue(id: $id) {
    comments(first: $first) {
      nodes { id body bodyData createdAt user { name email } }
    }
  }
}`

	var resp struct {
		Issue *struct {
			Comments struct {
				Nodes []struct {
					ID        string `json:"id"`
					Body      string `json:"body"`
					BodyData  string `json:"bodyData"`
					CreatedAt string `json:"createdAt"`
					User      *struct {
						Name  string `json:"name"`
						Email string `json:"email"`
					} `json:"user"`
				} `json:"nodes"`
			} `json:"comments"`
		} `json:"issue"`
	}

	vars := map[string]any{"id": issueID}
	if limit > 0 {
		vars["first"] = limit
	}
	if err := c.do(ctx, query, vars, &resp); err != nil {
		return nil, err
	}
	if resp.Issue == nil {
		return nil, ErrNotFound
	}

	comments := make([]Comment, 0, len(resp.Issue.Comments.Nodes))
	for _, node := range resp.Issue.Comments.Nodes {
		comment := Comment{
			ID:        node.ID,
			CreatedAt: node.CreatedAt,
			Body:      node.Body,
			BodyData:  node.BodyData,
		}
		if node.User != nil {
			comment.UserName = node.User.Name
			comment.UserEmail = node.User.Email
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (c *Client) IssueAttachments(ctx context.Context, issueID string, limit int) ([]Attachment, error) {
	query := `query($id: String!, $first: Int) {
  issue(id: $id) {
    attachments(first: $first) {
      nodes { id title url source createdAt }
    }
  }
}`

	var resp struct {
		Issue *struct {
			Attachments struct {
				Nodes []struct {
					ID        string `json:"id"`
					Title     string `json:"title"`
					URL       string `json:"url"`
					Source    string `json:"source"`
					CreatedAt string `json:"createdAt"`
				} `json:"nodes"`
			} `json:"attachments"`
		} `json:"issue"`
	}

	vars := map[string]any{"id": issueID}
	if limit > 0 {
		vars["first"] = limit
	}
	if err := c.do(ctx, query, vars, &resp); err != nil {
		return nil, err
	}
	if resp.Issue == nil {
		return nil, ErrNotFound
	}

	attachments := make([]Attachment, 0, len(resp.Issue.Attachments.Nodes))
	for _, item := range resp.Issue.Attachments.Nodes {
		url := item.URL
		if url == "" {
			url = item.Source
		}
		attachments = append(attachments, Attachment{
			ID:        item.ID,
			Title:     item.Title,
			URL:       url,
			CreatedAt: item.CreatedAt,
		})
	}

	if len(attachments) == 0 {
		comments, err := c.IssueComments(ctx, issueID, limit)
		if err == nil {
			attachments = append(attachments, extractAttachmentsFromComments(comments)...)
		}
	}

	return attachments, nil
}

func (c *Client) IssueRelations(ctx context.Context, issueID string, limit int) (IssueRelationSet, error) {
	query := `query($id: String!, $first: Int) {
  issue(id: $id) {
    relations(first: $first) { nodes { id type issue { id } relatedIssue { id } } }
    inverseRelations(first: $first) { nodes { id type issue { id } relatedIssue { id } } }
  }
}`

	type relationNode struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Issue struct {
			ID string `json:"id"`
		} `json:"issue"`
		RelatedIssue struct {
			ID string `json:"id"`
		} `json:"relatedIssue"`
	}
	var resp struct {
		Issue *struct {
			Relations *struct {
				Nodes []relationNode `json:"nodes"`
			} `json:"relations"`
			InverseRelations *struct {
				Nodes []relationNode `json:"nodes"`
			} `json:"inverseRelations"`
		} `json:"issue"`
	}

	vars := map[string]any{"id": issueID}
	if limit > 0 {
		vars["first"] = limit
	}
	if err := c.do(ctx, query, vars, &resp); err != nil {
		return IssueRelationSet{}, err
	}
	if resp.Issue == nil {
		return IssueRelationSet{}, ErrNotFound
	}

	result := IssueRelationSet{
		Relations:        []IssueRelation{},
		InverseRelations: []IssueRelation{},
	}
	if resp.Issue.Relations != nil {
		for _, node := range resp.Issue.Relations.Nodes {
			result.Relations = append(result.Relations, IssueRelation{
				ID:             node.ID,
				IssueID:        node.Issue.ID,
				RelatedIssueID: node.RelatedIssue.ID,
				Type:           node.Type,
			})
		}
	}
	if resp.Issue.InverseRelations != nil {
		for _, node := range resp.Issue.InverseRelations.Nodes {
			result.InverseRelations = append(result.InverseRelations, IssueRelation{
				ID:             node.ID,
				IssueID:        node.Issue.ID,
				RelatedIssueID: node.RelatedIssue.ID,
				Type:           node.Type,
			})
		}
	}

	return result, nil
}

func (c *Client) IssueRelationCreate(ctx context.Context, issueID, relatedIssueID, relationType string) (IssueRelation, error) {
	query := `mutation($input: IssueRelationCreateInput!) {
  issueRelationCreate(input: $input) {
    issueRelation { id type issue { id } relatedIssue { id } }
  }
}`
	var resp struct {
		IssueRelationCreate struct {
			IssueRelation *struct {
				ID    string `json:"id"`
				Type  string `json:"type"`
				Issue struct {
					ID string `json:"id"`
				} `json:"issue"`
				RelatedIssue struct {
					ID string `json:"id"`
				} `json:"relatedIssue"`
			} `json:"issueRelation"`
		} `json:"issueRelationCreate"`
	}
	input := map[string]any{
		"issueId":        issueID,
		"relatedIssueId": relatedIssueID,
		"type":           relationType,
	}
	if err := c.do(ctx, query, map[string]any{"input": input}, &resp); err != nil {
		return IssueRelation{}, err
	}
	if resp.IssueRelationCreate.IssueRelation == nil {
		return IssueRelation{}, ErrNotFound
	}
	return IssueRelation{
		ID:             resp.IssueRelationCreate.IssueRelation.ID,
		IssueID:        resp.IssueRelationCreate.IssueRelation.Issue.ID,
		RelatedIssueID: resp.IssueRelationCreate.IssueRelation.RelatedIssue.ID,
		Type:           resp.IssueRelationCreate.IssueRelation.Type,
	}, nil
}

func (c *Client) IssueRelationDelete(ctx context.Context, relationID string) error {
	query := `mutation($id: String!) {
  issueRelationDelete(id: $id) {
    success
  }
}`
	var resp struct {
		IssueRelationDelete *struct {
			Success bool `json:"success"`
		} `json:"issueRelationDelete"`
	}
	if err := c.do(ctx, query, map[string]any{"id": relationID}, &resp); err != nil {
		return err
	}
	if resp.IssueRelationDelete == nil {
		return ErrNotFound
	}
	if !resp.IssueRelationDelete.Success {
		return fmt.Errorf("relation delete failed")
	}
	return nil
}

func extractAttachmentsFromComments(comments []Comment) []Attachment {
	seen := map[string]struct{}{}
	attachments := []Attachment{}
	for _, comment := range comments {
		body := comment.Body
		if body == "" {
			body = comment.BodyData
		}
		if body == "" {
			continue
		}
		for _, item := range parseUploadsLinks(body) {
			if _, ok := seen[item.URL]; ok {
				continue
			}
			seen[item.URL] = struct{}{}
			item.CommentID = comment.ID
			item.CreatedAt = comment.CreatedAt
			attachments = append(attachments, item)
		}
	}
	return attachments
}

func parseUploadsLinks(text string) []Attachment {
	matches := uploadMarkdownLinkRe.FindAllStringSubmatch(text, -1)
	urls := map[string]Attachment{}
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		url := match[2]
		title := match[1]
		if url == "" {
			continue
		}
		urls[url] = Attachment{
			ID:       url,
			Title:    title,
			URL:      url,
			FileName: preferredFileName(title, url),
		}
	}
	rawMatches := uploadURLRe.FindAllString(text, -1)
	for _, url := range rawMatches {
		if url == "" {
			continue
		}
		if _, ok := urls[url]; ok {
			continue
		}
		urls[url] = Attachment{
			ID:       url,
			URL:      url,
			FileName: preferredFileName("", url),
		}
	}
	out := make([]Attachment, 0, len(urls))
	for _, item := range urls {
		out = append(out, item)
	}
	return out
}

func preferredFileName(title, urlStr string) string {
	title = strings.TrimSpace(title)
	if title != "" && strings.Contains(title, ".") {
		return sanitizeFileName(title)
	}
	parsed, err := url.Parse(urlStr)
	if err == nil {
		base := path.Base(parsed.Path)
		if base != "." && base != "/" && base != "" {
			return sanitizeFileName(base)
		}
	}
	if title != "" {
		return sanitizeFileName(title)
	}
	return "attachment"
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	if name == "" {
		return "attachment"
	}
	return name
}

var (
	uploadMarkdownLinkRe = regexp.MustCompile(`\[([^\]]+)\]\((https?://uploads\.linear\.app/[^\)\s]+)\)`)
	uploadURLRe          = regexp.MustCompile(`https?://uploads\.linear\.app/[^\s\)]+`)
)

func (c *Client) Issues(ctx context.Context, filter IssueFilter, limit int, after string) (IssuePage, error) {
	query := `query($filter: IssueFilter, $first: Int, $after: String) {
  issues(filter: $filter, first: $first, after: $after) {
    nodes {
      id
      identifier
      title
      url
      priority
      state { name }
      assignee { name }
      team { key }
      cycle { name }
    }
    pageInfo { hasNextPage endCursor }
  }
}`
	vars := map[string]any{}
	if limit > 0 {
		vars["first"] = limit
	}
	if after != "" {
		vars["after"] = after
	}
	vars["filter"] = buildIssueFilter(filter)

	var resp struct {
		Issues struct {
			Nodes []struct {
				ID         string `json:"id"`
				Identifier string `json:"identifier"`
				Title      string `json:"title"`
				URL        string `json:"url"`
				Priority   int    `json:"priority"`
				State      struct {
					Name string `json:"name"`
				} `json:"state"`
				Assignee *struct {
					Name string `json:"name"`
				} `json:"assignee"`
				Team struct {
					Key string `json:"key"`
				} `json:"team"`
				Cycle *struct {
					Name string `json:"name"`
				} `json:"cycle"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"issues"`
	}
	if err := c.do(ctx, query, vars, &resp); err != nil {
		return IssuePage{}, err
	}

	page := IssuePage{
		Nodes:    make([]IssueSummary, 0, len(resp.Issues.Nodes)),
		PageInfo: PageInfo{HasNextPage: resp.Issues.PageInfo.HasNextPage, EndCursor: resp.Issues.PageInfo.EndCursor},
	}
	for _, node := range resp.Issues.Nodes {
		assignee := ""
		if node.Assignee != nil {
			assignee = node.Assignee.Name
		}
		cycle := ""
		if node.Cycle != nil {
			cycle = node.Cycle.Name
		}
		page.Nodes = append(page.Nodes, IssueSummary{
			ID:         node.ID,
			Identifier: node.Identifier,
			Title:      node.Title,
			URL:        node.URL,
			State:      node.State.Name,
			Assignee:   assignee,
			TeamKey:    node.Team.Key,
			Cycle:      cycle,
			Priority:   node.Priority,
		})
	}
	return page, nil
}

func (c *Client) IssueCreate(ctx context.Context, input map[string]any) (IssueSummary, error) {
	query := `mutation($input: IssueCreateInput!) {
  issueCreate(input: $input) {
    issue { id identifier title url }
  }
}`
	var resp struct {
		IssueCreate struct {
			Issue *IssueSummary `json:"issue"`
		} `json:"issueCreate"`
	}
	if err := c.do(ctx, query, map[string]any{"input": input}, &resp); err != nil {
		return IssueSummary{}, err
	}
	if resp.IssueCreate.Issue == nil {
		return IssueSummary{}, ErrNotFound
	}
	return *resp.IssueCreate.Issue, nil
}

func (c *Client) IssueUpdate(ctx context.Context, input map[string]any) (IssueSummary, error) {
	id, _ := input["id"].(string)
	if id == "" {
		return IssueSummary{}, errors.New("issue id is required")
	}
	trimmed := map[string]any{}
	for key, value := range input {
		if key == "id" {
			continue
		}
		trimmed[key] = value
	}
	query := `mutation($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) {
    issue { id identifier title url }
  }
}`
	var resp struct {
		IssueUpdate struct {
			Issue *IssueSummary `json:"issue"`
		} `json:"issueUpdate"`
	}
	if err := c.do(ctx, query, map[string]any{"id": id, "input": trimmed}, &resp); err != nil {
		return IssueSummary{}, err
	}
	if resp.IssueUpdate.Issue == nil {
		return IssueSummary{}, ErrNotFound
	}
	return *resp.IssueUpdate.Issue, nil
}

func (c *Client) IssueComment(ctx context.Context, issueID, body string) (string, error) {
	query := `mutation($input: CommentCreateInput!) {
  commentCreate(input: $input) {
    comment { id }
  }
}`
	var resp struct {
		CommentCreate struct {
			Comment *struct {
				ID string `json:"id"`
			} `json:"comment"`
		} `json:"commentCreate"`
	}
	if err := c.do(ctx, query, map[string]any{"input": map[string]any{"issueId": issueID, "body": body}}, &resp); err != nil {
		return "", err
	}
	if resp.CommentCreate.Comment == nil {
		return "", ErrNotFound
	}
	return resp.CommentCreate.Comment.ID, nil
}

func (c *Client) Cycles(ctx context.Context, teamID string, current bool, limit int, after string) (CyclePage, error) {
	query := `query($filter: CycleFilter, $first: Int, $after: String) {
  cycles(filter: $filter, first: $first, after: $after) {
    nodes { id name number startsAt endsAt isActive }
    pageInfo { hasNextPage endCursor }
  }
}`
	vars := map[string]any{}
	if limit > 0 {
		vars["first"] = limit
	}
	if after != "" {
		vars["after"] = after
	}

	filter := map[string]any{
		"team": map[string]any{"id": map[string]any{"eq": teamID}},
	}
	if current {
		filter["isActive"] = map[string]any{"eq": true}
	}
	vars["filter"] = filter

	var resp struct {
		Cycles struct {
			Nodes    []cycleNode `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"cycles"`
	}

	err := c.do(ctx, query, vars, &resp)
	if err != nil {
		var gqlErr gqlErrors
		if errors.As(err, &gqlErr) && gqlErr.hasUnknownField("cycles") {
			return c.cyclesViaTeam(ctx, teamID, current, limit, after)
		}
		return CyclePage{}, err
	}

	return mapCycles(resp.Cycles.Nodes, resp.Cycles.PageInfo.HasNextPage, resp.Cycles.PageInfo.EndCursor), nil
}

func (c *Client) cyclesViaTeam(ctx context.Context, teamID string, current bool, limit int, after string) (CyclePage, error) {
	query := `query($id: ID!, $first: Int, $after: String) {
  team(id: $id) {
    cycles(first: $first, after: $after) {
      nodes { id name number startsAt endsAt isActive }
      pageInfo { hasNextPage endCursor }
    }
  }
}`
	vars := map[string]any{"id": teamID}
	if limit > 0 {
		vars["first"] = limit
	}
	if after != "" {
		vars["after"] = after
	}
	var resp struct {
		Team *struct {
			Cycles struct {
				Nodes    []cycleNode `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"cycles"`
		} `json:"team"`
	}
	if err := c.do(ctx, query, vars, &resp); err != nil {
		return CyclePage{}, err
	}
	if resp.Team == nil {
		return CyclePage{}, ErrNotFound
	}

	page := mapCycles(resp.Team.Cycles.Nodes, resp.Team.Cycles.PageInfo.HasNextPage, resp.Team.Cycles.PageInfo.EndCursor)
	if !current {
		return page, nil
	}

	filtered := make([]Cycle, 0, len(page.Nodes))
	for _, cycle := range page.Nodes {
		if cycle.IsActive == "true" {
			filtered = append(filtered, cycle)
		}
	}
	page.Nodes = filtered
	return page, nil
}

func mapCycles(nodes []cycleNode, hasNext bool, endCursor string) CyclePage {
	cycles := make([]Cycle, 0, len(nodes))
	for _, node := range nodes {
		isActive := "false"
		if node.IsActive {
			isActive = "true"
		}
		cycles = append(cycles, Cycle{
			ID:       node.ID,
			Name:     node.Name,
			Number:   fmt.Sprintf("%d", node.Number),
			StartsAt: node.StartsAt,
			EndsAt:   node.EndsAt,
			IsActive: isActive,
		})
	}
	return CyclePage{Nodes: cycles, PageInfo: PageInfo{HasNextPage: hasNext, EndCursor: endCursor}}
}

func (c *Client) Cycle(ctx context.Context, id string) (Cycle, error) {
	query := `query($id: ID!) {
  cycle(id: $id) { id name number startsAt endsAt isActive }
}`
	var resp struct {
		Cycle *struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Number   int    `json:"number"`
			StartsAt string `json:"startsAt"`
			EndsAt   string `json:"endsAt"`
			IsActive bool   `json:"isActive"`
		} `json:"cycle"`
	}
	if err := c.do(ctx, query, map[string]any{"id": id}, &resp); err != nil {
		return Cycle{}, err
	}
	if resp.Cycle == nil {
		return Cycle{}, ErrNotFound
	}
	active := "false"
	if resp.Cycle.IsActive {
		active = "true"
	}
	return Cycle{
		ID:       resp.Cycle.ID,
		Name:     resp.Cycle.Name,
		Number:   fmt.Sprintf("%d", resp.Cycle.Number),
		StartsAt: resp.Cycle.StartsAt,
		EndsAt:   resp.Cycle.EndsAt,
		IsActive: active,
	}, nil
}

func buildIssueFilter(filter IssueFilter) map[string]any {
	if filter.TeamID == "" &&
		filter.AssigneeID == "" &&
		filter.StateID == "" &&
		len(filter.LabelIDs) == 0 &&
		filter.ProjectID == "" &&
		filter.CycleID == "" &&
		filter.Search == "" &&
		filter.Priority == nil {
		return nil
	}
	out := map[string]any{}
	if filter.TeamID != "" {
		out["team"] = map[string]any{"id": map[string]any{"eq": filter.TeamID}}
	}
	if filter.AssigneeID != "" {
		out["assignee"] = map[string]any{"id": map[string]any{"eq": filter.AssigneeID}}
	}
	if filter.StateID != "" {
		out["state"] = map[string]any{"id": map[string]any{"eq": filter.StateID}}
	}
	if len(filter.LabelIDs) > 0 {
		out["labels"] = map[string]any{"id": map[string]any{"in": filter.LabelIDs}}
	}
	if filter.ProjectID != "" {
		out["project"] = map[string]any{"id": map[string]any{"eq": filter.ProjectID}}
	}
	if filter.CycleID != "" {
		out["cycle"] = map[string]any{"id": map[string]any{"eq": filter.CycleID}}
	}
	if filter.Search != "" {
		out["title"] = map[string]any{"contains": filter.Search}
	}
	if filter.Priority != nil {
		out["priority"] = map[string]any{"eq": *filter.Priority}
	}
	return out
}

func isLikelyID(value string) bool {
	if len(value) < 30 {
		return false
	}
	if strings.Count(value, "-") >= 4 {
		return true
	}
	return false
}
