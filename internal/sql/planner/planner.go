package planner

import "github.com/hainn191297/myDb/internal/sql/parser"

// Plan describes a physical execution plan produced from an AST.
type Plan struct {
    Root Operator
}

// Operator is the core interface implemented by executors.
type Operator interface {
    Name() string
}

// Build transforms the parser AST into an executable plan.
func Build(ast parser.AST) (Plan, error) {
    return Plan{}, ErrNotImplemented
}

var ErrNotImplemented = errPlaceholder("planner not implemented")

type errPlaceholder string

func (e errPlaceholder) Error() string { return string(e) }
