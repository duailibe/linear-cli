package cli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/duailibe/linear-cli/internal/auth"
)

func TestAuthStatusJSONEnv(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	store := auth.NewStore("/tmp/does-not-exist")

	deps := Dependencies{
		In:        bytes.NewBuffer(nil),
		Out:       &out,
		Err:       &errOut,
		Now:       time.Now,
		AuthStore: store,
		NewClient: nil,
	}

	// Use env key.
	t.Setenv("LINEAR_API_KEY", "env-key")

	code := ExecuteWith(deps, []string{"auth", "status", "--json"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", code, errOut.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}

	if payload["authenticated"] != true {
		t.Fatalf("expected authenticated true")
	}
	if payload["source"] != "env" {
		t.Fatalf("expected source env")
	}
}
