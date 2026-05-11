# SQL Parser & Lexer - Comprehensive Test Cases

## Overview

This document provides comprehensive test cases for all components of the SQL Parser & Lexer, organized by layer and functionality. Each test case includes:
- **Test Name**: Descriptive identifier
- **Input**: What is being tested
- **Expected Output**: What should happen
- **Acceptance Criteria**: How to verify success
- **Category**: Which layer/component

---

## Layer 1: Type Definitions Tests

### Test 1.1: Expression Interface Implementation
**Category**: Type Definitions
**Input**: All Expr types (LiteralExpr, ColumnRefExpr, BinaryExpr, UnaryExpr)
**Expected Output**: All types implement Expr interface
**Acceptance Criteria**:
- [ ] LiteralExpr has exprNode() method
- [ ] ColumnRefExpr has exprNode() method
- [ ] BinaryExpr has exprNode() method
- [ ] UnaryExpr has exprNode() method
- [ ] All types compile without errors

### Test 1.2: BinaryOp String Representation
**Category**: Type Definitions
**Input**: All BinaryOp values
**Expected Output**: Correct string representation
**Test Cases**:
- [ ] OpEquals → "="
- [ ] OpNotEquals → "!="
- [ ] OpLessThan → "<"
- [ ] OpLessOrEqual → "<="
- [ ] OpGreaterThan → ">"
- [ ] OpGreaterOrEqual → ">="
- [ ] OpAnd → "AND"
- [ ] OpOr → "OR"

### Test 1.3: UnaryOp String Representation
**Category**: Type Definitions
**Input**: All UnaryOp values
**Expected Output**: Correct string representation
**Test Cases**:
- [ ] OpNot → "NOT"
- [ ] OpIsNull → "IS NULL"
- [ ] OpIsNotNull → "IS NOT NULL"

### Test 1.4: ValueType Enum
**Category**: Type Definitions
**Input**: All ValueType values
**Expected Output**: Correct enum values
**Test Cases**:
- [ ] TypeInt defined
- [ ] TypeText defined
- [ ] TypeBool defined
- [ ] TypeFloat defined
- [ ] TypeNull defined

---

## Layer 2: Lexer/Tokenizer Tests

### Test 2.1: Tokenize Numbers
**Category**: Lexer
**Input**: Various number formats
**Expected Output**: tokenNumber tokens
**Test Cases**:
- [ ] "123" → tokenNumber("123")
- [ ] "45.67" → tokenNumber("45.67")
- [ ] "0" → tokenNumber("0")
- [ ] "999999" → tokenNumber("999999")
- [ ] "3.14159" → tokenNumber("3.14159")

### Test 2.2: Tokenize Strings
**Category**: Lexer
**Input**: Quoted strings
**Expected Output**: tokenString tokens with quotes preserved
**Test Cases**:
- [ ] "'hello'" → tokenString("'hello'")
- [ ] '"world"' → tokenString('"world"')
- [ ] "'it\\'s'" → tokenString("'it\\'s'") (escaped quote)
- [ ] "''" → tokenString("''") (empty string)
- [ ] "'with spaces'" → tokenString("'with spaces'")

### Test 2.3: Tokenize Operators
**Category**: Lexer
**Input**: Comparison operators
**Expected Output**: tokenOperator tokens
**Test Cases**:
- [ ] "=" → tokenOperator("=")
- [ ] "!=" → tokenOperator("!=")
- [ ] "<" → tokenOperator("<")
- [ ] "<=" → tokenOperator("<=")
- [ ] ">" → tokenOperator(">")
- [ ] ">=" → tokenOperator(">=")

### Test 2.4: Tokenize Keywords
**Category**: Lexer
**Input**: SQL keywords
**Expected Output**: tokenKeyword tokens (uppercase)
**Test Cases**:
- [ ] "AND" → tokenKeyword("AND")
- [ ] "and" → tokenKeyword("AND") (case-insensitive)
- [ ] "OR" → tokenKeyword("OR")
- [ ] "NOT" → tokenKeyword("NOT")
- [ ] "IS" → tokenKeyword("IS")
- [ ] "NULL" → tokenKeyword("NULL")

### Test 2.5: Tokenize Identifiers
**Category**: Lexer
**Input**: Column names and identifiers
**Expected Output**: tokenIdent tokens (case-sensitive)
**Test Cases**:
- [ ] "id" → tokenIdent("id")
- [ ] "email" → tokenIdent("email")
- [ ] "user_id" → tokenIdent("user_id")
- [ ] "Name" → tokenIdent("Name") (preserves case)
- [ ] "STATUS" → tokenIdent("STATUS")

### Test 2.6: Tokenize Parentheses
**Category**: Lexer
**Input**: Parentheses
**Expected Output**: tokenLParen and tokenRParen
**Test Cases**:
- [ ] "(" → tokenLParen("(")
- [ ] ")" → tokenRParen(")")
- [ ] "()" → [tokenLParen, tokenRParen]

### Test 2.7: Tokenize Whitespace Handling
**Category**: Lexer
**Input**: Expressions with various whitespace
**Expected Output**: Whitespace skipped, tokens preserved
**Test Cases**:
- [ ] "  id  =  1  " → [tokenIdent, tokenOperator, tokenNumber, tokenEOF]
- [ ] "id\t=\t1" → [tokenIdent, tokenOperator, tokenNumber, tokenEOF]
- [ ] "id\n=\n1" → [tokenIdent, tokenOperator, tokenNumber, tokenEOF]

### Test 2.8: Tokenize Complex Expression
**Category**: Lexer
**Input**: Full WHERE clause
**Expected Output**: Complete token stream
**Test Cases**:
- [ ] "id = 1 AND status = 'active'" → [tokenIdent, tokenOperator, tokenNumber, tokenKeyword, tokenIdent, tokenOperator, tokenString, tokenEOF]
- [ ] "NOT (x > 5 OR y < 10)" → [tokenKeyword, tokenLParen, tokenIdent, tokenOperator, tokenNumber, tokenKeyword, tokenIdent, tokenOperator, tokenNumber, tokenRParen, tokenEOF]

### Test 2.9: Tokenize EOF
**Category**: Lexer
**Input**: Any input
**Expected Output**: EOF token at end
**Test Cases**:
- [ ] "" → [tokenEOF]
- [ ] "id" → [tokenIdent, tokenEOF]
- [ ] "id = 1" → [tokenIdent, tokenOperator, tokenNumber, tokenEOF]

---

## Layer 3: Expression Parser Tests

### Test 3.1: Parse Simple Equality
**Category**: Expression Parser
**Input**: "id = 1"
**Expected Output**: BinaryExpr{Left: ColumnRefExpr{Name: "id"}, Op: OpEquals, Right: LiteralExpr{Value: 1, Type: TypeInt}}
**Acceptance Criteria**:
- [ ] Left operand is ColumnRefExpr with name "id"
- [ ] Operator is OpEquals
- [ ] Right operand is LiteralExpr with value 1 and type TypeInt

### Test 3.2: Parse Complex AND Expression
**Category**: Expression Parser
**Input**: "id = 1 AND status = 'active'"
**Expected Output**: BinaryExpr with OpAnd, left and right are BinaryExpr
**Acceptance Criteria**:
- [ ] Root operator is OpAnd
- [ ] Left side: id = 1
- [ ] Right side: status = 'active'
- [ ] String value has quotes removed

### Test 3.3: Parse OR Expression
**Category**: Expression Parser
**Input**: "x > 5 OR y < 10"
**Expected Output**: BinaryExpr with OpOr
**Acceptance Criteria**:
- [ ] Root operator is OpOr
- [ ] Left side: x > 5
- [ ] Right side: y < 10

### Test 3.4: Parse NOT Expression
**Category**: Expression Parser
**Input**: "NOT active"
**Expected Output**: UnaryExpr{Op: OpNot, Expr: ColumnRefExpr{Name: "active"}}
**Acceptance Criteria**:
- [ ] Operator is OpNot
- [ ] Operand is ColumnRefExpr with name "active"

### Test 3.5: Parse Multiple NOT
**Category**: Expression Parser
**Input**: "NOT NOT x"
**Expected Output**: UnaryExpr{Op: OpNot, Expr: UnaryExpr{Op: OpNot, Expr: ColumnRefExpr}}
**Acceptance Criteria**:
- [ ] Outer UnaryExpr has OpNot
- [ ] Inner UnaryExpr has OpNot
- [ ] Innermost is ColumnRefExpr

### Test 3.6: Parse IS NULL
**Category**: Expression Parser
**Input**: "email IS NULL"
**Expected Output**: UnaryExpr{Op: OpIsNull, Expr: ColumnRefExpr{Name: "email"}}
**Acceptance Criteria**:
- [ ] Operator is OpIsNull
- [ ] Operand is ColumnRefExpr

### Test 3.7: Parse IS NOT NULL
**Category**: Expression Parser
**Input**: "email IS NOT NULL"
**Expected Output**: UnaryExpr{Op: OpIsNotNull, Expr: ColumnRefExpr{Name: "email"}}
**Acceptance Criteria**:
- [ ] Operator is OpIsNotNull
- [ ] Operand is ColumnRefExpr

### Test 3.8: Parse Parenthesized Expression
**Category**: Expression Parser
**Input**: "(id = 1)"
**Expected Output**: BinaryExpr{Left: ColumnRefExpr, Op: OpEquals, Right: LiteralExpr}
**Acceptance Criteria**:
- [ ] Parentheses are consumed
- [ ] Inner expression is parsed correctly

### Test 3.9: Parse Operator Precedence
**Category**: Expression Parser
**Input**: "a = 1 OR b = 2 AND c = 3"
**Expected Output**: BinaryExpr{Left: (a=1), Op: OpOr, Right: BinaryExpr{Left: (b=2), Op: OpAnd, Right: (c=3)}}
**Acceptance Criteria**:
- [ ] AND has higher precedence than OR
- [ ] Parsed as: a = 1 OR (b = 2 AND c = 3)

### Test 3.10: Parse Integer Literal
**Category**: Expression Parser
**Input**: "123"
**Expected Output**: LiteralExpr{Value: int64(123), Type: TypeInt}
**Acceptance Criteria**:
- [ ] Value is int64
- [ ] Type is TypeInt

### Test 3.11: Parse Float Literal
**Category**: Expression Parser
**Input**: "45.67"
**Expected Output**: LiteralExpr{Value: float64(45.67), Type: TypeFloat}
**Acceptance Criteria**:
- [ ] Value is float64
- [ ] Type is TypeFloat

### Test 3.12: Parse String Literal
**Category**: Expression Parser
**Input**: "'hello'"
**Expected Output**: LiteralExpr{Value: "hello", Type: TypeText}
**Acceptance Criteria**:
- [ ] Quotes are removed
- [ ] Value is string
- [ ] Type is TypeText

### Test 3.13: Parse Boolean Literal
**Category**: Expression Parser
**Input**: "true" and "false"
**Expected Output**: LiteralExpr{Value: bool, Type: TypeBool}
**Acceptance Criteria**:
- [ ] "true" → LiteralExpr{Value: true, Type: TypeBool}
- [ ] "false" → LiteralExpr{Value: false, Type: TypeBool}
- [ ] Case-insensitive

### Test 3.14: Parse NULL Literal
**Category**: Expression Parser
**Input**: "NULL"
**Expected Output**: LiteralExpr{Value: nil, Type: TypeNull}
**Acceptance Criteria**:
- [ ] Value is nil
- [ ] Type is TypeNull

### Test 3.15: Parse Column Reference
**Category**: Expression Parser
**Input**: "email"
**Expected Output**: ColumnRefExpr{Name: "email"}
**Acceptance Criteria**:
- [ ] Name is "email"
- [ ] Type is ColumnRefExpr

### Test 3.16: Parse Empty Expression
**Category**: Expression Parser
**Input**: ""
**Expected Output**: nil (not an error)
**Acceptance Criteria**:
- [ ] Returns nil
- [ ] No error

### Test 3.17: Expression Error - Missing Right Operand
**Category**: Expression Parser
**Input**: "id ="
**Expected Output**: Error
**Acceptance Criteria**:
- [ ] Error message indicates missing right operand
- [ ] Error is not nil

### Test 3.18: Expression Error - Missing Left Operand
**Category**: Expression Parser
**Input**: "= 1"
**Expected Output**: Error
**Acceptance Criteria**:
- [ ] Error message indicates unexpected token
- [ ] Error is not nil

### Test 3.19: Expression Error - Unclosed Parenthesis
**Category**: Expression Parser
**Input**: "(id = 1"
**Expected Output**: Error
**Acceptance Criteria**:
- [ ] Error message indicates unclosed parenthesis
- [ ] Error is not nil

### Test 3.20: Expression Error - Unexpected Token After Expression
**Category**: Expression Parser
**Input**: "id = 1 EXTRA"
**Expected Output**: Error
**Acceptance Criteria**:
- [ ] Error message indicates unexpected token
- [ ] Error is not nil

---

## Layer 4: Statement Parser Tests

### Test 4.1: Parse Transaction Control - BEGIN
**Category**: Statement Parser
**Input**: "BEGIN"
**Expected Output**: AST{Type: BeginStmt}
**Acceptance Criteria**:
- [ ] Type is BeginStmt
- [ ] No other fields populated

### Test 4.2: Parse Transaction Control - COMMIT
**Category**: Statement Parser
**Input**: "COMMIT"
**Expected Output**: AST{Type: CommitStmt}
**Acceptance Criteria**:
- [ ] Type is CommitStmt

### Test 4.3: Parse Transaction Control - ROLLBACK
**Category**: Statement Parser
**Input**: "ROLLBACK"
**Expected Output**: AST{Type: RollbackStmt}
**Acceptance Criteria**:
- [ ] Type is RollbackStmt

### Test 4.4: Parse SELECT - Simple
**Category**: Statement Parser
**Input**: "SELECT id, name FROM users"
**Expected Output**: AST{Type: SelectStmt, Columns: [id, name], TableName: users, SchemaName: public}
**Acceptance Criteria**:
- [ ] Type is SelectStmt
- [ ] Columns: [id, name]
- [ ] TableName: users
- [ ] SchemaName: public
- [ ] Where: empty
- [ ] WhereExpr: nil

### Test 4.5: Parse SELECT - With WHERE
**Category**: Statement Parser
**Input**: "SELECT id, name FROM users WHERE id = 1"
**Expected Output**: AST with Where and WhereExpr populated
**Acceptance Criteria**:
- [ ] Where: "id = 1"
- [ ] WhereExpr: BinaryExpr{...}
- [ ] Both fields populated

### Test 4.6: Parse SELECT - With Schema
**Category**: Statement Parser
**Input**: "SELECT id FROM public.users"
**Expected Output**: AST{SchemaName: public, TableName: users}
**Acceptance Criteria**:
- [ ] SchemaName: public
- [ ] TableName: users

### Test 4.7: Parse SELECT - Error Missing FROM
**Category**: Statement Parser
**Input**: "SELECT id, name"
**Expected Output**: Error
**Acceptance Criteria**:
- [ ] Error message contains "FROM"
- [ ] Error is not nil

### Test 4.8: Parse INSERT - Simple
**Category**: Statement Parser
**Input**: "INSERT INTO users VALUES (1, 'alice')"
**Expected Output**: AST{Type: InsertStmt, Insert: InsertSpec{Table: users, Values: [[1, 'alice']]}}
**Acceptance Criteria**:
- [ ] Type is InsertStmt
- [ ] Table: users
- [ ] Values: [[1, 'alice']]
- [ ] Columns: empty

### Test 4.9: Parse INSERT - With Columns
**Category**: Statement Parser
**Input**: "INSERT INTO users (id, name) VALUES (1, 'alice')"
**Expected Output**: AST with Columns populated
**Acceptance Criteria**:
- [ ] Columns: [id, name]
- [ ] Values: [[1, 'alice']]

### Test 4.10: Parse INSERT - Multiple Rows
**Category**: Statement Parser
**Input**: "INSERT INTO users VALUES (1, 'alice'), (2, 'bob')"
**Expected Output**: AST{Insert: InsertSpec{Values: [[1, 'alice'], [2, 'bob']]}}
**Acceptance Criteria**:
- [ ] Values has 2 rows
- [ ] First row: [1, 'alice']
- [ ] Second row: [2, 'bob']

### Test 4.11: Parse UPDATE - Simple
**Category**: Statement Parser
**Input**: "UPDATE users SET name = 'bob'"
**Expected Output**: AST{Type: UpdateStmt, Update: UpdateSpec{Table: users, SetClauses: {name: 'bob'}}}
**Acceptance Criteria**:
- [ ] Type is UpdateStmt
- [ ] Table: users
- [ ] SetClauses: {name: 'bob'}

### Test 4.12: Parse UPDATE - With WHERE
**Category**: Statement Parser
**Input**: "UPDATE users SET name = 'bob' WHERE id = 1"
**Expected Output**: AST with Where and WhereExpr
**Acceptance Criteria**:
- [ ] Where: "id = 1"
- [ ] WhereExpr: BinaryExpr{...}

### Test 4.13: Parse UPDATE - Multiple SET
**Category**: Statement Parser
**Input**: "UPDATE users SET name = 'bob', active = true"
**Expected Output**: AST{Update: UpdateSpec{SetClauses: {name: 'bob', active: true}}}
**Acceptance Criteria**:
- [ ] SetClauses has 2 entries
- [ ] name: 'bob'
- [ ] active: true

### Test 4.14: Parse DELETE - Simple
**Category**: Statement Parser
**Input**: "DELETE FROM users"
**Expected Output**: AST{Type: DeleteStmt, Delete: DeleteSpec{Table: users}}
**Acceptance Criteria**:
- [ ] Type is DeleteStmt
- [ ] Table: users
- [ ] Where: empty

### Test 4.15: Parse DELETE - With WHERE
**Category**: Statement Parser
**Input**: "DELETE FROM users WHERE id = 1"
**Expected Output**: AST with Where and WhereExpr
**Acceptance Criteria**:
- [ ] Where: "id = 1"
- [ ] WhereExpr: BinaryExpr{...}

### Test 4.16: Parse CREATE TABLE - Simple
**Category**: Statement Parser
**Input**: "CREATE TABLE users (id INT, name TEXT)"
**Expected Output**: AST{Type: CreateTableStmt, CreateTable: CreateTableSpec{Table: users, Columns: [...]}}
**Acceptance Criteria**:
- [ ] Type is CreateTableStmt
- [ ] Table: users
- [ ] Columns: 2 columns
- [ ] Column 1: {Name: id, Type: INT, Nullable: true, PrimaryKey: false}
- [ ] Column 2: {Name: name, Type: TEXT, Nullable: true, PrimaryKey: false}

### Test 4.17: Parse CREATE TABLE - With Constraints
**Category**: Statement Parser
**Input**: "CREATE TABLE users (id INT PRIMARY KEY, name TEXT NOT NULL)"
**Expected Output**: AST with constraints
**Acceptance Criteria**:
- [ ] Column 1: {PrimaryKey: true, Nullable: false}
- [ ] Column 2: {Nullable: false, PrimaryKey: false}

### Test 4.18: Parse DROP TABLE
**Category**: Statement Parser
**Input**: "DROP TABLE users"
**Expected Output**: AST{Type: DropTableStmt, DropTable: DropTableSpec{Table: users}}
**Acceptance Criteria**:
- [ ] Type is DropTableStmt
- [ ] Table: users

### Test 4.19: Parse CREATE INDEX
**Category**: Statement Parser
**Input**: "CREATE INDEX idx_email ON users (email)"
**Expected Output**: AST{Type: CreateIndexStmt, CreateIndex: CreateIndexSpec{IndexName: idx_email, Table: users, Columns: [email]}}
**Acceptance Criteria**:
- [ ] Type is CreateIndexStmt
- [ ] IndexName: idx_email
- [ ] Table: users
- [ ] Columns: [email]
- [ ] Unique: false

### Test 4.20: Parse CREATE UNIQUE INDEX
**Category**: Statement Parser
**Input**: "CREATE UNIQUE INDEX idx_email ON users (email)"
**Expected Output**: AST with Unique: true
**Acceptance Criteria**:
- [ ] Unique: true

### Test 4.21: Parse DROP INDEX
**Category**: Statement Parser
**Input**: "DROP INDEX idx_email ON users"
**Expected Output**: AST{Type: DropIndexStmt, DropIndex: DropIndexSpec{IndexName: idx_email, Table: users}}
**Acceptance Criteria**:
- [ ] Type is DropIndexStmt
- [ ] IndexName: idx_email
- [ ] Table: users

---

## Layer 5: Integration Tests

### Test 5.1: Input Normalization
**Category**: Integration
**Input**: "  SELECT id FROM users  ;  "
**Expected Output**: Parsed correctly
**Acceptance Criteria**:
- [ ] Leading/trailing whitespace trimmed
- [ ] Semicolon removed
- [ ] Parsed as normal SELECT

### Test 5.2: Case Insensitivity
**Category**: Integration
**Input**: "select id from users where id = 1"
**Expected Output**: Parsed correctly
**Acceptance Criteria**:
- [ ] Keywords recognized (lowercase)
- [ ] AST correct

### Test 5.3: Schema.Table Notation
**Category**: Integration
**Input**: "SELECT id FROM public.users"
**Expected Output**: AST{SchemaName: public, TableName: users}
**Acceptance Criteria**:
- [ ] Schema extracted: public
- [ ] Table extracted: users

### Test 5.4: Default Schema
**Category**: Integration
**Input**: "SELECT id FROM users"
**Expected Output**: AST{SchemaName: public, TableName: users}
**Acceptance Criteria**:
- [ ] SchemaName defaults to "public"

### Test 5.5: Complex WHERE Clause
**Category**: Integration
**Input**: "SELECT * FROM users WHERE (id = 1 OR id = 2) AND status = 'active'"
**Expected Output**: Correct expression tree
**Acceptance Criteria**:
- [ ] Parentheses respected
- [ ] Operator precedence correct
- [ ] All operators parsed

### Test 5.6: Empty Statement
**Category**: Integration
**Input**: ""
**Expected Output**: Error ErrEmptyStatement
**Acceptance Criteria**:
- [ ] Error is ErrEmptyStatement
- [ ] Error message clear

### Test 5.7: Unsupported Statement
**Category**: Integration
**Input**: "ALTER TABLE users ADD COLUMN age INT"
**Expected Output**: Error
**Acceptance Criteria**:
- [ ] Error message indicates unsupported
- [ ] Error is not nil

### Test 5.8: Backward Compatibility - String WHERE
**Category**: Integration
**Input**: "SELECT * FROM users WHERE id = 1"
**Expected Output**: AST with both Where and WhereExpr
**Acceptance Criteria**:
- [ ] Where: "id = 1" (string)
- [ ] WhereExpr: BinaryExpr{...} (structured)

---

## Edge Cases & Stress Tests

### Test 6.1: Large Statement
**Category**: Edge Case
**Input**: SELECT with 100 columns
**Expected Output**: Parsed correctly
**Acceptance Criteria**:
- [ ] All columns extracted
- [ ] No performance degradation

### Test 6.2: Deeply Nested Parentheses
**Category**: Edge Case
**Input**: "((((id = 1))))"
**Expected Output**: Parsed correctly
**Acceptance Criteria**:
- [ ] All parentheses handled
- [ ] Correct expression tree

### Test 6.3: Long String Values
**Category**: Edge Case
**Input**: "INSERT INTO users VALUES (1, 'very long string with many characters...')"
**Expected Output**: Parsed correctly
**Acceptance Criteria**:
- [ ] String value preserved
- [ ] No truncation

### Test 6.4: Special Characters in Strings
**Category**: Edge Case
**Input**: "INSERT INTO users VALUES (1, 'O\\'Brien')"
**Expected Output**: Parsed correctly
**Acceptance Criteria**:
- [ ] Escaped quote handled
- [ ] Value: O'Brien

### Test 6.5: Unicode Characters
**Category**: Edge Case
**Input**: "SELECT * FROM users WHERE name = '日本語'"
**Expected Output**: Parsed correctly
**Acceptance Criteria**:
- [ ] Unicode preserved
- [ ] No encoding issues

### Test 6.6: Multiple Spaces Between Tokens
**Category**: Edge Case
**Input**: "SELECT    id    FROM    users"
**Expected Output**: Parsed correctly
**Acceptance Criteria**:
- [ ] Extra spaces ignored
- [ ] Tokens extracted correctly

### Test 6.7: Tab and Newline Characters
**Category**: Edge Case
**Input**: "SELECT\tid\nFROM\tusers"
**Expected Output**: Parsed correctly
**Acceptance Criteria**:
- [ ] Whitespace variants handled
- [ ] Parsed correctly

---

## Summary

**Total Test Cases**: 100+

**Coverage by Layer**:
- Layer 1 (Types): 4 tests
- Layer 2 (Lexer): 9 tests
- Layer 3 (Expression Parser): 20 tests
- Layer 4 (Statement Parser): 21 tests
- Layer 5 (Integration): 8 tests
- Edge Cases: 7 tests

**Success Criteria**:
- [ ] All 100+ test cases pass
- [ ] Code coverage > 90%
- [ ] No regressions
- [ ] All error cases handled
- [ ] Performance acceptable
