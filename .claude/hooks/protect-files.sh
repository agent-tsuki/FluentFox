#!/usr/bin/env bash
# PreToolUse hook for Edit/Write: keeps Claude out of secret and generated files.
set -uo pipefail
HOOK_INPUT="$(cat)"
source "$(dirname "$0")/lib.sh"

file="$(json_field '.tool_input.file_path')"
name="$(basename "$file")"

case "$name" in
  .env.example) exit 0 ;;
  .env|.env.*) block "Blocked: $name holds real secrets. Edit .env.example and tell the user which values to set." ;;
  go.sum) block "Blocked: go.sum is generated. Run 'go mod tidy' instead." ;;
esac
case "$file" in
  */.git/*|*/.vercel/*) block "Blocked: $file is managed by a tool, not edited by hand." ;;
esac
exit 0
