# Project Context

## Identity

MyDb is a learning-oriented relational database prototype written in Go. The code should remain approachable, well-tested, and easy to trace from SQL entrypoints down to storage internals.

## Architecture Map

- `cmd/mydbd`: daemon and CLI wiring
- `internal/sql`: lexer, parser, planner, and query processing code
- `internal/storage`: WAL, buffer pool, page formats, and disk-facing storage code
- `internal/txn`: transactions, locks, and sessions
- `internal/server`: gRPC and HTTP surfaces
- `internal/config`: runtime configuration
- `.kiro`: optional spec-driven workflow, feature specs, architecture notes, and AI-DLC heavy-thinking protocol

## Core Commands

- `go build ./cmd/mydbd`
- `go run ./cmd/mydbd --help`
- `go test ./...`
- `go test -bench . ./internal/...`
- `go fmt ./... && go vet ./...`

## Engineering Bias

- Prefer idiomatic Go and small cohesive functions.
- Keep tests beside the package they cover.
- Wrap errors with context using `fmt.Errorf("...: %w", err)`.
- Preserve the teaching value of the code. Simple, traceable designs beat clever abstractions.
- Update `.kiro` specs or implementation notes when architecture, terminology, or feature behavior changes.

## Current Optional Agent Workflow

`.kiro` is the active optional workflow for this repository. For complex work, read `.kiro/AI-DLC-HEAVYSKILL.md` and the relevant `.kiro/specs/{feature}/` documents before implementing.
