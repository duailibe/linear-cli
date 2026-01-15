# Development

## Build

```bash
make build
```

The binary is written to `bin/linear`.

## Test

```bash
make test
# or
go test ./...
```

## Smoke tests

Smoke tests run the CLI against the Linear API using a playground key. Each run
creates a new issue and updates it to validate the update flow.

Required:

- `LINEAR_API_KEY`
- `LINEAR_SMOKE_TEAM`

Optional:


Run them with:

```bash
scripts/smoke_test.sh
```

## Lint

```bash
make lint
```

## Formatting and tidy

```bash
make fmt
make tidy
```

## Versioning

Version metadata is injected at build time via ldflags:

- `VERSION` uses `git describe --tags --always --dirty`
- `COMMIT` uses the short Git SHA
- `DATE` uses the UTC build timestamp

Run `linear --version` to see the embedded version, commit, and build date.

## Release basics

- Tag releases with semantic versions (for example `v0.1.0`).
- The Makefile uses the tag to populate the version string automatically.
