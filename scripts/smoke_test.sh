#!/usr/bin/env bash
set -euo pipefail

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required for smoke tests" >&2
  exit 1
fi

if [[ -z "${LINEAR_API_KEY:-}" ]]; then
  echo "LINEAR_API_KEY is required" >&2
  exit 1
fi

if [[ -z "${LINEAR_SMOKE_TEAM:-}" ]]; then
  echo "LINEAR_SMOKE_TEAM is required (team key, e.g. DUA)" >&2
  exit 1
fi

TEAM="${LINEAR_SMOKE_TEAM}"
PREFIX="${LINEAR_SMOKE_PREFIX:-${TEAM}-}"
STATE="In Progress"
SMOKE_TS="$(date -u +%Y%m%d%H%M%S)"
SMOKE_TITLE="Smoke Test Issue ${SMOKE_TS}"
SMOKE_UPDATED_TITLE="Smoke Test Updated ${SMOKE_TS}"
SMOKE_DESCRIPTION="Smoke test description ${SMOKE_TS}"
SMOKE_COMMENT="Smoke test comment ${SMOKE_TS}"

if [[ ! -x "./bin/linear" ]]; then
  echo "expected ./bin/linear (run make build first)" >&2
  exit 1
fi

LINEAR=( ./bin/linear )

run() {
  echo "> $*" >&2
  "$@"
}

run "${LINEAR[@]}" whoami --json | jq -e '.id and .email' >/dev/null

run "${LINEAR[@]}" team list --json | jq -e --arg team "$TEAM" '.[] | select(.key == $team)' >/dev/null

issues_json=$(run "${LINEAR[@]}" issue list --team "$TEAM" --limit 20 --json)

issue_count=$(echo "$issues_json" | jq '.nodes | length')
if [[ "$issue_count" -lt 1 ]]; then
  echo "no existing issues found for team $TEAM; continuing with create flow" >&2
else
  bad_ids=$(echo "$issues_json" | jq -r --arg prefix "$PREFIX" '.nodes[].identifier | select(startswith($prefix) | not)')
  if [[ -n "$bad_ids" ]]; then
    echo "issue identifiers missing prefix $PREFIX:" >&2
    echo "$bad_ids" >&2
    exit 1
  fi

  issue_id=$(echo "$issues_json" | jq -r '.nodes[0].identifier')

  run "${LINEAR[@]}" issue view "$issue_id" --json | jq -e '.id and .state' >/dev/null
  run "${LINEAR[@]}" issue view "$issue_id" --comments --comments-limit 1 --json | jq -e 'has("comments") | not or (.comments | type == "array")' >/dev/null
  run "${LINEAR[@]}" issue attachments "$issue_id" --limit 5 --json >/dev/null
fi
run "${LINEAR[@]}" cycle list --team "$TEAM" --current --json >/dev/null

created_issue_id=$(run "${LINEAR[@]}" issue create --team "$TEAM" --title "$SMOKE_TITLE" --description "$SMOKE_DESCRIPTION" --priority 2 --json | jq -r '.identifier')
if [[ "$created_issue_id" != "$PREFIX"* ]]; then
  echo "created issue identifier missing prefix $PREFIX: $created_issue_id" >&2
  exit 1
fi
run "${LINEAR[@]}" issue update "$created_issue_id" --title "$SMOKE_UPDATED_TITLE" --description "$SMOKE_DESCRIPTION updated" --priority 1 --json | jq -e '.id' >/dev/null
run "${LINEAR[@]}" issue update "$created_issue_id" --state "$STATE" --json | jq -e '.id' >/dev/null
run "${LINEAR[@]}" issue comment "$created_issue_id" --body "$SMOKE_COMMENT" --json | jq -e '.id' >/dev/null
run "${LINEAR[@]}" issue view "$created_issue_id" --json | jq -e --arg state "$STATE" --arg title "$SMOKE_UPDATED_TITLE" '.state == $state and .title == $title and .priority == 1' >/dev/null

echo "smoke tests passed" >&2
