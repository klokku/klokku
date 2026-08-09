# Go Style Guide for Klokku

This document outlines the coding conventions and best practices for Go development in the Klokku project. Following these guidelines ensures consistency, readability, and maintainability across the codebase.

## Table of Contents

1. [Package Organization](#package-organization)
2. [Code Structure](#code-structure)
3. [Naming Conventions](#naming-conventions)
4. [Error Handling](#error-handling)
5. [Documentation](#documentation)
6. [Testing](#testing)
7. [Concurrency](#concurrency)
8. [Database Access](#database-access)
9. [Common Pitfalls](#common-pitfalls)

## Package Organization

- **Domain-Driven Design**: Organize packages by domain/feature (e.g., `budget_plan`, `current_event`, `user`).
- **Separation of Concerns**: Use the following package hierarchy:
  - `internal/`: Code that's not meant for external import (app wiring, config, database, CLI, test utils)
  - `pkg/`: Code that can be imported by other projects; each package follows the four-file layering (model, repository, service, handler) described in `contributing/architecture.md`
  - `migrations/`: Forward-only database migration scripts (golang-migrate, numbered `NNNN_description.up.sql`)
  - `db/init.sql`: PostgreSQL database/schema bootstrap; table schemas are created by migrations
- **Package Naming**: Use singular nouns for package names (e.g., `user` not `users`).
- **Avoid Circular Dependencies**: Ensure packages form a directed acyclic graph.

## Code Structure

### Architectural Patterns

- **Service-Oriented Architecture**: Organize code into services, repositories, and handlers.
- **Dependency Injection**: Pass dependencies through constructors rather than creating them inside functions.
- **Interface Segregation**: Define focused interfaces with minimal methods.
- **Repository Pattern**: Use repositories to abstract data access.

### File Organization

- **Interface and Implementation Separation**: Define interfaces at the top of the file, followed by implementations.
- **Related Functionality**: Group related functions together.
- **Helper Functions**: Place helper functions close to where they're used.

## Naming Conventions

- **CamelCase**: Use CamelCase for exported names and camelCase for non-exported names.
- **Descriptive Names**: Use descriptive, unabbreviated names for variables, functions, and types.
- **Interface Naming**: Name interfaces based on their behavior (e.g., `Reader`, `Writer`).
- **Method Naming**:
  - Use action verbs for methods that perform operations (e.g., `GetAll`, `Store`, `Update`).
  - Use `Find` prefix for methods that return a single entity or nil.
  - Use `Get` prefix for methods that return collections.
- **Acronyms**: Treat acronyms as words in names (e.g., `HttpServer` not `HTTPServer`).

## Error Handling

- **Error Wrapping**: Wrap errors with `fmt.Errorf("context: %w", err)` to add context as errors propagate up the call stack. Use `errors.Is` / `errors.As` to inspect wrapped errors.
- **Typed Errors**: Define sentinel error variables with `errors.New` for specific error conditions (e.g., `ErrUserNotFound`, `ErrNoCurrentEvent`, `ErrPlanNotFound`). Package-prefix them with `Err` and place them near the top of the relevant file.
- **Error Propagation**: Return errors to the caller rather than handling them locally when appropriate.
- **Early Returns**: Use early returns for error conditions to minimize nesting.
- **Logging**: Log errors with context before returning them.
- **No Panic**: Avoid using `panic` in production code; return errors instead.

## Documentation

- **Self-Documenting Code**: Write clear, descriptive code that explains itself.
- **Interface Documentation**: Document interfaces and their methods with comments.
- **Complex Logic**: Prioritize clear, self-documenting code. Add comments only when they provide additional value or clarify non-obvious context.
- **Package Documentation**: Include a package comment in at least one file per package.

## Testing

- **Table-Driven Tests**: Use table-driven tests for testing multiple scenarios, with `t.Run` subtests.
- **Two-Layer Testing**:
    - `service_test.go` - stub-based tests for business logic; no Docker required. Use in-memory stub repositories (e.g. `repository_stub.go`).
    - `repository_test.go` - testcontainers-based tests for SQL/repository behavior; requires Docker. The Postgres container is snapshotted after migrations and restored between tests for isolation (see `internal/test_utils/postgres.go`).
- **Mocking**: Use interfaces and dependency injection to facilitate stubbing; prefer stub implementations over mocks.
- **Test Naming**: Name tests with the pattern `Test<Function>_<Scenario>`.
- **Test Coverage**: Aim to test critical paths and edge cases thoroughly. Avoid focusing solely on coverage metrics; instead, prioritize meaningful, high-value tests for complex and critical logic.


## Concurrency

- **Context Usage**: Pass `context.Context` as the first parameter to functions that perform I/O.
- **Goroutine Management**: Ensure goroutines are properly managed and cleaned up.
- **Mutex Usage**: Use mutexes to protect shared state.
- **Channel Patterns**: Follow established channel patterns for communication between goroutines.

## Database Access

- **Driver**: PostgreSQL via `pgx/v5` (connection pool `*pgxpool.Pool`); repositories are constructed with `NewRepository(db)`.
- **Prepared Statements**: Use prepared statements for all database operations.
- **Resource Cleanup**: Use `defer` to ensure resources (rows, connections) are properly closed.
- **Transaction Management**: Use transactions (`pgx.Tx`) for operations that require atomicity; roll back on error (checking `pgx.ErrTxClosed`).
- **SQL Injection Prevention**: Use parameterized queries to prevent SQL injection.
- **NULL Handling**: Use `sql.Null*` types (or pgx nullable types) for columns that can be NULL.
- **No Rows**: Detect missing rows with `errors.Is(err, pgx.ErrNoRows)`.

## Database Migrations

- **Forward-Only**: Add migrations as `NNNN_description.up.sql`; production schema changes are not rolled back with down migrations.
- **Version Compatibility**: Every migration must work with both the currently deployed application version and the next version so mixed-version deployments remain safe.
- **Expand**: Introduce additive schema such as nullable columns, columns with safe defaults, new tables, or parallel indexes and constraints.
- **Migrate**: Deploy code that can read and write both old and new representations, then backfill existing data as needed.
- **Contract**: Remove obsolete schema only in a later release, after all application versions that depend on it have been retired.
- **Unsafe Changes**: Never rename or drop a column, remove a table, or tighten a constraint in the same release that stops using the old schema.
- **Fresh Installs**: `db/init.sql` bootstraps PostgreSQL; the full table schema is produced by applying every migration in order.

## Common Pitfalls

### Avoid

- **Global State**: Minimize the use of global variables.
- **Naked Returns**: Avoid naked returns in long functions.
- **Interface Pollution**: Don't create interfaces with too many methods.
- **Premature Optimization**: Write clear code first, then optimize if necessary.
- **Ignoring Errors**: Always check and handle errors appropriately.

### Prefer

- **Immutability**: Prefer creating new objects over modifying existing ones.
- **Composition Over Inheritance**: Use composition to build complex types.
- **Explicit Over Implicit**: Be explicit about your intentions in code.
- **Simplicity**: Keep functions and methods small and focused.
- **Consistency**: Follow established patterns in the codebase.

---

This style guide is a living document and may evolve as the project grows and as Go best practices evolve.
