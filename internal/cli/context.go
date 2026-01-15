package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/duailibe/linear-cli/internal/linear"
)

type commandContext struct {
	deps   Dependencies
	global *GlobalOptions
}

func (c *commandContext) resolveAPIKey() (string, string, error) {
	if c.global.APIKey != "" {
		return c.global.APIKey, "flag", nil
	}
	if env := os.Getenv("LINEAR_API_KEY"); env != "" {
		return env, "env", nil
	}
	if c.deps.AuthStore != nil {
		data, ok, err := c.deps.AuthStore.Load()
		if err != nil {
			return "", "", err
		}
		if ok && data.APIKey != "" {
			return data.APIKey, "file", nil
		}
	}
	return "", "", errors.New("no Linear API key found; run 'linear auth login' or set LINEAR_API_KEY")
}

func (c *commandContext) apiClient() (linear.API, error) {
	key, _, err := c.resolveAPIKey()
	if err != nil {
		return nil, err
	}
	if c.deps.NewClient == nil {
		return nil, fmt.Errorf("no API client configured")
	}
	return c.deps.NewClient(key, c.global.Timeout), nil
}
