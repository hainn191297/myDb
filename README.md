# myDb

**myDb** is a learning project to build a relational database in Go from first principles. The repository captures the full journey—from log-structured storage and a B+Tree index to SQL parsing, transactions, and operational tooling. Everything is intentionally simple so the implementation can stay hackable and educational.

> Status: early design phase. See `architecture.md` for the current high-level layout of interfaces and layers.

## Project Goals

- Understand how a SQL engine fits together (parser → planner → executor → storage).
- Implement transactional guarantees (WAL, locking, isolation choices) instead of relying on existing libraries.
- Gain confidence with Go systems programming patterns: goroutines, channels, memory management, diagnostics.
- Document trade-offs and lessons learned so future improvements stay deliberate.

## Planned Capabilities

| Area | Scope |
| --- | --- |
| Client / SQL | gRPC API, SQL lexer/parser, rule-based planner, streaming results |
| Transactions | Session manager, two-phase locking for writes, configurable isolation, retries/timeouts |
| Storage Engine | WAL-backed KV store, B+Tree index, buffer pool + page cache, checkpointing/fync policy |
| Ops & Tooling | etcd-backed config, rate limiting, metrics, tracing, structured logging |

For details, read `architecture.md`, which walks through the request lifecycle and roles of each layer.

## Getting Started

1. Install Go ≥ 1.22 and make sure `$GOPATH/bin` is on your `PATH`.
2. Clone this repo:

   ```bash
   git clone https://github.com/hainn191297/myDb.git
   cd myDb
   ```

3. Bootstrap modules and run the first tests:

   ```bash
   go mod tidy
   go test ./...
   ```

As the storage engine lands, documentation will be updated with concrete build/run instructions.

## Development Workflow

- Prefer vertical slices: add a small feature end-to-end (e.g., WAL append → recovery test) before moving on.
- Add unit tests per package (`parser`, `planner`, `storage/wal`, `txn/locks`) and keep them fast.
- Use Go benchmarks (`go test -bench . ./...`) to watch for regressions as data structures evolve.
- Capture TODOs in code and mirror larger ideas here in the roadmap to keep scope manageable.

## Roadmap

1. Minimal embedded KV with append-only log + in-memory index.
2. WAL + page format + buffer manager with fsync policies.
3. B+Tree indexes and range scans.
4. SQL lexer/parser and basic executor operators (scan, filter, project, insert/update/delete).
5. Transaction manager with locking + retries and gRPC protocol for clients.
6. Observability layer (metrics, tracing, logging) and configuration via etcd.
7. Stretch goals: query planner improvements, replication/failover, columnar storage experiments.

## Documentation

- `architecture.md`: layered diagram + detailed explanation of the client interface, transaction system, storage engine, and ops model.
- `docs/` (future): deep dives on WAL format, locking, buffer manager, and operational playbooks.

## Learning Resources

- *Database Internals* by Alex Petrov — excellent coverage of storage engines and consensus.
- *Designing Data-Intensive Applications* by Martin Kleppmann — background on transactions and distributed systems.
- CMU 15-445/645 (Intro to Database Systems) lectures for implementation patterns.

## Contributing

This is currently a personal learning project, but feedback and discussion are welcome. File issues with questions or suggestions, or fork the repo to try alternative designs and share your results.
