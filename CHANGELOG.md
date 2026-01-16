# Changelog

## Unreleased

### Changed
- Attachment downloads now fail if no unique filename is available (after 99 attempts) instead of risking an overwrite.
- `linear auth status` now reports API key configuration (not authentication).
- Attachment filename sanitization now guards against `.` and `..` to avoid directory traversal (sanitized at download time only).

## v0.2.0 (2026-01-15)

- Simplify CLI, drop schema cache and API URL configuration

## v0.1.0 (2026-01-15)

- Full issue workflow: list, view, create, update, close, reopen, and comment from the command line.
- Powerful filtering: by team, assignee, state, labels, project, cycle, priority, or title search.
- Issue dependencies: create and manage blocks/blocked-by relationships.
- Attachment downloads: fetch attachments from issues and comments to local files.
- Script-friendly: `--json` output everywhere, stdin support for descriptions and comments.
- Simple auth: store your API key with `linear auth login` or use `LINEAR_API_KEY`.
