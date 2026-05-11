# SQL Parser & Lexer Requirements

## Introduction

The SQL Parser & Lexer is a core component of myDb that converts raw SQL strings into Abstract Syntax Trees (ASTs) for downstream processing by the planner and executor. The parser supports a subset of SQL including DDL (CREATE/DROP TABLE/INDEX), DML (SELECT/INSERT/UPDATE/DELETE), TCL (BEGIN/COMMIT/ROLLBACK), and expression parsing for WHERE clauses. The lexer tokenizes input strings into meaningful tokens for expression parsing.

The parser is designed to be learning-oriented and maintainable, with clear separation between statement-level parsing and expression parsing. It provides structured AST representations that enable the planner to build efficient execution plans.

## Glossary

- **AST**: Abstract Syntax Tree - a structured representation of a parsed SQL statement
- **Lexer**: A tokenizer that converts input strings into tokens (used for expression parsing)
- **Parser**: The component that converts SQL strings into AST structures
- **Statement**: A complete SQL command (SELECT, INSERT, CREATE TABLE, etc.)
- **Expression**: A WHERE clause or value expression (e.g., `id = 1 AND status = 'active'`)
- **Token**: A lexical unit produced by the lexer (identifier, number, operator, keyword, etc.)
- **Schema**: A namespace for tables (defaults to "public" if not specified)
- **DDL**: Data Definition Language (CREATE TABLE, DROP TABLE, CREATE INDEX, DROP INDEX)
- **DML**: Data Manipulation Language (SELECT, INSERT, UPDATE, DELETE)
- **TCL**: Transaction Control Language (BEGIN, COMMIT, ROLLBACK)
- **Operator**: A symbol representing an operation (=, !=, <, >, <=, >=, AND, OR, NOT, IS NULL, IS NOT NULL)
- **Literal**: A constant value (number, string, boolean, null)
- **Column Reference**: A reference to a column by name
- **Binary Expression**: An expression with two operands and an operator (e.g., `a = b`)
- **Unary Expression**: An expression with one operand and an operator (e.g., `NOT x`, `col IS NULL`)

## Requirements

### Requirement 1: Parse Transaction Control Statements

**User Story:** As a server, I want to parse transaction control statements, so that I can manage transaction boundaries.

#### Acceptance Criteria

1. WHEN a BEGIN statement is provided, THE Parser SHALL parse it into an AST with Type = BeginStmt
2. WHEN a COMMIT statement is provided, THE Parser SHALL parse it into an AST with Type = CommitStmt
3. WHEN a ROLLBACK statement is provided, THE Parser SHALL parse it into an AST with Type = RollbackStmt
4. WHEN a statement is provided with leading/trailing whitespace or semicolon, THE Parser SHALL trim it before parsing
5. WHEN an empty statement is provided, THE Parser SHALL return ErrEmptyStatement

### Requirement 2: Parse SELECT Statements

**User Story:** As a planner, I want to parse SELECT statements, so that I can build query execution plans.

#### Acceptance Criteria

1. WHEN a SELECT statement is provided with column list and FROM clause, THE Parser SHALL extract the column names into AST.Columns
2. WHEN a SELECT statement is provided with a table name, THE Parser SHALL extract the table name into AST.TableName
3. WHEN a SELECT statement is provided with schema.table notation, THE Parser SHALL extract both schema and table names
4. WHEN a SELECT statement is provided without schema prefix, THE Parser SHALL default SchemaName to "public"
5. WHEN a SELECT statement is provided with a WHERE clause, THE Parser SHALL extract the WHERE expression into AST.Where (string) and AST.WhereExpr (structured)
6. WHEN a SELECT statement is provided without a WHERE clause, THE Parser SHALL leave AST.Where empty and AST.WhereExpr nil
7. WHEN a SELECT statement is missing the FROM clause, THE Parser SHALL return an error
8. WHEN a SELECT statement is missing the column list, THE Parser SHALL return an error
9. WHEN a SELECT statement is missing the table name, THE Parser SHALL return an error

### Requirement 3: Parse INSERT Statements

**User Story:** As an executor, I want to parse INSERT statements, so that I can insert rows into tables.

#### Acceptance Criteria

1. WHEN an INSERT statement is provided with VALUES clause, THE Parser SHALL extract table name, schema, and rows into InsertSpec
2. WHEN an INSERT statement is provided with column list, THE Parser SHALL extract column names into InsertSpec.Columns
3. WHEN an INSERT statement is provided without column list, THE Parser SHALL leave InsertSpec.Columns empty
4. WHEN an INSERT statement is provided with multiple rows, THE Parser SHALL parse all rows into InsertSpec.Values as a list of lists
5. WHEN an INSERT statement is provided with single row, THE Parser SHALL parse it into InsertSpec.Values with one row
6. WHEN an INSERT statement is provided with quoted string values, THE Parser SHALL preserve quotes in the raw value strings
7. WHEN an INSERT statement is provided with numeric values, THE Parser SHALL preserve them as string representations
8. WHEN an INSERT statement is provided with boolean values, THE Parser SHALL preserve them as string representations
9. WHEN an INSERT statement is provided with schema.table notation, THE Parser SHALL extract both schema and table names
10. WHEN an INSERT statement is provided without schema prefix, THE Parser SHALL default SchemaName to "public"
11. WHEN an INSERT statement is missing the table name, THE Parser SHALL return an error
12. WHEN an INSERT statement is missing the VALUES clause, THE Parser SHALL return an error
13. WHEN an INSERT statement has unbalanced parentheses in VALUES, THE Parser SHALL return an error

### Requirement 4: Parse UPDATE Statements

**User Story:** As an executor, I want to parse UPDATE statements, so that I can modify existing rows.

#### Acceptance Criteria

1. WHEN an UPDATE statement is provided with SET clause, THE Parser SHALL extract column-value pairs into UpdateSpec.SetClauses
2. WHEN an UPDATE statement is provided with WHERE clause, THE Parser SHALL extract the WHERE expression into UpdateSpec.Where (string) and AST.WhereExpr (structured)
3. WHEN an UPDATE statement is provided without WHERE clause, THE Parser SHALL leave UpdateSpec.Where empty and AST.WhereExpr nil
4. WHEN an UPDATE statement is provided with schema.table notation, THE Parser SHALL extract both schema and table names
5. WHEN an UPDATE statement is provided without schema prefix, THE Parser SHALL default SchemaName to "public"
6. WHEN an UPDATE statement is provided with multiple SET clauses, THE Parser SHALL parse all of them into the map
7. WHEN an UPDATE statement is missing the table name, THE Parser SHALL return an error
8. WHEN an UPDATE statement is missing the SET clause, THE Parser SHALL return an error

### Requirement 5: Parse DELETE Statements

**User Story:** As an executor, I want to parse DELETE statements, so that I can remove rows from tables.

#### Acceptance Criteria

1. WHEN a DELETE statement is provided with FROM clause, THE Parser SHALL extract the table name into DeleteSpec.Table
2. WHEN a DELETE statement is provided with WHERE clause, THE Parser SHALL extract the WHERE expression into DeleteSpec.Where (string) and AST.WhereExpr (structured)
3. WHEN a DELETE statement is provided without WHERE clause, THE Parser SHALL leave DeleteSpec.Where empty and AST.WhereExpr nil
4. WHEN a DELETE statement is provided with schema.table notation, THE Parser SHALL extract both schema and table names
5. WHEN a DELETE statement is provided without schema prefix, THE Parser SHALL default SchemaName to "public"
6. WHEN a DELETE statement is missing the FROM clause, THE Parser SHALL return an error
7. WHEN a DELETE statement is missing the table name, THE Parser SHALL return an error

### Requirement 6: Parse CREATE TABLE Statements

**User Story:** As a catalog, I want to parse CREATE TABLE statements, so that I can create table schemas.

#### Acceptance Criteria

1. WHEN a CREATE TABLE statement is provided with column definitions, THE Parser SHALL extract all columns into CreateTableSpec.Columns
2. WHEN a CREATE TABLE statement is provided with column type, THE Parser SHALL extract the type (INT, TEXT, BOOL, FLOAT) into ColumnSpec.Type
3. WHEN a CREATE TABLE statement is provided with NOT NULL constraint, THE Parser SHALL set ColumnSpec.Nullable to false
4. WHEN a CREATE TABLE statement is provided without NOT NULL constraint, THE Parser SHALL set ColumnSpec.Nullable to true
5. WHEN a CREATE TABLE statement is provided with PRIMARY KEY constraint, THE Parser SHALL set ColumnSpec.PrimaryKey to true and Nullable to false
6. WHEN a CREATE TABLE statement is provided with schema.table notation, THE Parser SHALL extract both schema and table names
7. WHEN a CREATE TABLE statement is provided without schema prefix, THE Parser SHALL default SchemaName to "public"
8. WHEN a CREATE TABLE statement is missing the table name, THE Parser SHALL return an error
9. WHEN a CREATE TABLE statement is missing column definitions, THE Parser SHALL return an error
10. WHEN a CREATE TABLE statement has empty column list, THE Parser SHALL return an error
11. WHEN a CREATE TABLE statement has incomplete column definition, THE Parser SHALL return an error
12. WHEN a CREATE TABLE statement has unbalanced parentheses, THE Parser SHALL return an error

### Requirement 7: Parse DROP TABLE Statements

**User Story:** As a catalog, I want to parse DROP TABLE statements, so that I can remove table schemas.

#### Acceptance Criteria

1. WHEN a DROP TABLE statement is provided with table name, THE Parser SHALL extract the table name into DropTableSpec.Table
2. WHEN a DROP TABLE statement is provided with schema.table notation, THE Parser SHALL extract both schema and table names
3. WHEN a DROP TABLE statement is provided without schema prefix, THE Parser SHALL default SchemaName to "public"
4. WHEN a DROP TABLE statement is missing the table name, THE Parser SHALL return an error

### Requirement 8: Parse CREATE INDEX Statements

**User Story:** As a catalog, I want to parse CREATE INDEX statements, so that I can create indexes on tables.

#### Acceptance Criteria

1. WHEN a CREATE INDEX statement is provided with index name and column list, THE Parser SHALL extract them into CreateIndexSpec
2. WHEN a CREATE INDEX statement is provided with ON clause, THE Parser SHALL extract the table name
3. WHEN a CREATE INDEX statement is provided with UNIQUE keyword, THE Parser SHALL set CreateIndexSpec.Unique to true
4. WHEN a CREATE INDEX statement is provided without UNIQUE keyword, THE Parser SHALL set CreateIndexSpec.Unique to false
5. WHEN a CREATE INDEX statement is provided with schema.table notation, THE Parser SHALL extract both schema and table names
6. WHEN a CREATE INDEX statement is provided without schema prefix, THE Parser SHALL default SchemaName to "public"
7. WHEN a CREATE INDEX statement is provided with multiple columns, THE Parser SHALL extract all columns into CreateIndexSpec.Columns
8. WHEN a CREATE INDEX statement is missing the ON clause, THE Parser SHALL return an error
9. WHEN a CREATE INDEX statement is missing the column list, THE Parser SHALL return an error
10. WHEN a CREATE INDEX statement is missing the index name, THE Parser SHALL return an error

### Requirement 9: Parse DROP INDEX Statements

**User Story:** As a catalog, I want to parse DROP INDEX statements, so that I can remove indexes.

#### Acceptance Criteria

1. WHEN a DROP INDEX statement is provided with index name and ON clause, THE Parser SHALL extract them into DropIndexSpec
2. WHEN a DROP INDEX statement is provided with table name, THE Parser SHALL extract the table name
3. WHEN a DROP INDEX statement is provided with schema.table notation, THE Parser SHALL extract both schema and table names
4. WHEN a DROP INDEX statement is provided without schema prefix, THE Parser SHALL default SchemaName to "public"
5. WHEN a DROP INDEX statement is missing the ON clause, THE Parser SHALL return an error
6. WHEN a DROP INDEX statement is missing the table name, THE Parser SHALL return an error
7. WHEN a DROP INDEX statement is missing the index name, THE Parser SHALL return an error

### Requirement 10: Tokenize Expression Input

**User Story:** As an expression parser, I want to tokenize WHERE clause strings, so that I can parse them into structured expressions.

#### Acceptance Criteria

1. WHEN a WHERE clause string is provided, THE Lexer SHALL tokenize it into a sequence of tokens
2. WHEN a token is a number (integer or float), THE Lexer SHALL classify it as tokenNumber
3. WHEN a token is a quoted string (single or double quotes), THE Lexer SHALL classify it as tokenString and preserve the quotes
4. WHEN a token is an identifier (column name), THE Lexer SHALL classify it as tokenIdent
5. WHEN a token is an operator (=, !=, <, >, <=, >=), THE Lexer SHALL classify it as tokenOperator
6. WHEN a token is a keyword (AND, OR, NOT, IS, NULL), THE Lexer SHALL classify it as tokenKeyword
7. WHEN a token is a left parenthesis, THE Lexer SHALL classify it as tokenLParen
8. WHEN a token is a right parenthesis, THE Lexer SHALL classify it as tokenRParen
9. WHEN a WHERE clause string ends, THE Lexer SHALL append a tokenEOF token
10. WHEN a WHERE clause string contains whitespace, THE Lexer SHALL skip it and not produce tokens for it
11. WHEN a WHERE clause string contains escaped characters in strings, THE Lexer SHALL handle them correctly

### Requirement 11: Parse Binary Expressions

**User Story:** As an expression evaluator, I want to parse binary expressions, so that I can evaluate WHERE clauses.

#### Acceptance Criteria

1. WHEN a binary expression with comparison operator is provided, THE Expression_Parser SHALL parse it into a BinaryExpr with the correct operator
2. WHEN a binary expression with = operator is provided, THE Expression_Parser SHALL set Op to OpEquals
3. WHEN a binary expression with != operator is provided, THE Expression_Parser SHALL set Op to OpNotEquals
4. WHEN a binary expression with < operator is provided, THE Expression_Parser SHALL set Op to OpLessThan
5. WHEN a binary expression with <= operator is provided, THE Expression_Parser SHALL set Op to OpLessOrEqual
6. WHEN a binary expression with > operator is provided, THE Expression_Parser SHALL set Op to OpGreaterThan
7. WHEN a binary expression with >= operator is provided, THE Expression_Parser SHALL set Op to OpGreaterOrEqual
8. WHEN a binary expression with AND operator is provided, THE Expression_Parser SHALL set Op to OpAnd
9. WHEN a binary expression with OR operator is provided, THE Expression_Parser SHALL set Op to OpOr
10. WHEN a binary expression is provided, THE Expression_Parser SHALL extract the left operand into BinaryExpr.Left
11. WHEN a binary expression is provided, THE Expression_Parser SHALL extract the right operand into BinaryExpr.Right

### Requirement 12: Parse Unary Expressions

**User Story:** As an expression evaluator, I want to parse unary expressions, so that I can handle NOT and IS NULL operations.

#### Acceptance Criteria

1. WHEN a NOT expression is provided, THE Expression_Parser SHALL parse it into a UnaryExpr with Op = OpNot
2. WHEN an IS NULL expression is provided, THE Expression_Parser SHALL parse it into a UnaryExpr with Op = OpIsNull
3. WHEN an IS NOT NULL expression is provided, THE Expression_Parser SHALL parse it into a UnaryExpr with Op = OpIsNotNull
4. WHEN a unary expression is provided, THE Expression_Parser SHALL extract the operand into UnaryExpr.Expr
5. WHEN multiple NOT operators are provided, THE Expression_Parser SHALL parse them recursively

### Requirement 13: Parse Literal Values

**User Story:** As an expression evaluator, I want to parse literal values, so that I can evaluate expressions against data.

#### Acceptance Criteria

1. WHEN an integer literal is provided, THE Expression_Parser SHALL parse it into a LiteralExpr with Type = TypeInt and Value as int64
2. WHEN a float literal is provided, THE Expression_Parser SHALL parse it into a LiteralExpr with Type = TypeFloat and Value as float64
3. WHEN a string literal is provided, THE Expression_Parser SHALL parse it into a LiteralExpr with Type = TypeText and Value as string (quotes removed)
4. WHEN a boolean literal (true/false) is provided, THE Expression_Parser SHALL parse it into a LiteralExpr with Type = TypeBool and Value as bool
5. WHEN a NULL literal is provided, THE Expression_Parser SHALL parse it into a LiteralExpr with Type = TypeNull and Value as nil

### Requirement 14: Parse Column References

**User Story:** As an expression evaluator, I want to parse column references, so that I can identify which columns are used in expressions.

#### Acceptance Criteria

1. WHEN a column name is provided in an expression, THE Expression_Parser SHALL parse it into a ColumnRefExpr
2. WHEN a column reference is provided, THE Expression_Parser SHALL extract the column name into ColumnRefExpr.Name

### Requirement 15: Handle Expression Operator Precedence

**User Story:** As an expression evaluator, I want expressions to be parsed with correct operator precedence, so that complex WHERE clauses are evaluated correctly.

#### Acceptance Criteria

1. WHEN an expression with mixed operators is provided, THE Expression_Parser SHALL apply precedence: OR (lowest) < AND < NOT < comparison (highest)
2. WHEN an expression with parentheses is provided, THE Expression_Parser SHALL evaluate parenthesized sub-expressions first
3. WHEN an expression with multiple AND operators is provided, THE Expression_Parser SHALL associate them left-to-right
4. WHEN an expression with multiple OR operators is provided, THE Expression_Parser SHALL associate them left-to-right

### Requirement 16: Handle Expression Parsing Errors

**User Story:** As an expression evaluator, I want to detect malformed expressions, so that I can report errors to the user.

#### Acceptance Criteria

1. WHEN an expression with missing right operand is provided, THE Expression_Parser SHALL return an error
2. WHEN an expression with missing left operand is provided, THE Expression_Parser SHALL return an error
3. WHEN an expression with missing operator is provided, THE Expression_Parser SHALL return an error
4. WHEN an expression with incomplete AND is provided, THE Expression_Parser SHALL return an error
5. WHEN an expression with unclosed parenthesis is provided, THE Expression_Parser SHALL return an error
6. WHEN an expression with unexpected token after complete expression is provided, THE Expression_Parser SHALL return an error
7. WHEN an empty WHERE clause is provided, THE Expression_Parser SHALL return nil (not an error)

### Requirement 17: Identify Statement Type

**User Story:** As a server, I want to identify the type of SQL statement, so that I can route it to the appropriate handler.

#### Acceptance Criteria

1. WHEN a SELECT statement is provided, THE Parser SHALL set AST.Type to SelectStmt
2. WHEN an INSERT statement is provided, THE Parser SHALL set AST.Type to InsertStmt
3. WHEN an UPDATE statement is provided, THE Parser SHALL set AST.Type to UpdateStmt
4. WHEN a DELETE statement is provided, THE Parser SHALL set AST.Type to DeleteStmt
5. WHEN a CREATE TABLE statement is provided, THE Parser SHALL set AST.Type to CreateTableStmt
6. WHEN a DROP TABLE statement is provided, THE Parser SHALL set AST.Type to DropTableStmt
7. WHEN a CREATE INDEX statement is provided, THE Parser SHALL set AST.Type to CreateIndexStmt
8. WHEN a DROP INDEX statement is provided, THE Parser SHALL set AST.Type to DropIndexStmt
9. WHEN a BEGIN statement is provided, THE Parser SHALL set AST.Type to BeginStmt
10. WHEN a COMMIT statement is provided, THE Parser SHALL set AST.Type to CommitStmt
11. WHEN a ROLLBACK statement is provided, THE Parser SHALL set AST.Type to RollbackStmt

### Requirement 18: Handle Unsupported Statements

**User Story:** As a server, I want to reject unsupported SQL statements, so that users know what is not yet implemented.

#### Acceptance Criteria

1. WHEN a statement that is not supported is provided, THE Parser SHALL return an error
2. WHEN an unsupported statement error is returned, THE error message SHALL indicate which statement was unsupported

### Requirement 19: Integrate WHERE Expressions with DML Statements

**User Story:** As a planner, I want WHERE clauses to be available as structured expressions, so that I can build efficient query plans.

#### Acceptance Criteria

1. WHEN a SELECT statement with WHERE clause is parsed, THE Parser SHALL populate both AST.Where (string) and AST.WhereExpr (structured)
2. WHEN an UPDATE statement with WHERE clause is parsed, THE Parser SHALL populate both UpdateSpec.Where (string) and AST.WhereExpr (structured)
3. WHEN a DELETE statement with WHERE clause is parsed, THE Parser SHALL populate both DeleteSpec.Where (string) and AST.WhereExpr (structured)
4. WHEN a WHERE clause cannot be parsed into a structured expression, THE Parser SHALL return an error with context about the invalid WHERE clause

### Requirement 20: Maintain Backward Compatibility

**User Story:** As a developer, I want the parser to maintain backward compatibility, so that existing code continues to work during migration.

#### Acceptance Criteria

1. WHEN a DML statement is parsed, THE Parser SHALL populate both string-based WHERE fields (deprecated) and structured WhereExpr fields
2. WHEN code accesses the string-based WHERE fields, THE Parser SHALL provide the raw WHERE clause string
3. WHEN code accesses the structured WhereExpr field, THE Parser SHALL provide the parsed expression tree

