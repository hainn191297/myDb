# Repository Guidelines

This repository prototypes a Go-based relational database, so every change should keep the learning-oriented codebase approachable, well-tested, and easy to trace from SQL entrypoints down to storage internals.

## Project Structure & Module Organization

- `cmd/mydbd` hosts the primary daemon/CLI wiring that bootstraps configuration, server endpoints, and telemetry.
- `internal/` holds the real logic: `sql/` (lexer, parser, planner), `storage/` (WAL, buffer pool, page formats), `txn/` (locks, sessions), `server/` (gRPC/HTTP surfaces), and `config/` (runtime options).
- Top-level docs (`architecture.md`, `knowledge.md`, `docs/`, `plantUML/`) capture design decisions; keep them updated whenever architecture or terminology changes.
- Tests live beside the code they exercise; add focused fixtures under each package instead of a global test directory.

## Build, Test, and Development Commands

- `go build ./cmd/mydbd` — compile the daemon and verify module wiring.
- `go run ./cmd/mydbd --help` — quick smoke run to confirm CLI flags and dependency injection.
- `go test ./...` — run the complete unit test suite; required before every PR.
- `go test -bench . ./internal/...` — run microbenchmarks when touching index, WAL, or planner hot paths.
- `go fmt ./... && go vet ./...` — enforce formatting and static checks prior to committing.

## Coding Style & Naming Conventions

- Stick to idiomatic Go: tabs for indentation, mixedCaps for exported identifiers, `lowerCamel` for locals.
- Keep functions short and cohesive; prefer pure helpers under `internal/pkg`-style directories when code spans packages.
- Always run `gofmt` (or `goimports`) on every touched file; avoid custom linters unless configured here.
- Error values should wrap context via `fmt.Errorf("...: %w", err)` and bubble upwards until `server` logs them.

## Testing Guidelines

- Use Go's `testing` package with table-driven tests (`TestComponentScenario`) colocated with the code under test.
- Cover all new branches, especially WAL recovery, planner rewrites, and lock arbitration; add regression cases before bugfixes.
- For concurrency-sensitive code, add `t.Parallel()` only when data races are impossible, and lean on `-race` locally when touching shared state.

## Commit & Pull Request Guidelines

- Follow the existing short, imperative commit style (`Enhance Go CI workflow...`, `Add CodeQL analysis...`). Keep subject ≤72 characters and explain context in the body when needed.
- Each PR should describe motivation, summarize architectural impacts, link to any relevant issues/notes, and include screenshots or log excerpts if behavior changes.
- Ensure CI passes (`go test ./...`, lint/format commands above) and mention any follow-up TODOs explicitly in the PR description.

## Security & Configuration Tips

- Configuration structs live in `internal/config`; keep secrets outside the repo and inject via environment variables or runtime flags.
- Never commit real credentials or production dataset snapshots; if you need fixtures, anonymize them and store under `docs/fixtures` with clear provenance notes.
