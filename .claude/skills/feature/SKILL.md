---
name: feature
description: Build a new FluentFox API feature or endpoint with design first, then code, review and tests, following the existing code patterns. Use when the user asks to add a feature, endpoint or new behaviour.
---

Build the feature in this order. Do not skip a step.

1. **Design.** Run the `go-architect` agent with the request. Show the
   design. If it has open questions or changes existing code, wait for the
   user's approval.
2. **Implement.** Follow `CLAUDE.md` and the design. Add new code; leave
   existing code alone unless the design says otherwise.
3. **Wire.** Construct new pieces in `cmd/api/main.go` and register routes in
   the domain's `routes.go`.
4. **Review.** Run the `go-reviewer` agent on the diff. Fix blockers and
   majors. Fix minors or say why not.
5. **Test.** Run the `test-writer` agent for each new service method.
6. **Verify.** Run `go build ./...` and show the output.

Finish with a short summary: what was built, files changed, test results,
and anything the user must decide.
