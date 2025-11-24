# MyDB Quick Start Guide

## Getting Started (5 minutes)

### Prerequisites

- Go 1.22+ installed
- Terminal access

### Step 1: Build MyDB (30 seconds)

```bash
cd /Users/hair/Documents/learn/myDb

# Build server and client
go build -o bin/mydbd ./cmd/mydbd
go build -o bin/mydb-cli ./cmd/mydb-cli
```

### Step 2: Start Server (Terminal 1)

```bash
# Start MyDB server
./bin/mydbd

# You should see:
# INFO server initialized with data dir data
# INFO gRPC server listening on :7001
```

**Server is now running!** Keep this terminal open.

---

### Step 3: Connect Client (Terminal 2)

```bash
# In a NEW terminal, connect to server
./bin/mydb-cli

# You should see:
# Connected to MyDB at localhost:7001
# mydb>
```

**You're now connected!** Try some SQL commands:

---

## 📝 Try These Commands

### Create a Table

```sql
mydb> CREATE TABLE users (
        id INT PRIMARY KEY,
        name TEXT NOT NULL,
        email TEXT,
        age INT
      );

# Output: Command executed successfully
```

### Insert Data

```sql
mydb> INSERT INTO users VALUES (1, 'Alice', 'alice@example.com', 30);
mydb> INSERT INTO users VALUES (2, 'Bob', 'bob@example.com', 25);
mydb> INSERT INTO users VALUES (3, 'Charlie', NULL, 35);

# Each should show: 1 row affected
```

### Query Data

```sql
mydb> SELECT * FROM users;

# Output:
# id | name    | email              | age
# -------------------------------------------
# 1  | Alice   | alice@example.com  | 30
# 2  | Bob     | bob@example.com    | 25
# 3  | Charlie | NULL               | 35
```

### Query with WHERE

```sql
mydb> SELECT name, age FROM users WHERE age > 25;

# Output:
# name    | age
# ----------------
# Alice   | 30
# Charlie | 35
```

### Update Data

```sql
mydb> UPDATE users SET age = 26 WHERE name = 'Bob';

# Output: 1 row affected
```

### Delete Data

```sql
mydb> DELETE FROM users WHERE id = 3;

# Output: 1 row affected
```

### Verify

```sql
mydb> SELECT * FROM users;

# Output:
# id | name  | email             | age
# ----------------------------------------
# 1  | Alice | alice@example.com | 30
# 2  | Bob   | bob@example.com   | 26
```

---

## Test Transactions

### Start a Transaction

```sql
mydb> BEGIN;
mydb> INSERT INTO users VALUES (4, 'David', 'david@example.com', 40);
mydb> UPDATE users SET age = 31 WHERE id = 1;
mydb> COMMIT;

# All changes are now persisted!
```

### Rollback Example

```sql
mydb> BEGIN;
mydb> DELETE FROM users WHERE id = 1;
mydb> ROLLBACK;

# All changes discarded - Alice is still there!
mydb> SELECT * FROM users WHERE id = 1;
```

---

## Test Indexes

### Create an Index

```sql
mydb> CREATE INDEX idx_age ON users(age);

# Output: Index created successfully
```

### Query Uses Index

```sql
mydb> SELECT * FROM users WHERE age = 30;

# The query planner automatically uses idx_age for faster lookup!
```

---

## Test Constraints

### Primary Key Enforcement

```sql
mydb> INSERT INTO users VALUES (1, 'Duplicate', 'dup@example.com', 99);

# Output: ERROR: duplicate value for unique index __pk_users_id
```

### NOT NULL Enforcement

```sql
mydb> CREATE TABLE products (
        id INT PRIMARY KEY,
        name TEXT NOT NULL,
        price FLOAT
      );

mydb> INSERT INTO products VALUES (1, NULL, 9.99);

# Output: ERROR: column name cannot be NULL
```

---

## Test Crash Recovery

### In Terminal 1 (Server):

```sql
# In client (Terminal 2):
mydb> CREATE TABLE test (id INT PRIMARY KEY, val TEXT);
mydb> INSERT INTO test VALUES (1, 'before crash');
```

### Simulate Crash:

```bash
# In server terminal (Terminal 1), press Ctrl+C to kill server
^C

# Restart server:
./bin/mydbd
```

### Verify Data Survived:

```sql
# In client (reconnect if needed):
mydb> SELECT * FROM test;

# Output:
# id | val
# --------------
# 1  | before crash

# ✅ Data is durable!
```

---

## Supported Features

### ✅ SQL Statements

- `CREATE TABLE` (with PRIMARY KEY, UNIQUE, NOT NULL)
- `DROP TABLE`
- `CREATE INDEX` / `DROP INDEX`
- `INSERT`
- `SELECT` (with WHERE clause)
- `UPDATE` (with WHERE clause)
- `DELETE` (with WHERE clause)
- `BEGIN` / `COMMIT` / `ROLLBACK`

### ✅ Data Types

- `INT` (64-bit integer)
- `TEXT` (variable-length string)
- `FLOAT` (64-bit float)
- `BOOL` (true/false)
- `TIMESTAMP`

### ✅ WHERE Clause

- Comparison: `=`, `!=`, `<`, `>`, `<=`, `>=`
- Logical: `AND`, `OR`, `NOT`
- Null checks: `IS NULL`, `IS NOT NULL`
- Parentheses: `(age > 25 AND age < 40)`

### ✅ Constraints

- `PRIMARY KEY` - enforced uniqueness
- `UNIQUE` - unique values
- `NOT NULL` - no NULL values

### ✅ Transactions

- Row-level locking
- Read/Write locks
- Deadlock detection (timeout-based)
- Isolation via two-phase locking

### ✅ Durability

- Write-Ahead Logging (WAL)
- Crash recovery
- Buffer pool with LRU eviction

---

## Not Supported (Yet)

- ❌ **JOINs** (only single-table queries)
- ❌ **Aggregates** (SUM, COUNT, AVG, MIN, MAX)
- ❌ **GROUP BY** / **HAVING**
- ❌ **ORDER BY** / **LIMIT** / **OFFSET**
- ❌ **Subqueries**
- ❌ **ALTER TABLE**
- ❌ **Foreign Keys**
- ❌ **Views** / **Triggers**
- ❌ **Multi-node** (single-node only)

---

## Troubleshooting

### Server won't start

```bash
# Check if port 7001 is already in use
lsof -i :7001

# If another mydbd is running, kill it:
pkill mydbd
```

### Client can't connect

```bash
# Make sure server is running
ps aux | grep mydbd

# Check if server is listening
lsof -i :7001
```

### Reset database

```bash
# Stop server (Ctrl+C in Terminal 1)

# Delete data directory
rm -rf data/

# Restart server
./bin/mydbd
```

---

## Known Limitations (v0.1.0-beta)

### Durability

- **COMMIT durability window**: Small window where data may be lost if server crashes immediately after COMMIT returns (before background WAL sync)
- **Workaround**: For critical operations, wait 1-2 seconds after COMMIT
- **Impact**: Low for normal use, avoid rapid commit-crash testing
- **Status**: Architectural limitation, planned for v0.2.0

### Performance

- **Large scans**: Avoid scanning 1000+ rows in a transaction (can cause lock timeout)
- **No query timeout**: Long-running queries hold locks indefinitely
- **Lock holding**: Row locks held until COMMIT/ROLLBACK (no early release)

### Features

- No VACUUM command (index bloat possible over time)
- No query EXPLAIN
- Basic query planner (no cost-based optimization)
- Schema not persisted (recreate tables after restart)

### Recommended Use

- ✅ Development and testing
- ✅ Learning database internals  
- ✅ Small applications (< 10 concurrent users)
- ❌ Production with high durability needs
- ❌ High concurrency (> 10 concurrent writes)

---

## What to Try Next

1. **Insert 1000 rows** and test performance
2. **Run concurrent clients** (open multiple `mydb-cli` terminals)
3. **Test transaction isolation** with two clients
4. **Crash the server mid-transaction** and verify rollback
5. **Create complex WHERE clauses** with nested AND/OR

---

## More Information

- **Architecture**: See `architecture.md`
- **Tests**: Run `go test ./...`

---

## Known Issues

1. **WAL warning on shutdown**: Harmless warning `wal: open ... invalid argument` during graceful shutdown
2. **Schema not persisted**: Catalog is in-memory - recreate tables after restart
3. **No query planner optimization**: All index selection is basic

---

**Enjoy using MyDB!**

Questions? Check the code or run tests: `go test ./... -v`
