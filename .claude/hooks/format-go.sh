#!/usr/bin/env bash
# PostToolUse hook: runs goimports on a Go file right after Claude edits it.
# A file is only formatted if its committed version was already formatted,
# so a small edit never turns into a whole-file reformat of existing code.
set -uo pipefail
HOOK_INPUT="$(cat)"
source "$(dirname "$0")/lib.sh"

file="$(json_field '.tool_input.file_path')"
[[ "$file" == *.go && -f "$file" ]] || exit 0
echo "$file" >> "$(session_edits_file)"

project_dir="${CLAUDE_PROJECT_DIR:-$(pwd)}"
committed="$(git -C "$project_dir" show "HEAD:${file#"$project_dir"/}" 2>/dev/null)"
if [[ -n "$committed" && -n "$(goimports -local "$MODULE_PATH" -l <<<"$committed")" ]]; then
  exit 0
fi

if ! out="$(goimports -local "$MODULE_PATH" -w "$file" 2>&1)"; then
  block "goimports failed on $file. Fix the syntax error:
$out"
fi
exit 0
