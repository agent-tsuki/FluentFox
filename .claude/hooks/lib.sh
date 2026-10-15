#!/usr/bin/env bash
# Shared helpers for the hook scripts. Sourced, never run directly.

export PATH="$(go env GOPATH 2>/dev/null)/bin:$PATH"
MODULE_PATH="github.com/fluentfox/api"

# block prints a message and exits 2, which makes Claude Code stop the action
# and show the message to Claude.
block() {
  echo "$1" >&2
  exit 2
}

# json_field prints the value at a jq path from the hook's JSON input.
json_field() {
  jq -r "$1 // empty" <<<"$HOOK_INPUT"
}

# session_edits_file lists the Go files Claude edited in this session, so the
# stop check judges only Claude's work, not the user's own uncommitted changes.
session_edits_file() {
  local session
  session="$(json_field '.session_id')"
  echo "${TMPDIR:-/tmp}/claude-go-edits-${session:-default}"
}
