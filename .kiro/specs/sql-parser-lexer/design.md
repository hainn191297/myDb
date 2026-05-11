# SQL Parser & Lexer Design

## Overview

The SQL Parser & Lexer is a two-tier system that converts raw SQL strings into Abstract Syntax Trees (ASTs) for downstream processing. The design prioritizes clarity and maintainability over performance, making it suitable for a learning-oriented database system.

**Architecture:**
- **Statement Parser** (`parser.go`): Handles top-level SQL statement parsing (DDL, DML, TCL)
- **Expression Parser** (`expr/parser.go`): Handles WHERE clause and expression parsing
- **Lexer/Tokenizer** (`expr/tokenizer.go`): Converts expression strings into tokens
- **AST Types** (`expr/expr.go`): Defines expression node types

## Design Principles

1. **Separation of Concerns**: Statement parsing is separate from expression parsing
2. **Recursive Descent**: Both parsers use recursive descent for clarity
3. **Operator Precedence**: Expression parser respects SQL operator precedence
4. **Error Reporting**: Clear error messages with context
5. **Backward Compatibility**: Dual support for string-based and structured WHERE clauses

## Layer Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Application Layer                             │
│              (Server, Planner, Executor)                         │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Parser Interface Layer                          │
│                                                                   │
│  Parse(ctx, sql) → AST                                           │
│  ├─ Validates input (trim, normalize)                            │
│  ├─ Routes to appropriate statement parser                       │
│  └─ Returns structured AST or error                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Statement    │  │ Statement    │  │ Statement    │
│ Parsers      │  │ Parsers      │  │ Parsers      │
│              │  │              │  │              │
│ DDL:         │  │ DML:         │  │ TCL:         │
│ - CREATE TBL │  │ - SELECT     │  │ - BEGIN      │
│ - DROP TBL   │  │ - INSERT     │  │ - COMMIT     │
│ - CREATE IDX │  │ - UPDATE     │  │ - ROLLBACK   │
│ - DROP IDX   │  │ - DELETE     │  │              │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
       └─────────────────┼─────────────────┘
                         │
                         ▼
        ┌────────────────────────────────┐
        │  Expression Parser Layer        │
        │                                 │
        │  ParseExpr(whereClause) → Expr  │
        │  ├─ Tokenizes input             │
        │  ├─ Applies operator precedence │
        │  └─ Returns Expr tree or error  │
        └────────────────┬────────────────┘
                         │
                         ▼
        ┌────────────────────────────────┐
        │  Lexer/Tokenizer Layer          │
        │                                 │
        │  tokenize(input) → []token      │
        │  ├─ Single-pass tokenization    │
        │  ├─ Recognizes all token types  │
        │  └─ Returns token stream + EOF  │
        └────────────────┬────────────────┘
                         │
                         ▼
        ┌────────────────────────────────┐
        │  AST Type Definitions           │
        │                                 │
        │  Expr (interface)               │
        │  ├─ LiteralExpr                 │
        │  ├─ ColumnRefExpr               │
        │  ├─ BinaryExpr                  │
        │  └─ UnaryExpr                   │
        │                                 │
        │  AST (root)                     │
        │  ├─ CreateTableSpec             │
        │  ├─ InsertSpec                  │
        │  ├─ UpdateSpec                  │
        │  ├─ DeleteSpec                  │
        │  └─ ... (other specs)           │
        └────────────────────────────────┘
```

### Layer Responsibilities

**Layer 1: Application Layer**
- Consumers: Server, Planner, Executor
- Uses: `Parse()` function
- Receives: SQL string
- Returns: AST or error

**Layer 2: Parser Interface Layer**
- Entry point: `Parse(ctx, sql)`
- Responsibilities:
  - Input validation (trim, normalize)
  - Statement type detection
  - Routing to appropriate parser
  - Error handling and reporting
- Returns: Fully populated AST

**Layer 3: Statement Parsers**
- DDL Parsers: CREATE/DROP TABLE, CREATE/DROP INDEX
- DML Parsers: SELECT, INSERT, UPDATE, DELETE
- TCL Parsers: BEGIN, COMMIT, ROLLBACK
- Each parser:
  - Extracts specific components
  - Validates required fields
  - Delegates WHERE clause parsing to Expression Parser
  - Returns statement-specific Spec

**Layer 4: Expression Parser Layer**
- Entry point: `ParseExpr(whereClause)`
- Responsibilities:
  - Tokenization delegation
  - Recursive descent parsing
  - Operator precedence handling
  - Expression tree construction
- Returns: Expr tree or error

**Layer 5: Lexer/Tokenizer Layer**
- Entry point: `tokenize(input)`
- Responsibilities:
  - Single-pass character scanning
  - Token recognition and classification
  - Whitespace handling
  - Escape sequence handling
- Returns: Token stream with EOF

**Layer 6: Type Definitions**
- AST types: Root structure for all statements
- Spec types: Statement-specific details
- Expr types: Expression tree nodes
- Operator enums: BinaryOp, UnaryOp
- Value types: TypeInt, TypeText, etc.

### Data Flow Through Layers

```
SQL String Input
    ↓
[Layer 2] Parse() - Normalize & Route
    ↓
[Layer 3] Statement Parser - Extract Components
    ├─ For DML with WHERE:
    │   ↓
    │ [Layer 4] ParseExpr() - Parse WHERE clause
    │   ↓
    │ [Layer 5] tokenize() - Tokenize expression
    │   ↓
    │ [Layer 4] Recursive Descent - Build Expr tree
    │   ↓
    │ Return Expr to Statement Parser
    └─ Populate AST with Expr
    ↓
[Layer 2] Return AST to Application
    ↓
Application Layer (Server, Planner, Executor)
```

### Dependency Graph

```
Layer 6 (Types)
    ↑
    │ (depends on)
    │
Layer 5 (Lexer)
    ↑
    │
Layer 4 (Expression Parser)
    ↑
    │
Layer 3 (Statement Parsers)
    ↑
    │
Layer 2 (Parser Interface)
    ↑
    │
Layer 1 (Application)
```

### Error Handling Flow

```
Error at any layer
    ↓
Wrap with context (if not already wrapped)
    ↓
Return error up the stack
    ↓
Layer 2 (Parse) catches and returns to Application
    ↓
Application handles error (logs, returns to client)
```

## Component Architecture

### 1. Statement Parser (`internal/sql/parser/parser.go`)

**Responsibility**: Parse complete SQL statements into AST structures

**Key Types**:
```
AST
├── Type: StatementType (SELECT, INSERT, UPDATE, DELETE, CREATE_TABLE, DROP_TABLE, CREATE_INDEX, DROP_INDEX, BEGIN, COMMIT, ROLLBACK)
├── SchemaName: string (defaults to "public")
├── TableName: string
├── Columns: []string
├── Where: string (deprecated, for backward compat)
├── WhereExpr: expr.Expr (structured WHERE clause)
├── CreateTable: *CreateTableSpec
├── DropTable: *DropTableSpec
├── CreateIndex: *CreateIndexSpec
├── DropIndex: *DropIndexSpec
├── Insert: *InsertSpec
├── Update: *UpdateSpec
└── Delete: *DeleteSpec
```

**Spec Types**:
- `CreateTableSpec`: Schema, Table, Columns (with type, nullable, primaryKey)
- `DropTableSpec`: Schema, Table
- `CreateIndexSpec`: Schema, Table, IndexName, Columns, Unique flag
- `DropIndexSpec`: Schema, Table, IndexName
- `InsertSpec`: Schema, Table, Columns (optional), Values (list of rows)
- `UpdateSpec`: Schema, Table, SetClauses (map), Where (string)
- `DeleteSpec`: Schema, Table, Where (string)

**Parsing Flow**:
1. Trim and normalize input (remove leading/trailing whitespace and semicolons)
2. Identify statement type by prefix matching (case-insensitive)
3. Route to appropriate parser function
4. Extract components using string manipulation and regex-like patterns
5. Validate required fields
6. For DML statements with WHERE clauses, delegate to expression parser

**Key Functions**:
- `Parse(ctx, sql)`: Main entry point, routes to specific parsers
- `parseSelect(sql)`: Handles SELECT statements
- `parseInsert(ctx, sql)`: Handles INSERT statements
- `parseUpdate(sql)`: Handles UPDATE statements
- `parseDelete(sql)`: Handles DELETE statements
- `parseCreateTable(sql)`: Handles CREATE TABLE statements
- `parseDropTable(sql)`: Handles DROP TABLE statements
- `parseCreateIndex(sql)`: Handles CREATE INDEX statements
- `parseDropIndex(sql)`: Handles DROP INDEX statements
- `parseColumnDefs(columnsPart)`: Parses column definitions
- `parseColumnDef(def)`: Parses single column definition
- `parseSetClauses(setClauseStr)`: Parses UPDATE SET clauses
- `parseValues(valuesStr)`: Parses INSERT VALUES
- `splitSchemaTable(name)`: Splits "schema.table" notation
- `splitColumns(part)`: Splits comma-separated column list

**Error Handling**:
- Returns `ErrEmptyStatement` for empty input
- Returns descriptive errors for missing required clauses
- Returns errors for malformed syntax (unbalanced parentheses, etc.)
- Propagates expression parsing errors with context

### 2. Expression Parser (`internal/sql/expr/parser.go`)

**Responsibility**: Parse WHERE clauses and expressions into structured ASTs

**Key Types**:
```
Expr (interface)
├── LiteralExpr: Value (any), Type (ValueType)
├── ColumnRefExpr: Name (string)
├── BinaryExpr: Left (Expr), Op (BinaryOp), Right (Expr)
└── UnaryExpr: Op (UnaryOp), Expr (Expr)

BinaryOp: OpEquals, OpNotEquals, OpLessThan, OpLessOrEqual, OpGreaterThan, OpGreaterOrEqual, OpAnd, OpOr

UnaryOp: OpNot, OpIsNull, OpIsNotNull

ValueType: TypeInt, TypeText, TypeBool, TypeFloat, TypeNull
```

**Parsing Strategy**: Recursive descent with operator precedence

**Precedence (low to high)**:
1. OR (lowest)
2. AND
3. NOT
4. Comparison operators (=, !=, <, >, <=, >=)
5. IS NULL / IS NOT NULL
6. Primary (literals, column refs, parentheses) (highest)

**Parsing Functions**:
- `ParseExpr(whereClause)`: Main entry point, returns nil for empty input
- `parseOr()`: Handles OR expressions (lowest precedence)
- `parseAnd()`: Handles AND expressions
- `parseNot()`: Handles NOT expressions (recursive for multiple NOTs)
- `parseComparison()`: Handles comparison and IS NULL operators
- `parsePrimary()`: Handles literals, column refs, and parenthesized expressions
- `parseOperator(op)`: Converts operator string to BinaryOp enum

**Key Design Decisions**:
- Left-associative for AND/OR (a AND b AND c = (a AND b) AND c)
- Recursive NOT handling (NOT NOT x is valid)
- Parentheses override precedence
- Empty WHERE clause returns nil (not an error)
- Validates all tokens are consumed after parsing

### 3. Lexer/Tokenizer (`internal/sql/expr/tokenizer.go`)

**Responsibility**: Convert expression strings into tokens

**Token Types**:
```
tokenIdent: Column names, identifiers (id, email, status)
tokenNumber: Integers and floats (123, 45.67)
tokenString: Quoted strings ('hello', "world")
tokenOperator: Comparison operators (=, !=, <, >, <=, >=)
tokenKeyword: SQL keywords (AND, OR, NOT, IS, NULL)
tokenLParen: Left parenthesis (
tokenRParen: Right parenthesis )
tokenEOF: End of input
```

**Tokenization Algorithm**:
1. Iterate through input character by character
2. Skip whitespace
3. Recognize numbers (integers and floats with decimal point)
4. Recognize quoted strings (single or double quotes, with escape handling)
5. Recognize parentheses
6. Recognize operators (single and two-character: !=, <=, >=)
7. Recognize keywords and identifiers
8. Append EOF token at end

**Key Functions**:
- `tokenize(input)`: Main tokenization function
- `isWhitespace(ch)`: Check if character is whitespace
- `isDigit(ch)`: Check if character is digit
- `isAlpha(ch)`: Check if character is alphabetic
- `isAlphaNum(ch)`: Check if character is alphanumeric
- `isOperatorChar(ch)`: Check if character is operator
- `isKeyword(word)`: Check if word is SQL keyword
- `ParseInt(s)`: Parse string as int64
- `ParseFloat(s)`: Parse string as float64
- `StripQuotes(s)`: Remove surrounding quotes from string

**Key Design Decisions**:
- Single-pass tokenization (no lookahead needed)
- Handles escaped characters in strings
- Keywords are case-insensitive (converted to uppercase)
- Identifiers preserve case
- Numbers can be integers or floats

## Data Flow

### SELECT Statement Parsing
```
"SELECT id, name FROM users WHERE id = 1"
    ↓
Parse(ctx, sql)
    ↓
parseSelect(sql)
    ↓
Extract columns: [id, name]
Extract table: users (schema: public)
Extract WHERE: "id = 1"
    ↓
ParseExpr("id = 1")
    ↓
tokenize("id = 1")
    ↓
[tokenIdent(id), tokenOperator(=), tokenNumber(1), tokenEOF]
    ↓
parseOr() → parseAnd() → parseNot() → parseComparison()
    ↓
BinaryExpr{
  Left: ColumnRefExpr{Name: "id"},
  Op: OpEquals,
  Right: LiteralExpr{Value: 1, Type: TypeInt}
}
    ↓
AST{
  Type: SelectStmt,
  SchemaName: "public",
  TableName: "users",
  Columns: [id, name],
  Where: "id = 1",
  WhereExpr: BinaryExpr{...}
}
```

### INSERT Statement Parsing
```
"INSERT INTO users (id, name) VALUES (1, 'alice'), (2, 'bob')"
    ↓
Parse(ctx, sql)
    ↓
parseInsert(ctx, sql)
    ↓
Extract table: users (schema: public)
Extract columns: [id, name]
Parse VALUES: [[1, 'alice'], [2, 'bob']]
    ↓
AST{
  Type: InsertStmt,
  Insert: InsertSpec{
    Schema: "public",
    Table: "users",
    Columns: [id, name],
    Values: [[1, 'alice'], [2, 'bob']]
  }
}
```

### CREATE TABLE Statement Parsing
```
"CREATE TABLE users (id INT PRIMARY KEY, name TEXT NOT NULL)"
    ↓
Parse(ctx, sql)
    ↓
parseCreateTable(sql)
    ↓
Extract table: users (schema: public)
Parse columns:
  - id: INT, NOT NULL, PRIMARY KEY
  - name: TEXT, NOT NULL
    ↓
AST{
  Type: CreateTableStmt,
  CreateTable: CreateTableSpec{
    Schema: "public",
    Table: "users",
    Columns: [
      {Name: "id", Type: "INT", Nullable: false, PrimaryKey: true},
      {Name: "name", Type: "TEXT", Nullable: false, PrimaryKey: false}
    ]
  }
}
```

## Error Handling Strategy

**Parser Errors**:
- Empty statement → `ErrEmptyStatement`
- Missing required clauses → Descriptive error with clause name
- Malformed syntax → Error with context (e.g., "unbalanced parentheses")
- Unsupported statements → Error indicating statement type

**Expression Parser Errors**:
- Invalid operators → Error with operator name
- Unexpected tokens → Error with token value
- Incomplete expressions → Error with context (e.g., "missing right operand")
- Unclosed parentheses → Error with position

**Error Propagation**:
- Expression parsing errors are wrapped with context in statement parser
- All errors include the problematic SQL/expression for debugging

## Backward Compatibility

The parser maintains dual support for WHERE clauses:
- **String-based** (`AST.Where`, `UpdateSpec.Where`, `DeleteSpec.Where`): Raw WHERE clause string
- **Structured** (`AST.WhereExpr`): Parsed expression tree

This allows:
- Old code to continue using string-based WHERE clauses
- New code to use structured expressions for optimization
- Gradual migration path

## Testing Strategy

**Unit Tests** (`*_test.go` files):
- `TestParseTxnStatements`: Transaction control statements
- `TestParseSelect`: SELECT with various clauses
- `TestParseErrors`: Error cases
- `TestParseSelectDefaultSchema`: Schema defaulting
- `TestParseCreateTable`: CREATE TABLE variations
- `TestParseInsert`: INSERT with/without columns
- `TestParseInsertWithSchema`: Schema.table notation
- `TestParseUpdate`: UPDATE with/without WHERE
- `TestParseUpdateNoWhere`: UPDATE without WHERE clause
- `TestParseCreateIndex`: CREATE INDEX variations
- `TestParseDropIndex`: DROP INDEX
- `TestParsePrimaryKey`: PRIMARY KEY constraints
- `TestParseSimpleEquality`: Simple expressions
- `TestParseComplexAnd`: Complex AND expressions
- `TestParseOrExpression`: OR expressions
- `TestParseNot`: NOT expressions
- `TestParseIsNull`: IS NULL expressions

**Test Coverage**:
- All statement types
- Schema.table notation
- Default schema handling
- Column constraints (NOT NULL, PRIMARY KEY)
- WHERE clause parsing
- Expression operator precedence
- Error cases

## Performance Considerations

**Current Approach**:
- Single-pass tokenization: O(n) where n = input length
- Recursive descent parsing: O(n) for well-formed input
- No lookahead or backtracking needed
- Suitable for typical SQL statement sizes

**Optimization Opportunities** (future):
- Caching parsed statements
- Pre-compiled statement templates
- Parallel parsing for batch operations
- Streaming result processing

## Extension Points

**Adding New Statement Types**:
1. Add new `StatementType` constant
2. Add new `*Spec` type if needed
3. Add case in `Parse()` switch
4. Implement `parse<StatementType>()` function
5. Add tests

**Adding New Operators**:
1. Add new `BinaryOp` or `UnaryOp` constant
2. Update `isKeyword()` if keyword-based
3. Update `parseOperator()` or tokenizer
4. Update precedence in expression parser if needed
5. Add tests

**Adding New Data Types**:
1. Add new `ValueType` constant
2. Update `parsePrimary()` to recognize new type
3. Update tokenizer if needed
4. Add tests

## Integration Points

**Upstream** (consumers of parser):
- `internal/sql/planner`: Uses AST to build execution plans
- `internal/sql/executor`: Uses AST and WhereExpr for execution

**Downstream** (dependencies):
- `internal/errors`: Error definitions
- `internal/logging`: Debug logging

## Future Enhancements

1. **JOIN Support**: Parse JOIN clauses for multi-table queries
2. **Aggregate Functions**: Parse COUNT, SUM, AVG, etc.
3. **GROUP BY / HAVING**: Parse grouping and filtering
4. **ORDER BY**: Parse sorting specifications
5. **LIMIT / OFFSET**: Parse pagination
6. **Subqueries**: Parse nested SELECT statements
7. **UNION**: Parse UNION operations
8. **Window Functions**: Parse OVER clauses
9. **Type Casting**: Parse CAST expressions
10. **Function Calls**: Parse built-in functions

## Summary

The SQL Parser & Lexer provides a clean, maintainable foundation for SQL parsing in myDb. Its two-tier design (statement parsing + expression parsing) allows for clear separation of concerns while maintaining good error reporting and extensibility. The recursive descent approach with explicit operator precedence makes the code easy to understand and modify, which aligns with the learning-oriented goals of the project.
