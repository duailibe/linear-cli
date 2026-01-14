package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/alecthomas/kong"

	"github.com/duailibe/linear-cli/internal/auth"
	"github.com/duailibe/linear-cli/internal/linear"
)

func Execute() int {
	return Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}

func Run(args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	storePath, err := auth.DefaultStorePath()
	if err != nil {
		_, _ = errOut.Write([]byte(err.Error() + "\n"))
		return 1
	}

	deps := Dependencies{
		In:        in,
		Out:       out,
		Err:       errOut,
		Now:       time.Now,
		AuthStore: auth.NewStore(storePath),
		NewClient: linear.NewClient,
	}

	return ExecuteWith(deps, args)
}

func ExecuteWith(deps Dependencies, args []string) (code int) {
	cli := &CLI{}

	parser, err := kong.New(
		cli,
		kong.Name("linear"),
		kong.Description("Manage Linear issues and cycles from the terminal"),
		kong.Vars(kong.Vars{
			"version": VersionOutput(),
		}),
		kong.Writers(deps.Out, deps.Err),
		kong.Exit(func(code int) { panic(exitPanic{Code: code}) }),
	)
	if err != nil {
		_, _ = deps.Err.Write([]byte(err.Error() + "\n"))
		return 1
	}

	defer func() {
		if r := recover(); r != nil {
			if exit := parseExitPanic(r); exit != nil {
				code = exit.Code
				return
			}
			panic(r)
		}
	}()

	kctx, err := parser.Parse(args)
	if err != nil {
		return handleExit(deps, wrapParseError(err))
	}

	kctx.BindTo(context.Background(), (*context.Context)(nil))
	kctx.Bind(&commandContext{deps: deps, global: &cli.GlobalOptions})

	if err := kctx.Run(); err != nil {
		return handleExit(deps, err)
	}
	return 0
}

type exitPanic struct {
	Code int
}

func parseExitPanic(val any) *exitPanic {
	switch cast := val.(type) {
	case exitPanic:
		return &cast
	case *exitPanic:
		return cast
	default:
		return nil
	}
}

func wrapParseError(err error) error {
	if err == nil {
		return nil
	}
	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		return exitError(2, parseErr)
	}
	return err
}

func handleExit(deps Dependencies, err error) int {
	if err == nil {
		return 0
	}
	var exitErr ExitError
	if errors.As(err, &exitErr) {
		if exitErr.Err != nil {
			_, _ = deps.Err.Write([]byte(exitErr.Err.Error() + "\n"))
		}
		return exitErr.Code
	}
	_, _ = fmt.Fprintf(deps.Err, "%v\n", err)
	return 1
}
