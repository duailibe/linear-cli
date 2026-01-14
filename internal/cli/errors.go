package cli

import (
	"errors"

	"github.com/duailibe/linear-cli/internal/linear"
)

func mapErrorToExitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, linear.ErrUnauthorized) {
		return 3
	}
	if errors.Is(err, linear.ErrNotFound) {
		return 4
	}
	if errors.Is(err, linear.ErrRateLimited) {
		return 5
	}
	return 1
}
