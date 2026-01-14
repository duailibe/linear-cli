package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Store a Linear API key"`
	Status AuthStatusCmd `cmd:"" help:"Show authentication status"`
	Logout AuthLogoutCmd `cmd:"" help:"Remove stored authentication"`
}

type AuthLoginCmd struct{}

type AuthStatusCmd struct{}

type AuthLogoutCmd struct{}

func (c *AuthLoginCmd) Run(ctx *commandContext) error {
	apiKey := ctx.global.APIKey
	if apiKey == "" {
		if ctx.global.NoInput {
			return exitError(2, errors.New("API key required with --no-input"))
		}
		key, err := readAPIKey(ctx.deps.In)
		if err != nil {
			return exitError(1, err)
		}
		apiKey = key
	}

	if strings.TrimSpace(apiKey) == "" {
		return exitError(2, errors.New("API key cannot be empty"))
	}

	if ctx.deps.AuthStore == nil {
		return exitError(1, errors.New("no auth store configured"))
	}
	if err := ctx.deps.AuthStore.Save(strings.TrimSpace(apiKey), ctx.deps.Now()); err != nil {
		return exitError(1, err)
	}

	out := outputFor(ctx)
	if out.JSON {
		return out.PrintJSON(map[string]any{
			"saved": true,
			"path":  ctx.deps.AuthStore.Path,
		})
	}
	_, _ = fmt.Fprintf(ctx.deps.Out, "Saved API key to %s\n", ctx.deps.AuthStore.Path)
	return nil
}

func (c *AuthStatusCmd) Run(ctx *commandContext) error {
	key, source, err := ctx.resolveAPIKey()
	authed := err == nil && key != ""
	out := outputFor(ctx)
	if out.JSON {
		return out.PrintJSON(map[string]any{
			"authenticated": authed,
			"source":        source,
		})
	}
	if authed {
		_, _ = fmt.Fprintf(ctx.deps.Out, "Authenticated via %s\n", source)
		return nil
	}
	_, _ = fmt.Fprintln(ctx.deps.Out, "Not authenticated")
	return exitError(3, errors.New("no API key configured"))
}

func (c *AuthLogoutCmd) Run(ctx *commandContext) error {
	if ctx.deps.AuthStore == nil {
		return exitError(1, errors.New("no auth store configured"))
	}
	if err := ctx.deps.AuthStore.Delete(); err != nil {
		return exitError(1, err)
	}
	out := outputFor(ctx)
	if out.JSON {
		return out.PrintJSON(map[string]any{
			"deleted": true,
			"path":    ctx.deps.AuthStore.Path,
		})
	}
	_, _ = fmt.Fprintln(ctx.deps.Out, "Logged out")
	return nil
}

func readAPIKey(r io.Reader) (string, error) {
	if file, ok := r.(*os.File); ok {
		if term.IsTerminal(int(file.Fd())) {
			_, _ = fmt.Fprint(os.Stdout, "Linear API key: ")
			b, err := term.ReadPassword(int(file.Fd()))
			_, _ = fmt.Fprintln(os.Stdout)
			if err != nil {
				return "", fmt.Errorf("read API key: %w", err)
			}
			return string(b), nil
		}
	}
	reader := bufio.NewReader(r)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read API key: %w", err)
	}
	return strings.TrimSpace(line), nil
}
