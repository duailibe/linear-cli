# Architecture Spec

This document describes how the CLI is wired, what each package owns, and the
behavioral contract of the commands and API layer as implemented in the code.

## Overview

Runtime flow:

1. `cmd/linear/main.go` calls `cli.Execute()`.
2. `cli.Execute()` delegates to `Run()`/`ExecuteWith()` which build dependencies
   (IO, clock, auth store, API client).
3. Kong parses flags and subcommands into `internal/cli` command structs.
4. Kong binds a `context.Context` and a `commandContext` (dependencies + global options).
5. Command `Run(...)` methods resolve inputs, call the `internal/linear` API, and
   render output.

## Package map

- `cmd/linear/`: entry point (`main.go`).
- `internal/cli/`: CLI command structs, Kong wiring, output formatting, error
  handling, and version output.
- `internal/linear/`: GraphQL client, queries/mutations, ID resolution, and
  CLI-friendly shapes.
- `internal/auth/`: file-based auth store (XDG-aware).

## CLI lifecycle and dependency injection

- `Execute()` uses `os.Args[1:]`, `os.Stdin`, `os.Stdout`, and `os.Stderr`.
- `Run()` builds `Dependencies`:
  - `AuthStore` from `auth.DefaultStorePath()`
  - `NewClient` as `linear.NewClient`
  - `Now` as `time.Now`
- `ExecuteWith()` creates the Kong parser with name/description/version and
  binds:
  - `context.Context` for command `Run(ctx, ...)` signatures
  - `commandContext` for dependency access and global options
- Kong exit is handled via a panic/unwrap mechanism so it can be converted into
  process exit codes.

## Global options and configuration

Global flags (`internal/cli/types.go`):

- `--json`: output JSON instead of tables
- `--no-color`: parsed but currently unused (no color output exists)
- `-q, --quiet`: parsed but currently unused
- `-v, --verbose`: parsed but currently unused
- `--no-input`: disable interactive prompts
- `-y, --yes`: parsed but currently unused
- `--timeout`: API timeout (default `10s`)
- `--api-key`: explicit API key (overrides env and stored auth)

## Auth resolution and storage

Resolution order (`commandContext.resolveAPIKey()`):

1. `--api-key`
2. `LINEAR_API_KEY` environment variable
3. Stored auth file

Auth file location (`internal/auth/store.go`):

- `$XDG_DATA_HOME/linear/auth.json` if `XDG_DATA_HOME` is set
- otherwise `~/.local/share/linear/auth.json`

File format:

```json
{
  "api_key": "<token>",
  "saved_at": "2025-01-01T00:00:00Z"
}
```

File permissions:

- directory: `0700`
- file: `0600`

Auth commands:

- `auth login` prompts for a key (hidden input on TTY) unless `--api-key` is set.
  With `--no-input`, the key must be provided via `--api-key`.
- `auth status` prints whether a key is configured and returns exit code `3`
  when no key is available.
- `auth logout` deletes the auth file.

## Output layer

- JSON output uses `json.Encoder` with two-space indentation.
- Table output uses `text/tabwriter` with a 2-space column padding.
- Commands choose their own column sets; JSON output returns the underlying
  struct or map as-is.

## Error handling and exit codes

General:

- Kong parse errors are wrapped as exit code `2`.
- Non-`ExitError` failures return exit code `1`.

API error mapping (`mapErrorToExitCode`):

- `ErrUnauthorized` -> `3`
- `ErrNotFound` -> `4`
- `ErrRateLimited` -> `5`

Command-level validation:

- Missing required flags or invalid input generally return exit code `2`.
- `issue close` / `issue reopen` return exit code `4` when the required workflow
  state type is missing.

## Commands and behaviors

### Auth

- `auth login`: saves a trimmed API key to the auth store.
- `auth status`: reports source (`flag`, `env`, `file`, or `none`) and exits `3`
  when not configured.
- `auth logout`: removes the stored auth file.

### Whoami

- Uses `linear.API.Me()` to fetch the current user.
- Output columns: `ID`, `Name`, `Email`.

### Team

- `team list`: lists teams from the API.
- Output columns: `ID`, `Key`, `Name`.

### Cycle

- `cycle list`: requires `--team` (key or ID). Optional `--current` filters to
  active cycles. Supports `--limit` and `--after` for pagination.
- `cycle view`: expects a cycle ID.
- Output columns: `ID`, `Name`, `Number`, `Starts`, `Ends`, `Active`.

### Issue

#### issue list

Filters:

- `--team` (key or ID) sets `teamId`.
- `--assignee` accepts `me`, a user ID, or an email.
- `--state` requires `--team` unless the value looks like an ID.
- `--cycle` requires `--team` unless the value looks like an ID.
- `--label` accepts comma-separated names or IDs.
- `--project` accepts name or ID.
- `--search` matches issue titles (`contains`).
- `--priority` sets priority when >= 0 (default is `-1`, meaning unset).

Output columns: `ID`, `Title`, `State`, `Assignee`, `Team`, `Cycle`.

#### issue view

- Fetches a single issue with labels, project, and timestamps.
- `--comments` optionally fetches comments; `--comments-limit` defaults to 20.
- Human output prints a summary table, then URL, labels, description, and
  timestamps when present.

#### issue create

- Requires `--team` and `--title`.
- `--description` accepts `-` to read from stdin.
- Resolves team, assignee, state, project, cycle, and labels before creation.
- Applies relation flags:
  - `--blocks`
  - `--blocked-by`
- Output columns: `ID`, `Title`, `URL`.

#### issue update

- Accepts an issue ID or identifier; resolves to a canonical issue ID.
- `--team` is optional; if omitted and a state/cycle name is provided, the issue
  is fetched to determine its team.
- `--description` accepts `-` to read from stdin.
- Relation flags:
  - `--blocks`, `--blocked-by`
  - `--remove-blocks`, `--remove-blocked-by`
- Output columns: `ID`, `Title`, `URL`.

#### issue close / issue reopen

- Transitions the issue to the workflow state type `completed` or `unstarted`.
- Finds the state by scanning the team’s workflow states; errors if no match.

#### issue comment

- `--body` accepts `-` to read from stdin; body is required.
- Returns the new comment ID (JSON) or prints a confirmation line.

#### issue attachments

- Requests attachments for the issue and downloads them to a directory
  (default: `attachments`).
- Also parses `uploads.linear.app` links in the issue description.
- If no attachments are returned by the API, it also parses
  `uploads.linear.app` links in comments.
- File naming:
  - prefers attachment filename/title/url path
  - sanitizes path separators and colons
  - uses `-1`, `-2`, ... suffixes unless `--overwrite` is set
- Downloads use a temp file and atomic rename. Authorization is only sent to
  hosts ending in `linear.app`.
- Output columns: `ID`, `Title`, `Path`.

## Linear API client

### HTTP and GraphQL

- Default API endpoint: `https://api.linear.app/graphql`.
- A single `http.Client` is created with the CLI timeout.
- Requests are JSON-encoded GraphQL payloads with `query` + `variables`.
- `Authorization` header uses a normalized token:
  - trims whitespace
  - strips a leading `Bearer ` prefix if present

### Error mapping

- HTTP `401`/`403` -> `ErrUnauthorized`
- HTTP `429`/`503`/`504` -> `ErrRateLimited`
- GraphQL errors are aggregated as a single error.
- `ErrNotFound` is returned when expected nodes are missing or null.

### Query fallbacks

- `Me()` tries `viewer`, and falls back to `me` if `viewer` is unsupported.
- `Cycles()` uses the top-level `cycles` query, and falls back to `team.cycles`
  when the `cycles` field is unavailable.

### ID and name resolution

- Team: accepts ID or key.
- User: accepts `me`, ID, or email.
- State: accepts ID or resolves by name within a team.
- Label: accepts ID or resolves by exact name (one-by-one).
- Project: accepts ID or resolves by exact name.
- Cycle: accepts ID or `current` (resolved to the active cycle for the team).
- Issue: `ResolveIssueID` queries `issue(id: $value)` and returns the resulting ID.

### Pagination

- `issue list` and `cycle list` return a `page_info` object with `has_next_page`
  and `end_cursor`.
- The CLI does not auto-paginate; users must pass `--after` manually.
