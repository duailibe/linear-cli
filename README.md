# linear-cli

A fast, no-nonsense CLI for [Linear](https://linear.app). Manage issues, cycles, and teams without leaving your terminal.

Built in Go. Works with personal API keys. Plays nice with scripts.

## Install

```bash
go install github.com/duailibe/linear-cli/cmd/linear@latest
```

Or clone and build:

```bash
git clone https://github.com/duailibe/linear-cli
cd linear-cli
make build
```

The binary is written to `bin/linear` by default.

## Authentication

The CLI looks for credentials in this order:

1. `--api-key`
2. `LINEAR_API_KEY`
3. Stored credentials from `linear auth login`

Recommended: export an environment variable in your shell profile:

```bash
export LINEAR_API_KEY=lin_api_...
```

Or store it locally:

```bash
linear auth login
```

Check which API key source is configured (use `whoami` to verify):

```bash
linear auth status
linear whoami
```

Credentials are stored at `~/.local/share/linear/auth.json` (respects `$XDG_DATA_HOME`).

## What you can do

```
linear issue list        List and filter issues
linear issue view        View issue details, comments, and uploads
linear issue create      Create new issues
linear issue update      Update existing issues
linear issue close       Close an issue
linear issue reopen      Reopen a closed issue
linear issue comment     Add comments
linear issue uploads     Download uploads

linear cycle list        List team cycles
linear cycle view        View cycle details

linear team list         List teams

linear auth login        Store API key
linear auth status       Show API key configuration
linear auth logout       Remove stored credentials

linear whoami            Show current user
```

## Examples

**List issues in the current cycle:**

```bash
linear issue list --team ENG --cycle current
```

**Create an issue with dependencies:**

```bash
linear issue create --team ENG --title "Ship new auth" \
  --priority 2 --blocked-by ENG-101 --blocks ENG-220
```

**Pipe a description from a file:**

```bash
cat spec.md | linear issue create --team ENG --title "New feature" --description -
```

**Add a quick comment:**

```bash
linear issue comment ENG-123 --body "Fixed in latest deploy"
```

**Download uploads from an issue:**

```bash
linear issue uploads ENG-456 --dir ./downloads
```

**Get JSON output for scripting:**

```bash
linear issue list --team ENG --json | jq '.nodes[].identifier'
```

## Usage reference

Issue references are passed to the `issue(id: ...)` GraphQL query. In most
workspaces this accepts either the issue ID or the issue identifier (for example
ENG-123). If you see "not found" errors, use the canonical ID from `--json`
output.

### Global flags

```
--json          Output JSON instead of tables
--no-color      Disable colored output
--quiet, -q     Suppress non-essential output
--verbose, -v   Enable verbose diagnostics
--no-input      Disable interactive prompts
--yes, -y       Auto-confirm prompts
--timeout       API request timeout (default 10s)
--api-key       API key (overrides env/stored auth)
--version       Print version and exit
```

Notes:

- `--no-input` is enforced in `linear auth login`; you must pass `--api-key` when it is set.
- `--no-color`, `--quiet`, `--verbose`, and `--yes` are currently accepted for forward
  compatibility, but not all commands change behavior yet.

### Auth

#### `linear auth login`

Store a Linear API key in the local auth file.

- Reads the key from `--api-key` if provided.
- Otherwise prompts on stdin (unless `--no-input` is set).

```bash
linear auth login
linear auth login --api-key "$LINEAR_API_KEY"
```

#### `linear auth status`

Show whether an API key is configured and report its source.
This does not verify the key; use `linear whoami` to validate access.

```bash
linear auth status
```

#### `linear auth logout`

Remove the stored API key.

```bash
linear auth logout
```

### Whoami

#### `linear whoami`

Print the authenticated Linear user (verifies API key access).

```bash
linear whoami
```

### Teams

#### `linear team list`

List teams you have access to.

```bash
linear team list
```

### Cycles

#### `linear cycle list`

List cycles for a team.

```
--team       Team key or ID (required)
--current    Only show current/active cycles
--limit      Maximum number of cycles to fetch (default 20)
--after      Pagination cursor
```

```bash
linear cycle list --team ENG --current
```

#### `linear cycle view`

View a cycle by ID.

```bash
linear cycle view <cycle-id>
```

### Issues

#### `linear issue list`

List issues with filters.

```
--team       Team key or ID
--assignee   Assignee (me, id, or email)
--state      Workflow state name or ID
--label      Comma-separated label names or IDs
--project    Project name or ID
--cycle      Cycle ID or 'current'
--search     Search issue titles
--priority   Priority (0-4)
--limit      Maximum number of issues (default 50)
--after      Pagination cursor
```

Notes:

- `--state` requires `--team` when using state names. If you pass a state ID,
  `--team` can be omitted.
- `--cycle current` requires `--team`.

```bash
linear issue list --team ENG --cycle current --assignee me
```

#### `linear issue view`

View issue details.

Argument:

```
<issue-id>    Issue ID or identifier
```

Flags:

```
--comments          Include comments
--comments-limit    Maximum number of comments (default 20)
--uploads           Include uploads
--uploads-limit     Maximum number of uploads/comments to scan (default 50)
```

```bash
linear issue view ENG-123 --comments
```

#### `linear issue create`

Create a new issue.

```
--team         Team key or ID (required)
--title        Issue title (required)
--description  Issue description or '-' for stdin
--assignee     Assignee (me, id, or email)
--state        Workflow state name or ID
--priority     Priority (0-4)
--project      Project name or ID
--cycle        Cycle ID or 'current'
--labels       Comma-separated label names or IDs
--blocks       Comma-separated issue IDs or keys this issue blocks
--blocked-by   Comma-separated issue IDs or keys blocking this issue
```

```bash
linear issue create --team ENG --title "Bug in auth" --priority 1
```

#### `linear issue update`

Update an existing issue.

```
<issue-id>    Issue ID or identifier
--team        Team key or ID
--title       Issue title
--description Issue description or '-' for stdin
--assignee    Assignee (me, id, or email)
--state       Workflow state name or ID
--priority    Priority (0-4)
--project     Project name or ID
--cycle       Cycle ID or 'current'
--labels      Comma-separated label names or IDs
--blocks      Comma-separated issue IDs or keys this issue blocks
--blocked-by  Comma-separated issue IDs or keys blocking this issue
--remove-blocks       Comma-separated issue IDs or keys to remove from blocks
--remove-blocked-by   Comma-separated issue IDs or keys to remove from blocked-by
```

Notes:

- If you set `--state` or `--cycle` without `--team`, the CLI fetches the issue
  to determine the team before resolving names.

```bash
linear issue update ENG-123 --state "In Progress"
```

#### `linear issue close`

Set the issue to the workflow state of type `completed`.

```bash
linear issue close ENG-123
```

#### `linear issue reopen`

Set the issue to the workflow state of type `unstarted`.

```bash
linear issue reopen ENG-123
```

#### `linear issue comment`

Add a comment to an issue.

```
<issue-id>    Issue ID or identifier
--body        Comment body or '-' for stdin
```

```bash
linear issue comment ENG-123 --body "Working on this"
```

#### `linear issue uploads`

Download uploads from the issue description and comments (uploads.linear.app only).

```
<issue-id>    Issue ID or identifier
--dir         Directory to save uploads (default "uploads")
--limit       Maximum number of comments to scan (default 50)
--overwrite   Overwrite existing files
```

```bash
linear issue uploads ENG-123 --dir ./downloads
```

## Configuration

### API key resolution

The CLI looks for credentials in this order:

1. `--api-key`
2. `LINEAR_API_KEY`
3. Stored credentials from `linear auth login`

### Auth storage

Stored credentials live in:

- `$XDG_DATA_HOME/linear/auth.json` when XDG_DATA_HOME is set
- `~/.local/share/linear/auth.json` otherwise

The file is created with restrictive permissions.

### Request timeout

The `--timeout` flag accepts Go duration strings (for example `10s`, `1m`, `1m30s`).

```bash
linear --timeout 30s issue list --team ENG
```

### Non-interactive mode

`--no-input` disables prompts. For `linear auth login`, this means you must pass
`--api-key` explicitly or the command fails.

## Output and errors

### Output formats

By default, commands render human-readable tables to stdout. Pass `--json` to
return JSON suitable for scripts.

```bash
linear issue list --team ENG --json | jq '.nodes[].identifier'
linear issue view ENG-123 --json | jq '{id, title, state}'
```

### Table columns

The default table output includes these columns:

- `linear issue list`: ID, Title, State, Assignee, Team, Cycle
- `linear issue view`: ID, Title, State, Assignee, Team, Cycle, Project, Priority
- `linear issue create/update/close/reopen`: ID, Title, URL
- `linear issue comment`: prints a confirmation line with the new comment ID
- `linear issue uploads`: ID, Title, Path
- `linear cycle list/view`: ID, Name, Number, Starts, Ends, Active
- `linear team list`: ID, Key, Name
- `linear whoami`: ID, Name, Email

`linear issue view` prints additional lines for URL, labels, description, timestamps,
comments (when `--comments` is provided), and uploads (when `--uploads` is provided).

### JSON shapes

The JSON output mirrors the internal types in `internal/linear/types.go`. Common
shapes include:

- `IssuePage`: `{ nodes: [IssueSummary], page_info: { has_next_page, end_cursor } }`
- `CyclePage`: `{ nodes: [Cycle], page_info: { has_next_page, end_cursor } }`
- `IssueDetail`: detailed issue fields plus optional `comments` and `uploads`
- `User`, `Team`, and `Cycle` objects with straightforward scalar fields
- `IssueComment` creation returns `{ id: "..." }`

### Exit codes

Errors are printed to stderr and return non-zero exit codes. The most common
codes are:

- `0`: success
- `1`: general error
- `2`: usage error (invalid flags, missing required inputs)
- `3`: authentication error (missing/invalid API key)
- `4`: not found
- `5`: rate limited or temporarily unavailable

Use these codes when scripting the CLI in CI or automation.

## Architecture and development docs

Code-focused documentation lives in `docs/`:

- `docs/README.md`
- `docs/architecture.md`
- `docs/development.md`

## AI Coding Agents

This repo includes a SKILL file for AI coding agents. Install it with:

```bash
npx add-skill duailibe/linear-cli
```

## Development

```bash
go test ./...
```

## Smoke tests

Smoke tests run the CLI against a playground workspace. Each run creates a new
issue and updates it to validate the update flow.

Required environment variables:

- `LINEAR_API_KEY` (playground API key)
- `LINEAR_SMOKE_TEAM` (team key, for example `DUA`)

Optional environment variables:


Run:

```bash
scripts/smoke_test.sh
```

## License

MIT
