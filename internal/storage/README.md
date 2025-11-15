# Storage Engine Overview

This document captures the foundational storage-engine concepts that shape the myDb implementation roadmap. It consolidates the reference material into practical guidance for building the Go-based storage stack.

## 1. Goals of the Storage Engine

A database storage engine must:

- **Persist data durably** on disk even across crashes (the D in ACID).
- **Serve queries quickly** despite disk being orders of magnitude slower than RAM.
- **Organize records** so different workloads (point lookups, ranges, scans) remain efficient.
- **Support transactions, concurrency, and recovery** through well-defined interfaces.

These goals naturally lead to a layered architecture rather than direct disk access.

## 2. High-Level Architecture

```bash
Application → Query Processor → Storage Engine Layers → Disk
```

| Layer | Responsibility |
| --- | --- |
| Record/Data Model | Represent tuples/rows/fields consistently. |
| File Manager | Owns the physical files per table/index. |
| Page Manager | Slices files into fixed-size pages (4–16 KB). |
| Buffer Manager | Caches hot pages in RAM, handles replacement. |
| Access Methods | Provide logical structures (B+Tree, hash, heap). |
| Disk Storage | Hardware blocks/sectors where bytes live durably. |

## 3. Disk Concepts

- **Sector:** hardware minimum I/O unit (512 B or 4 KB). DBMS rarely touches sectors directly.
- **Block:** OS-level grouping of sectors (e.g., 4 KB, 8 KB) used for read/write system calls.
- **Page:** DBMS unit of transfer. All tables and indexes are built from pages, which are loaded into buffer frames. Query cost models reason about page I/O counts.

## 4. File Organizations

| Organization | Strengths | Weaknesses |
| --- | --- | --- |
| Heap File | Fast inserts, simple appends. | Point lookups require full scans. |
| Sorted File | Efficient range scans thanks to ordering. | Inserts/updates expensive (re-sorting). |
| Hash File | O(1) equality lookups. | No natural support for range queries. |

myDb will begin with heap files + B+Tree indexes, then add sorted/hash variants as workloads demand.

## 5. Access Methods

- **B+Tree Indexes** – ubiquitous due to O(log n) search/insert/delete, good cache locality, and natural range scan support. The planner uses them as access paths for selections and joins.
- **Hash Indexes** – excel at equality predicates (e.g., `WHERE id = …`) but cannot satisfy range scans without scanning every bucket.

## 6. Buffer Manager

The buffer pool keeps frequently touched pages in RAM to minimize disk I/O.

- Tracks page frames, pin counts, and dirty state.
- Uses replacement policies (LRU, Clock, etc.) to evict cold pages.
- Drives overall cost models: every join/scan aims to minimize page faults and maximize hit ratios.

## 7. Write-Ahead Logging (WAL)

Durability and crash recovery hinge on WAL:

1. Append the change record to the log on disk.
2. Only then flush dirty data pages.

On crash, WAL enables **redo** (reapply committed changes) and **undo** (rollback incomplete transactions), keeping ACID guarantees intact.

## 8. Space Management

Common mechanisms:

- **Page directories** for quick page location.
- **Free-space maps** to find pages with room for new tuples.
- **Extents** (groups of pages) for efficient sequential allocation.
- **Segments** to represent logical objects (tables, indexes).

## 9. Clustered vs Non-Clustered Indexes

| Type | Characteristics |
| --- | --- |
| Clustered | Table data physically ordered by the index key; great for range scans; one per table. |
| Non-Clustered | Separate structure storing key → pointer/page references; multiple per table; ideal for selective lookups across different columns. |

## 10. Storage Flow Example

Example query: `SELECT * FROM employee WHERE ssn = '123';`

1. Parser builds AST; optimizer picks an access path (B+Tree on `ssn`).
2. Buffer manager loads the B+Tree root page.
3. Access method traverses down to the leaf and fetches the tuple page.
4. Result is returned to the executor.
5. For updates, WAL is written first, then the modified page flushes later.

## 11. Deep-Dive Topics

Areas to explore further (future docs under `internal/storage/`):

1. **Buffer Pool internals** – page tables, dirty tracking, flush strategies, replacement tuning.
2. **WAL & Recovery** – undo/redo logging, ARIES-style algorithms, checkpointing.
3. **B+Tree mechanics** – node format, split/merge, clustering strategies, I/O cost in joins.
4. **File organizations** – comparative analysis and workload-based selection.
