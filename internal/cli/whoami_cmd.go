package cli

import "context"

type WhoamiCmd struct{}

func (c *WhoamiCmd) Run(ctx context.Context, cmdCtx *commandContext) error {
	client, err := cmdCtx.apiClient()
	if err != nil {
		return exitError(3, err)
	}
	user, err := client.Me(ctx)
	if err != nil {
		return exitError(mapErrorToExitCode(err), err)
	}

	out := outputFor(cmdCtx)
	if out.JSON {
		return out.PrintJSON(user)
	}

	return out.PrintTable([]string{"ID", "Name", "Email"}, [][]string{{user.ID, user.Name, user.Email}})
}
