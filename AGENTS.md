# AGENTS.md

Project overview
- This repository is a Go command-line client for Linear built on their GraphQL API.
- The CLI entry point is `cmd/linear/main.go`, which calls `cli.Execute()`.
- The CLI uses Kong for command/flag parsing and a small output layer for tables and JSON.

Code layout and responsibilities
- `cmd/linear/main.go`: entry point, calls `cli.Execute()`.
- `internal/cli/`: command definitions (auth, issue, cycle, team, whoami), output formatting, context wiring, and error handling.
- `internal/linear/`: GraphQL client, queries/mutations. Owns API shapes and CLI-friendly struct translations.
- `internal/auth/`: API key storage and retrieval (file-based, XDG-compliant).

Key patterns
- Commands live in `internal/cli/*_cmd.go` files as Kong command structs.
- Commands implement `Run(...)` methods and receive `*commandContext` (and `context.Context` when needed).
- Use `exitError(code, err)` for error returns with specific exit codes.
- Resolve human-friendly names (team keys, state names) to IDs via `client.Resolve*` methods.

Versioning
- Build metadata lives in `internal/cli/version.go` (version/commit/date).
- `make` / `make build` injects version info via ldflags (`VERSION`, `COMMIT`, `DATE` in `Makefile`).
- `linear --version` prints `linear version <version>` and optional commit/date lines.
- Use semantic versioning (e.g., `v0.1.0`).

Runtime behavior and configuration
- Global flags: `--json`, `--no-color`, `--quiet/-q`, `--verbose/-v`, `--no-input`, `--yes/-y`, `--timeout`, `--api-key`.
- Auth precedence: `--api-key` flag → `LINEAR_API_KEY` env → stored token (`~/.local/share/linear/auth.json`).
- Default API endpoint: `https://api.linear.app/graphql`.

Output expectations
- Human-readable tables by default; `--json` for machine output.
- Errors go to stderr with non-zero exit codes (see `internal/cli/errors.go`).

Testing
- Prefer `make test` (or `go test ./...`) after code changes.
- `make` defaults to `build`; use `make build` explicitly for clarity.
- Linting: `make lint` (golangci-lint).
- Test files follow `*_test.go` convention alongside source files.

Commits
- Use conventional commits: `type(scope): message` (e.g., `feat(issue): add search flag`, `fix(auth): handle expired tokens`).
- Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`.
- After any user-facing changes in the code, check if README.md, SKILL.md, and docs/ also need to be updated.
- Keep docs/ up to date when commands, flags, or outputs change.

Releasing
- Ensure notable changes are listed under `## Unreleased` in `CHANGELOG.md`.
- Run `scripts/release.sh vX.Y.Z` to cut a release (updates changelog, commits, tags, and pushes).
