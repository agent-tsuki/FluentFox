#!/usr/bin/env bash
# UserPromptSubmit hook: this text is added to Claude's context with every
# prompt, so the core rules stay in view during long sessions.
cat <<'MSG'
FluentFox rules (full text in CLAUDE.md): do not refactor or reformat existing code unless asked; write new code in the style of internal/auth; one use case per struct with a NewX constructor; no magic numbers or strings; wrap every error; short "why" comments only.
MSG
