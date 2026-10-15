#!/usr/bin/env bash
# Stop hook: before Claude finishes, the packages it edited must build, pass
# go vet and tests, and add no new lint issues on the lines it changed.
# After MAX_ATTEMPTS failed rounds it lets Claude stop, so a problem Claude
# cannot fix never becomes an endless loop.
set -uo pipefail
HOOK_INPUT="$(cat)"
source "$(dirname "$0")/lib.sh"

MAX_ATTEMPTS=3
cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0

edits_file="$(session_edits_file)"
[[ -s "$edits_file" ]] || exit 0

root="$(pwd -P)"
packages="$(sort -u "$edits_file" | while read -r f; do
  [[ -f "$f" ]] || continue
  dir="$(cd "$(dirname "$f")" && pwd -P)"
  [[ "$dir" == "$root"* ]] && echo ".${dir#"$root"}"
done | sort -u)"
[[ -n "$packages" ]] || exit 0

counter_file="${edits_file}.attempts"

run_checks() {
  local out
  if ! out="$(go build ./... 2>&1)"; then
    echo "go build failed:"; echo "$out"; return 1
  fi
  # shellcheck disable=SC2086
  if ! out="$(go vet $packages 2>&1)"; then
    echo "go vet failed:"; echo "$out"; return 1
  fi
  # shellcheck disable=SC2086
  if ! out="$(go test -short -count=1 $packages 2>&1)"; then
    echo "go test failed:"; echo "$out"; return 1
  fi
  if ! out="$(lint_issues)"; then
    echo "golangci-lint could not run:"; echo "$out"; return 1
  fi
  if [[ -n "$out" ]]; then
    echo "golangci-lint found issues in your changes:"; echo "$out" | head -60; return 1
  fi
}

# lint_issues prints one line per lint issue on changed lines. Formatting
# issues in files whose committed version was already unformatted are dropped:
# fixing them means reformatting the user's existing code, which is not allowed.
lint_issues() {
  local json
  # shellcheck disable=SC2086
  json="$(golangci-lint run --config .claude/golangci.yml --new-from-rev=HEAD \
    --output.json.path=stdout --output.text.path=/dev/null $packages 2>/dev/null | head -1)"
  jq -e '.Issues' >/dev/null 2>&1 <<<"$json" || { echo "no JSON output from golangci-lint"; return 1; }

  jq -r '.Issues[]? | [.FromLinter, .Pos.Filename, (.Pos.Line|tostring), .Text] | @tsv' <<<"$json" |
    while IFS=$'\t' read -r linter file line text; do
      file="${file#../}"
      if [[ "$linter" == "goimports" ]] && committed_unformatted "$file"; then
        continue
      fi
      echo "$file:$line: $text ($linter)"
    done
}

committed_unformatted() {
  local committed
  committed="$(git show "HEAD:$1" 2>/dev/null)" || return 1
  [[ -n "$(goimports -local "$MODULE_PATH" -l <<<"$committed")" ]]
}

if report="$(run_checks)"; then
  rm -f "$counter_file" "$edits_file"
  exit 0
fi

attempts=$(( $(cat "$counter_file" 2>/dev/null || echo 0) + 1 ))
echo "$attempts" > "$counter_file"
if (( attempts > MAX_ATTEMPTS )); then
  rm -f "$counter_file"
  echo "verify-go: still failing after $MAX_ATTEMPTS attempts. Tell the user what is still broken." >&2
  exit 0
fi

block "Quality check failed (attempt $attempts of $MAX_ATTEMPTS). Fix these before finishing:
$report"
