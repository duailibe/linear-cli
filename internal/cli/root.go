package cli

import "github.com/alecthomas/kong"

type CLI struct {
	GlobalOptions `embed:""`

	Version kong.VersionFlag `help:"Print version and exit"`

	Auth   AuthCmd   `cmd:"" help:"Manage authentication"`
	Whoami WhoamiCmd `cmd:"" help:"Show current Linear user"`
	Issue  IssueCmd  `cmd:"" help:"Manage issues"`
	Cycle  CycleCmd  `cmd:"" help:"Manage cycles"`
	Team   TeamCmd   `cmd:"" help:"Manage teams"`
}

func outputFor(ctx *commandContext) output {
	return output{Out: ctx.deps.Out, JSON: ctx.global.JSON}
}
