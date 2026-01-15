package cli

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/duailibe/linear-cli/internal/auth"
	"github.com/duailibe/linear-cli/internal/linear"
)

type Dependencies struct {
	In        io.Reader
	Out       io.Writer
	Err       io.Writer
	Now       func() time.Time
	AuthStore *auth.Store
	NewClient func(token string, timeout time.Duration) linear.API
}

type GlobalOptions struct {
	JSON    bool          `help:"output JSON"`
	NoColor bool          `name:"no-color" help:"disable color output"`
	Quiet   bool          `short:"q" help:"suppress non-essential output"`
	Verbose bool          `short:"v" help:"enable verbose diagnostics"`
	NoInput bool          `name:"no-input" help:"disable interactive prompts"`
	Yes     bool          `short:"y" help:"assume yes for confirmations"`
	Timeout time.Duration `help:"API request timeout" default:"10s"`
	APIKey  string        `name:"api-key" help:"Linear API key (overrides env and stored auth)"`
}

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit %d", e.Code)
	}
	return e.Err.Error()
}

func exitError(code int, err error) error {
	if err == nil {
		return ExitError{Code: code, Err: errors.New("unknown error")}
	}
	return ExitError{Code: code, Err: err}
}
