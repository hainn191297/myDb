# Query Planner Requirements

## Introduction

The Query Planner transforms parsed SQL Abstract Syntax Trees (ASTs) into optimized physical execution plans. It serves as the critical bridge between the SQL Parser and the Executor, making intelligent decisions about data access paths, join ordering, and optimization strategies. This document specifies the functional, non-functional, and integration requirements for the Query Planner system.

## Glossary

- **Semantic_Analyzer**: Component that validates table and column references against the schema catalog
- **Query_Optimizer**: Component that applies logical transformation rules and explores alternative plan shapes
- **Cost_Estimator**: Component that estimates row counts, I/O costs, and CPU costs for different access paths
- **Plan_Generator**: Component that constructs physical plan nodes with concrete operators
- **Physical_Plan**: Executable representation of a query consisting of operator nodes
- **Operator**: A plan node representing a specific operation (scan, filter, join, etc.)
- **Cardinality**: Estimated number of rows produced by an operation
- **Selectivity**: Fraction of rows that satisfy a predicate (0.0 to 1.0)
- **Index_Scan**: Access path using an index to retrieve rows
- **Sequential_Scan**: Access path that reads all table pages sequentially
- **Join_Condition**: Predicate that specifies how two tables are joined
- **Predicate_Pushdown**: Optimization that moves filters closer to data sources
- **Cost_Model**: Algorithm for estimating execution costs of different plan alternatives
- **Table_Statistics**: Metadata about table size, row count, and column distributions
- **Schema_Catalog**: System catalog containing table definitions and index information

## Requirements

### Requirement 1: Semantic Analysis and Validation

**User Story:** As a query planner, I want to validate semantic correctness of parsed queries, so that invalid queries are rejected before optimization and execution.

#### Acceptance Criteria

1. WHEN a SELECT query is analyzed THEN the Semantic_Analyzer SHALL validate that all referenced tables exist in the Schema_Catalog
2. WHEN a SELECT query is analyzed THEN the Semantic_Analyzer SHALL validate that all referenced columns exist in their source tables
3. WHEN a SELECT query is analyzed THEN the Semantic_Analyzer SHALL detect and reject ambiguous column references (same column name in multiple tables without qualification)
4. WHEN a SELECT query is analyzed THEN the Semantic_Analyzer SHALL validate that column data types are compatible with operations applied to them
5. WHEN an INSERT query is analyzed THEN the Semantic_Analyzer SHALL validate that the target table exists and all specified columns exist
6. WHEN an UPDATE query is analyzed THEN the Semantic_Analyzer SHALL validate that the target table exists and all referenced columns exist
7. WHEN a DELETE query is analyzed THEN the Semantic_Analyzer SHALL validate that the target table exists
8. WHEN a query contains a WHERE clause THEN the Semantic_Analyzer SHALL validate that all column references in the predicate are valid
9. IF a table or column reference is invalid THEN the Semantic_Analyzer SHALL return a descriptive error message identifying the invalid reference

### Requirement 2: Plan Node Creation for All Operation Types

**User Story:** As a query planner, I want to create appropriate plan nodes for all supported SQL operations, so that the executor can process any valid query.

#### Acceptance Criteria

1. WHEN a SELECT query is planned THEN the Plan_Generator SHALL create a SeqScanOp or IndexScanOp node for each table in the FROM clause
2. WHEN a SELECT query contains a WHERE clause THEN the Plan_Generator SHALL create a FilterOp node to apply the predicate
3. WHEN a SELECT query contains a column list THEN the Plan_Generator SHALL create a ProjectionOp node to select specific columns
4. WHEN a SELECT query contains a JOIN THEN the Plan_Generator SHALL create a JoinOp node with the appropriate join type (INNER, LEFT, RIGHT, FULL)
5. WHEN a SELECT query contains an ORDER BY clause THEN the Plan_Generator SHALL create a SortOp node with the specified sort keys
6. WHEN a SELECT query contains a GROUP BY clause THEN the Plan_Generator SHALL create an AggregateOp node with grouping columns and aggregate functions
7. WHEN a SELECT query contains a LIMIT clause THEN the Plan_Generator SHALL create a LimitOp node with the specified limit and offset
8. WHEN an INSERT query is planned THEN the Plan_Generator SHALL create an InsertOp node with column mappings and value expressions
9. WHEN an UPDATE query is planned THEN the Plan_Generator SHALL create an UpdateOp node with the target table and column assignments
10. WHEN a DELETE query is planned THEN the Plan_Generator SHALL create a DeleteOp node with the target table and filter conditions
11. WHEN a CREATE TABLE query is planned THEN the Plan_Generator SHALL create a CreateTableOp node with table schema
12. WHEN a CREATE INDEX query is planned THEN the Plan_Generator SHALL create a CreateIndexOp node with index definition
13. WHEN a DROP TABLE query is planned THEN the Plan_Generator SHALL create a DropTableOp node with the target table
14. WHEN a DROP INDEX query is planned THEN the Plan_Generator SHALL create a DropIndexOp node with the target index

### Requirement 3: Cost Estimation for Cardinality

**User Story:** As a query optimizer, I want to estimate row counts for query operations, so that I can compare alternative plans and select the most efficient one.

#### Acceptance Criteria

1. WHEN a table is scanned without filters THEN the Cost_Estimator SHALL estimate cardinality as the table's row count from statistics
2. WHEN a table is scanned with an equality predicate THEN the Cost_Estimator SHALL estimate cardinality as (table_rows / distinct_values) for the filtered column
3. WHEN a table is scanned with a range predicate THEN the Cost_Estimator SHALL estimate cardinality based on the value distribution of the filtered column
4. WHEN a table is scanned with an AND predicate THEN the Cost_Estimator SHALL estimate cardinality by multiplying the selectivity factors of both conditions
5. WHEN a table is scanned with an OR predicate THEN the Cost_Estimator SHALL estimate cardinality by adding the selectivity factors (capped at 1.0)
6. WHEN a filter is applied to a result set THEN the Cost_Estimator SHALL estimate output cardinality as (input_cardinality * selectivity)
7. WHEN a join is performed THEN the Cost_Estimator SHALL estimate output cardinality as (left_cardinality * right_cardinality * join_selectivity)
8. WHEN cardinality is estimated THEN the Cost_Estimator SHALL ensure the result is at least 1 row
9. WHEN cardinality is estimated THEN the Cost_Estimator SHALL ensure the result does not exceed the source table's row count

### Requirement 4: Cost Estimation for I/O Operations

**User Story:** As a query optimizer, I want to estimate I/O costs for different access paths, so that I can select the most efficient data access strategy.

#### Acceptance Criteria

1. WHEN a sequential scan is planned THEN the Cost_Estimator SHALL estimate I/O cost as (table_size_bytes / page_size) * sequential_read_cost
2. WHEN an index scan is planned THEN the Cost_Estimator SHALL estimate I/O cost as (index_pages + data_pages) * random_read_cost
3. WHEN an index scan is planned THEN the Cost_Estimator SHALL estimate index pages based on the index height and key range
4. WHEN an index scan is planned THEN the Cost_Estimator SHALL estimate data pages based on the number of rows to fetch and average row size
5. WHEN comparing sequential scan vs index scan THEN the Cost_Estimator SHALL select the access path with lower total I/O cost
6. WHEN a range scan is performed on an index THEN the Cost_Estimator SHALL estimate I/O cost based on the key range boundaries

### Requirement 5: Cost Estimation for CPU Operations

**User Story:** As a query optimizer, I want to estimate CPU costs for different operations, so that I can account for processing overhead in plan selection.

#### Acceptance Criteria

1. WHEN a filter operation is planned THEN the Cost_Estimator SHALL estimate CPU cost as (row_count * predicate_evaluation_cost)
2. WHEN a projection operation is planned THEN the Cost_Estimator SHALL estimate CPU cost as (row_count * column_count * column_extraction_cost)
3. WHEN a sort operation is planned THEN the Cost_Estimator SHALL estimate CPU cost as (row_count * log2(row_count) * comparison_cost)
4. WHEN a nested loop join is planned THEN the Cost_Estimator SHALL estimate CPU cost as (left_rows * right_rows * comparison_cost)
5. WHEN an aggregate operation is planned THEN the Cost_Estimator SHALL estimate CPU cost based on the number of rows and aggregate functions

### Requirement 6: Index Selection Strategy

**User Story:** As a query optimizer, I want to automatically select the best index for query filters, so that queries use efficient access paths.

#### Acceptance Criteria

1. WHEN a query contains a WHERE clause with indexed columns THEN the Query_Optimizer SHALL identify candidate indexes that cover the filtered columns
2. WHEN multiple indexes are available THEN the Query_Optimizer SHALL estimate I/O cost for each candidate index
3. WHEN index cost is lower than sequential scan cost THEN the Query_Optimizer SHALL select the index for the access path
4. WHEN index cost is higher than sequential scan cost THEN the Query_Optimizer SHALL select sequential scan instead
5. WHEN a filter contains an equality predicate on an indexed column THEN the Query_Optimizer SHALL prefer that index
6. WHEN a filter contains a range predicate on an indexed column THEN the Query_Optimizer SHALL consider that index for range scan
7. WHEN a filter contains multiple indexed columns THEN the Query_Optimizer SHALL select the index with the best selectivity
8. WHEN no indexes are available for the filter THEN the Query_Optimizer SHALL plan a sequential scan

### Requirement 7: Join Ordering Optimization

**User Story:** As a query optimizer, I want to determine the optimal order for joining multiple tables, so that intermediate result sizes are minimized.

#### Acceptance Criteria

1. WHEN a query joins multiple tables THEN the Query_Optimizer SHALL determine a join order that minimizes total cost
2. WHEN join ordering is determined THEN the Query_Optimizer SHALL consider the cardinality of each table
3. WHEN join ordering is determined THEN the Query_Optimizer SHALL prioritize joining smaller tables first to reduce intermediate result sizes
4. WHEN a join condition exists between two tables THEN the Query_Optimizer SHALL use that condition to connect them in the join tree
5. WHEN multiple join orders are possible THEN the Query_Optimizer SHALL estimate cost for each order and select the minimum
6. WHEN a query contains 10 or more tables THEN the Query_Optimizer SHALL use heuristic-based ordering to avoid exponential search time

### Requirement 8: Predicate Pushdown Optimization

**User Story:** As a query optimizer, I want to apply predicates as early as possible in the plan, so that fewer rows are processed by downstream operations.

#### Acceptance Criteria

1. WHEN a filter is applied after a join THEN the Query_Optimizer SHALL detect if the filter references only one table
2. WHEN a filter references only one table THEN the Query_Optimizer SHALL push the filter down to apply before the join
3. WHEN a filter is pushed down THEN the Query_Optimizer SHALL reduce the cardinality estimate for that table
4. WHEN a filter is pushed down THEN the Query_Optimizer SHALL recalculate join cardinality based on reduced input
5. WHEN multiple filters can be pushed down THEN the Query_Optimizer SHALL push all applicable filters to their source tables

### Requirement 9: Plan Validation

**User Story:** As a query planner, I want to validate that generated plans are correct before execution, so that invalid plans do not reach the executor.

#### Acceptance Criteria

1. WHEN a plan is generated THEN the Plan_Generator SHALL validate that all referenced tables exist in the Schema_Catalog
2. WHEN a plan is generated THEN the Plan_Generator SHALL validate that all referenced columns exist in their source tables
3. WHEN a plan is generated THEN the Plan_Generator SHALL validate that join conditions reference columns from both joined tables
4. WHEN a plan is generated THEN the Plan_Generator SHALL validate that filter predicates reference valid columns
5. WHEN a plan is generated THEN the Plan_Generator SHALL validate that projection columns exist in the child operator's output
6. WHEN a plan is generated THEN the Plan_Generator SHALL validate that all operators have valid cost estimates
7. IF a plan validation fails THEN the Plan_Generator SHALL return a descriptive error message

### Requirement 10: Error Handling and Reporting

**User Story:** As a database user, I want clear error messages when query planning fails, so that I can understand and fix the problem.

#### Acceptance Criteria

1. WHEN a table is not found THEN the Semantic_Analyzer SHALL return error "table {schema}.{table} not found"
2. WHEN a column is not found THEN the Semantic_Analyzer SHALL return error "column {column} not found in table {schema}.{table}"
3. WHEN a column reference is ambiguous THEN the Semantic_Analyzer SHALL return error "ambiguous column reference: {column}"
4. WHEN data types are incompatible THEN the Semantic_Analyzer SHALL return error "type mismatch: expected {expected_type}, got {actual_type}"
5. WHEN a join condition is invalid THEN the Query_Optimizer SHALL return error "invalid join condition: {condition}"
6. WHEN an unsupported operation is encountered THEN the Plan_Generator SHALL return error "unsupported operation: {operation}"
7. WHEN an error occurs THEN the error message SHALL include context about which query component failed

### Requirement 11: Performance Requirements

**User Story:** As a database administrator, I want query planning to complete quickly, so that query execution latency is minimized.

#### Acceptance Criteria

1. WHEN a simple SELECT query is planned THEN the Query_Planner SHALL complete planning in less than 10 milliseconds
2. WHEN a query with 5 tables is planned THEN the Query_Planner SHALL complete planning in less than 10 milliseconds
3. WHEN a query with 10 tables is planned THEN the Query_Planner SHALL complete planning in less than 50 milliseconds
4. WHEN a query with complex predicates is planned THEN the Query_Planner SHALL complete planning in less than 100 milliseconds
5. WHEN table statistics are cached THEN the Query_Planner SHALL reuse cached statistics without querying the catalog

### Requirement 12: Scalability Requirements

**User Story:** As a database administrator, I want the query planner to handle large schemas and complex queries, so that the system can support enterprise workloads.

#### Acceptance Criteria

1. WHEN a query references 100 or more tables THEN the Query_Planner SHALL successfully generate a valid plan
2. WHEN a query contains 10 or more joins THEN the Query_Planner SHALL successfully generate a valid plan
3. WHEN a query contains deeply nested predicates THEN the Query_Planner SHALL successfully validate and optimize the query
4. WHEN a schema contains 1000 or more tables THEN the Query_Planner SHALL efficiently look up table definitions

### Requirement 13: Correctness Requirements

**User Story:** As a database developer, I want all generated plans to be semantically correct, so that query results are accurate.

#### Acceptance Criteria

1. WHEN a plan is generated THEN the plan SHALL include all tables referenced in the FROM clause
2. WHEN a plan is generated THEN the plan SHALL apply all predicates from the WHERE clause
3. WHEN a plan is generated THEN the plan SHALL select only the columns specified in the SELECT list
4. WHEN a plan is generated THEN the plan SHALL apply joins with the correct join conditions
5. WHEN a plan is generated THEN the plan SHALL apply sorting with the correct sort keys and order
6. WHEN a plan is generated THEN the plan SHALL apply grouping with the correct grouping columns
7. WHEN a plan is generated THEN the plan SHALL apply aggregation with the correct aggregate functions
8. WHEN a plan is generated THEN the plan SHALL apply LIMIT and OFFSET correctly

### Requirement 14: Determinism Requirements

**User Story:** As a database developer, I want query planning to be deterministic, so that the same query always produces the same plan.

#### Acceptance Criteria

1. WHEN the same query is planned multiple times THEN the Query_Planner SHALL generate identical plans
2. WHEN table statistics do not change THEN the Query_Planner SHALL generate identical plans for the same query
3. WHEN the schema does not change THEN the Query_Planner SHALL generate identical plans for the same query

### Requirement 15: Integration with Parser

**User Story:** As a query planner, I want to accept parsed AST structures from the parser, so that I can process any valid SQL query.

#### Acceptance Criteria

1. WHEN the Query_Planner receives an AST from the Parser THEN it SHALL accept the AST structure
2. WHEN the Query_Planner receives a SELECT AST THEN it SHALL extract table references, columns, predicates, and join conditions
3. WHEN the Query_Planner receives an INSERT AST THEN it SHALL extract the target table and value expressions
4. WHEN the Query_Planner receives an UPDATE AST THEN it SHALL extract the target table and column assignments
5. WHEN the Query_Planner receives a DELETE AST THEN it SHALL extract the target table and filter conditions
6. WHEN the Query_Planner receives a DDL AST THEN it SHALL extract the operation type and parameters

### Requirement 16: Integration with Schema Catalog

**User Story:** As a query planner, I want to query the schema catalog for table and index information, so that I can make informed optimization decisions.

#### Acceptance Criteria

1. WHEN the Query_Planner needs table metadata THEN it SHALL query the Schema_Catalog for table definitions
2. WHEN the Query_Planner needs column information THEN it SHALL query the Schema_Catalog for column definitions
3. WHEN the Query_Planner needs index information THEN it SHALL query the Schema_Catalog for index definitions
4. WHEN the Query_Planner needs table statistics THEN it SHALL query the Schema_Catalog for row counts and column distributions
5. WHEN the Schema_Catalog is unavailable THEN the Query_Planner SHALL return an appropriate error

### Requirement 17: Integration with Executor

**User Story:** As a query executor, I want to receive well-formed physical plans from the query planner, so that I can execute them correctly.

#### Acceptance Criteria

1. WHEN the Query_Planner generates a plan THEN it SHALL return a Physical_Plan structure
2. WHEN the Query_Planner generates a plan THEN the plan SHALL contain a root Operator node
3. WHEN the Query_Planner generates a plan THEN each Operator node SHALL have a Name, EstimatedCost, EstimatedRows, and OutputColumns
4. WHEN the Query_Planner generates a plan THEN the plan SHALL be ready for immediate execution by the Executor
5. WHEN the Query_Planner generates a plan THEN the plan SHALL include all necessary metadata for the Executor to access data

### Requirement 18: Integration with Transaction Manager

**User Story:** As a transaction manager, I want to receive plan information for transaction tracking, so that I can manage locks and isolation levels.

#### Acceptance Criteria

1. WHEN the Query_Planner generates a plan THEN it SHALL identify which tables are accessed
2. WHEN the Query_Planner generates a plan THEN it SHALL identify the access type (read or write) for each table
3. WHEN the Query_Planner generates a plan THEN it SHALL provide this information to the Transaction_Manager

### Requirement 19: Documentation and Maintainability

**User Story:** As a database developer, I want clear documentation of planning algorithms, so that I can understand and maintain the code.

#### Acceptance Criteria

1. WHEN the Query_Planner is implemented THEN the code SHALL include comments explaining the cardinality estimation algorithm
2. WHEN the Query_Planner is implemented THEN the code SHALL include comments explaining the cost estimation algorithm
3. WHEN the Query_Planner is implemented THEN the code SHALL include comments explaining the index selection strategy
4. WHEN the Query_Planner is implemented THEN the code SHALL include comments explaining the join ordering algorithm
5. WHEN the Query_Planner is implemented THEN the code SHALL include comments explaining the predicate pushdown optimization

### Requirement 20: Comprehensive Test Coverage

**User Story:** As a database developer, I want comprehensive test coverage of the query planner, so that bugs are caught early.

#### Acceptance Criteria

1. WHEN the Query_Planner is tested THEN unit tests SHALL cover all plan node types
2. WHEN the Query_Planner is tested THEN unit tests SHALL cover all cost estimation algorithms
3. WHEN the Query_Planner is tested THEN unit tests SHALL cover all optimization rules
4. WHEN the Query_Planner is tested THEN unit tests SHALL cover all error conditions
5. WHEN the Query_Planner is tested THEN property-based tests SHALL verify universal properties of plan generation
6. WHEN the Query_Planner is tested THEN integration tests SHALL verify end-to-end query planning and execution

