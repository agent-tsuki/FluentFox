---
name: go-review
description: Review the current Go changes in FluentFox against CLAUDE.md, SOLID, design and comment rules. Use when the user asks to review or check code quality of their changes.
---

1. Run the `go-reviewer` agent on the current diff, or on the files or
   commits the user named.
2. Show the findings, most severe first.
3. Fix nothing unless the user asks. If they ask, fix only the reported
   lines, then run `go build ./...` and show the output.
