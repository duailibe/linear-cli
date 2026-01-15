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
go build -o linear ./cmd/linear
```

## Authentication

Set your API key via environment (preferred):

```bash
export LINEAR_API_KEY=lin_api_...
```

Or store it locally:

```bash
linear auth login
```

Credentials are stored at `~/.local/share/linear/auth.json` (respects `$XDG_DATA_HOME`).

## What you can do

```
linear issue list       List and filter issues
linear issue view       View issue details and comments
linear issue create     Create new issues
linear issue update     Update existing issues
linear issue close      Close an issue
linear issue reopen     Reopen a closed issue
linear issue comment    Add comments
linear issue attachments Download attachments

linear cycle list       List team cycles
linear cycle view       View cycle details

linear team list        List teams

linear auth login       Store API key
linear auth status      Check auth status
linear auth logout      Remove stored credentials

linear whoami           Show current user
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

**Download all attachments from an issue:**

```bash
linear issue attachments ENG-456 --dir ./downloads
```

**Get JSON output for scripting:**

```bash
linear issue list --team ENG --json | jq '.nodes[].identifier'
```

## Output

- Human-readable tables by default
- `--json` for machine-parseable output
- `--quiet` to suppress non-essential messages
- `--no-color` if your terminal doesn't like colors

## Global flags

```
--json          Output JSON instead of tables
--quiet, -q     Suppress non-essential output
--verbose, -v   Enable verbose diagnostics
--no-color      Disable colored output
--no-input      Disable interactive prompts
--yes, -y       Auto-confirm prompts
--timeout       API request timeout (default 10s)
--api-url       Custom GraphQL endpoint
--api-key       API key (overrides env/stored auth)
```

## AI Coding Agents

This repo includes a SKILL file for AI coding agents. Install it with:

```bash
npx add-skill duailibe/linear-cli
```

## Development

```bash
go test ./...
```

## License

MIT
