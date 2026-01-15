# Architecture

This CLI is intentionally small and direct. The main flow is:

1. `cmd/linear/main.go` calls `cli.Execute()`.
2. Kong parses flags and subcommands in `internal/cli`.
3. Commands resolve user-friendly inputs (team keys, emails, state names).
4. The GraphQL client in `internal/linear` executes queries and mutations.
5. The output layer prints tables or JSON.

## Command wiring

- Commands are defined as Kong structs in `internal/cli/*_cmd.go`.
- Each command implements a `Run(...)` method.
- `commandContext` provides dependency injection for IO, auth store, and API client.

## Name resolution

Human-friendly values are resolved before requests are made:

- Team keys -> team IDs
- User emails or "me" -> user IDs
- State names -> workflow state IDs
- Labels/projects -> their IDs

This logic lives in `internal/linear` as `Resolve*` helpers.

## GraphQL client

The client in `internal/linear`:

- Uses a single HTTP client with a configurable timeout.
- Normalizes the auth token and attaches it to requests.
- Maps common HTTP failures to `ErrUnauthorized`, `ErrNotFound`, and `ErrRateLimited`.

## Output layer

Output is minimal and deterministic:

- `--json` prints structured data for scripting.
- Tables are plain tab-separated text via Go's tabwriter.
