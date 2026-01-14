package cli

import (
	"context"
	"errors"
)

type CycleCmd struct {
	List CycleListCmd `cmd:"" help:"List cycles for a team"`
	View CycleViewCmd `cmd:"" help:"View cycle details"`
}

type CycleListCmd struct {
	Team    string `help:"Team key or ID"`
	Current bool   `help:"Only show current/active cycles"`
	Limit   int    `help:"Maximum number of cycles to fetch" default:"20"`
	After   string `help:"Pagination cursor"`
}

type CycleViewCmd struct {
	CycleID string `arg:"" name:"cycle-id" help:"Cycle ID"`
}

func (c *CycleListCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	if c.Team == "" {
		return exitError(2, errors.New("--team is required"))
	}
	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}

	teamID, err := client.ResolveTeamID(ctx, c.Team)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	page, err := client.Cycles(ctx, teamID, c.Current, c.Limit, c.After)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(page)
	}

	rows := make([][]string, 0, len(page.Nodes))
	for _, cycle := range page.Nodes {
		rows = append(rows, []string{cycle.ID, cycle.Name, cycle.Number, cycle.StartsAt, cycle.EndsAt, cycle.IsActive})
	}
	return out.PrintTable([]string{"ID", "Name", "Number", "Starts", "Ends", "Active"}, rows)
}

func (c *CycleViewCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}

	cycle, err := client.Cycle(ctx, c.CycleID)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(cycle)
	}

	rows := [][]string{{cycle.ID, cycle.Name, cycle.Number, cycle.StartsAt, cycle.EndsAt, cycle.IsActive}}
	return out.PrintTable([]string{"ID", "Name", "Number", "Starts", "Ends", "Active"}, rows)
}
