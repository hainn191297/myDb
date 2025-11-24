# PlantUML Diagrams - MyDB Architecture

This directory contains PlantUML diagrams documenting MyDB's architecture and flows.

## Overview Diagrams

### 1. `architecture_mydb.puml`
**Complete system architecture** showing all layers:
- Client Layer (mydb-cli, gRPC clients)
- Server Layer (gRPC API, Sessions, SQL Processing, Transactions, Catalog)
- Storage Layer (Providers, Engines, Buffer Pool, Page Management, WAL)
- Disk Layer (Data files, Index files, WAL files)

**Use case**: Understanding overall system structure

---

### 2. `architecture_rdbms.puml`
**Generic RDBMS reference architecture** for comparison:
- Traditional database components
- Pluggable storage engines
- General design patterns

**Use case**: Learning context for database systems

---

## Module-Specific Flows

### SQL Processing

**File**: `internal/sql/sql_flow.puml`

**Covers**:
- Parser: Tokenization, AST building, expression parsing
- Planner: IndexScan vs SeqScan selection, operator tree
- Executor: Volcano iterator, expression evaluation, locking

**Examples**:
1. `SELECT * FROM users WHERE age > 25` (full flow)
2. `INSERT INTO users VALUES (1, 'Alice', 30)` (with uniqueness check)
3. `UPDATE users SET age = 31 WHERE id = 1` (with index maintenance)

---

### Storage Layer

**File**: `internal/storage/storage_flow.puml`

**Covers**:
- Complete INSERT flow (data + index)
- Buffer Manager page lifecycle
- WAL logging and flush
- Index maintenance (B+Tree)
- SELECT with index lookup
- Crash recovery process

**Flow**:
```
Executor → Provider → Engine → Buffer Manager → Global Pool → WAL → File Manager → Disk
```

**Key Concepts**:
- Page pinning/unpinning
- Dirty page tracking
- LRU eviction
- WAL-before-write durability

---

**File**: `internal/storage/buffer_flow.puml`

**Focused on**: Buffer pool operations
- Page load (cache hit/miss)
- Eviction with WAL flush
- Mark dirty / Unpin
- FlushAll mechanism

---

**File**: `internal/storage/wal/wal_flow.puml`

**Focused on**: Write-Ahead Logging
- Append log record
- Sync to disk
- Write data page
- Explicit flush

---

**File**: `internal/storage/engine/heap/heap_flow.puml`

**Focused on**: Heap table operations
- Insert into slotted page
- Scan all pages
- Buffer manager integration

---

### Transaction & Locking

**File**: `internal/txn/transaction_flow.puml`

**Covers**:
- Transaction lifecycle (BEGIN/COMMIT/ROLLBACK)
- Lock acquisition (read/write locks)
- Concurrent transaction isolation
- Deadlock detection via timeout
- Lock wait queue

**Scenarios**:
1. Simple transaction
2. Concurrent access (read waiting for write)
3. Deadlock (timeout-based resolution)

**Lock Compatibility Matrix**:
```
|         | No Lock | Read  | Write |
|---------|---------|-------|-------|
| Read    | ✓       | ✓     | ✗     |
| Write   | ✓       | ✗     | ✗     |
```

---

### Server & Session Management

**File**: `internal/server/server_flow.puml`

**Covers**:
- gRPC server startup
- Session creation and reuse
- Request routing (SELECT/DML/DDL/Transaction)
- Response formatting (QueryResult/CommandResult/ErrorResult)
- GetMetadata RPC

**Session Features**:
- Unique session IDs (timestamp-based)
- Transaction state per session
- Thread-safe session map

---

### Schema Catalog

**File**: `internal/schema/schema_catalog_flow.puml`

**Covers**:
- Catalog startup (load from system tables)
- CREATE TABLE/DROP TABLE
- CREATE INDEX/DROP INDEX
- Metadata lookup (in-memory cache)

**System Tables**:
- `__system.__catalog_tables`
- `__system.__catalog_indexes`

---

## How to Use

### Viewing Diagrams

**Option 1: VS Code Extension**
```bash
# Install PlantUML extension
code --install-extension jebbs.plantuml
```

**Option 2: Online Viewer**
Visit: https://www.plantuml.com/plantuml/uml/

**Option 3: Command Line**
```bash
# Install PlantUML
brew install plantuml

# Generate PNG
plantuml internal/sql/sql_flow.puml

# Generate SVG
plantuml -tsvg internal/sql/sql_flow.puml
```

---

### Updating Diagrams

When code changes, update corresponding diagrams:

1. **Architecture changes** → `architecture_mydb.puml`
2. **SQL processing changes** → `internal/sql/sql_flow.puml`
3. **Storage changes** → `internal/storage/storage_flow.puml`
4. **Transaction changes** → `internal/txn/transaction_flow.puml`
5. **Server API changes** → `internal/server/server_flow.puml`

---

## Diagram Index

| Diagram | Location | Purpose |
|---------|----------|---------|
| MyDB Architecture | `plantUML/architecture_mydb.puml` | Complete system overview |
| RDBMS Reference | `plantUML/architecture_rdbms.puml` | Generic database architecture |
| SQL Processing | `internal/sql/sql_flow.puml` | Parser → Planner → Executor |
| Storage Complete | `internal/storage/storage_flow.puml` | End-to-end storage flow |
| Buffer Pool | `internal/storage/buffer_flow.puml` | Page caching details |
| WAL | `internal/storage/wal/wal_flow.puml` | Write-ahead logging |
| Heap Engine | `internal/storage/engine/heap/heap_flow.puml` | Heap table operations |
| Transactions | `internal/txn/transaction_flow.puml` | Locking and concurrency |
| Server/Sessions | `internal/server/server_flow.puml` | gRPC and session handling |
| Schema Catalog | `internal/schema/schema_catalog_flow.puml` | Metadata management |

---

## Quick Reference

### Data Flow (SELECT)
```
Client → Server → Parser → Planner → Executor
  → Provider → HeapEngine → BufferManager → GlobalPool
  → PageCache (hit) → return data
```

### Data Flow (INSERT)
```
Executor → HeapEngine.Put() → BufferManager.GetPage()
  → GlobalPool (pin) → modify page → Unpin(dirty=true)
  → BTreeEngine.Insert() (index) → mark dirty
  → On flush: WAL → Data → Sync
```

### Transaction Flow
```
BEGIN → Executor (acquire locks) → modify data
  → COMMIT → flush dirty pages → release locks
```

---

**Maintained by**: MyDB Development Team
**Last Updated**: 2024-11-24
