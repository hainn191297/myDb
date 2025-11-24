package expr

import (
	"fmt"
)

// ParseExpr parses a WHERE clause string into an Expr AST.
// Returns nil if whereClause is empty.
// Example: "id = 1 AND status = 'active'" -> BinaryExpr{OpAnd, ...}
func ParseExpr(whereClause string) (Expr, error) {
	if whereClause == "" {
		return nil, nil
	}

	tokens := tokenize(whereClause)
	p := &parser{tokens: tokens, pos: 0}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	// Verify we consumed all tokens
	if p.current().typ != tokenEOF {
		return nil, fmt.Errorf("unexpected token after expression: %s", p.current().val)
	}

	return expr, nil
}

// parser implements recursive descent parsing for WHERE expressions.
type parser struct {
	tokens []token
	pos    int
}

// current returns the current token without advancing.
func (p *parser) current() token {
	if p.pos >= len(p.tokens) {
		return token{typ: tokenEOF}
	}
	return p.tokens[p.pos]
}

// advance moves to the next token.
func (p *parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// match checks if current token matches the given value and advances if so.
func (p *parser) match(val string) bool {
	if p.current().val == val {
		p.advance()
		return true
	}
	return false
}

// expect verifies current token matches and consumes it, or returns error.
func (p *parser) expect(val string) error {
	if !p.match(val) {
		return fmt.Errorf("expected %s, got %s", val, p.current().val)
	}
	return nil
}

// Operator precedence (low to high):
// OR
// AND
// NOT
// comparison operators (=, !=, <, >, <=, >=)
// primary (literals, column refs, parentheses)

// parseOr parses OR expressions (lowest precedence).
func (p *parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.match("OR") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Left:  left,
			Op:    OpOr,
			Right: right,
		}
	}

	return left, nil
}

// parseAnd parses AND expressions.
func (p *parser) parseAnd() (Expr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.match("AND") {
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		// Validate right operand is not empty
		if right == nil {
			return nil, fmt.Errorf("expected expression after AND")
		}
		left = &BinaryExpr{
			Left:  left,
			Op:    OpAnd,
			Right: right,
		}
	}

	return left, nil
}

// parseNot parses NOT expressions.
func (p *parser) parseNot() (Expr, error) {
	if p.match("NOT") {
		expr, err := p.parseNot() // Recursive for multiple NOTs
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{
			Op:   OpNot,
			Expr: expr,
		}, nil
	}

	return p.parseComparison()
}

// parseComparison parses comparison operators (=, !=, <, >, <=, >=).
func (p *parser) parseComparison() (Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// Check for IS NULL / IS NOT NULL
	if p.match("IS") {
		if p.match("NOT") {
			if err := p.expect("NULL"); err != nil {
				return nil, err
			}
			return &UnaryExpr{
				Op:   OpIsNotNull,
				Expr: left,
			}, nil
		}
		if err := p.expect("NULL"); err != nil {
			return nil, err
		}
		return &UnaryExpr{
			Op:   OpIsNull,
			Expr: left,
		}, nil
	}

	// Check for comparison operators
	tok := p.current()
	if tok.typ == tokenOperator {
		op, err := parseOperator(tok.val)
		if err != nil {
			return nil, err
		}
		p.advance()

		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}

		return &BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
		}, nil
	}

	return left, nil
}

// parsePrimary parses primary expressions (literals, column refs, parentheses).
func (p *parser) parsePrimary() (Expr, error) {
	tok := p.current()

	// Parenthesized expression
	if tok.typ == tokenLParen {
		p.advance()
		expr, err := p.parseOr() // Start from top precedence
		if err != nil {
			return nil, err
		}
		if err := p.expect(")"); err != nil {
			return nil, err
		}
		return expr, nil
	}

	// Number literal
	if tok.typ == tokenNumber {
		p.advance()

		// Try int first, then float
		if intVal, ok := ParseInt(tok.val); ok {
			return &LiteralExpr{
				Value: intVal,
				Type:  TypeInt,
			}, nil
		}

		if floatVal, ok := ParseFloat(tok.val); ok {
			return &LiteralExpr{
				Value: floatVal,
				Type:  TypeFloat,
			}, nil
		}

		return nil, fmt.Errorf("invalid number: %s", tok.val)
	}

	// String literal
	if tok.typ == tokenString {
		p.advance()
		return &LiteralExpr{
			Value: StripQuotes(tok.val),
			Type:  TypeText,
		}, nil
	}

	// Boolean keywords (true/false) or NULL
	if tok.typ == tokenKeyword || tok.typ == tokenIdent {
		p.advance()
		switch tok.val {
		case "TRUE", "true":
			return &LiteralExpr{Value: true, Type: TypeBool}, nil
		case "FALSE", "false":
			return &LiteralExpr{Value: false, Type: TypeBool}, nil
		case "NULL":
			return &LiteralExpr{Value: nil, Type: TypeNull}, nil
		}

		// Otherwise it's a column reference
		return &ColumnRefExpr{Name: tok.val}, nil
	}

	// Identifier (column name)
	if tok.typ == tokenIdent {
		p.advance()
		return &ColumnRefExpr{Name: tok.val}, nil
	}

	return nil, fmt.Errorf("unexpected token: %s", tok.val)
}

// parseOperator converts operator string to BinaryOp.
func parseOperator(op string) (BinaryOp, error) {
	switch op {
	case "=":
		return OpEquals, nil
	case "!=", "<>":
		return OpNotEquals, nil
	case "<":
		return OpLessThan, nil
	case "<=":
		return OpLessOrEqual, nil
	case ">":
		return OpGreaterThan, nil
	case ">=":
		return OpGreaterOrEqual, nil
	default:
		return 0, fmt.Errorf("unknown operator: %s", op)
	}
}
