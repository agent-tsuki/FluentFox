#!/usr/bin/env bash
# PreToolUse hook for Bash: blocks commands that leak secrets or destroy work.
set -uo pipefail
HOOK_INPUT="$(cat)"
source "$(dirname "$0")/lib.sh"

cmd="$(json_field '.tool_input.command')"

# Matches a real env file used as a path, like `cat .env.local` or `< ./.env`.
# Prose such as "never read .env* files" does not match. .env.example is
# removed first so it never triggers, even next to a real env file.
ENV_FILE_PATTERN='(^|[[:space:]/=<"'"'"'])\.env(\.(local|development|test|production))?([[:space:];|&>)"'"'"']|$)'
if sed 's/\.env\.example//g' <<<"$cmd" | grep -Eq "$ENV_FILE_PATTERN"; then
  block "Blocked: this command touches a real .env file. Secrets stay out of the chat. Use .env.example for variable names."
fi
if grep -Eq 'git push.*(--force|-f( |$))' <<<"$cmd"; then
  block "Blocked: force push. Ask the user to do it themselves."
fi
if grep -Eq 'git (reset --hard|clean -[a-z]*f|checkout -- \.( |$)|restore \.( |$))' <<<"$cmd"; then
  block "Blocked: this would throw away uncommitted work. Ask the user first."
fi
if grep -Eq -- '--no-verify' <<<"$cmd"; then
  block "Blocked: do not skip git hooks. Fix the failing check instead."
fi
if grep -Eq 'rm -rf? +(/|~|\$HOME|\.)( |$)' <<<"$cmd"; then
  block "Blocked: recursive delete of a root, home or project directory."
fi
exit 0
