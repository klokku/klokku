# AGENTS.md

Guidance for AI agents working on the Klokku codebase.

## Project Overview

Klokku is a time-planning and time-tracking application written in Go.
The repository produces two binaries:

- `klokku` - the HTTP API server (`main.go`, built with `make build`).
- `klokku-cli` - a Cobra-based CLI for interacting with the API, primarily aimed at AI agents and scripting (`cmd/klokku-cli/main.go` -> `internal/cli/`, built with `make build-cli`).

The frontend lives in a separate repository (`klokku-ui`) and is optionally served by the Go backend when `frontend.enabled` is true.

## Tech Stack

- **Language**: Go 1.26+ (module path `github.com/klokku/klokku`).
- **Database**: PostgreSQL 18, accessed via `pgx/v5` connection pool.
- **Migrations**: `golang-migrate` v4, forward-only files in `/migrations` (numbered `NNNN_description.up.sql`).
- **HTTP**: Gorilla Mux.
- **Config**: koanf (YAML + `KLOKKU_`-prefixed env vars), defined in `internal/config/config.go`.
- **Logging**: logrus, imported as `log`.
- **API docs**: Swagger via `swag`/`http-swagger`; generated docs live in `/docs` and are served at `/swagger/`.
- **Testing**: testify (assert/require); repository tests use `testcontainers-go` with a real Postgres container.

## Common Commands

```bash
make build           # build the klokku server binary
make build-cli       # build the klokku-cli binary
make build-all       # build both
make run             # go run main.go

make fmt             # go fmt
make vet             # go vet (runs fmt first)
make cilint          # golangci-lint run (CI uses v2.11.3, --timeout=10m)
make swagger         # regenerate Swagger docs from annotations (run after changing handlers)

go test ./... -v     # run all tests (requires Docker for repository tests)
go test ./pkg/<pkg>/... -v -run TestServiceImpl_GetPlan   # run a specific test
```

Lint and tests must pass before committing. CI runs `go test ./... -v` and `golangci-lint`.

## Repository Tests Require Docker

Repository-level tests spin up a real Postgres container via testcontainers (see `internal/test_utils/postgres.go`).
Docker must be running. The container is snapshotted after migrations and restored between tests for isolation.
Service-level tests do NOT need Docker - they use in-memory stub repositories (e.g. `repository_stub.go`).
Prefer stubs for business-logic tests; use testcontainers only when you are testing SQL/repository behavior.

## Architecture

See `contributing/architecture.md` and `contributing/guidelines.md` for the full style guide. Key points:

- `/pkg/` - public domain packages, organized by feature (`budget_plan`, `calendar`, `current_event`, `stats`, `user`, `weekly_plan`, `webhook`, `clickup`, `calendar_provider`, `budget_plan_report`). Each is importable by other projects.
- `/internal/` - private code: `app` (wiring, routes, middleware), `config`, `database`, `event_bus`, `rest` (HTTP helpers), `cli` (CLI commands), `test_utils`, `utils`.
- `/migrations/` - forward-only golang-migrate SQL files; all table schemas are created through migrations.
- `/db/init.sql` - PostgreSQL database/schema bootstrap used before migrations run.
- `/contributing/`, `/docs/`, `/scripts/`, `/skills/` - supporting files.

Each domain package follows a strict four-file layering:

1. `<domain>.go` - domain models (plain structs, no logic).
2. `repository.go` - `Repository` interface + `RepositoryImpl` using `*pgxpool.Pool`, constructed by `NewRepository(db)`.
3. `service.go` - `Service` interface + `ServiceImpl`, constructed by `NewService(repo, ...)`. Business logic lives here; it reads the current user from context via `user.CurrentId(ctx)`.
4. `handler.go` - HTTP `Handler` struct with methods wired to routes, constructed by `NewHandler(svc)`; methods carry `// @Summary` Swagger annotations.

Testing files: `service_test.go` (stub-based) and `repository_test.go` (testcontainers-based) alongside the implementation.

## Authentication

There is no token-based auth. The `X-User-Id` header carries a user UID.
The middleware in `internal/app/middleware.go` resolves it to a `user.User` and stores it in the request context (`user.WithUser`).
Services obtain the user with `user.CurrentId(ctx)` or `user.Current(ctx)`; always propagate `context.Context` as the first parameter.
When adding an endpoint that needs the current user, do not read the header yourself - read the context.

## Wiring: Adding a New Feature / Endpoint

Follow the existing pattern end-to-end:

1. **Domain model**: add structs to `pkg/<domain>/<domain>.go`.
2. **Repository**: define methods on the `Repository` interface and implement them in `repository.go` using prepared statements/parameterized queries. Add a stub implementation in `repository_stub.go` if service tests need it.
3. **Service**: add methods to the `Service` interface and `ServiceImpl` in `service.go`. Inject dependencies via the constructor; do not create them inside.
4. **Handler**: add HTTP methods to `Handler` in `handler.go` with Swagger annotations.
5. **Wiring**: instantiate the new repo/service/handler in `internal/app/dependencies.go` `BuildDependencies` and add fields to the `Dependencies` struct.
6. **Routes**: register the route in `internal/app/routes.go` under the `/api/...` prefix.
7. **Swagger**: run `make swagger` to regenerate `/docs` after adding/changing annotations. Commit the regenerated `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`.
8. **Migrations** (if schema changes): add a backward-compatible `migrations/NNNN_description.up.sql`. Do not duplicate table definitions in `db/init.sql`.
9. **Tests**: add table-driven tests. Use stubs in `service_test.go` for logic; use testcontainers in `repository_test.go` for SQL. Name tests `Test<Type>_<Scenario>`.

## Migration Compatibility

- Migrations are forward-only and must remain compatible with both the currently deployed application and the next application version.
- Prefer additive changes: new nullable columns, columns with safe defaults, new tables, and parallel indexes or constraints.
- Use an expand-migrate-contract sequence for incompatible schema changes: expand the schema, deploy code that can use both representations and backfill data, then contract only after every old application version has been retired.
- Never rename or drop a column, remove a table, or tighten a constraint in the same release that stops using the old schema.
- Fresh installations bootstrap PostgreSQL with `db/init.sql` and then apply the complete migration sequence.

## Conventions (from `contributing/guidelines.md`)

- Singular package names (`user`, not `users`).
- Exported CamelCase, unexported camelCase; acronyms as words (`HttpServer`).
- Methods: action verbs (`GetAll`, `Store`, `Update`); `FindX` returns `X, error` (nil-able), `GetX` returns collections.
- Interfaces at top of file, then implementations; small focused interfaces.
- Error handling: wrap with `fmt.Errorf("...: %w", err)`; define typed errors (e.g. `ErrUserNotFound`); log with context; never `panic` in production code; early returns to reduce nesting.
- No naked returns in long functions; no global state; prefer immutability.
- Context as first parameter for any I/O function.
- Database: prepared statements, parameterized queries, `sql.Null*` / pgx nullable types for nullable columns, `defer` for resource cleanup, transactions for atomic operations.
- Comments: self-documenting code first; add comments only for non-obvious context. Package comment in at least one file per package.

## CLI (`klokku-cli`)

Cobra-based; entry point `cmd/klokku-cli/main.go` delegates to `internal/cli/cmd`.
REST client code lives under `internal/cli/api`, output formatting under `internal/cli/output`, config under `internal/cli/config`.
The `skills/klokku-cli/SKILL.md` file documents CLI usage for AI agents and should be kept in sync with CLI commands.
Released via GoReleaser (`.goreleaser.yaml`) on unprefixed Git tags.

## Git Workflow

- Target branch: `main`.
- Branch naming prefixes: `feature/`, `bugfix/`, `tech/`, `upgrade/` (e.g. `feature/budget-plan-report`).
- PRs are squash-merged; the resulting commit message uses imperative mood and ends with `(#NN)` referencing the PR number (e.g. `Add budget plan report feature (#27)`).
- Write commit/PR messages in the imperative ("Add ...", "Fix ...", "Update ...").
- Do not force-push to `main`; never force-push shared branches.
- Never commit secrets or the `.env` file (it is gitignored).

## Before Opening a PR

1. `make fmt && make vet`
2. `make cilint` (or `golangci-lint run`)
3. `go test ./... -v` (Docker running)
4. `make swagger` if you touched handlers, and commit regenerated docs.
5. Confirm every new migration is backward compatible with the previously deployed application version.
