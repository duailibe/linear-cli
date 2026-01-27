package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/duailibe/linear-cli/internal/linear"
)

type IssueCmd struct {
	List        IssueListCmd        `cmd:"" help:"List issues"`
	View        IssueViewCmd        `cmd:"" help:"View issue details"`
	Create      IssueCreateCmd      `cmd:"" help:"Create an issue"`
	Update      IssueUpdateCmd      `cmd:"" help:"Update an issue"`
	Close       IssueCloseCmd       `cmd:"" help:"Close an issue"`
	Reopen      IssueReopenCmd      `cmd:"" help:"Reopen an issue"`
	Comment     IssueCommentCmd     `cmd:"" help:"Add a comment to an issue"`
	Uploads     IssueUploadsCmd     `cmd:"" help:"Download issue uploads from the issue description and comments"`
}

type IssueListCmd struct {
	Team     string `help:"Team key or ID"`
	Assignee string `help:"Assignee (me, id, or email)"`
	State    string `help:"Workflow state name or ID"`
	Labels   string `name:"label" help:"Comma-separated label names or IDs"`
	Project  string `help:"Project name or ID"`
	Cycle    string `help:"Cycle ID or 'current'"`
	Search   string `help:"Search issue titles"`
	Priority int    `help:"Priority (0-4)" default:"-1"`
	Limit    int    `help:"Maximum number of issues" default:"50"`
	After    string `help:"Pagination cursor"`
}

type IssueViewCmd struct {
	IssueID          string `arg:"" name:"issue-id" help:"Issue ID"`
	Comments         bool   `help:"Include comments"`
	CommentsLimit    int    `name:"comments-limit" help:"Maximum number of comments" default:"20"`
	Uploads          bool   `help:"Include uploads"`
	UploadsLimit     int    `name:"uploads-limit" help:"Maximum number of uploads/comments to scan" default:"50"`
}

type IssueCreateCmd struct {
	Team        string `help:"Team key or ID"`
	Title       string `help:"Issue title"`
	Description string `help:"Issue description or '-' for stdin"`
	Assignee    string `help:"Assignee (me, id, or email)"`
	State       string `help:"Workflow state name or ID"`
	Priority    int    `help:"Priority (0-4)" default:"-1"`
	Project     string `help:"Project name or ID"`
	Cycle       string `help:"Cycle ID or 'current'"`
	Labels      string `help:"Comma-separated label names or IDs"`
	Blocks      string `help:"Comma-separated issue IDs or keys this issue blocks"`
	BlockedBy   string `name:"blocked-by" help:"Comma-separated issue IDs or keys blocking this issue"`
}

type IssueUpdateCmd struct {
	IssueID         string `arg:"" name:"issue-id" help:"Issue ID"`
	Team            string `help:"Team key or ID"`
	Title           string `help:"Issue title"`
	Description     string `help:"Issue description or '-' for stdin"`
	Assignee        string `help:"Assignee (me, id, or email)"`
	State           string `help:"Workflow state name or ID"`
	Priority        int    `help:"Priority (0-4)" default:"-1"`
	Project         string `help:"Project name or ID"`
	Cycle           string `help:"Cycle ID or 'current'"`
	Labels          string `help:"Comma-separated label names or IDs"`
	Blocks          string `help:"Comma-separated issue IDs or keys this issue blocks"`
	BlockedBy       string `name:"blocked-by" help:"Comma-separated issue IDs or keys blocking this issue"`
	RemoveBlocks    string `name:"remove-blocks" help:"Comma-separated issue IDs or keys to remove from blocks"`
	RemoveBlockedBy string `name:"remove-blocked-by" help:"Comma-separated issue IDs or keys to remove from blocked-by"`
}

type IssueCloseCmd struct {
	IssueID string `arg:"" name:"issue-id" help:"Issue ID"`
}

type IssueReopenCmd struct {
	IssueID string `arg:"" name:"issue-id" help:"Issue ID"`
}

type IssueCommentCmd struct {
	IssueID string `arg:"" name:"issue-id" help:"Issue ID"`
	Body    string `help:"Comment body or '-' for stdin"`
}

func (c *IssueListCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}

	filter := linear.IssueFilter{}
	if c.Team != "" {
		teamID, resolveErr := client.ResolveTeamID(ctx, c.Team)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		filter.TeamID = teamID
	}
	if c.Assignee != "" {
		assigneeID, resolveErr := client.ResolveUserID(ctx, c.Assignee)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		filter.AssigneeID = assigneeID
	}
	if c.State != "" {
		if filter.TeamID == "" && looksLikeID(c.State) {
			filter.StateID = c.State
		} else {
			if filter.TeamID == "" {
				return exitError(2, errors.New("--state requires --team to resolve state name"))
			}
			stateID, resolveErr := client.ResolveStateID(ctx, filter.TeamID, c.State)
			if resolveErr != nil {
				return exitError(mapErrorToExitCode(resolveErr), resolveErr)
			}
			filter.StateID = stateID
		}
	}
	if c.Labels != "" {
		labels, resolveErr := client.ResolveLabelIDs(ctx, splitComma(c.Labels))
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		filter.LabelIDs = labels
	}
	if c.Project != "" {
		projectID, resolveErr := client.ResolveProjectID(ctx, c.Project)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		filter.ProjectID = projectID
	}
	if c.Cycle != "" {
		if filter.TeamID == "" && looksLikeID(c.Cycle) {
			filter.CycleID = c.Cycle
		} else {
			if filter.TeamID == "" {
				return exitError(2, errors.New("--cycle requires --team to resolve 'current'"))
			}
			cycleID, resolveErr := client.ResolveCycleID(ctx, filter.TeamID, c.Cycle)
			if resolveErr != nil {
				return exitError(mapErrorToExitCode(resolveErr), resolveErr)
			}
			filter.CycleID = cycleID
		}
	}
	if c.Search != "" {
		filter.Search = c.Search
	}
	if c.Priority >= 0 {
		filter.Priority = &c.Priority
	}

	page, err := client.Issues(ctx, filter, c.Limit, c.After)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(page)
	}
	rows := make([][]string, 0, len(page.Nodes))
	for _, issue := range page.Nodes {
		rows = append(rows, []string{issue.Identifier, issue.Title, issue.State, issue.Assignee, issue.TeamKey, issue.Cycle})
	}
	return out.PrintTable([]string{"ID", "Title", "State", "Assignee", "Team", "Cycle"}, rows)
}

func (c *IssueViewCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}
	issue, err := client.Issue(ctx, c.IssueID)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	if c.Comments {
		comments, err := client.IssueComments(ctx, issue.ID, c.CommentsLimit)
		if err != nil {
			return exitError(mapErrorToExitCode(err), err)
		}
		issue.Comments = comments
	}
	if c.Uploads {
		uploads, err := client.IssueUploads(ctx, issue.ID, c.UploadsLimit)
		if err != nil {
			return exitError(mapErrorToExitCode(err), err)
		}
		issue.Uploads = uploads
	}
	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(issue)
	}
	rows := [][]string{{
		issue.Identifier,
		issue.Title,
		issue.State,
		issue.Assignee,
		issue.TeamKey,
		issue.Cycle,
		issue.Project,
		fmt.Sprintf("%d", issue.Priority),
	}}
	if err := out.PrintTable([]string{"ID", "Title", "State", "Assignee", "Team", "Cycle", "Project", "Priority"}, rows); err != nil {
		return err
	}
	if issue.URL != "" {
		_, _ = fmt.Fprintf(cmdCtx.deps.Out, "\nURL: %s\n", issue.URL)
	}
	if len(issue.Labels) > 0 {
		_, _ = fmt.Fprintf(cmdCtx.deps.Out, "Labels: %s\n", strings.Join(issue.Labels, ", "))
	}
	if issue.Description != "" {
		_, _ = fmt.Fprintf(cmdCtx.deps.Out, "\nDescription:\n%s\n", issue.Description)
	}
	if c.Uploads {
		if len(issue.Uploads) == 0 {
			_, _ = fmt.Fprintln(cmdCtx.deps.Out, "\nUploads: none")
		} else {
			_, _ = fmt.Fprintln(cmdCtx.deps.Out, "\nUploads:")
			for _, attachment := range issue.Uploads {
				name := attachment.Title
				if name == "" {
					name = attachment.FileName
				}
				if name == "" {
					name = attachment.URL
				}
				if name == "" {
					name = attachment.ID
				}
				if attachment.URL != "" && attachment.URL != name {
					_, _ = fmt.Fprintf(cmdCtx.deps.Out, "- %s (%s)\n", name, attachment.URL)
				} else {
					_, _ = fmt.Fprintf(cmdCtx.deps.Out, "- %s\n", name)
				}
			}
		}
	}
	if issue.CreatedAt != "" || issue.UpdatedAt != "" {
		_, _ = fmt.Fprintf(cmdCtx.deps.Out, "\nCreated: %s\nUpdated: %s\n", issue.CreatedAt, issue.UpdatedAt)
	}
	if c.Comments && len(issue.Comments) > 0 {
		_, _ = fmt.Fprintln(cmdCtx.deps.Out, "\nComments:")
		for _, comment := range issue.Comments {
			author := comment.UserName
			if author == "" {
				author = comment.UserEmail
			}
			body := comment.Body
			if body == "" {
				body = comment.BodyData
			}
			if author != "" {
				_, _ = fmt.Fprintf(cmdCtx.deps.Out, "- %s (%s): %s\n", author, comment.CreatedAt, body)
			} else {
				_, _ = fmt.Fprintf(cmdCtx.deps.Out, "- %s: %s\n", comment.CreatedAt, body)
			}
		}
	}
	return nil
}

func (c *IssueCreateCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	if c.Team == "" {
		return exitError(2, errors.New("--team is required"))
	}
	if c.Title == "" {
		return exitError(2, errors.New("--title is required"))
	}

	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}

	teamID, err := client.ResolveTeamID(ctx, c.Team)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	input := map[string]any{
		"teamId": teamID,
		"title":  c.Title,
	}

	description, err := readOptionalBody(c.Description, cmdCtx.deps.In)
	if err != nil {
		return exitError(1, err)
	}
	if description != "" {
		input["description"] = description
	}

	if c.Assignee != "" {
		assigneeID, resolveErr := client.ResolveUserID(ctx, c.Assignee)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		input["assigneeId"] = assigneeID
	}
	if c.State != "" {
		stateID, resolveErr := client.ResolveStateID(ctx, teamID, c.State)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		input["stateId"] = stateID
	}
	if c.Priority >= 0 {
		input["priority"] = c.Priority
	}
	if c.Project != "" {
		projectID, resolveErr := client.ResolveProjectID(ctx, c.Project)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		input["projectId"] = projectID
	}
	if c.Cycle != "" {
		cycleID, resolveErr := client.ResolveCycleID(ctx, teamID, c.Cycle)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		input["cycleId"] = cycleID
	}
	if c.Labels != "" {
		labelIDs, resolveErr := client.ResolveLabelIDs(ctx, splitComma(c.Labels))
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		input["labelIds"] = labelIDs
	}

	issue, err := client.IssueCreate(ctx, input)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	if err := applyIssueRelations(ctx, client, issue.ID, issueRelationFlags{
		Blocks:    c.Blocks,
		BlockedBy: c.BlockedBy,
	}, false); err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(issue)
	}
	rows := [][]string{{issue.Identifier, issue.Title, issue.URL}}
	return out.PrintTable([]string{"ID", "Title", "URL"}, rows)
}

func (c *IssueUpdateCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}
	issueID, err := client.ResolveIssueID(ctx, c.IssueID)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	input := map[string]any{"id": issueID}
	teamID := ""
	if c.Team != "" {
		teamID, err = client.ResolveTeamID(ctx, c.Team)
		if err != nil {
			return exitError(mapErrorToExitCode(err), err)
		}
	}

	if c.Title != "" {
		input["title"] = c.Title
	}

	description, err := readOptionalBody(c.Description, cmdCtx.deps.In)
	if err != nil {
		return exitError(1, err)
	}
	if description != "" {
		input["description"] = description
	}

	if c.Assignee != "" {
		assigneeID, resolveErr := client.ResolveUserID(ctx, c.Assignee)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		input["assigneeId"] = assigneeID
	}

	if c.State != "" {
		if teamID == "" {
			issueResp, resolveErr := client.Issue(ctx, c.IssueID)
			if resolveErr != nil {
				return exitError(mapErrorToExitCode(resolveErr), resolveErr)
			}
			teamID = issueResp.TeamID
		}
		stateID, resolveErr := client.ResolveStateID(ctx, teamID, c.State)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		input["stateId"] = stateID
	}
	if c.Priority >= 0 {
		input["priority"] = c.Priority
	}
	if c.Project != "" {
		projectID, resolveErr := client.ResolveProjectID(ctx, c.Project)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		input["projectId"] = projectID
	}
	if c.Cycle != "" {
		if teamID == "" {
			issueResp, resolveErr := client.Issue(ctx, c.IssueID)
			if resolveErr != nil {
				return exitError(mapErrorToExitCode(resolveErr), resolveErr)
			}
			teamID = issueResp.TeamID
		}
		cycleID, resolveErr := client.ResolveCycleID(ctx, teamID, c.Cycle)
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		input["cycleId"] = cycleID
	}
	if c.Labels != "" {
		labelIDs, resolveErr := client.ResolveLabelIDs(ctx, splitComma(c.Labels))
		if resolveErr != nil {
			return exitError(mapErrorToExitCode(resolveErr), resolveErr)
		}
		input["labelIds"] = labelIDs
	}

	issue, err := client.IssueUpdate(ctx, input)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	if err := applyIssueRelations(ctx, client, issueID, issueRelationFlags{
		Blocks:          c.Blocks,
		BlockedBy:       c.BlockedBy,
		RemoveBlocks:    c.RemoveBlocks,
		RemoveBlockedBy: c.RemoveBlockedBy,
	}, true); err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(issue)
	}
	rows := [][]string{{issue.Identifier, issue.Title, issue.URL}}
	return out.PrintTable([]string{"ID", "Title", "URL"}, rows)
}

func (c *IssueCloseCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	return issueSetState(ctx, cmdCtx, c.IssueID, "completed")
}

func (c *IssueReopenCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	return issueSetState(ctx, cmdCtx, c.IssueID, "unstarted")
}

func issueSetState(ctx context.Context, cmdCtx *commandContext, issueRef string, stateType string) error {
	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}
	issue, err := client.Issue(ctx, issueRef)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}
	states, err := client.WorkflowStates(ctx, issue.TeamID)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}
	stateID := ""
	for _, state := range states {
		if strings.EqualFold(state.Type, stateType) {
			stateID = state.ID
			break
		}
	}
	if stateID == "" {
		return exitError(4, fmt.Errorf("no workflow state of type %s", stateType))
	}
	updated, err := client.IssueUpdate(ctx, map[string]any{"id": issue.ID, "stateId": stateID})
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}
	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(updated)
	}
	rows := [][]string{{updated.Identifier, updated.Title, updated.URL}}
	return out.PrintTable([]string{"ID", "Title", "URL"}, rows)
}

func (c *IssueCommentCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}
	issueID, err := client.ResolveIssueID(ctx, c.IssueID)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}
	text, err := readOptionalBody(c.Body, cmdCtx.deps.In)
	if err != nil {
		return exitError(1, err)
	}
	if strings.TrimSpace(text) == "" {
		return exitError(2, errors.New("comment body is required"))
	}
	commentID, err := client.IssueComment(ctx, issueID, text)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}
	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(map[string]string{"id": commentID})
	}
	_, _ = fmt.Fprintf(cmdCtx.deps.Out, "Comment added: %s\n", commentID)
	return nil
}

type issueRelationFlags struct {
	Blocks          string
	BlockedBy       string
	RemoveBlocks    string
	RemoveBlockedBy string
}

func splitComma(input string) []string {
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func readOptionalBody(flagValue string, r io.Reader) (string, error) {
	if flagValue == "" {
		return "", nil
	}
	if flagValue != "-" {
		return flagValue, nil
	}
	reader := bufio.NewReader(r)
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return string(data), nil
}

func looksLikeID(value string) bool {
	if len(value) < 30 {
		return false
	}
	return strings.Count(value, "-") >= 4
}

func applyIssueRelations(ctx context.Context, client linear.API, issueID string, flags issueRelationFlags, fetchExisting bool) error {
	addBlocks := uniqueStrings(splitComma(flags.Blocks))
	addBlockedBy := uniqueStrings(splitComma(flags.BlockedBy))
	removeBlocks := uniqueStrings(splitComma(flags.RemoveBlocks))
	removeBlockedBy := uniqueStrings(splitComma(flags.RemoveBlockedBy))

	if len(addBlocks) == 0 && len(addBlockedBy) == 0 && len(removeBlocks) == 0 && len(removeBlockedBy) == 0 {
		return nil
	}

	outgoing := map[string]string{}
	incoming := map[string]string{}
	if fetchExisting {
		existing, err := client.IssueRelations(ctx, issueID, 200)
		if err != nil {
			return err
		}
		for _, rel := range existing.Relations {
			if strings.EqualFold(rel.Type, "blocks") {
				outgoing[rel.RelatedIssueID] = rel.ID
			}
		}
		for _, rel := range existing.InverseRelations {
			if strings.EqualFold(rel.Type, "blocks") {
				incoming[rel.IssueID] = rel.ID
			}
		}
	}

	resolveID := func(ref string) (string, error) {
		id, err := client.ResolveIssueID(ctx, ref)
		if err != nil {
			return "", err
		}
		if id == issueID {
			return "", fmt.Errorf("cannot relate issue to itself")
		}
		return id, nil
	}

	for _, ref := range removeBlocks {
		targetID, err := resolveID(ref)
		if err != nil {
			return err
		}
		if relationID := outgoing[targetID]; relationID != "" {
			if err := client.IssueRelationDelete(ctx, relationID); err != nil {
				return err
			}
		}
	}
	for _, ref := range removeBlockedBy {
		targetID, err := resolveID(ref)
		if err != nil {
			return err
		}
		if relationID := incoming[targetID]; relationID != "" {
			if err := client.IssueRelationDelete(ctx, relationID); err != nil {
				return err
			}
		}
	}

	for _, ref := range addBlocks {
		targetID, err := resolveID(ref)
		if err != nil {
			return err
		}
		if outgoing[targetID] != "" {
			continue
		}
		if _, err := client.IssueRelationCreate(ctx, issueID, targetID, "blocks"); err != nil {
			return err
		}
	}
	for _, ref := range addBlockedBy {
		targetID, err := resolveID(ref)
		if err != nil {
			return err
		}
		if incoming[targetID] != "" {
			continue
		}
		if _, err := client.IssueRelationCreate(ctx, targetID, issueID, "blocks"); err != nil {
			return err
		}
	}

	return nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
