# Query Planner Architecture & Implementation Guide

## Overview

The Query Planner is the intelligent core of the database system, transforming parsed SQL Abstract Syntax Trees (ASTs) into optimized physical execution plans. This document provides a comprehensive guide to the architecture, design decisions, and implementation strategy.

## System Architecture

### High-Level Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        SQL Query                                 │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    SQL Parser & Lexer                            │
│  (Transforms SQL string → AST)                                  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    QUERY PLANNER                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ 1. Semantic Analyzer                                     │   │
│  │    - Validate table/column references                    │   │
│  │    - Check data type compatibility                       │   │
│  │    - Detect ambiguous references                         │   │
│  └──────────────────────────────────────────────────────────┘   │
│                             │                                    │
│                             ▼                                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ 2. Query Optimizer                                       │   │
│  │    - Apply optimization rules                            │   │
│  │    - Select best index                                   │   │
│  │    - Determine join order                                │   │
│  │    - Push down predicates                                │   │
│  └──────────────────────────────────────────────────────────┘   │
│                             │                                    │
│                             ▼                                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ 3. Cost Estimator                                        │   │
│  │    - Estimate cardinality                                │   │
│  │    - Estimate I/O costs                                  │   │
│  │    - Estimate CPU costs                                  │   │
│  │    - Compare plan alternatives                           │   │
│  └──────────────────────────────────────────────────────────┘   │
│                             │                                    │
│                             ▼                                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ 4. Plan Generator                                        │   │
│  │    - Create operator nodes                               │   │
│  │    - Build plan tree                                     │   │
│  │    - Validate plan correctness                           │   │
│  └──────────────────────────────────────────────────────────┘   │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Physical Execution Plan                       │
│  (Tree of operators ready for execution)                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Query Executor                                │
│  (Executes plan and returns results)                            │
└─────────────────────────────────────────────────────────────────┘
```

### Component Interactions

```
┌──────────────────────────────────────────────────────────────────┐
│                    Query Planner                                  │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Semantic Analyzer                                       │    │
│  │ ├─ Validates against Schema Catalog                     │    │
│  │ └─ Returns validated AST or error                       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                          │                                       │
│                          ▼                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Query Optimizer                                         │    │
│  │ ├─ Applies optimization rules                           │    │
│  │ ├─ Queries Schema Catalog for indexes                   │    │
│  │ ├─ Queries Cost Estimator for alternatives              │    │
│  │ └─ Returns optimized plan shape                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                          │                                       │
│                          ▼                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Cost Estimator                                          │    │
│  │ ├─ Queries Schema Catalog for statistics                │    │
│  │ ├─ Estimates cardinality                                │    │
│  │ ├─ Estimates I/O costs                                  │    │
│  │ ├─ Estimates CPU costs                                  │    │
│  │ └─ Returns cost estimates                               │    │
│  └─────────────────────────────────────────────────────────┘    │
│                          │                                       │
│                          ▼                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Plan Generator                                          │    │
│  │ ├─ Creates operator nodes                               │    │
│  │ ├─ Validates plan against Schema Catalog                │    │
│  │ ├─ Embeds cost estimates                                │    │
│  │ └─ Returns physical plan                                │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Dependencies:                                                    │
│  ├─ Schema Catalog (table/index/column metadata)                 │
│  ├─ Table Statistics (row counts, distributions)                 │
│  └─ Expression Evaluator (for predicate analysis)                │
└──────────────────────────────────────────────────────────────────┘
```

## Plan Node Hierarchy

```
Operator (interface)
├── TableAccessOp
│   ├── SeqScanOp
│   │   └── Full table scan, sequential I/O
│   └── IndexScanOp
│       └── Index-based scan, random I/O
├── FilterOp
│   └── Apply WHERE predicates
├── ProjectionOp
│   └── Select specific columns
├── JoinOp (interface)
│   ├── NestedLoopJoinOp
│   │   └── Cartesian product with filter
│   ├── HashJoinOp (future)
│   │   └── Hash-based join
│   └── SortMergeJoinOp (future)
│       └── Sort-based join
├── SortOp
│   └── ORDER BY implementation
├── AggregateOp
│   └── GROUP BY and aggregate functions
├── LimitOp
│   └── LIMIT and OFFSET
├── DML Operations
│   ├── InsertOp
│   ├── UpdateOp
│   ├── DeleteOp
│   └── CreateTableOp
└── DDL Operations
    ├── DropTableOp
    ├── CreateIndexOp
    └── DropIndexOp
```

## Cost Estimation Model

### Cardinality Estimation

```
Base Cardinality = Table Row Count (from statistics)

For Equality Predicate (col = value):
  Selectivity = 1 / DistinctValues(col)
  Output Rows = Base Cardinality * Selectivity

For Range Predicate (col < value):
  Selectivity = EstimateRangeSelectivity(col, value)
  Output Rows = Base Cardinality * Selectivity

For AND Predicate (pred1 AND pred2):
  Selectivity = Selectivity(pred1) * Selectivity(pred2)
  Output Rows = Base Cardinality * Selectivity

For OR Predicate (pred1 OR pred2):
  Selectivity = MIN(1.0, Selectivity(pred1) + Selectivity(pred2))
  Output Rows = Base Cardinality * Selectivity

For Join (table1 JOIN table2 ON condition):
  Output Rows = Cardinality(table1) * Cardinality(table2) * JoinSelectivity
```

### I/O Cost Estimation

```
Sequential Scan:
  Pages = CEIL(TableSize / PageSize)
  IOCost = Pages * SequentialReadCost

Index Scan:
  IndexPages = EstimateIndexPages(index, keyRange)
  DataPages = CEIL(RowsToFetch * AvgRowSize / PageSize)
  IOCost = (IndexPages + DataPages) * RandomReadCost

Access Path Selection:
  IF IndexScanCost < SeqScanCost THEN
    Use Index Scan
  ELSE
    Use Sequential Scan
  END IF
```

### CPU Cost Estimation

```
Filter Operation:
  CPUCost = RowCount * PredicateEvaluationCost

Projection Operation:
  CPUCost = RowCount * ColumnCount * ColumnExtractionCost

Sort Operation:
  CPUCost = RowCount * LOG2(RowCount) * ComparisonCost

Nested Loop Join:
  CPUCost = LeftRows * RightRows * ComparisonCost

Aggregate Operation:
  CPUCost = RowCount * AggregateFunctionCount * AggregationCost
```

## Optimization Strategies

### 1. Index Selection

```
Algorithm: SelectBestIndex(table, filter)
  1. Extract indexed columns from filter
  2. Find all indexes covering those columns
  3. For each candidate index:
     - Estimate I/O cost
     - Compare with sequential scan cost
  4. Select index with lowest cost
  5. If no index beats sequential scan, use sequential scan
```

### 2. Join Ordering

```
Algorithm: OrderJoins(tables, joinConditions)
  1. Calculate cardinality for each table
  2. Sort tables by cardinality (ascending)
  3. Join smallest table first
  4. For each remaining table:
     - Find join condition
     - Add to join tree
  5. Return join order
```

### 3. Predicate Pushdown

```
Algorithm: PushdownPredicates(plan)
  1. For each filter after a join:
     - Analyze filter predicate
     - If predicate references only one table:
       - Move filter before join
       - Recalculate cardinality
  2. Repeat until no more filters can be pushed down
```

## Data Structures

### Cost Structure

```go
type Cost struct {
    IOCost    float64  // I/O operations cost
    CPUCost   float64  // CPU operations cost
    TotalCost float64  // Total estimated cost
}
```

### Operator Interface

```go
type Operator interface {
    Name() string                    // Operator type name
    EstimatedCost() Cost             // Estimated execution cost
    EstimatedRows() int64            // Estimated output rows
    OutputColumns() []string         // Output column names
}
```

### Plan Node Examples

```go
// Sequential Scan
type SeqScanOp struct {
    Schema          string
    Table           string
    Columns         []string
    Filter          expr.Expr
    EstimatedCost   Cost
    EstimatedRowCnt int64
}

// Index Scan
type IndexScanOp struct {
    Schema          string
    Table           string
    IndexName       string
    Columns         []string
    Filter          expr.Expr
    StartKey        []byte
    EndKey          []byte
    IsRangeScan     bool
    EstimatedCost   Cost
    EstimatedRowCnt int64
}

// Filter
type FilterOp struct {
    Child           Operator
    Predicate       expr.Expr
    EstimatedCost   Cost
    EstimatedRowCnt int64
}

// Join
type NestedLoopJoinOp struct {
    Left            Operator
    Right           Operator
    Condition       expr.Expr
    Type            JoinType
    EstimatedCost   Cost
    EstimatedRowCnt int64
}
```

## Integration Points

### Upstream: Parser

**Input**: `parser.AST` structure
- SELECT, INSERT, UPDATE, DELETE, CREATE TABLE, CREATE INDEX, DROP TABLE, DROP INDEX

**Extraction**:
- Table references from FROM clause
- Column references from SELECT/WHERE/ORDER BY
- Predicates from WHERE clause
- Join conditions from ON clause
- Aggregate functions from GROUP BY/SELECT

### Downstream: Executor

**Output**: `Physical Plan` with root `Operator`

**Plan Properties**:
- Each operator has Name, EstimatedCost, EstimatedRows, OutputColumns
- Plan is ready for immediate execution
- All metadata needed for data access is embedded

### Schema Catalog

**Queries**:
- Table definitions (columns, types, constraints)
- Index definitions (columns, uniqueness)
- Table statistics (row count, size, column distributions)

**Usage**:
- Validation of table/column references
- Cost estimation using statistics
- Index selection

## Testing Strategy

### Property-Based Testing

Properties to validate:

1. **Semantic Validation Completeness**: Invalid references are rejected
2. **Plan Node Creation Correctness**: Appropriate operators for all query types
3. **Cardinality Estimation Accuracy**: Estimates within bounds (1 ≤ est ≤ table_rows)
4. **I/O Cost Estimation Consistency**: Index scan cost < sequential scan when selective
5. **CPU Cost Estimation Monotonicity**: Cost increases with row count
6. **Index Selection Optimality**: Selected index has lowest cost
7. **Join Ordering Cost Minimization**: Selected order has lowest cost
8. **Predicate Pushdown Correctness**: Pushed filters reduce intermediate cardinality
9. **Plan Validation Completeness**: Invalid plans are detected
10. **Error Message Specificity**: Errors include context
11. **Scalability**: Handles 100+ tables, 10+ joins
12. **Plan Completeness**: All tables, predicates, columns included
13. **Planning Determinism**: Same query produces identical plans

### Unit Testing

- Test each operator type creation
- Test cost estimation algorithms
- Test index selection logic
- Test join ordering logic
- Test predicate pushdown
- Test validation logic
- Test error handling

### Integration Testing

- End-to-end: Parse SQL → Plan → Execute
- Multi-table JOINs with various join types
- Complex WHERE clauses with AND/OR
- GROUP BY with HAVING
- ORDER BY with multiple keys
- LIMIT and OFFSET
- DML operations (INSERT, UPDATE, DELETE)
- DDL operations (CREATE/DROP TABLE/INDEX)

### Performance Testing

- Simple SELECT: < 10ms
- 5-table query: < 10ms
- 10-table query: < 50ms
- Complex predicates: < 100ms

## Implementation Roadmap

### Phase 1: Foundation (Wave 0-1)
- [ ] Core interfaces and types
- [ ] Plan node hierarchy
- [ ] Cost structures

### Phase 2: Semantic Analysis (Wave 1-2)
- [ ] Table/column validation
- [ ] Type checking
- [ ] Ambiguous reference detection

### Phase 3: Plan Generation (Wave 2-4)
- [ ] SELECT plan generation
- [ ] DML plan generation
- [ ] DDL plan generation

### Phase 4: Cost Estimation (Wave 4-6)
- [ ] Cardinality estimation
- [ ] I/O cost estimation
- [ ] CPU cost estimation

### Phase 5: Optimization (Wave 7-10)
- [ ] Index selection
- [ ] Join ordering
- [ ] Predicate pushdown

### Phase 6: Validation & Error Handling (Wave 11-12)
- [ ] Plan validation
- [ ] Error handling
- [ ] Error messages

### Phase 7: Integration (Wave 12-13)
- [ ] Parser integration
- [ ] Catalog integration
- [ ] Executor integration
- [ ] Transaction manager integration

### Phase 8: Testing & Performance (Wave 13-14)
- [ ] Property-based tests
- [ ] Unit tests
- [ ] Integration tests
- [ ] Performance benchmarks

## Key Design Decisions

1. **Cost-Based Optimization**: Use estimated costs to select best plan
2. **Heuristic Join Ordering**: Use cardinality-based heuristics for large queries
3. **Predicate Pushdown**: Move filters close to data sources
4. **Index Selection**: Automatic index selection based on cost
5. **Modular Components**: Separate semantic analysis, optimization, cost estimation
6. **Deterministic Planning**: Same query always produces same plan
7. **Error Context**: Include context in error messages

## Performance Targets

- Simple SELECT: < 10ms
- 5-table query: < 10ms
- 10-table query: < 50ms
- Complex predicates: < 100ms
- 100+ table schema: Efficient lookup
- 10+ table joins: Heuristic-based ordering

## Future Enhancements

1. Advanced join algorithms (hash join, sort-merge join)
2. Subquery optimization
3. Window functions
4. Materialized views
5. Parallel execution
6. Adaptive planning
7. Query caching
8. Explain plans
9. Statistics collection
10. Machine learning-based cost estimation

## References

- Design Document: `/Users/steven/Documents/learn/myDb/.kiro/specs/query-planner/design.md`
- Requirements: `/Users/steven/Documents/learn/myDb/.kiro/specs/query-planner/requirements.md`
- Tasks: `/Users/steven/Documents/learn/myDb/.kiro/specs/query-planner/tasks.md`
- Parser: `/Users/steven/Documents/learn/myDb/internal/sql/parser/`
- Schema Catalog: `/Users/steven/Documents/learn/myDb/internal/schema/`
- Executor: `/Users/steven/Documents/learn/myDb/internal/sql/executor/`

## Next Steps

1. Review this architecture document
2. Open `/Users/steven/Documents/learn/myDb/.kiro/specs/query-planner/tasks.md`
3. Start with Phase 1 (Foundation) tasks
4. Follow the task dependency graph for optimal parallel execution
5. Run tests after each phase
6. Reference requirements for acceptance criteria
7. Use property tests to validate correctness properties

---

**Status**: Ready for implementation
**Estimated Effort**: 40-50 hours
**Complexity**: High (core optimization logic)
**Priority**: Critical (enables efficient query execution)
