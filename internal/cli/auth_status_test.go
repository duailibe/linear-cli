package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
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

	if payload["configured"] != true {
		t.Fatalf("expected configured true")
	}
	if payload["source"] != "env" {
		t.Fatalf("expected source env")
	}
}

func TestAuthStatusJSONNone(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	store := auth.NewStore(filepath.Join(t.TempDir(), "auth.json"))

	deps := Dependencies{
		In:        bytes.NewBuffer(nil),
		Out:       &out,
		Err:       &errOut,
		Now:       time.Now,
		AuthStore: store,
		NewClient: nil,
	}

	t.Setenv("LINEAR_API_KEY", "")

	code := ExecuteWith(deps, []string{"auth", "status", "--json"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", code, errOut.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}

	if payload["configured"] != false {
		t.Fatalf("expected configured false")
	}
	if payload["source"] != "none" {
		t.Fatalf("expected source none")
	}
}

func TestAuthStatusTextNone(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	store := auth.NewStore(filepath.Join(t.TempDir(), "auth.json"))

	deps := Dependencies{
		In:        bytes.NewBuffer(nil),
		Out:       &out,
		Err:       &errOut,
		Now:       time.Now,
		AuthStore: store,
		NewClient: nil,
	}

	t.Setenv("LINEAR_API_KEY", "")

	code := ExecuteWith(deps, []string{"auth", "status"})
	if code != 3 {
		t.Fatalf("expected exit 3, got %d (stderr: %s)", code, errOut.String())
	}
	if !strings.Contains(out.String(), "No API key configured") {
		t.Fatalf("expected stdout to mention no API key configured")
	}
	if !strings.Contains(errOut.String(), "no API key configured") {
		t.Fatalf("expected stderr to mention no api key configured")
	}
}
