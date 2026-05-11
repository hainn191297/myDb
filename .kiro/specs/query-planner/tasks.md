# Implementation Plan: Query Planner

## Overview

This implementation plan transforms the Query Planner design into executable tasks. The Query Planner bridges the SQL Parser and Executor, making intelligent decisions about data access paths, join ordering, and optimization strategies. Implementation follows a foundation-first approach: core interfaces and types, then semantic analysis, plan generation, cost estimation, optimization, and finally comprehensive testing.

## Tasks

- [ ] 1. Foundation: Core Interfaces and Types
  - [ ] 1.1 Define Cost estimation structures and interfaces
    - Create `Cost` struct with IOCost, CPUCost, TotalCost fields
    - Define `CostEstimator` interface with methods for cardinality, I/O, and CPU estimation
    - Add cost comparison and aggregation helper functions
    - _Requirements: 3.1, 4.1, 5.1_

  - [ ] 1.2 Extend Operator interface with cost and cardinality tracking
    - Add `EstimatedCost()` and `EstimatedRows()` methods to Operator interface
    - Add `OutputColumns()` method to track column lineage
    - Ensure all existing operators (SeqScanOp, IndexScanOp, etc.) implement new methods
    - _Requirements: 2.1, 2.2, 2.3_

  - [ ] 1.3 Create plan node hierarchy for all operator types
    - Implement `FilterOp` for WHERE clause predicates
    - Implement `ProjectionOp` for SELECT column lists
    - Implement `JoinOp` interface with JoinType enum (INNER, LEFT, RIGHT, FULL)
    - Implement `NestedLoopJoinOp` as concrete join implementation
    - Implement `SortOp` for ORDER BY with SortKey struct
    - Implement `AggregateOp` for GROUP BY and aggregate functions
    - Implement `LimitOp` for LIMIT and OFFSET
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7_

  - [ ]* 1.4 Write unit tests for core operator types
    - Test each operator's Name(), EstimatedCost(), EstimatedRows(), OutputColumns() methods
    - Test operator creation with various configurations
    - _Requirements: 20.1_

- [ ] 2. Semantic Analysis and Validation
  - [ ] 2.1 Create SemanticAnalyzer component
    - Define `SemanticAnalyzer` struct with reference to schema catalog
    - Implement `Analyze(ast parser.AST) error` method
    - Add logging for validation steps
    - _Requirements: 1.1, 1.2, 1.3_

  - [ ] 2.2 Implement table and column reference validation
    - Validate all table references exist in schema catalog
    - Validate all column references exist in their source tables
    - Return descriptive errors with table/column names
    - _Requirements: 1.1, 1.2, 1.9_

  - [ ] 2.3 Implement ambiguous column reference detection
    - Detect when same column name appears in multiple tables without qualification
    - Return error "ambiguous column reference: {column}"
    - Handle qualified column references (schema.table.column)
    - _Requirements: 1.3, 1.9_

  - [ ] 2.4 Implement data type compatibility validation
    - Validate column data types are compatible with operations applied to them
    - Check type compatibility for predicates, assignments, and expressions
    - Return error "type mismatch: expected {expected_type}, got {actual_type}"
    - _Requirements: 1.4, 1.9_

  - [ ] 2.5 Implement DML statement validation
    - Validate INSERT: target table exists, all specified columns exist
    - Validate UPDATE: target table exists, all referenced columns exist
    - Validate DELETE: target table exists
    - _Requirements: 1.5, 1.6, 1.7_

  - [ ] 2.6 Implement WHERE clause validation
    - Validate all column references in predicates are valid
    - Validate predicate expressions are well-formed
    - _Requirements: 1.8_

  - [ ]* 2.7 Write unit tests for semantic analysis
    - Test valid table/column references pass validation
    - Test invalid references return appropriate errors
    - Test ambiguous column detection
    - Test type compatibility checking
    - _Requirements: 20.1, 20.4_

- [ ] 3. Plan Generation for SELECT Queries
  - [ ] 3.1 Implement basic SELECT plan generation
    - Extract table references from FROM clause
    - Create SeqScanOp or IndexScanOp for each table
    - Handle single-table SELECT queries
    - _Requirements: 2.1, 15.2_

  - [ ] 3.2 Implement WHERE clause plan generation
    - Create FilterOp node for WHERE predicates
    - Apply filter after table scan
    - Handle complex predicates (AND, OR, comparisons)
    - _Requirements: 2.2, 15.2_

  - [ ] 3.3 Implement SELECT column list plan generation
    - Create ProjectionOp node for column selection
    - Track output columns through plan tree
    - Handle SELECT * expansion
    - _Requirements: 2.3, 15.2_

  - [ ] 3.4 Implement JOIN plan generation
    - Create JoinOp nodes for JOIN clauses
    - Support INNER, LEFT, RIGHT, FULL join types
    - Extract join conditions from ON clauses
    - _Requirements: 2.4, 15.2_

  - [ ] 3.5 Implement ORDER BY plan generation
    - Create SortOp node with sort keys and direction
    - Handle multiple sort columns
    - _Requirements: 2.5, 15.2_

  - [ ] 3.6 Implement GROUP BY and aggregate plan generation
    - Create AggregateOp node with grouping columns
    - Support COUNT, SUM, AVG, MIN, MAX aggregate functions
    - Handle HAVING clause filtering
    - _Requirements: 2.6, 15.2_

  - [ ] 3.7 Implement LIMIT and OFFSET plan generation
    - Create LimitOp node with limit and offset values
    - Apply limit after sorting and aggregation
    - _Requirements: 2.7, 15.2_

  - [ ]* 3.8 Write unit tests for SELECT plan generation
    - Test single-table SELECT plans
    - Test multi-table JOIN plans
    - Test complex queries with WHERE, ORDER BY, GROUP BY, LIMIT
    - _Requirements: 20.1, 20.2_

- [ ] 4. Plan Generation for DML Statements
  - [ ] 4.1 Implement INSERT plan generation
    - Create InsertOp node with column mappings
    - Validate column count matches value count
    - Encode values using type system
    - _Requirements: 2.8, 15.3_

  - [ ] 4.2 Implement UPDATE plan generation
    - Create UpdateOp node with target table and column assignments
    - Validate all referenced columns exist
    - Encode assigned values using type system
    - _Requirements: 2.9, 15.4_

  - [ ] 4.3 Implement DELETE plan generation
    - Create DeleteOp node with target table and filter conditions
    - Validate table exists
    - _Requirements: 2.10, 15.5_

  - [ ]* 4.4 Write unit tests for DML plan generation
    - Test INSERT with various column configurations
    - Test UPDATE with different SET clauses
    - Test DELETE with and without WHERE clauses
    - _Requirements: 20.1, 20.2_

- [ ] 5. Plan Generation for DDL Statements
  - [ ] 5.1 Implement CREATE TABLE plan generation
    - Create CreateTableOp node with table schema
    - Validate column definitions
    - _Requirements: 2.11, 15.6_

  - [ ] 5.2 Implement CREATE INDEX plan generation
    - Create CreateIndexOp node with index definition
    - Validate index columns exist in table
    - _Requirements: 2.12, 15.6_

  - [ ] 5.3 Implement DROP TABLE plan generation
    - Create DropTableOp node with target table
    - _Requirements: 2.13, 15.6_

  - [ ] 5.4 Implement DROP INDEX plan generation
    - Create DropIndexOp node with target index
    - _Requirements: 2.14, 15.6_

  - [ ]* 5.5 Write unit tests for DDL plan generation
    - Test CREATE TABLE with various column types
    - Test CREATE INDEX with single and multi-column indexes
    - Test DROP TABLE and DROP INDEX
    - _Requirements: 20.1, 20.2_

- [ ] 6. Cardinality Estimation
  - [ ] 6.1 Implement base cardinality estimation
    - Estimate cardinality for table scans without filters
    - Use table statistics from schema catalog
    - Ensure minimum of 1 row
    - _Requirements: 3.1, 3.8_

  - [ ] 6.2 Implement selectivity estimation for equality predicates
    - Estimate selectivity as 1 / distinct_values for equality filters
    - Use column statistics from catalog
    - _Requirements: 3.2_

  - [ ] 6.3 Implement selectivity estimation for range predicates
    - Estimate selectivity based on value distribution
    - Support <, >, <=, >= operators
    - _Requirements: 3.3_

  - [ ] 6.4 Implement selectivity estimation for AND predicates
    - Multiply selectivity factors of both conditions
    - Handle nested AND expressions
    - _Requirements: 3.4_

  - [ ] 6.5 Implement selectivity estimation for OR predicates
    - Add selectivity factors (capped at 1.0)
    - Handle nested OR expressions
    - _Requirements: 3.5_

  - [ ] 6.6 Implement filter cardinality estimation
    - Estimate output cardinality as (input_cardinality * selectivity)
    - Ensure result does not exceed source table row count
    - _Requirements: 3.6, 3.9_

  - [ ] 6.7 Implement join cardinality estimation
    - Estimate output cardinality as (left_cardinality * right_cardinality * join_selectivity)
    - Use join condition selectivity
    - _Requirements: 3.7_

  - [ ]* 6.8 Write property tests for cardinality estimation
    - **Property 3: Cardinality Estimation Accuracy**
    - **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8, 3.9**
    - Test that estimated cardinality is within bounds (1 ≤ estimated ≤ table_rows)
    - Test that selectivity factors are between 0.0 and 1.0

  - [ ]* 6.9 Write unit tests for cardinality estimation
    - Test base cardinality for tables with known row counts
    - Test selectivity for various predicate types
    - Test cardinality bounds enforcement
    - _Requirements: 20.1, 20.3_

- [ ] 7. I/O Cost Estimation
  - [ ] 7.1 Implement sequential scan I/O cost estimation
    - Calculate I/O cost as (page_count * sequential_read_cost)
    - Use table size from statistics
    - _Requirements: 4.1_

  - [ ] 7.2 Implement index scan I/O cost estimation
    - Estimate index pages based on index height and key range
    - Estimate data pages based on rows to fetch and average row size
    - Calculate total I/O cost as (index_pages + data_pages) * random_read_cost
    - _Requirements: 4.2, 4.3, 4.4_

  - [ ] 7.3 Implement range scan I/O cost estimation
    - Estimate I/O cost based on key range boundaries
    - Handle partial index scans
    - _Requirements: 4.6_

  - [ ] 7.4 Implement access path comparison
    - Compare sequential scan vs index scan costs
    - Select access path with lower total I/O cost
    - _Requirements: 4.5_

  - [ ]* 7.5 Write unit tests for I/O cost estimation
    - Test sequential scan cost calculation
    - Test index scan cost calculation
    - Test access path selection logic
    - _Requirements: 20.1, 20.3_

- [ ] 8. CPU Cost Estimation
  - [ ] 8.1 Implement filter operation CPU cost estimation
    - Estimate CPU cost as (row_count * predicate_evaluation_cost)
    - Use predicate complexity for evaluation cost
    - _Requirements: 5.1_

  - [ ] 8.2 Implement projection operation CPU cost estimation
    - Estimate CPU cost as (row_count * column_count * column_extraction_cost)
    - _Requirements: 5.2_

  - [ ] 8.3 Implement sort operation CPU cost estimation
    - Estimate CPU cost as (row_count * log2(row_count) * comparison_cost)
    - Use O(n log n) complexity for comparison-based sort
    - _Requirements: 5.3_

  - [ ] 8.4 Implement nested loop join CPU cost estimation
    - Estimate CPU cost as (left_rows * right_rows * comparison_cost)
    - _Requirements: 5.4_

  - [ ] 8.5 Implement aggregate operation CPU cost estimation
    - Estimate CPU cost based on row count and aggregate functions
    - Account for grouping overhead
    - _Requirements: 5.5_

  - [ ]* 8.6 Write unit tests for CPU cost estimation
    - Test cost calculation for each operation type
    - Test cost monotonicity with increasing row counts
    - _Requirements: 20.1, 20.3_

- [ ] 9. Index Selection Strategy
  - [ ] 9.1 Implement index candidate identification
    - Extract indexed columns from WHERE clause predicates
    - Find all indexes that cover the filtered columns
    - _Requirements: 6.1, 6.2_

  - [ ] 9.2 Implement index cost estimation
    - Estimate I/O cost for each candidate index
    - Compare with sequential scan cost
    - _Requirements: 6.2, 6.3_

  - [ ] 9.3 Implement equality predicate index selection
    - Prefer indexes for equality predicates on indexed columns
    - _Requirements: 6.5_

  - [ ] 9.4 Implement range predicate index selection
    - Consider indexes for range predicates (<, >, <=, >=)
    - _Requirements: 6.6_

  - [ ] 9.5 Implement multi-column index selection
    - Select index with best selectivity for multiple indexed columns
    - _Requirements: 6.7_

  - [ ] 9.6 Implement fallback to sequential scan
    - Select sequential scan when no indexes are available
    - Select sequential scan when index cost exceeds sequential scan cost
    - _Requirements: 6.4, 6.8_

  - [ ]* 9.7 Write property tests for index selection
    - **Property 6: Index Selection Optimality**
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8**
    - Test that selected index has lower cost than sequential scan
    - Test that sequential scan is selected when no indexes available

  - [ ]* 9.8 Write unit tests for index selection
    - Test index identification for various predicates
    - Test cost comparison logic
    - Test fallback to sequential scan
    - _Requirements: 20.1, 20.3_

- [ ] 10. Join Ordering Optimization
  - [ ] 10.1 Implement join order enumeration
    - Extract all tables and join conditions from query
    - Generate candidate join orders
    - _Requirements: 7.1, 7.2_

  - [ ] 10.2 Implement cardinality-based join ordering
    - Calculate cardinality for each table
    - Prioritize joining smaller tables first
    - _Requirements: 7.3_

  - [ ] 10.3 Implement join condition matching
    - Find join conditions between tables
    - Build join tree with correct connections
    - _Requirements: 7.4_

  - [ ] 10.4 Implement cost-based join ordering
    - Estimate cost for each candidate join order
    - Select order with minimum total cost
    - _Requirements: 7.5_

  - [ ] 10.5 Implement heuristic-based ordering for large queries
    - Use greedy heuristics for queries with 10+ tables
    - Avoid exponential search time
    - _Requirements: 7.6_

  - [ ]* 10.6 Write property tests for join ordering
    - **Property 7: Join Ordering Cost Minimization**
    - **Validates: Requirements 7.1, 7.2, 7.3, 7.4, 7.5, 7.6**
    - Test that selected join order has lower cost than alternatives

  - [ ]* 10.7 Write unit tests for join ordering
    - Test join order generation for multi-table queries
    - Test cost comparison logic
    - Test heuristic ordering for large queries
    - _Requirements: 20.1, 20.3_

- [ ] 11. Predicate Pushdown Optimization
  - [ ] 11.1 Implement filter detection after joins
    - Identify filters applied after join operations
    - Analyze filter predicates for table references
    - _Requirements: 8.1_

  - [ ] 11.2 Implement single-table filter pushdown
    - Detect filters that reference only one table
    - Push filter down to apply before join
    - _Requirements: 8.2_

  - [ ] 11.3 Implement cardinality recalculation
    - Recalculate table cardinality after filter pushdown
    - Recalculate join cardinality based on reduced input
    - _Requirements: 8.3, 8.4_

  - [ ] 11.4 Implement multi-filter pushdown
    - Push all applicable filters to their source tables
    - Handle complex predicate trees
    - _Requirements: 8.5_

  - [ ]* 11.5 Write property tests for predicate pushdown
    - **Property 8: Predicate Pushdown Correctness**
    - **Validates: Requirements 8.1, 8.2, 8.3, 8.4, 8.5**
    - Test that pushed-down filters reduce intermediate cardinality

  - [ ]* 11.6 Write unit tests for predicate pushdown
    - Test filter detection and pushdown
    - Test cardinality recalculation
    - Test multi-filter scenarios
    - _Requirements: 20.1, 20.3_

- [ ] 12. Plan Validation Framework
  - [ ] 12.1 Implement plan node validation
    - Validate all referenced tables exist in catalog
    - Validate all referenced columns exist in source tables
    - _Requirements: 9.1, 9.2_

  - [ ] 12.2 Implement join condition validation
    - Validate join conditions reference columns from both joined tables
    - _Requirements: 9.3_

  - [ ] 12.3 Implement filter predicate validation
    - Validate filter predicates reference valid columns
    - _Requirements: 9.4_

  - [ ] 12.4 Implement projection column validation
    - Validate projection columns exist in child operator's output
    - _Requirements: 9.5_

  - [ ] 12.5 Implement cost estimate validation
    - Validate all operators have valid cost estimates
    - _Requirements: 9.6_

  - [ ] 12.6 Implement comprehensive plan validation
    - Orchestrate all validation checks
    - Return descriptive error messages on validation failure
    - _Requirements: 9.7_

  - [ ]* 12.7 Write unit tests for plan validation
    - Test validation of valid plans
    - Test detection of invalid table/column references
    - Test detection of invalid join conditions
    - _Requirements: 20.1, 20.4_

- [ ] 13. Error Handling and Reporting
  - [ ] 13.1 Implement table not found error handling
    - Return error "table {schema}.{table} not found"
    - Include context about which query component failed
    - _Requirements: 10.1, 10.7_

  - [ ] 13.2 Implement column not found error handling
    - Return error "column {column} not found in table {schema}.{table}"
    - Include context about which query component failed
    - _Requirements: 10.2, 10.7_

  - [ ] 13.3 Implement ambiguous column reference error handling
    - Return error "ambiguous column reference: {column}"
    - _Requirements: 10.3_

  - [ ] 13.4 Implement type mismatch error handling
    - Return error "type mismatch: expected {expected_type}, got {actual_type}"
    - _Requirements: 10.4_

  - [ ] 13.5 Implement invalid join condition error handling
    - Return error "invalid join condition: {condition}"
    - _Requirements: 10.5_

  - [ ] 13.6 Implement unsupported operation error handling
    - Return error "unsupported operation: {operation}"
    - _Requirements: 10.6_

  - [ ] 13.7 Implement error context wrapping
    - Wrap errors with context using fmt.Errorf
    - Propagate errors up the call stack
    - _Requirements: 10.7_

  - [ ]* 13.8 Write unit tests for error handling
    - Test each error type is returned correctly
    - Test error messages include appropriate context
    - _Requirements: 20.1, 20.4_

- [ ] 14. Integration with Parser and Catalog
  - [ ] 14.1 Implement AST acceptance and extraction
    - Accept parsed AST structures from parser
    - Extract table references, columns, predicates, join conditions
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.6_

  - [ ] 14.2 Implement schema catalog queries
    - Query catalog for table definitions
    - Query catalog for column definitions
    - Query catalog for index definitions
    - Query catalog for table statistics
    - _Requirements: 16.1, 16.2, 16.3, 16.4_

  - [ ] 14.3 Implement catalog error handling
    - Handle catalog unavailability gracefully
    - Return appropriate errors
    - _Requirements: 16.5_

  - [ ] 14.4 Implement plan output for executor
    - Return Physical_Plan structure with root Operator
    - Ensure each Operator has Name, EstimatedCost, EstimatedRows, OutputColumns
    - Ensure plan is ready for immediate execution
    - _Requirements: 17.1, 17.2, 17.3, 17.4_

  - [ ] 14.5 Implement plan metadata for transaction manager
    - Identify which tables are accessed by plan
    - Identify access type (read or write) for each table
    - Provide this information to Transaction_Manager
    - _Requirements: 18.1, 18.2, 18.3_

  - [ ]* 14.6 Write integration tests
    - Test end-to-end planning from parser AST to executor plan
    - Test catalog integration
    - Test transaction manager integration
    - _Requirements: 20.2, 20.6_

- [ ] 15. Performance Optimization and Caching
  - [ ] 15.1 Implement table statistics caching
    - Cache table statistics in memory
    - Reuse cached statistics without querying catalog
    - _Requirements: 11.5_

  - [ ] 15.2 Implement plan generation performance monitoring
    - Add timing instrumentation to plan generation
    - Log performance metrics
    - _Requirements: 11.1, 11.2, 11.3, 11.4_

  - [ ] 15.3 Implement query plan caching (optional)
    - Cache frequently used plans
    - Invalidate cache on schema changes
    - _Requirements: 11.5_

  - [ ]* 15.4 Write performance benchmarks
    - Benchmark simple SELECT query planning (target: <10ms)
    - Benchmark 5-table query planning (target: <10ms)
    - Benchmark 10-table query planning (target: <50ms)
    - Benchmark complex predicate planning (target: <100ms)
    - _Requirements: 11.1, 11.2, 11.3, 11.4_

- [ ] 16. Checkpoint - Core Planning Complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 17. Comprehensive Testing Suite
  - [ ] 17.1 Write property tests for semantic validation
    - **Property 1: Semantic Validation Completeness**
    - **Validates: Requirements 1.1, 1.2, 1.3, 1.4**
    - Test that invalid table/column references are rejected

  - [ ] 17.2 Write property tests for plan node creation
    - **Property 2: Plan Node Creation Correctness**
    - **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 2.10, 2.11, 2.12, 2.13, 2.14**
    - Test that appropriate operators are created for all query types

  - [ ] 17.3 Write property tests for plan completeness
    - **Property 12: Plan Completeness**
    - **Validates: Requirements 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7, 13.8**
    - Test that all tables, predicates, and columns are included in plan

  - [ ] 17.4 Write property tests for planning determinism
    - **Property 13: Planning Determinism**
    - **Validates: Requirements 14.1, 14.2, 14.3**
    - Test that same query produces identical plans

  - [ ] 17.5 Write integration tests for complex queries
    - Test multi-table JOINs with various join types
    - Test complex WHERE clauses with AND/OR predicates
    - Test GROUP BY with HAVING clauses
    - Test ORDER BY with multiple sort keys
    - Test LIMIT and OFFSET
    - _Requirements: 20.2, 20.6_

  - [ ] 17.6 Write integration tests for DML operations
    - Test INSERT with various column configurations
    - Test UPDATE with complex SET clauses
    - Test DELETE with complex WHERE clauses
    - _Requirements: 20.2, 20.6_

  - [ ] 17.7 Write integration tests for DDL operations
    - Test CREATE TABLE with various column types
    - Test CREATE INDEX with single and multi-column indexes
    - Test DROP TABLE and DROP INDEX
    - _Requirements: 20.2, 20.6_

  - [ ] 17.8 Write scalability tests
    - Test planning with 100+ table schema
    - Test planning with 10+ table JOINs
    - Test planning with deeply nested predicates
    - _Requirements: 12.1, 12.2, 12.3, 12.4_

  - [ ] 17.9 Write documentation and code comments
    - Document cardinality estimation algorithm
    - Document cost estimation algorithm
    - Document index selection strategy
    - Document join ordering algorithm
    - Document predicate pushdown optimization
    - _Requirements: 19.1, 19.2, 19.3, 19.4, 19.5_

- [ ] 18. Final Checkpoint - All Tests Pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP, but are strongly recommended for production quality
- Each task references specific requirements for traceability
- Property-based tests validate universal correctness properties across all valid inputs
- Unit tests validate specific examples and edge cases
- Integration tests verify end-to-end query planning and execution
- Performance benchmarks ensure planning meets latency targets
- All code follows Go idioms and repository guidelines (tabs, mixedCaps, gofmt)
- Error handling uses fmt.Errorf with context wrapping
- Tests use table-driven approach colocated with code under test

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3"] },
    { "id": 1, "tasks": ["1.4", "2.1", "2.2", "2.3", "2.4", "2.5", "2.6"] },
    { "id": 2, "tasks": ["2.7", "3.1", "3.2", "3.3", "3.4", "3.5", "3.6", "3.7"] },
    { "id": 3, "tasks": ["3.8", "4.1", "4.2", "4.3", "4.4", "5.1", "5.2", "5.3", "5.4"] },
    { "id": 4, "tasks": ["5.5", "6.1", "6.2", "6.3", "6.4", "6.5", "6.6", "6.7"] },
    { "id": 5, "tasks": ["6.8", "6.9", "7.1", "7.2", "7.3", "7.4", "7.5"] },
    { "id": 6, "tasks": ["8.1", "8.2", "8.3", "8.4", "8.5", "8.6"] },
    { "id": 7, "tasks": ["9.1", "9.2", "9.3", "9.4", "9.5", "9.6"] },
    { "id": 8, "tasks": ["9.7", "9.8", "10.1", "10.2", "10.3", "10.4", "10.5"] },
    { "id": 9, "tasks": ["10.6", "10.7", "11.1", "11.2", "11.3", "11.4"] },
    { "id": 10, "tasks": ["11.5", "11.6", "12.1", "12.2", "12.3", "12.4", "12.5", "12.6"] },
    { "id": 11, "tasks": ["12.7", "13.1", "13.2", "13.3", "13.4", "13.5", "13.6", "13.7"] },
    { "id": 12, "tasks": ["13.8", "14.1", "14.2", "14.3", "14.4", "14.5"] },
    { "id": 13, "tasks": ["14.6", "15.1", "15.2", "15.3", "15.4"] },
    { "id": 14, "tasks": ["17.1", "17.2", "17.3", "17.4", "17.5", "17.6", "17.7", "17.8", "17.9"] }
  ]
}
```
