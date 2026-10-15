# FluentFox API

Go backend for FoxSensei, a Japanese learning app. Stack: Go, Huma on Gin,
PostgreSQL via GORM, zap logging.

## Golden rule: do not change existing code unless asked

- Read the existing code first and write new code the same way.
- Never refactor, rename, reformat or "clean up" code the task did not ask
  you to touch. If you see a problem in existing code, mention it in your
  reply instead of fixing it.
- Keep a diff to the lines the task needs.

## Commands

- Build: `go build ./...`
- Lint: `golangci-lint run --config .claude/golangci.yml --new-from-rev=HEAD ./...`
- Test: `make test` (needs Docker for the test database)
- Migrate: `make migrate`

Hooks in `.claude/settings.json` run goimports on files you edit and run
build, vet, tests and lint on your changes before you finish. Lint only
reports issues on lines you changed. Fix what they report.

## How the code is written today (follow this)

```
cmd/api/main.go        wiring: config, DB, mailer, repos, services, routes
config/config.go       typed Config loaded with viper; require() for mandatory keys
internal/<domain>/     one folder per feature (auth, users, kana, quiz ...)
  routes.go            RegisterRoutes(api huma.API, h *Handler) with huma.Register
  handler.go           Handler struct; one method per endpoint
  service.go           one struct per use case, built with NewX(...)
  repository.go        consumer-side interface (e.g. auth.UserRepo)
  dto.go               request/response structs with json, validate and doc tags
internal/users/        models.go (GORM models) and repository.go (*users.Repository)
internal/common/       shared constants.go and enum types.go
internal/services/     MailService, shared across domains
pkg/                   infrastructure with no domain knowledge:
                       exceptions, humautil, mailer, token, middleware, database ...
```

**Handlers.** Signature is
`func (h *Handler) X(ctx context.Context, input *humautil.Input[Req]) (*humautil.Output[humautil.APIResponse[Resp]], error)`.
Get the logger with `middleware.LoggerFromContext(ctx, h.logger)`.
Return errors through `humautil.MapErr(err, log)` and success through
`humautil.OK(...)`. No business rules in handlers.

**Services.** One struct per use case (`AuthService`, `LoginService`,
`TokenVerificationService` ...). Dependencies come in through the `NewX`
constructor: the repo interface, other services, `*zap.Logger`.

**Repositories.** The consuming domain declares the interface it needs
(`auth.UserRepo`); `*users.Repository` satisfies it. Add new methods to
both. Repository methods take `ctx` first and wrap errors with
`fmt.Errorf("...: %w", err)`.

**Transactions.** Use `repo.Transaction(ctx, func(tx *gorm.DB) error {...})`
and pass `tx` to the repository write methods, like `registerUser` does.

**Errors.** Errors the client sees are `*exceptions.AppError` from
`pkg/exceptions`. Return an existing sentinel (`exceptions.ErrTokenExpired`)
or add a new sentinel there. Use `exceptions.Wrap(status, code, msg, cause)`
only when you need to keep a cause.

**Constants.** Shared constants live in `internal/common/constants.go`.
Enum types with their values live in `internal/common/types.go`.

**Config.** New env values go in the `Config` struct and `Load()` in
`config/config.go`, with `require()` when the app cannot run without them.
Document the key in the README or `.env.example`.

**Wiring.** Construct new repos and services in `cmd/api/main.go`, next to
the existing ones.

## Quality rules for new code

- **Single responsibility.** One use case per service struct. If a struct
  needs more than about five dependencies, split it.
- **Depend on interfaces you define in the consuming package.** Keep them to
  the methods you call. Accept interfaces, return concrete types.
- **No magic numbers or strings in logic.** Name them in
  `internal/common/constants.go`. Prefer typed values such as
  `time.Duration`.
- **Never ignore an error.** Wrap it with context using `%w`. No `panic`
  outside `main` and config loading.
- **Context first.** Every function that does I/O takes `ctx context.Context`
  as its first parameter. Do not use `context.Background()` inside a request.
- **Small functions.** About 50 lines at most, nesting three levels at most.
- **No duplication.** Reuse helpers in `pkg/` and the domain before writing
  new ones.
- **Prefer a small named struct over `map[string]string`** for related values.
- **Security.** Passwords use argon2id. Lookup tokens are stored as SHA-256
  hashes. Never log passwords, raw tokens or full request bodies.

## Comments

Write a comment only where a reader would otherwise stop and ask "why?".
If the function or variable name already says what it does, write no comment.

- Explain **why**, or a rule that is not obvious from the code: security
  choices, transaction or lock order, rollback behaviour, workarounds.
- One or two sentences in plain English. No banners, no commented-out code.
- No doc comments that restate the name (`// NewX builds an X.`,
  `// GetUser gets the user.`). Leave them out.

Good: `// Hash before storing so a leaked table cannot be used to verify emails.`
Bad: `// hash the token` above `hash := sha256.Sum256(token)`.

## Tests

- Table-driven tests with `t.Run`, cases named by behaviour.
- Test services with small hand-written fakes of their repo interface.

## Never

- Read or edit real `.env*` files. Use `.env.example` for variable names.
- Force push, `git reset --hard`, or skip git hooks.

## Workflow for a new feature

1. `go-architect` agent: short design within the patterns above.
2. Implement it.
3. `go-reviewer` agent on the diff; fix its findings.
4. `test-writer` agent for new service methods.

The `/feature` and `/go-review` skills run these steps.
