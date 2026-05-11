package planner

import (
	"github.com/hainn191297/myDb/internal/sql/expr"
)

// FilterOp applies predicates to rows from a child operator.
// It filters rows based on a WHERE clause expression.
type FilterOp struct {
	Child           Operator
	Predicate       expr.Expr
	Cost_           Cost
	EstimatedRowCnt int64
}

func (f *FilterOp) Name() string { return "Filter" }

func (f *FilterOp) EstimatedCost() Cost {
	return f.Cost_
}

func (f *FilterOp) EstimatedRows() int64 {
	return f.EstimatedRowCnt
}

func (f *FilterOp) OutputColumns() []string {
	if f.Child == nil {
		return []string{}
	}
	return f.Child.OutputColumns()
}

// ProjectionOp selects specific columns from a child operator.
// It implements the SELECT column list projection.
type ProjectionOp struct {
	Child           Operator
	Columns         []string
	Cost_           Cost
	EstimatedRowCnt int64
}

func (p *ProjectionOp) Name() string { return "Projection" }

func (p *ProjectionOp) EstimatedCost() Cost {
	return p.Cost_
}

func (p *ProjectionOp) EstimatedRows() int64 {
	return p.EstimatedRowCnt
}

func (p *ProjectionOp) OutputColumns() []string {
	if len(p.Columns) == 0 {
		return []string{"*"}
	}
	return p.Columns
}

// JoinType enumerates the types of joins supported.
type JoinType string

const (
	InnerJoin JoinType = "INNER"
	LeftJoin  JoinType = "LEFT"
	RightJoin JoinType = "RIGHT"
	FullJoin  JoinType = "FULL"
)

// JoinOp is the interface for all join operations.
type JoinOp interface {
	Operator
	LeftChild() Operator
	RightChild() Operator
	JoinCondition() expr.Expr
	JoinType() JoinType
}

// NestedLoopJoinOp implements a nested loop join algorithm.
// For each row from the left child, it scans all rows from the right child
// and evaluates the join condition.
type NestedLoopJoinOp struct {
	Left            Operator
	Right           Operator
	Condition       expr.Expr
	Type            JoinType
	Cost_           Cost
	EstimatedRowCnt int64
}

func (n *NestedLoopJoinOp) Name() string { return "NestedLoopJoin" }

func (n *NestedLoopJoinOp) EstimatedCost() Cost {
	return n.Cost_
}

func (n *NestedLoopJoinOp) EstimatedRows() int64 {
	return n.EstimatedRowCnt
}

func (n *NestedLoopJoinOp) OutputColumns() []string {
	leftCols := []string{}
	rightCols := []string{}

	if n.Left != nil {
		leftCols = n.Left.OutputColumns()
	}
	if n.Right != nil {
		rightCols = n.Right.OutputColumns()
	}

	// Combine columns from both sides
	result := make([]string, 0, len(leftCols)+len(rightCols))
	result = append(result, leftCols...)
	result = append(result, rightCols...)
	return result
}

func (n *NestedLoopJoinOp) LeftChild() Operator {
	return n.Left
}

func (n *NestedLoopJoinOp) RightChild() Operator {
	return n.Right
}

func (n *NestedLoopJoinOp) JoinCondition() expr.Expr {
	return n.Condition
}

func (n *NestedLoopJoinOp) JoinType() JoinType {
	return n.Type
}

// SortKey represents a column to sort by with direction.
type SortKey struct {
	Column    string
	Ascending bool
}

// SortOp sorts rows from a child operator by specified columns.
// It implements ORDER BY clause execution.
type SortOp struct {
	Child           Operator
	SortKeys        []SortKey
	Cost_           Cost
	EstimatedRowCnt int64
}

func (s *SortOp) Name() string { return "Sort" }

func (s *SortOp) EstimatedCost() Cost {
	return s.Cost_
}

func (s *SortOp) EstimatedRows() int64 {
	return s.EstimatedRowCnt
}

func (s *SortOp) OutputColumns() []string {
	if s.Child == nil {
		return []string{}
	}
	return s.Child.OutputColumns()
}

// AggregateFunc represents an aggregate function in a GROUP BY clause.
type AggregateFunc struct {
	Function string // COUNT, SUM, AVG, MIN, MAX
	Column   string
	Alias    string
}

// AggregateOp performs aggregation and grouping.
// It implements GROUP BY and aggregate function execution.
type AggregateOp struct {
	Child           Operator
	GroupByColumns  []string
	Aggregates      []AggregateFunc
	Cost_           Cost
	EstimatedRowCnt int64
}

func (a *AggregateOp) Name() string { return "Aggregate" }

func (a *AggregateOp) EstimatedCost() Cost {
	return a.Cost_
}

func (a *AggregateOp) EstimatedRows() int64 {
	return a.EstimatedRowCnt
}

func (a *AggregateOp) OutputColumns() []string {
	// Output columns are grouping columns + aggregate aliases
	result := make([]string, 0, len(a.GroupByColumns)+len(a.Aggregates))
	result = append(result, a.GroupByColumns...)

	for _, agg := range a.Aggregates {
		if agg.Alias != "" {
			result = append(result, agg.Alias)
		} else {
			result = append(result, agg.Function+"("+agg.Column+")")
		}
	}

	return result
}

// LimitOp limits the number of rows returned from a child operator.
// It implements LIMIT and OFFSET clause execution.
type LimitOp struct {
	Child           Operator
	Limit           int64
	Offset          int64
	Cost_           Cost
	EstimatedRowCnt int64
}

func (l *LimitOp) Name() string { return "Limit" }

func (l *LimitOp) EstimatedCost() Cost {
	return l.Cost_
}

func (l *LimitOp) EstimatedRows() int64 {
	return l.EstimatedRowCnt
}

func (l *LimitOp) OutputColumns() []string {
	if l.Child == nil {
		return []string{}
	}
	return l.Child.OutputColumns()
}
