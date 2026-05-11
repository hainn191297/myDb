# SQL Parser & Lexer Implementation Tasks

## Overview

This document outlines the implementation tasks for the SQL Parser & Lexer feature. The tasks are organized by component and dependency, with clear acceptance criteria and testing requirements.

**Total Estimated Tasks**: 25 core tasks + 15 test tasks = 40 tasks

## Task Dependency Graph

```
Foundation Tasks (Lexer & Expression Types)
├── Task 1: Define Expression AST Types
├── Task 2: Implement Lexer Tokenization
└── Task 3: Implement Expression Parser

Statement Parser Tasks
├── Task 4: Implement Transaction Control Parser
├── Task 5: Implement SELECT Parser
├── Task 6: Implement INSERT Parser
├── Task 7: Implement UPDATE Parser
├── Task 8: Implement DELETE Parser
├── Task 9: Implement CREATE TABLE Parser
├── Task 10: Implement DROP TABLE Parser
├── Task 11: Implement CREATE INDEX Parser
└── Task 12: Implement DROP INDEX Parser

Integration Tasks
├── Task 13: Integrate WHERE Expression Parsing
├── Task 14: Add Error Handling & Validation
└── Task 15: Add Backward Compatibility Support

Testing Tasks
├── Task 16-30: Unit Tests (one per parser function)
└── Task 31-40: Integration & Edge Case Tests
```

---

## Foundation Tasks

### Task 1: Define Expression AST Types

**Description**: Define all expression node types and operators for WHERE clause parsing.

**Acceptance Criteria**:
- [x] `Expr` interface defined with marker method `exprNode()`
- [x] `LiteralExpr` struct with `Value` (any) and `Type` (ValueType) fields
- [x] `ColumnRefExpr` struct with `Name` (string) field
- [x] `BinaryExpr` struct with `Left`, `Op`, `Right` fields
- [x] `UnaryExpr` struct with `Op` and `Expr` fields
- [x] `BinaryOp` enum with: OpEquals, OpNotEquals, OpLessThan, OpLessOrEqual, OpGreaterThan, OpGreaterOrEqual, OpAnd, OpOr
- [x] `UnaryOp` enum with: OpNot, OpIsNull, OpIsNotNull
- [x] `ValueType` enum with: TypeInt, TypeText, TypeBool, TypeFloat, TypeNull
- [x] All types implement `Expr` interface
- [x] `String()` methods for operators for debugging

**File**: `internal/sql/expr/expr.go`

**Dependencies**: None

**Estimated Effort**: 1 hour

---

### Task 2: Implement Lexer Tokenization

**Description**: Implement the lexer that converts WHERE clause strings into tokens.

**Acceptance Criteria**:
- [x] `tokenType` enum defined: tokenIdent, tokenNumber, tokenString, tokenOperator, tokenKeyword, tokenLParen, tokenRParen, tokenEOF
- [x] `token` struct with `typ`, `val`, `pos` fields
- [x] `tokenize(input string)` function implemented
- [x] Handles whitespace skipping
- [x] Recognizes integers and floats (including decimal points)
- [x] Recognizes single and double quoted strings with escape handling
- [x] Recognizes parentheses
- [x] Recognizes single and two-character operators (=, !=, <, >, <=, >=)
- [x] Recognizes keywords (AND, OR, NOT, IS, NULL) - case insensitive
- [x] Recognizes identifiers (column names) - case sensitive
- [x] Appends EOF token at end
- [x] Helper functions: `isWhitespace()`, `isDigit()`, `isAlpha()`, `isAlphaNum()`, `isOperatorChar()`, `isKeyword()`
- [x] Utility functions: `ParseInt()`, `ParseFloat()`, `StripQuotes()`

**File**: `internal/sql/expr/tokenizer.go`

**Dependencies**: None

**Estimated Effort**: 2 hours

**Test Coverage**:
- Tokenize simple expressions
- Tokenize numbers (int and float)
- Tokenize strings with quotes
- Tokenize operators
- Tokenize keywords
- Tokenize identifiers
- Handle whitespace
- Handle escaped characters

---

### Task 3: Implement Expression Parser

**Description**: Implement recursive descent parser for WHERE clause expressions with proper operator precedence.

**Acceptance Criteria**:
- [x] `ParseExpr(whereClause string)` main entry point
- [x] Returns nil for empty input (not an error)
- [x] `parser` struct with `tokens` and `pos` fields
- [x] `current()` method to get current token
- [x] `advance()` method to move to next token
- [x] `match(val string)` method for token matching
- [x] `expect(val string)` method for token validation
- [x] `parseOr()` function for OR expressions (lowest precedence)
- [x] `parseAnd()` function for AND expressions
- [x] `parseNot()` function for NOT expressions (recursive)
- [x] `parseComparison()` function for comparison and IS NULL operators
- [x] `parsePrimary()` function for literals, column refs, parentheses
- [x] `parseOperator(op string)` function to convert operator strings
- [x] Validates all tokens consumed after parsing
- [x] Proper error messages for malformed expressions
- [x] Handles operator precedence: OR < AND < NOT < comparison < primary
- [x] Left-associative for AND/OR
- [x] Parentheses override precedence

**File**: `internal/sql/expr/parser.go`

**Dependencies**: Task 1, Task 2

**Estimated Effort**: 3 hours

**Test Coverage**:
- Simple equality expressions
- Complex AND expressions
- OR expressions
- NOT expressions
- IS NULL / IS NOT NULL
- Operator precedence
- Parenthesized expressions
- Error cases (missing operands, unclosed parens, etc.)

---

## Statement Parser Tasks

### Task 4: Implement Transaction Control Parser

**Description**: Parse BEGIN, COMMIT, and ROLLBACK statements.

**Acceptance Criteria**:
- [x] Recognizes BEGIN statement → AST.Type = BeginStmt
- [x] Recognizes COMMIT statement → AST.Type = CommitStmt
- [x] Recognizes ROLLBACK statement → AST.Type = RollbackStmt
- [x] Case-insensitive matching
- [x] Returns appropriate AST with minimal fields

**File**: `internal/sql/parser/parser.go` - `Parse()` function

**Dependencies**: None

**Estimated Effort**: 0.5 hours

**Test Coverage**:
- Parse BEGIN
- Parse COMMIT
- Parse ROLLBACK
- Case variations

---

### Task 5: Implement SELECT Parser

**Description**: Parse SELECT statements with column list, FROM clause, and optional WHERE clause.

**Acceptance Criteria**:
- [x] `parseSelect(sql string)` function implemented
- [x] Extracts column names into AST.Columns
- [x] Extracts table name into AST.TableName
- [x] Extracts schema name into AST.SchemaName (defaults to "public")
- [x] Handles schema.table notation
- [x] Extracts WHERE clause into AST.Where (string)
- [x] Parses WHERE clause into AST.WhereExpr (structured)
- [x] Validates FROM clause exists
- [x] Validates column list exists and not empty
- [x] Validates table name exists
- [x] Returns descriptive errors for missing clauses
- [x] Sets AST.Type = SelectStmt

**File**: `internal/sql/parser/parser.go`

**Dependencies**: Task 3 (for WHERE parsing)

**Estimated Effort**: 2 hours

**Test Coverage**:
- Simple SELECT with all columns
- SELECT with specific columns
- SELECT with WHERE clause
- SELECT without WHERE clause
- SELECT with schema.table notation
- SELECT with default schema
- Error: missing FROM
- Error: missing columns
- Error: missing table name

---

### Task 6: Implement INSERT Parser

**Description**: Parse INSERT statements with optional column list and VALUES clause.

**Acceptance Criteria**:
- [x] `parseInsert(ctx context.Context, sql string)` function implemented
- [x] Extracts table name into InsertSpec.Table
- [x] Extracts schema name into InsertSpec.Schema (defaults to "public")
- [ ] Handles schema.table notation
- [x] Extracts optional column list into InsertSpec.Columns
- [x] Parses VALUES clause with one or more rows
- [x] Preserves quoted strings with quotes
- [x] Preserves numeric values as strings
- [x] Preserves boolean values as strings
- [x] Handles multiple rows: `VALUES (1, 'a'), (2, 'b')`
- [x] Validates VALUES clause exists
- [ ] Validates table name exists
- [x] Validates balanced parentheses
- [x] Returns descriptive errors
- [x] Sets AST.Type = InsertStmt
- [x] Logs debug information

**File**: `internal/sql/parser/parser.go`

**Dependencies**: None

**Estimated Effort**: 2.5 hours

**Test Coverage**:
- INSERT with column list
- INSERT without column list
- INSERT with single row
- INSERT with multiple rows
- INSERT with schema.table notation
- INSERT with default schema
- INSERT with quoted strings
- INSERT with numeric values
- INSERT with boolean values
- Error: missing table name
- Error: missing VALUES clause
- Error: unbalanced parentheses

---

### Task 7: Implement UPDATE Parser

**Description**: Parse UPDATE statements with SET clause and optional WHERE clause.

**Acceptance Criteria**:
- [x] `parseUpdate(sql string)` function implemented
- [x] Extracts table name into UpdateSpec.Table
- [x] Extracts schema name into UpdateSpec.Schema (defaults to "public")
- [ ] Handles schema.table notation
- [x] Parses SET clause into UpdateSpec.SetClauses (map[string]string)
- [x] Handles multiple SET clauses: `SET col1 = val1, col2 = val2`
- [x] Extracts WHERE clause into UpdateSpec.Where (string)
- [ ] Parses WHERE clause into AST.WhereExpr (structured)
- [x] Validates SET clause exists
- [ ] Validates table name exists
- [ ] Returns descriptive errors
- [x] Sets AST.Type = UpdateStmt

**File**: `internal/sql/parser/parser.go`

**Dependencies**: Task 3 (for WHERE parsing)

**Estimated Effort**: 2 hours

**Test Coverage**:
- UPDATE with single SET clause
- UPDATE with multiple SET clauses
- UPDATE with WHERE clause
- UPDATE without WHERE clause
- UPDATE with schema.table notation
- UPDATE with default schema
- Error: missing table name
- Error: missing SET clause

---

### Task 8: Implement DELETE Parser

**Description**: Parse DELETE statements with optional WHERE clause.

**Acceptance Criteria**:
- [x] `parseDelete(sql string)` function implemented
- [x] Extracts table name into DeleteSpec.Table
- [x] Extracts schema name into DeleteSpec.Schema (defaults to "public")
- [ ] Handles schema.table notation
- [x] Extracts WHERE clause into DeleteSpec.Where (string)
- [ ] Parses WHERE clause into AST.WhereExpr (structured)
- [ ] Validates FROM clause exists
- [ ] Validates table name exists
- [ ] Returns descriptive errors
- [x] Sets AST.Type = DeleteStmt

**File**: `internal/sql/parser/parser.go`

**Dependencies**: Task 3 (for WHERE parsing)

**Estimated Effort**: 1.5 hours

**Test Coverage**:
- DELETE with WHERE clause
- DELETE without WHERE clause
- DELETE with schema.table notation
- DELETE with default schema
- Error: missing FROM clause
- Error: missing table name

---

### Task 9: Implement CREATE TABLE Parser

**Description**: Parse CREATE TABLE statements with column definitions and constraints.

**Acceptance Criteria**:
- [x] `parseCreateTable(sql string)` function implemented
- [x] Extracts table name into CreateTableSpec.Table
- [x] Extracts schema name into CreateTableSpec.Schema (defaults to "public")
- [ ] Handles schema.table notation
- [x] `parseColumnDefs(columnsPart string)` function for parsing column list
- [x] `parseColumnDef(def string)` function for parsing single column
- [x] Extracts column name, type, nullable, primaryKey for each column
- [x] Recognizes types: INT, TEXT, BOOL, FLOAT (case-insensitive)
- [x] Recognizes NOT NULL constraint
- [x] Recognizes PRIMARY KEY constraint
- [x] PRIMARY KEY implies NOT NULL
- [ ] Validates table name exists
- [x] Validates column definitions exist and not empty
- [ ] Validates balanced parentheses
- [ ] Returns descriptive errors
- [x] Sets AST.Type = CreateTableStmt

**File**: `internal/sql/parser/parser.go`

**Dependencies**: None

**Estimated Effort**: 2.5 hours

**Test Coverage**:
- CREATE TABLE with simple columns
- CREATE TABLE with NOT NULL constraint
- CREATE TABLE with PRIMARY KEY constraint
- CREATE TABLE with mixed constraints
- CREATE TABLE with schema.table notation
- CREATE TABLE with default schema
- Error: missing table name
- Error: missing column definitions
- Error: empty column list
- Error: incomplete column definition
- Error: unbalanced parentheses

---

### Task 10: Implement DROP TABLE Parser

**Description**: Parse DROP TABLE statements.

**Acceptance Criteria**:
- [x] `parseDropTable(sql string)` function implemented
- [x] Extracts table name into DropTableSpec.Table
- [x] Extracts schema name into DropTableSpec.Schema (defaults to "public")
- [ ] Handles schema.table notation
- [ ] Validates table name exists
- [ ] Returns descriptive errors
- [x] Sets AST.Type = DropTableStmt

**File**: `internal/sql/parser/parser.go`

**Dependencies**: None

**Estimated Effort**: 1 hour

**Test Coverage**:
- DROP TABLE with simple name
- DROP TABLE with schema.table notation
- DROP TABLE with default schema
- Error: missing table name

---

### Task 11: Implement CREATE INDEX Parser

**Description**: Parse CREATE INDEX statements with optional UNIQUE keyword.

**Acceptance Criteria**:
- [x] `parseCreateIndex(sql string)` function implemented
- [x] Recognizes CREATE INDEX and CREATE UNIQUE INDEX
- [x] Extracts index name into CreateIndexSpec.IndexName
- [x] Extracts table name into CreateIndexSpec.Table
- [x] Extracts schema name into CreateIndexSpec.Schema (defaults to "public")
- [ ] Handles schema.table notation
- [x] Extracts column list into CreateIndexSpec.Columns
- [x] Sets CreateIndexSpec.Unique based on UNIQUE keyword
- [x] Validates ON clause exists
- [ ] Validates column list exists and not empty
- [x] Validates index name exists
- [ ] Validates balanced parentheses
- [ ] Returns descriptive errors
- [x] Sets AST.Type = CreateIndexStmt

**File**: `internal/sql/parser/parser.go`

**Dependencies**: None

**Estimated Effort**: 2 hours

**Test Coverage**:
- CREATE INDEX with single column
- CREATE INDEX with multiple columns
- CREATE UNIQUE INDEX
- CREATE INDEX with schema.table notation
- CREATE INDEX with default schema
- Error: missing ON clause
- Error: missing column list
- Error: missing index name

---

### Task 12: Implement DROP INDEX Parser

**Description**: Parse DROP INDEX statements.

**Acceptance Criteria**:
- [x] `parseDropIndex(sql string)` function implemented
- [x] Extracts index name into DropIndexSpec.IndexName
- [x] Extracts table name into DropIndexSpec.Table
- [x] Extracts schema name into DropIndexSpec.Schema (defaults to "public")
- [ ] Handles schema.table notation
- [ ] Validates ON clause exists
- [ ] Validates table name exists
- [ ] Validates index name exists
- [ ] Returns descriptive errors
- [x] Sets AST.Type = DropIndexStmt

**File**: `internal/sql/parser/parser.go`

**Dependencies**: None

**Estimated Effort**: 1.5 hours

**Test Coverage**:
- DROP INDEX with simple name
- DROP INDEX with schema.table notation
- DROP INDEX with default schema
- Error: missing ON clause
- Error: missing table name
- Error: missing index name

---

## Integration Tasks

### Task 13: Integrate WHERE Expression Parsing

**Description**: Integrate expression parser with DML statement parsers for WHERE clauses.

**Acceptance Criteria**:
- [x] SELECT parser calls `expr.ParseExpr()` for WHERE clause
- [x] UPDATE parser calls `expr.ParseExpr()` for WHERE clause
- [x] DELETE parser calls `expr.ParseExpr()` for WHERE clause
- [x] Populates both string-based WHERE and structured WhereExpr fields
- [x] Wraps expression parsing errors with statement context
- [x] Returns nil WhereExpr for empty WHERE clauses
- [x] Maintains backward compatibility with string-based WHERE

**File**: `internal/sql/parser/parser.go`

**Dependencies**: Task 3, Task 5, Task 7, Task 8

**Estimated Effort**: 1 hour

**Test Coverage**:
- SELECT with WHERE clause parsing
- UPDATE with WHERE clause parsing
- DELETE with WHERE clause parsing
- Error propagation from expression parser

---

### Task 14: Add Error Handling & Validation

**Description**: Implement comprehensive error handling and validation across all parsers.

**Acceptance Criteria**:
- [x] `ErrEmptyStatement` error defined and used
- [x] All required fields validated before returning AST
- [x] Descriptive error messages with context
- [x] Error messages indicate which clause is missing/invalid
- [x] Consistent error format across all parsers
- [x] Errors wrapped with context using `fmt.Errorf(...: %w, err)`
- [x] Validation for balanced parentheses
- [x] Validation for required keywords (FROM, SET, VALUES, etc.)
- [x] Validation for table/column names not empty

**File**: `internal/sql/parser/parser.go`, `internal/errors/errors.go`

**Dependencies**: All parser tasks

**Estimated Effort**: 1.5 hours

**Test Coverage**:
- All error cases from requirements
- Edge cases (empty strings, special characters, etc.)

---

### Task 15: Add Backward Compatibility Support

**Description**: Ensure dual support for string-based and structured WHERE clauses.

**Acceptance Criteria**:
- [x] AST.Where field populated with raw WHERE clause string
- [x] UpdateSpec.Where field populated with raw WHERE clause string
- [x] DeleteSpec.Where field populated with raw WHERE clause string
- [x] AST.WhereExpr field populated with parsed expression tree
- [x] Both fields populated simultaneously for DML statements
- [x] Documentation indicates string-based fields are deprecated
- [x] Code comments explain migration path
- [x] No breaking changes to existing code

**File**: `internal/sql/parser/parser.go`

**Dependencies**: Task 13

**Estimated Effort**: 0.5 hours

**Test Coverage**:
- Verify both fields populated
- Verify string-based fields contain raw WHERE clause
- Verify structured fields contain parsed expression

---

## Testing Tasks

### Task 16: Unit Tests - Transaction Control Parser

**Description**: Test transaction control statement parsing.

**File**: `internal/sql/parser/parser_test.go`

**Test Cases**:
- [x] TestParseTxnStatements (existing)
- [x] TestParseBegin
- [x] TestParseCommit
- [x] TestParseRollback
- [x] TestParseTxnWithWhitespace
- [x] TestParseTxnWithSemicolon

**Estimated Effort**: 0.5 hours

---

### Task 17: Unit Tests - SELECT Parser

**Description**: Test SELECT statement parsing.

**File**: `internal/sql/parser/parser_test.go`

**Test Cases**:
- [x] TestParseSelect (existing)
- [x] TestParseSelectWithWhere
- [x] TestParseSelectDefaultSchema (existing)
- [x] TestParseSelectWithSchemaTable
- [x] TestParseSelectMultipleColumns
- [x] TestParseSelectErrors
- [x] TestParseSelectMissingFrom
- [x] TestParseSelectMissingColumns
- [x] TestParseSelectMissingTable

**Estimated Effort**: 1 hour

---

### Task 18: Unit Tests - INSERT Parser

**Description**: Test INSERT statement parsing.

**File**: `internal/sql/parser/parser_dml_test.go`

**Test Cases**:
- [x] TestParseInsert (existing)
- [x] TestParseInsertWithColumns (existing)
- [x] TestParseInsertWithSchema (existing)
- [x] TestParseInsertMultipleRows
- [x] TestParseInsertWithQuotedStrings
- [x] TestParseInsertWithNumbers
- [x] TestParseInsertWithBooleans
- [x] TestParseInsertErrors
- [x] TestParseInsertMissingTable
- [x] TestParseInsertMissingValues
- [x] TestParseInsertUnbalancedParens

**Estimated Effort**: 1.5 hours

---

### Task 19: Unit Tests - UPDATE Parser

**Description**: Test UPDATE statement parsing.

**File**: `internal/sql/parser/parser_dml_test.go`

**Test Cases**:
- [x] TestParseUpdate (existing)
- [x] TestParseUpdateNoWhere (existing)
- [x] TestParseUpdateWithSchema
- [x] TestParseUpdateMultipleSetClauses
- [x] TestParseUpdateWithWhere
- [x] TestParseUpdateErrors
- [x] TestParseUpdateMissingTable
- [x] TestParseUpdateMissingSet

**Estimated Effort**: 1 hour

---

### Task 20: Unit Tests - DELETE Parser

**Description**: Test DELETE statement parsing.

**File**: `internal/sql/parser/parser_dml_test.go`

**Test Cases**:
- [x] TestParseDelete
- [x] TestParseDeleteWithWhere
- [x] TestParseDeleteWithSchema
- [x] TestParseDeleteErrors
- [x] TestParseDeleteMissingFrom
- [x] TestParseDeleteMissingTable

**Estimated Effort**: 0.75 hours

---

### Task 21: Unit Tests - CREATE TABLE Parser

**Description**: Test CREATE TABLE statement parsing.

**File**: `internal/sql/parser/parser_test.go`

**Test Cases**:
- [x] TestParseCreateTable (existing)
- [x] TestParseCreateTableWithConstraints
- [x] TestParseCreateTableWithPrimaryKey
- [x] TestParseCreateTableWithSchema
- [x] TestParseCreateTableMultipleColumns
- [x] TestParseCreateTableErrors
- [x] TestParseCreateTableMissingName
- [x] TestParseCreateTableMissingColumns
- [x] TestParseCreateTableEmptyColumns
- [x] TestParseCreateTableUnbalancedParens

**Estimated Effort**: 1.5 hours

---

### Task 22: Unit Tests - DROP TABLE Parser

**Description**: Test DROP TABLE statement parsing.

**File**: `internal/sql/parser/parser_test.go`

**Test Cases**:
- [x] TestParseDropTable
- [x] TestParseDropTableWithSchema
- [x] TestParseDropTableErrors
- [x] TestParseDropTableMissingTable

**Estimated Effort**: 0.5 hours

---

### Task 23: Unit Tests - CREATE INDEX Parser

**Description**: Test CREATE INDEX statement parsing.

**File**: `internal/sql/parser/parser_index_test.go`

**Test Cases**:
- [x] TestParseCreateIndex (existing)
- [x] TestParseCreateUniqueIndex
- [x] TestParseCreateIndexWithSchema
- [x] TestParseCreateIndexMultipleColumns
- [x] TestParseCreateIndexErrors
- [x] TestParseCreateIndexMissingOn
- [x] TestParseCreateIndexMissingColumns
- [x] TestParseCreateIndexMissingName

**Estimated Effort**: 1 hour

---

### Task 24: Unit Tests - DROP INDEX Parser

**Description**: Test DROP INDEX statement parsing.

**File**: `internal/sql/parser/parser_index_test.go`

**Test Cases**:
- [x] TestParseDropIndex (existing)
- [x] TestParseDropIndexWithSchema
- [x] TestParseDropIndexErrors
- [x] TestParseDropIndexMissingOn
- [x] TestParseDropIndexMissingTable
- [x] TestParseDropIndexMissingName

**Estimated Effort**: 0.75 hours

---

### Task 25: Unit Tests - Lexer

**Description**: Test lexer tokenization.

**File**: `internal/sql/expr/expr_test.go` (or new file)

**Test Cases**:
- [x] TestTokenizeNumbers
- [x] TestTokenizeStrings
- [x] TestTokenizeOperators
- [x] TestTokenizeKeywords
- [x] TestTokenizeIdentifiers
- [x] TestTokenizeParentheses
- [x] TestTokenizeWhitespace
- [x] TestTokenizeEscapedCharacters
- [x] TestTokenizeComplexExpression

**Estimated Effort**: 1 hour

---

### Task 26: Unit Tests - Expression Parser

**Description**: Test expression parser.

**File**: `internal/sql/expr/expr_test.go`

**Test Cases**:
- [x] TestParseSimpleEquality (existing)
- [x] TestParseComplexAnd (existing)
- [x] TestParseOrExpression (existing)
- [x] TestParseNot (existing)
- [x] TestParseIsNull (existing)
- [x] TestParseIsNotNull
- [x] TestParseParenthesizedExpression
- [x] TestParseOperatorPrecedence
- [x] TestParseMultipleNot
- [x] TestParseLiterals
- [x] TestParseColumnReferences
- [x] TestParseExpressionErrors
- [x] TestParseEmptyExpression

**Estimated Effort**: 1.5 hours

---

### Task 27: Integration Tests - WHERE Clause Parsing

**Description**: Test WHERE clause integration with DML statements.

**File**: `internal/sql/parser/parser_dml_test.go`

**Test Cases**:
- [x] TestSelectWithComplexWhere
- [x] TestUpdateWithComplexWhere
- [x] TestDeleteWithComplexWhere
- [x] TestWhereExprPopulation
- [x] TestWhereStringBackwardCompat

**Estimated Effort**: 1 hour

---

### Task 28: Integration Tests - Error Propagation

**Description**: Test error handling and propagation.

**File**: `internal/sql/parser/parser_test.go`

**Test Cases**:
- [x] TestParseErrors (existing)
- [x] TestExpressionErrorPropagation
- [x] TestMissingClauseErrors
- [x] TestMalformedSyntaxErrors
- [x] TestErrorMessages

**Estimated Effort**: 1 hour

---

### Task 29: Integration Tests - Schema Handling

**Description**: Test schema.table notation and default schema handling.

**File**: `internal/sql/parser/parser_test.go`

**Test Cases**:
- [x] TestSchemaTableNotation
- [x] TestDefaultSchemaHandling
- [x] TestSchemaAcrossAllStatements

**Estimated Effort**: 0.75 hours

---

### Task 30: Edge Case & Regression Tests

**Description**: Test edge cases and potential regressions.

**File**: `internal/sql/parser/parser_test.go`

**Test Cases**:
- [x] TestEmptyStatement
- [x] TestWhitespaceHandling
- [x] TestSemicolonHandling
- [x] TestCaseInsensitivity
- [x] TestSpecialCharactersInStrings
- [x] TestLargeStatements
- [x] TestUnicodeCharacters
- [x] TestCommentHandling (if applicable)

**Estimated Effort**: 1.5 hours

---

## Summary

**Total Estimated Effort**: ~40-45 hours

**Breakdown**:
- Foundation Tasks (1-3): 6 hours
- Statement Parser Tasks (4-12): 16 hours
- Integration Tasks (13-15): 3 hours
- Testing Tasks (16-30): 15-20 hours

**Recommended Implementation Order**:
1. Complete all Foundation Tasks first (Tasks 1-3)
2. Implement Statement Parser Tasks in order (Tasks 4-12)
3. Complete Integration Tasks (Tasks 13-15)
4. Implement Testing Tasks in parallel with implementation (Tasks 16-30)

**Quality Gates**:
- All unit tests pass
- All integration tests pass
- Code coverage > 90%
- No regressions in existing tests
- All error cases handled gracefully
- Documentation updated

**Success Criteria**:
- All 20 requirements satisfied
- All 30 tasks completed
- All tests passing
- Code review approved
- Ready for integration with planner and executor
