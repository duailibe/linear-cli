package cli

import "context"

type TeamCmd struct {
	List TeamListCmd `cmd:"" help:"List teams"`
}

type TeamListCmd struct{}

func (c *TeamListCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}
	teams, err := client.Teams(ctx)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}
	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(teams)
	}
	rows := make([][]string, 0, len(teams))
	for _, team := range teams {
		rows = append(rows, []string{team.ID, team.Key, team.Name})
	}
	return out.PrintTable([]string{"ID", "Key", "Name"}, rows)
}
