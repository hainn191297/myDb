# Query Planner Implementation Guide

## Current State

The Query Planner has a basic implementation with:
- ✅ Core operator types (SeqScanOp, IndexScanOp, InsertOp, UpdateOp, DeleteOp, etc.)
- ✅ Basic plan building for SELECT, INSERT, UPDATE, DELETE
- ✅ Index scan attempt logic
- ✅ Transaction control operators (BEGIN, COMMIT, ROLLBACK)
- ✅ DDL operators (CREATE TABLE, DROP TABLE, CREATE INDEX, DROP INDEX)

## What Needs to Be Enhanced

### Phase 1: Cost Estimation Framework (CRITICAL)
- [ ] Add Cost struct with IOCost, CPUCost, TotalCost
- [ ] Add EstimatedCost() and EstimatedRows() methods to Operator interface
- [ ] Add OutputColumns() method to track column lineage
- [ ] Implement cardinality estimation algorithms
- [ ] Implement I/O cost estimation
- [ ] Implement CPU cost estimation

### Phase 2: Semantic Analysis (CRITICAL)
- [ ] Create SemanticAnalyzer component
- [ ] Implement table/column validation
- [ ] Implement ambiguous column detection
- [ ] Implement data type compatibility checking
- [ ] Implement DML statement validation

### Phase 3: Advanced Plan Generation (IMPORTANT)
- [ ] Add FilterOp for WHERE clause predicates
- [ ] Add ProjectionOp for SELECT column lists
- [ ] Add JoinOp for multi-table queries
- [ ] Add SortOp for ORDER BY
- [ ] Add AggregateOp for GROUP BY
- [ ] Add LimitOp for LIMIT/OFFSET

### Phase 4: Optimization (IMPORTANT)
- [ ] Implement index selection strategy
- [ ] Implement join ordering algorithm
- [ ] Implement predicate pushdown optimization
- [ ] Implement optimization rule application

### Phase 5: Plan Validation (IMPORTANT)
- [ ] Implement comprehensive plan validation
- [ ] Implement error handling with context
- [ ] Implement error message generation

### Phase 6: Testing (CRITICAL)
- [ ] Property-based tests for all correctness properties
- [ ] Unit tests for all components
- [ ] Integration tests for end-to-end planning
- [ ] Performance benchmarks

## Implementation Priority

### Must Have (MVP)
1. Cost estimation framework
2. Semantic analysis
3. Plan validation
4. Error handling
5. Basic testing

### Should Have (Production)
1. Advanced plan generation (Filter, Projection, Join, Sort, Aggregate, Limit)
2. Optimization (index selection, join ordering, predicate pushdown)
3. Comprehensive testing
4. Performance optimization

### Nice to Have (Future)
1. Advanced join algorithms (hash join, sort-merge join)
2. Subquery optimization
3. Query caching
4. Explain plans
5. Statistics collection

## Next Steps

1. **Enhance Operator Interface**
   - Add Cost struct
   - Add EstimatedCost(), EstimatedRows(), OutputColumns() methods
   - Update all operator types

2. **Implement Cost Estimation**
   - Cardinality estimation with selectivity
   - I/O cost calculation
   - CPU cost calculation

3. **Implement Semantic Analysis**
   - Table/column validation
   - Type checking
   - Error handling

4. **Add Advanced Operators**
   - FilterOp, ProjectionOp, JoinOp, SortOp, AggregateOp, LimitOp

5. **Implement Optimization**
   - Index selection
   - Join ordering
   - Predicate pushdown

6. **Comprehensive Testing**
   - Property-based tests
   - Unit tests
   - Integration tests
   - Performance benchmarks

## File Structure

```
internal/sql/planner/
├── planner.go              # Main planner logic
├── operators.go            # Operator definitions (NEW)
├── cost_estimator.go       # Cost estimation (NEW)
├── semantic_analyzer.go    # Semantic analysis (NEW)
├── optimizer.go            # Optimization rules (NEW)
├── validator.go            # Plan validation (NEW)
├── planner_test.go         # Tests
└── planner_index_test.go   # Index tests
```

## Testing Strategy

### Property-Based Tests
- Semantic Validation Completeness
- Plan Node Creation Correctness
- Cardinality Estimation Accuracy
- I/O Cost Estimation Consistency
- CPU Cost Estimation Monotonicity
- Index Selection Optimality
- Join Ordering Cost Minimization
- Predicate Pushdown Correctness
- Plan Validation Completeness
- Error Message Specificity
- Scalability with Large Schemas
- Plan Completeness
- Planning Determinism

### Unit Tests
- Cost estimation algorithms
- Semantic analysis validation
- Plan generation for all statement types
- Optimization logic
- Plan validation
- Error handling

### Integration Tests
- End-to-end: Parse SQL → Plan → Execute
- Multi-table JOINs
- Complex WHERE clauses
- GROUP BY with HAVING
- ORDER BY with multiple keys
- LIMIT and OFFSET
- DML operations
- DDL operations

### Performance Tests
- Simple SELECT: < 10ms
- 5-table query: < 10ms
- 10-table query: < 50ms
- Complex predicates: < 100ms

## Key Algorithms to Implement

### Cardinality Estimation
```
Base Cardinality = Table Row Count
Selectivity = 1 / DistinctValues (for equality)
Output Rows = Base Cardinality * Selectivity
```

### I/O Cost Estimation
```
Sequential Scan: Pages * SequentialReadCost
Index Scan: (IndexPages + DataPages) * RandomReadCost
```

### CPU Cost Estimation
```
Filter: RowCount * PredicateEvaluationCost
Sort: RowCount * LOG2(RowCount) * ComparisonCost
Join: LeftRows * RightRows * ComparisonCost
```

### Index Selection
```
1. Extract indexed columns from filter
2. Find candidate indexes
3. Estimate cost for each
4. Select lowest cost (or sequential scan if cheaper)
```

### Join Ordering
```
1. Calculate cardinality for each table
2. Sort by cardinality (ascending)
3. Join smallest table first
4. Build join tree
```

## Code Style Guidelines

- Use tabs for indentation
- Use mixedCaps for exported identifiers
- Use lowerCamel for local variables
- Use fmt.Errorf("...: %w", err) for error wrapping
- Use table-driven tests
- Add comments explaining algorithms
- Run gofmt before committing

## References

- Design: `/Users/steven/Documents/learn/myDb/.kiro/specs/query-planner/design.md`
- Requirements: `/Users/steven/Documents/learn/myDb/.kiro/specs/query-planner/requirements.md`
- Tasks: `/Users/steven/Documents/learn/myDb/.kiro/specs/query-planner/tasks.md`
- Architecture: `/Users/steven/Documents/learn/myDb/.kiro/QUERY-PLANNER-ARCHITECTURE.md`

## Status

**Current**: Basic planner with operator types and plan building
**Target**: Full-featured planner with cost estimation, optimization, and comprehensive testing
**Effort**: 40-50 hours
**Complexity**: High (core optimization logic)
**Priority**: Critical (enables efficient query execution)

---

Ready to begin Phase 1: Cost Estimation Framework
