package executor

import "context"

import "github.com/hainn191297/myDb/internal/sql/planner"

// Executor walks a plan tree and produces row streams.
type Executor struct{}

// New builds an executor for the supplied plan.
func New(plan planner.Plan) *Executor {
    return &Executor{}
}

// Next executes the plan and returns whether more rows exist.
func (e *Executor) Next(ctx context.Context) (bool, error) {
    return false, ErrNotImplemented
}

var ErrNotImplemented = errPlaceholder("executor not implemented")

type errPlaceholder string

func (e errPlaceholder) Error() string { return string(e) }
