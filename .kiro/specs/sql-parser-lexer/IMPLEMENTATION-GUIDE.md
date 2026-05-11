# SQL Parser & Lexer - Implementation Guide

## Quick Start

This guide helps you implement the SQL Parser & Lexer feature using the comprehensive specification provided.

---

## 📋 Before You Start

### Review These Documents (in order):
1. **SPEC-SUMMARY.md** - 5 min overview
2. **requirements.md** - Understand what needs to be built
3. **design.md** - Understand how it should be built
4. **test-cases.md** - Understand how to verify it works
5. **tasks.md** - Understand the implementation steps

### Key Concepts:
- **Layer Architecture**: 6 layers from Application down to Type Definitions
- **Recursive Descent Parsing**: Both statement and expression parsers use this approach
- **Operator Precedence**: OR < AND < NOT < Comparison < Primary
- **Dual WHERE Support**: Both string-based and structured expressions

---

## 🚀 Implementation Phases

### Phase 1: Foundation (6 hours)

**Goal**: Build the base types and lexer that everything else depends on

#### Task 1.1: Define Expression AST Types
**File**: `internal/sql/expr/expr.go`

```go
// Define these types:
type Expr interface { exprNode() }
type LiteralExpr struct { Value any; Type ValueType }
type ColumnRefExpr struct { Name string }
type BinaryExpr struct { Left Expr; Op BinaryOp; Right Expr }
type UnaryExpr struct { Op UnaryOp; Expr Expr }

// Define these enums:
type BinaryOp int // OpEquals, OpNotEquals, OpLessThan, etc.
type UnaryOp int  // OpNot, OpIsNull, OpIsNotNull
type ValueType int // TypeInt, TypeText, TypeBool, TypeFloat, TypeNull
```

**Tests**: 
- [ ] All types compile
- [ ] All types implement Expr interface
- [ ] String() methods work for operators

**Acceptance Criteria**:
- [ ] All 4 Expr types defined
- [ ] All operators defined
- [ ] All value types defined
- [ ] No compilation errors

---

#### Task 1.2: Implement Lexer Tokenization
**File**: `internal/sql/expr/tokenizer.go`

```go
// Implement:
func tokenize(input string) []token
func isWhitespace(ch rune) bool
func isDigit(ch rune) bool
func isAlpha(ch rune) bool
func isAlphaNum(ch rune) bool
func isOperatorChar(ch byte) bool
func isKeyword(word string) bool
func ParseInt(s string) (int64, bool)
func ParseFloat(s string) (float64, bool)
func StripQuotes(s string) string
```

**Algorithm**:
1. Iterate through input character by character
2. Skip whitespace
3. Recognize: numbers, strings, operators, keywords, identifiers, parentheses
4. Append EOF token

**Tests** (from test-cases.md):
- [ ] Test 2.1-2.9: All tokenization cases
- [ ] 9 test cases total

**Acceptance Criteria**:
- [ ] Tokenizes all token types
- [ ] Handles whitespace correctly
- [ ] Handles escaped characters
- [ ] Appends EOF token

---

#### Task 1.3: Implement Expression Parser
**File**: `internal/sql/expr/parser.go`

```go
// Implement:
func ParseExpr(whereClause string) (Expr, error)
func (p *parser) parseOr() (Expr, error)
func (p *parser) parseAnd() (Expr, error)
func (p *parser) parseNot() (Expr, error)
func (p *parser) parseComparison() (Expr, error)
func (p *parser) parsePrimary() (Expr, error)
func parseOperator(op string) (BinaryOp, error)
```

**Algorithm**: Recursive descent with operator precedence
- parseOr() → parseAnd() → parseNot() → parseComparison() → parsePrimary()

**Tests** (from test-cases.md):
- [ ] Test 3.1-3.20: All expression parsing cases
- [ ] 20 test cases total

**Acceptance Criteria**:
- [ ] Parses all expression types
- [ ] Respects operator precedence
- [ ] Handles parentheses
- [ ] Returns nil for empty input
- [ ] Returns errors for malformed expressions

---

### Phase 2: Statement Parsers (16 hours)

**Goal**: Implement parsers for each SQL statement type

#### Task 2.1: Implement Transaction Control Parser
**File**: `internal/sql/parser/parser.go` - `Parse()` function

```go
// In Parse() function, add:
case strings.HasPrefix(upper, "BEGIN"):
    return AST{Type: BeginStmt}, nil
case strings.HasPrefix(upper, "COMMIT"):
    return AST{Type: CommitStmt}, nil
case strings.HasPrefix(upper, "ROLLBACK"):
    return AST{Type: RollbackStmt}, nil
```

**Tests**:
- [ ] Test 4.1-4.3: Transaction control parsing
- [ ] 3 test cases

---

#### Task 2.2: Implement SELECT Parser
**File**: `internal/sql/parser/parser.go` - `parseSelect()` function

```go
// Extract:
// 1. Column list (after SELECT)
// 2. Table name (after FROM)
// 3. Schema name (from schema.table or default to "public")
// 4. WHERE clause (if present)
// 5. Parse WHERE into structured expression

// Validate:
// - FROM clause exists
// - Column list not empty
// - Table name not empty
```

**Tests**:
- [ ] Test 4.4-4.7: SELECT parsing
- [ ] Test 5.1-5.5: Integration tests
- [ ] 6+ test cases

---

#### Task 2.3: Implement INSERT Parser
**File**: `internal/sql/parser/parser.go` - `parseInsert()` function

```go
// Extract:
// 1. Table name
// 2. Schema name (default to "public")
// 3. Optional column list
// 4. VALUES clause with one or more rows
// 5. Preserve quotes and types in values

// Validate:
// - VALUES clause exists
// - Balanced parentheses
// - Table name not empty
```

**Tests**:
- [ ] Test 4.8-4.10: INSERT parsing
- [ ] 5+ test cases

---

#### Task 2.4: Implement UPDATE Parser
**File**: `internal/sql/parser/parser.go` - `parseUpdate()` function

```go
// Extract:
// 1. Table name
// 2. Schema name (default to "public")
// 3. SET clauses (column = value pairs)
// 4. WHERE clause (if present)
// 5. Parse WHERE into structured expression

// Validate:
// - SET clause exists
// - Table name not empty
```

**Tests**:
- [ ] Test 4.11-4.13: UPDATE parsing
- [ ] 4+ test cases

---

#### Task 2.5: Implement DELETE Parser
**File**: `internal/sql/parser/parser.go` - `parseDelete()` function

```go
// Extract:
// 1. Table name
// 2. Schema name (default to "public")
// 3. WHERE clause (if present)
// 4. Parse WHERE into structured expression

// Validate:
// - FROM clause exists
// - Table name not empty
```

**Tests**:
- [ ] Test 4.14-4.15: DELETE parsing
- [ ] 3+ test cases

---

#### Task 2.6: Implement CREATE TABLE Parser
**File**: `internal/sql/parser/parser.go` - `parseCreateTable()` function

```go
// Extract:
// 1. Table name
// 2. Schema name (default to "public")
// 3. Column definitions:
//    - Column name
//    - Type (INT, TEXT, BOOL, FLOAT)
//    - Nullable (default true, false if NOT NULL or PRIMARY KEY)
//    - PrimaryKey (true if PRIMARY KEY constraint)

// Validate:
// - Table name not empty
// - Column definitions not empty
// - Balanced parentheses
```

**Tests**:
- [ ] Test 4.16-4.17: CREATE TABLE parsing
- [ ] 5+ test cases

---

#### Task 2.7: Implement DROP TABLE Parser
**File**: `internal/sql/parser/parser.go` - `parseDropTable()` function

```go
// Extract:
// 1. Table name
// 2. Schema name (default to "public")

// Validate:
// - Table name not empty
```

**Tests**:
- [ ] Test 4.18: DROP TABLE parsing
- [ ] 2+ test cases

---

#### Task 2.8: Implement CREATE INDEX Parser
**File**: `internal/sql/parser/parser.go` - `parseCreateIndex()` function

```go
// Extract:
// 1. Index name
// 2. Table name
// 3. Schema name (default to "public")
// 4. Column list
// 5. Unique flag (true if CREATE UNIQUE INDEX)

// Validate:
// - ON clause exists
// - Column list not empty
// - Index name not empty
```

**Tests**:
- [ ] Test 4.19-4.20: CREATE INDEX parsing
- [ ] 4+ test cases

---

#### Task 2.9: Implement DROP INDEX Parser
**File**: `internal/sql/parser/parser.go` - `parseDropIndex()` function

```go
// Extract:
// 1. Index name
// 2. Table name
// 3. Schema name (default to "public")

// Validate:
// - ON clause exists
// - Table name not empty
// - Index name not empty
```

**Tests**:
- [ ] Test 4.21: DROP INDEX parsing
- [ ] 3+ test cases

---

### Phase 3: Integration (3 hours)

**Goal**: Connect expression parser with statement parsers and add error handling

#### Task 3.1: Integrate WHERE Expression Parsing
**File**: `internal/sql/parser/parser.go`

```go
// In parseSelect(), parseUpdate(), parseDelete():
if wherePart != "" {
    whereExpr, err := expr.ParseExpr(wherePart)
    if err != nil {
        return AST{}, fmt.Errorf("parser: invalid WHERE clause: %w", err)
    }
    ast.WhereExpr = whereExpr
}
```

**Tests**:
- [ ] Test 5.8: Backward compatibility
- [ ] Test 5.1-5.7: Integration tests
- [ ] 3+ test cases

---

#### Task 3.2: Add Error Handling & Validation
**File**: `internal/sql/parser/parser.go`

```go
// Ensure all parsers:
// 1. Validate required fields
// 2. Return descriptive errors
// 3. Include context in error messages
// 4. Check for balanced parentheses
// 5. Verify required keywords present
```

**Tests**:
- [ ] All error cases from requirements
- [ ] 15+ test cases

---

#### Task 3.3: Add Backward Compatibility Support
**File**: `internal/sql/parser/parser.go`

```go
// Ensure all DML statements populate:
// 1. AST.Where (string) - raw WHERE clause
// 2. AST.WhereExpr (Expr) - parsed expression tree
// 3. UpdateSpec.Where (string) - for UPDATE
// 4. DeleteSpec.Where (string) - for DELETE
```

**Tests**:
- [ ] Test 5.8: Both fields populated
- [ ] 1+ test case

---

### Phase 4: Testing (15-20 hours)

**Goal**: Implement comprehensive test suite

#### Task 4.1-4.15: Unit Tests
**Files**: `internal/sql/parser/parser_test.go`, `internal/sql/expr/expr_test.go`

Implement tests for:
- [ ] Transaction control (Task 4.1)
- [ ] SELECT parser (Task 4.2)
- [ ] INSERT parser (Task 4.3)
- [ ] UPDATE parser (Task 4.4)
- [ ] DELETE parser (Task 4.5)
- [ ] CREATE TABLE parser (Task 4.6)
- [ ] DROP TABLE parser (Task 4.7)
- [ ] CREATE INDEX parser (Task 4.8)
- [ ] DROP INDEX parser (Task 4.9)
- [ ] Lexer (Task 4.10)
- [ ] Expression parser (Task 4.11)
- [ ] Integration tests (Task 4.12)
- [ ] Error propagation (Task 4.13)
- [ ] Schema handling (Task 4.14)
- [ ] Edge cases (Task 4.15)

**Total**: 15 test tasks

---

## 🧪 Testing Strategy

### Run Tests During Implementation

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/sql/parser/...
go test ./internal/sql/expr/...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestParseSelect ./internal/sql/parser/...
```

### Test Coverage Goals

- [ ] > 90% code coverage
- [ ] All 100+ test cases pass
- [ ] No regressions in existing tests
- [ ] All error cases handled

---

## 📊 Implementation Checklist

### Phase 1: Foundation
- [ ] Task 1.1: Expression AST Types
- [ ] Task 1.2: Lexer Tokenization
- [ ] Task 1.3: Expression Parser

### Phase 2: Statement Parsers
- [ ] Task 2.1: Transaction Control Parser
- [ ] Task 2.2: SELECT Parser
- [ ] Task 2.3: INSERT Parser
- [ ] Task 2.4: UPDATE Parser
- [ ] Task 2.5: DELETE Parser
- [ ] Task 2.6: CREATE TABLE Parser
- [ ] Task 2.7: DROP TABLE Parser
- [ ] Task 2.8: CREATE INDEX Parser
- [ ] Task 2.9: DROP INDEX Parser

### Phase 3: Integration
- [ ] Task 3.1: WHERE Expression Integration
- [ ] Task 3.2: Error Handling & Validation
- [ ] Task 3.3: Backward Compatibility

### Phase 4: Testing
- [ ] Task 4.1-4.15: Unit and Integration Tests

### Quality Assurance
- [ ] Code coverage > 90%
- [ ] All tests passing
- [ ] No regressions
- [ ] Code review approved
- [ ] Documentation updated

---

## 💡 Implementation Tips

### 1. Start with Tests
- Write test cases first (from test-cases.md)
- Implement code to pass tests
- Ensures requirements are met

### 2. Follow Layer Architecture
- Implement bottom-up: Types → Lexer → Expression Parser → Statement Parsers
- Each layer depends on layers below it
- Easier to debug and test

### 3. Use Table-Driven Tests
```go
tests := []struct {
    name    string
    input   string
    want    AST
    wantErr bool
}{
    {"simple select", "SELECT id FROM users", ...},
    {"with where", "SELECT id FROM users WHERE id = 1", ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test code
    })
}
```

### 4. Error Messages Matter
- Include context: what was expected, what was found
- Include position if possible
- Make debugging easier

### 5. Test Edge Cases
- Empty input
- Large input
- Special characters
- Unicode
- Whitespace variations

---

## 🔍 Debugging Tips

### 1. Print Token Stream
```go
tokens := tokenize(input)
for _, tok := range tokens {
    fmt.Printf("Token: %v = %q\n", tok.typ, tok.val)
}
```

### 2. Print Expression Tree
```go
expr, _ := ParseExpr(whereClause)
fmt.Printf("Expr: %#v\n", expr)
```

### 3. Use Verbose Tests
```bash
go test -v ./internal/sql/parser/...
```

### 4. Add Debug Logging
```go
logging.DebugContext(ctx, "[Parser] Parsing INSERT statement")
```

---

## 📚 Reference Documents

- **requirements.md**: What needs to be built
- **design.md**: How it should be built (with layer architecture)
- **test-cases.md**: How to verify it works (100+ test cases)
- **tasks.md**: Implementation tasks with dependencies
- **SPEC-SUMMARY.md**: Overview of entire specification

---

## ✅ Success Criteria

When implementation is complete:

- [ ] All 20 requirements satisfied
- [ ] All 30 tasks completed
- [ ] All 100+ test cases passing
- [ ] Code coverage > 90%
- [ ] No regressions in existing tests
- [ ] All error cases handled gracefully
- [ ] Documentation updated
- [ ] Code review approved
- [ ] Ready for integration with planner and executor

---

## 🎯 Next Steps

1. **Review** this guide and the specification documents
2. **Start** with Phase 1 (Foundation tasks)
3. **Implement** tests alongside code
4. **Run** tests frequently to verify progress
5. **Complete** all phases in order
6. **Verify** all success criteria met

---

**Good luck with the implementation! 🚀**

For questions, refer to the specification documents or the test cases for clarification.
