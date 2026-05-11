# MyDb - Complete RDBMS Architecture

## Full System Architecture with All Layers

```
╔════════════════════════════════════════════════════════════════════════════╗
║                         CLIENT LAYER                                       ║
║  ┌──────────────────────────────────────────────────────────────────────┐  ║
║  │  Interactive CLI Client  │  gRPC Client  │  HTTP Client (Future)    │  ║
║  └──────────────────────────────────────────────────────────────────────┘  ║
╚════════════════════════════════════════════════════════════════════════════╝
                                    ↓
╔════════════════════════════════════════════════════════════════════════════╗
║                      API & PROTOCOL LAYER                                  ║
║  ┌──────────────────────────────────────────────────────────────────────┐  ║
║  │  gRPC Service  │  Protocol Buffers  │  Session Management           │  ║
║  │  ExecuteSQL()  │  GetMetadata()     │  Connection Pooling           │  ║
║  └──────────────────────────────────────────────────────────────────────┘  ║
╚════════════════════════════════════════════════════════════════════════════╝
                                    ↓
╔════════════════════════════════════════════════════════════════════════════╗
║                    QUERY PROCESSING LAYER                                  ║
║  ┌──────────────────────────────────────────────────────────────────────┐  ║
║  │  ┌─────────────────────────────────────────────────────────────┐    │  ║
║  │  │  SQL Parser & Lexer                                         │    │  ║
║  │  │  ├─ Statement Parser (DDL, DML, TCL)                       │    │  ║
║  │  │  ├─ Expression Parser (WHERE clauses)                      │    │  ║
║  │  │  └─ Lexer/Tokenizer                                        │    │  ║
║  │  └─────────────────────────────────────────────────────────────┘    │  ║
║  │                              ↓                                       │  ║
║  │  ┌─────────────────────────────────────────────────────────────┐    │  ║
║  │  │  Query Planner & Optimizer                                 │    │  ║
║  │  │  ├─ Plan Builder (SELECT, INSERT, UPDATE, DELETE)         │    │  ║
║  │  │  ├─ Index Selection                                        │    │  ║
║  │  │  ├─ Join Ordering                                          │    │  ║
║  │  │  └─ Cost Estimation                                        │    │  ║
║  │  └─────────────────────────────────────────────────────────────┘    │  ║
║  │                              ↓                                       │  ║
║  │  ┌─────────────────────────────────────────────────────────────┐    │  ║
║  │  │  Query Executor                                             │    │  ║
║  │  │  ├─ Scan Operators (Sequential, Index)                     │    │  ║
║  │  │  ├─ Filter & Projection                                    │    │  ║
║  │  │  ├─ Join Operators                                         │    │  ║
║  │  │  ├─ Aggregate Functions                                    │    │  ║
║  │  │  └─ DML Operators (INSERT, UPDATE, DELETE)                │    │  ║
║  │  └─────────────────────────────────────────────────────────────┘    │  ║
║  └──────────────────────────────────────────────────────────────────────┘  ║
╚════════════════════════════════════════════════════════════════════════════╝
                                    ↓
╔════════════════════════════════════════════════════════════════════════════╗
║                    SCHEMA & METADATA LAYER                                 ║
║  ┌──────────────────────────────────────────────────────────────────────┐  ║
║  │  Schema Catalog                                                      │  ║
║  │  ├─ Table Definitions (columns, types, constraints)                 │  ║
║  │  ├─ Index Metadata                                                  │  ║
║  │  ├─ Primary Key & Foreign Key Constraints                           │  ║
║  │  ├─ Type System (INT, TEXT, BOOL, FLOAT, NULL)                     │  ║
║  │  └─ Schema Persistence (System Tables)                              │  ║
║  └──────────────────────────────────────────────────────────────────────┘  ║
╚════════════════════════════════════════════════════════════════════════════╝
                                    ↓
╔════════════════════════════════════════════════════════════════════════════╗
║                  TRANSACTION & CONCURRENCY LAYER                           ║
║  ┌──────────────────────────────────────────────────────────────────────┐  ║
║  │  Transaction Manager                                                 │  ║
║  │  ├─ Transaction State Management (ACTIVE, COMMITTED, ABORTED)       │  ║
║  │  ├─ ACID Guarantees                                                 │  ║
║  │  └─ Transaction Isolation Levels                                    │  ║
║  │                                                                      │  ║
║  │  Concurrency Control                                                │  ║
║  │  ├─ Two-Phase Locking (2PL)                                         │  ║
║  │  │  ├─ Shared Locks (Read)                                          │  ║
║  │  │  ├─ Exclusive Locks (Write)                                      │  ║
║  │  │  └─ Lock Manager                                                 │  ║
║  │  ├─ Deadlock Detection & Resolution                                 │  ║
║  │  └─ Lock Timeout & Retry Logic                                      │  ║
║  │                                                                      │  ║
║  │  Session Management                                                 │  ║
║  │  ├─ Session State (Active Transactions, Locks)                      │  ║
║  │  ├─ Session Expiry & Cleanup                                        │  ║
║  │  └─ Connection Pooling                                              │  ║
║  └──────────────────────────────────────────────────────────────────────┘  ║
╚════════════════════════════════════════════════════════════════════════════╝
                                    ↓
╔════════════════════════════════════════════════════════════════════════════╗
║                    STORAGE ENGINE LAYER                                    ║
║  ┌──────────────────────────────────────────────────────────────────────┐  ║
║  │  ┌─────────────────────────────────────────────────────────────┐    │  ║
║  │  │  Access Methods (Index Structures)                          │    │  ║
║  │  │  ├─ B+Tree Index                                            │    │  ║
║  │  │  │  ├─ Node Structure (Internal, Leaf)                     │    │  ║
║  │  │  │  ├─ Search Operations                                   │    │  ║
║  │  │  │  ├─ Insert/Delete with Rebalancing                     │    │  ║
║  │  │  │  └─ Range Scans                                         │    │  ║
║  │  │  ├─ Hash Index (Future)                                    │    │  ║
║  │  │  └─ Clustered Index (Primary Key)                          │    │  ║
║  │  └─────────────────────────────────────────────────────────────┘    │  ║
║  │                              ↓                                       │  ║
║  │  ┌─────────────────────────────────────────────────────────────┐    │  ║
║  │  │  Key-Value Store                                            │    │  ║
║  │  │  ├─ Table Data Storage (KV pairs)                           │    │  ║
║  │  │  ├─ Index Data Storage                                      │    │  ║
║  │  │  ├─ Get/Put/Delete Operations                              │    │  ║
║  │  │  └─ Range Queries                                           │    │  ║
║  │  └─────────────────────────────────────────────────────────────┘    │  ║
║  │                              ↓                                       │  ║
║  │  ┌─────────────────────────────────────────────────────────────┐    │  ║
║  │  │  Buffer Pool & Page Cache                                   │    │  ║
║  │  │  ├─ Page Frames (Fixed-size memory blocks)                 │    │  ║
║  │  │  ├─ LRU Replacement Policy                                 │    │  ║
║  │  │  ├─ Pin/Unpin Operations                                   │    │  ║
║  │  │  ├─ Dirty Page Tracking                                    │    │  ║
║  │  │  └─ Page Eviction & Flushing                               │    │  ║
║  │  └─────────────────────────────────────────────────────────────┘    │  ║
║  │                              ↓                                       │  ║
║  │  ┌─────────────────────────────────────────────────────────────┐    │  ║
║  │  │  Write-Ahead Logging (WAL)                                  │    │  ║
║  │  │  ├─ Log Records (REDO, UNDO)                               │    │  ║
║  │  │  ├─ Log Buffer                                             │    │  ║
║  │  │  ├─ Fsync Policy (Immediate, Batch, Delayed)              │    │  ║
║  │  │  ├─ Checkpointing                                          │    │  ║
║  │  │  └─ Recovery (Redo/Undo)                                   │    │  ║
║  │  └─────────────────────────────────────────────────────────────┘    │  ║
║  │                              ↓                                       │  ║
║  │  ┌─────────────────────────────────────────────────────────────┐    │  ║
║  │  │  File Manager & Page Format                                 │    │  ║
║  │  │  ├─ Table Files (Heap Files)                               │    │  ║
║  │  │  ├─ Index Files                                            │    │  ║
║  │  │  ├─ Log Files                                              │    │  ║
║  │  │  ├─ Page Format (Header, Data, Footer)                     │    │  ║
║  │  │  ├─ Serialization/Deserialization                          │    │  ║
║  │  │  └─ File I/O Operations                                    │    │  ║
║  │  └─────────────────────────────────────────────────────────────┘    │  ║
║  └──────────────────────────────────────────────────────────────────────┘  ║
╚════════════════════════════════════════════════════════════════════════════╝
                                    ↓
╔════════════════════════════════════════════════════════════════════════════╗
║                    DISK & PERSISTENCE LAYER                                ║
║  ┌──────────────────────────────────────────────────────────────────────┐  ║
║  │  Operating System File System                                        │  ║
║  │  ├─ Data Directory Structure                                        │  ║
║  │  ├─ File Handles & Descriptors                                      │  ║
║  │  ├─ Read/Write System Calls                                         │  ║
║  │  └─ Fsync for Durability                                            │  ║
║  │                                                                      │  ║
║  │  Physical Storage (Disk)                                            │  ║
║  │  ├─ Blocks & Sectors                                                │  ║
║  │  ├─ Persistent Data Files                                           │  ║
║  │  ├─ Log Files                                                       │  ║
║  │  └─ Checkpoint Files                                                │  ║
║  └──────────────────────────────────────────────────────────────────────┘  ║
╚════════════════════════════════════════════════════════════════════════════╝
                                    ↓
╔════════════════════════════════════════════════════════════════════════════╗
║                  OBSERVABILITY & OPERATIONS LAYER                          ║
║  ┌──────────────────────────────────────────────────────────────────────┐  ║
║  │  Logging                                                             │  ║
║  │  ├─ Structured Logging (JSON)                                       │  ║
║  │  ├─ Log Levels (DEBUG, INFO, WARN, ERROR)                           │  ║
║  │  ├─ Context Propagation (Trace IDs)                                 │  ║
║  │  └─ Log Rotation & Retention                                        │  ║
║  │                                                                      │  ║
║  │  Metrics & Monitoring                                               │  ║
║  │  ├─ Query Performance Metrics                                       │  ║
║  │  ├─ Storage Metrics (Pages, Cache Hit Rate)                         │  ║
║  │  ├─ Transaction Metrics (Throughput, Latency)                       │  ║
║  │  ├─ Lock Contention Metrics                                         │  ║
║  │  └─ System Health Metrics                                           │  ║
║  │                                                                      │  ║
║  │  Tracing & Debugging                                                │  ║
║  │  ├─ Distributed Tracing (Trace Context)                             │  ║
║  │  ├─ Span Creation & Propagation                                     │  ║
║  │  ├─ Performance Profiling                                           │  ║
║  │  └─ Debug Logging                                                   │  ║
║  │                                                                      │  ║
║  │  Configuration Management                                           │  ║
║  │  ├─ Runtime Configuration                                           │  ║
║  │  ├─ Buffer Pool Size                                                │  ║
║  │  ├─ Lock Timeout Settings                                           │  ║
║  │  ├─ WAL Fsync Policy                                                │  ║
║  │  └─ Log Level Configuration                                         │  ║
║  └──────────────────────────────────────────────────────────────────────┘  ║
╚════════════════════════════════════════════════════════════════════════════╝
```

---

## Layer Descriptions

### 1. **Client Layer**
- **Interactive CLI Client**: Command-line interface for users
- **gRPC Client**: Programmatic access via gRPC
- **HTTP Client**: Future REST API support

### 2. **API & Protocol Layer**
- **gRPC Service**: Remote procedure calls
- **Protocol Buffers**: Message serialization
- **Session Management**: User sessions and connections
- **Connection Pooling**: Reuse connections efficiently

### 3. **Query Processing Layer**
- **SQL Parser & Lexer**: Parse SQL into AST
- **Query Planner**: Build execution plans
- **Query Executor**: Execute plans and return results

### 4. **Schema & Metadata Layer**
- **Schema Catalog**: Table and index definitions
- **Type System**: Data type definitions and validation
- **Constraint Management**: Primary keys, foreign keys, unique constraints
- **System Tables**: Persistent schema storage

### 5. **Transaction & Concurrency Layer**
- **Transaction Manager**: ACID guarantees
- **Concurrency Control**: Two-phase locking
- **Lock Manager**: Manage locks and detect deadlocks
- **Session Management**: Track active transactions

### 6. **Storage Engine Layer**
- **Access Methods**: B+Tree, Hash indexes
- **Key-Value Store**: Core data storage
- **Buffer Pool**: In-memory page cache
- **Write-Ahead Logging**: Durability and recovery
- **File Manager**: Disk I/O and page format

### 7. **Disk & Persistence Layer**
- **File System**: OS-level file operations
- **Physical Storage**: Actual disk storage

### 8. **Observability & Operations Layer**
- **Logging**: Structured logging with context
- **Metrics**: Performance and health monitoring
- **Tracing**: Distributed tracing support
- **Configuration**: Runtime configuration management

---

## Data Flow Example: SELECT Query

```
User Input: "SELECT id, name FROM users WHERE id = 1"
    ↓
[Client Layer] CLI receives input
    ↓
[API Layer] gRPC ExecuteSQL() call
    ↓
[Query Processing] 
    ├─ Parser: Parse SQL → AST
    ├─ Planner: Build execution plan (use index on id)
    └─ Executor: Execute plan
        ├─ Lock: Acquire read lock on users table
        ├─ Index Scan: Find row with id=1 using B+Tree
        ├─ Buffer Pool: Load page from cache or disk
        ├─ Filter: Apply WHERE clause
        ├─ Project: Select id, name columns
        └─ Unlock: Release read lock
    ↓
[Storage Engine]
    ├─ B+Tree: Search for key
    ├─ Buffer Pool: Check cache, load if needed
    ├─ File Manager: Read page from disk if needed
    └─ WAL: Log read operation (if needed)
    ↓
[Disk] Read data from disk
    ↓
[Result] Return rows to client
    ↓
[Observability] Log query, record metrics
```

---

## Data Flow Example: INSERT Query

```
User Input: "INSERT INTO users VALUES (1, 'Alice')"
    ↓
[Client Layer] CLI receives input
    ↓
[API Layer] gRPC ExecuteSQL() call
    ↓
[Query Processing]
    ├─ Parser: Parse SQL → AST
    ├─ Planner: Build insert plan
    └─ Executor: Execute plan
        ├─ Lock: Acquire exclusive lock on users table
        ├─ Validate: Check constraints (PK, NOT NULL, etc.)
        ├─ Find Space: Locate page with free space
        ├─ Insert Row: Add row to page
        ├─ Update Indexes: Update all indexes
        └─ Unlock: Release exclusive lock
    ↓
[Transaction Layer]
    ├─ Log: Write REDO log record
    └─ Commit: Mark transaction as committed
    ↓
[Storage Engine]
    ├─ Buffer Pool: Mark page as dirty
    ├─ WAL: Write log record to disk
    └─ File Manager: Flush dirty pages (eventually)
    ↓
[Disk] Write data to disk
    ↓
[Result] Return success to client
    ↓
[Observability] Log insert, record metrics
```

---

## Component Dependencies

```
Client Layer
    ↓
API Layer
    ↓
Query Processing Layer
    ├─ depends on → Schema & Metadata Layer
    ├─ depends on → Transaction & Concurrency Layer
    └─ depends on → Storage Engine Layer
    ↓
Transaction & Concurrency Layer
    ├─ depends on → Storage Engine Layer
    └─ depends on → Observability Layer
    ↓
Storage Engine Layer
    ├─ depends on → Schema & Metadata Layer
    ├─ depends on → Disk & Persistence Layer
    └─ depends on → Observability Layer
    ↓
Disk & Persistence Layer
    ↓
Observability & Operations Layer
```

---

## Key Architectural Principles

### 1. **Layered Architecture**
- Each layer has clear responsibilities
- Layers communicate through well-defined interfaces
- Lower layers don't depend on higher layers

### 2. **Separation of Concerns**
- Query processing separate from storage
- Concurrency control separate from transaction management
- Logging separate from business logic

### 3. **ACID Guarantees**
- **Atomicity**: Transaction manager + WAL
- **Consistency**: Schema validation + constraints
- **Isolation**: Concurrency control (2PL)
- **Durability**: WAL + fsync

### 4. **Performance Optimization**
- Buffer pool minimizes disk I/O
- Index selection for fast lookups
- Query planning for efficient execution
- Lock optimization to reduce contention

### 5. **Observability**
- Structured logging at each layer
- Metrics for monitoring
- Tracing for debugging
- Configuration for tuning

---

## Future Enhancements

### Short Term
- [ ] Query optimization (cost-based planning)
- [ ] More index types (Hash, Bitmap)
- [ ] Aggregate functions (COUNT, SUM, AVG)
- [ ] GROUP BY and HAVING
- [ ] ORDER BY and LIMIT

### Medium Term
- [ ] JOIN operations (Nested Loop, Hash Join, Sort-Merge)
- [ ] Subqueries and CTEs
- [ ] Window functions
- [ ] Replication and failover
- [ ] Distributed transactions

### Long Term
- [ ] Columnar storage
- [ ] Query vectorization
- [ ] Machine learning integration
- [ ] Sharding and partitioning
- [ ] Advanced concurrency (MVCC)

---

## Summary

This architecture provides a complete, layered RDBMS implementation with:
- ✅ Clear separation of concerns
- ✅ ACID guarantees
- ✅ Efficient query processing
- ✅ Robust storage engine
- ✅ Comprehensive observability
- ✅ Extensible design for future features

Each layer can be developed, tested, and optimized independently while maintaining clear interfaces with other layers.
