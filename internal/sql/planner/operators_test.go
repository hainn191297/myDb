package planner

import (
	"testing"

	"github.com/hainn191297/myDb/internal/sql/expr"
)

// TestSeqScanOp tests the SeqScanOp operator.
func TestSeqScanOp(t *testing.T) {
	tests := []struct {
		name           string
		op             *SeqScanOp
		expectedName   string
		expectedRows   int64
		expectedCols   []string
	}{
		{
			name: "basic seq scan",
			op: &SeqScanOp{
				Schema:          "public",
				Table:           "users",
				Columns:         []string{"id", "name"},
				EstimatedRowCnt: 1000,
				Cost_:           Cost{IOCost: 10.0, CPUCost: 5.0, TotalCost: 15.0},
			},
			expectedName: "SeqScan",
			expectedRows: 1000,
			expectedCols: []string{"id", "name"},
		},
		{
			name: "seq scan with wildcard",
			op: &SeqScanOp{
				Schema:          "public",
				Table:           "users",
				Columns:         []string{},
				EstimatedRowCnt: 500,
				Cost_:           Cost{IOCost: 5.0, CPUCost: 2.5, TotalCost: 7.5},
			},
			expectedName: "SeqScan",
			expectedRows: 500,
			expectedCols: []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Name(); got != tt.expectedName {
				t.Errorf("Name() = %v, want %v", got, tt.expectedName)
			}
			if got := tt.op.EstimatedRows(); got != tt.expectedRows {
				t.Errorf("EstimatedRows() = %v, want %v", got, tt.expectedRows)
			}
			if got := tt.op.OutputColumns(); len(got) != len(tt.expectedCols) {
				t.Errorf("OutputColumns() length = %v, want %v", len(got), len(tt.expectedCols))
			}
			cost := tt.op.EstimatedCost()
			if cost.TotalCost == 0 {
				t.Errorf("EstimatedCost() TotalCost should not be zero")
			}
		})
	}
}

// TestIndexScanOp tests the IndexScanOp operator.
func TestIndexScanOp(t *testing.T) {
	tests := []struct {
		name           string
		op             *IndexScanOp
		expectedName   string
		expectedRows   int64
		expectedCols   []string
	}{
		{
			name: "basic index scan",
			op: &IndexScanOp{
				Schema:          "public",
				Table:           "users",
				IndexName:       "idx_users_id",
				Columns:         []string{"id", "name"},
				EstimatedRowCnt: 100,
				Cost_:           Cost{IOCost: 2.0, CPUCost: 1.0, TotalCost: 3.0},
			},
			expectedName: "IndexScan",
			expectedRows: 100,
			expectedCols: []string{"id", "name"},
		},
		{
			name: "index scan with wildcard",
			op: &IndexScanOp{
				Schema:          "public",
				Table:           "users",
				IndexName:       "idx_users_id",
				Columns:         []string{},
				EstimatedRowCnt: 50,
				Cost_:           Cost{IOCost: 1.0, CPUCost: 0.5, TotalCost: 1.5},
			},
			expectedName: "IndexScan",
			expectedRows: 50,
			expectedCols: []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Name(); got != tt.expectedName {
				t.Errorf("Name() = %v, want %v", got, tt.expectedName)
			}
			if got := tt.op.EstimatedRows(); got != tt.expectedRows {
				t.Errorf("EstimatedRows() = %v, want %v", got, tt.expectedRows)
			}
			if got := tt.op.OutputColumns(); len(got) != len(tt.expectedCols) {
				t.Errorf("OutputColumns() length = %v, want %v", len(got), len(tt.expectedCols))
			}
		})
	}
}

// TestFilterOp tests the FilterOp operator.
func TestFilterOp(t *testing.T) {
	tests := []struct {
		name           string
		op             *FilterOp
		expectedName   string
		expectedRows   int64
	}{
		{
			name: "basic filter",
			op: &FilterOp{
				Child: &SeqScanOp{
					Schema:          "public",
					Table:           "users",
					Columns:         []string{"id", "name"},
					EstimatedRowCnt: 1000,
				},
				Predicate:       &expr.BinaryExpr{Op: expr.OpEquals},
				EstimatedRowCnt: 100,
				Cost_:           Cost{IOCost: 10.0, CPUCost: 10.0, TotalCost: 20.0},
			},
			expectedName: "Filter",
			expectedRows: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Name(); got != tt.expectedName {
				t.Errorf("Name() = %v, want %v", got, tt.expectedName)
			}
			if got := tt.op.EstimatedRows(); got != tt.expectedRows {
				t.Errorf("EstimatedRows() = %v, want %v", got, tt.expectedRows)
			}
			// Filter should inherit columns from child
			cols := tt.op.OutputColumns()
			if len(cols) == 0 {
				t.Errorf("OutputColumns() should inherit from child")
			}
		})
	}
}

// TestProjectionOp tests the ProjectionOp operator.
func TestProjectionOp(t *testing.T) {
	tests := []struct {
		name           string
		op             *ProjectionOp
		expectedName   string
		expectedRows   int64
		expectedCols   []string
	}{
		{
			name: "basic projection",
			op: &ProjectionOp{
				Child: &SeqScanOp{
					Schema:          "public",
					Table:           "users",
					Columns:         []string{"id", "name", "email"},
					EstimatedRowCnt: 1000,
				},
				Columns:         []string{"id", "name"},
				EstimatedRowCnt: 1000,
				Cost_:           Cost{IOCost: 10.0, CPUCost: 5.0, TotalCost: 15.0},
			},
			expectedName: "Projection",
			expectedRows: 1000,
			expectedCols: []string{"id", "name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Name(); got != tt.expectedName {
				t.Errorf("Name() = %v, want %v", got, tt.expectedName)
			}
			if got := tt.op.EstimatedRows(); got != tt.expectedRows {
				t.Errorf("EstimatedRows() = %v, want %v", got, tt.expectedRows)
			}
			cols := tt.op.OutputColumns()
			if len(cols) != len(tt.expectedCols) {
				t.Errorf("OutputColumns() length = %v, want %v", len(cols), len(tt.expectedCols))
			}
		})
	}
}

// TestNestedLoopJoinOp tests the NestedLoopJoinOp operator.
func TestNestedLoopJoinOp(t *testing.T) {
	tests := []struct {
		name           string
		op             *NestedLoopJoinOp
		expectedName   string
		expectedRows   int64
		expectedType   JoinType
	}{
		{
			name: "inner join",
			op: &NestedLoopJoinOp{
				Left: &SeqScanOp{
					Schema:          "public",
					Table:           "users",
					Columns:         []string{"id", "name"},
					EstimatedRowCnt: 1000,
				},
				Right: &SeqScanOp{
					Schema:          "public",
					Table:           "orders",
					Columns:         []string{"user_id", "amount"},
					EstimatedRowCnt: 5000,
				},
				Type:            InnerJoin,
				EstimatedRowCnt: 5000,
				Cost_:           Cost{IOCost: 50.0, CPUCost: 100.0, TotalCost: 150.0},
			},
			expectedName: "NestedLoopJoin",
			expectedRows: 5000,
			expectedType: InnerJoin,
		},
		{
			name: "left join",
			op: &NestedLoopJoinOp{
				Left: &SeqScanOp{
					Schema:          "public",
					Table:           "users",
					Columns:         []string{"id", "name"},
					EstimatedRowCnt: 1000,
				},
				Right: &SeqScanOp{
					Schema:          "public",
					Table:           "orders",
					Columns:         []string{"user_id", "amount"},
					EstimatedRowCnt: 5000,
				},
				Type:            LeftJoin,
				EstimatedRowCnt: 1000,
				Cost_:           Cost{IOCost: 50.0, CPUCost: 100.0, TotalCost: 150.0},
			},
			expectedName: "NestedLoopJoin",
			expectedRows: 1000,
			expectedType: LeftJoin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Name(); got != tt.expectedName {
				t.Errorf("Name() = %v, want %v", got, tt.expectedName)
			}
			if got := tt.op.EstimatedRows(); got != tt.expectedRows {
				t.Errorf("EstimatedRows() = %v, want %v", got, tt.expectedRows)
			}
			if got := tt.op.JoinType(); got != tt.expectedType {
				t.Errorf("JoinType() = %v, want %v", got, tt.expectedType)
			}
			if tt.op.LeftChild() == nil {
				t.Errorf("LeftChild() should not be nil")
			}
			if tt.op.RightChild() == nil {
				t.Errorf("RightChild() should not be nil")
			}
			// Output columns should combine both sides
			cols := tt.op.OutputColumns()
			if len(cols) == 0 {
				t.Errorf("OutputColumns() should combine left and right columns")
			}
		})
	}
}

// TestSortOp tests the SortOp operator.
func TestSortOp(t *testing.T) {
	tests := []struct {
		name           string
		op             *SortOp
		expectedName   string
		expectedRows   int64
	}{
		{
			name: "basic sort",
			op: &SortOp{
				Child: &SeqScanOp{
					Schema:          "public",
					Table:           "users",
					Columns:         []string{"id", "name"},
					EstimatedRowCnt: 1000,
				},
				SortKeys: []SortKey{
					{Column: "name", Ascending: true},
				},
				EstimatedRowCnt: 1000,
				Cost_:           Cost{IOCost: 10.0, CPUCost: 50.0, TotalCost: 60.0},
			},
			expectedName: "Sort",
			expectedRows: 1000,
		},
		{
			name: "multi-column sort",
			op: &SortOp{
				Child: &SeqScanOp{
					Schema:          "public",
					Table:           "users",
					Columns:         []string{"id", "name", "age"},
					EstimatedRowCnt: 1000,
				},
				SortKeys: []SortKey{
					{Column: "age", Ascending: false},
					{Column: "name", Ascending: true},
				},
				EstimatedRowCnt: 1000,
				Cost_:           Cost{IOCost: 10.0, CPUCost: 50.0, TotalCost: 60.0},
			},
			expectedName: "Sort",
			expectedRows: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Name(); got != tt.expectedName {
				t.Errorf("Name() = %v, want %v", got, tt.expectedName)
			}
			if got := tt.op.EstimatedRows(); got != tt.expectedRows {
				t.Errorf("EstimatedRows() = %v, want %v", got, tt.expectedRows)
			}
			// Sort should inherit columns from child
			cols := tt.op.OutputColumns()
			if len(cols) == 0 {
				t.Errorf("OutputColumns() should inherit from child")
			}
		})
	}
}

// TestAggregateOp tests the AggregateOp operator.
func TestAggregateOp(t *testing.T) {
	tests := []struct {
		name           string
		op             *AggregateOp
		expectedName   string
		expectedRows   int64
	}{
		{
			name: "count aggregate",
			op: &AggregateOp{
				Child: &SeqScanOp{
					Schema:          "public",
					Table:           "users",
					Columns:         []string{"id", "name", "age"},
					EstimatedRowCnt: 1000,
				},
				GroupByColumns: []string{},
				Aggregates: []AggregateFunc{
					{Function: "COUNT", Column: "*", Alias: "count"},
				},
				EstimatedRowCnt: 1,
				Cost_:           Cost{IOCost: 10.0, CPUCost: 20.0, TotalCost: 30.0},
			},
			expectedName: "Aggregate",
			expectedRows: 1,
		},
		{
			name: "group by aggregate",
			op: &AggregateOp{
				Child: &SeqScanOp{
					Schema:          "public",
					Table:           "users",
					Columns:         []string{"id", "name", "age"},
					EstimatedRowCnt: 1000,
				},
				GroupByColumns: []string{"age"},
				Aggregates: []AggregateFunc{
					{Function: "COUNT", Column: "*", Alias: "count"},
					{Function: "AVG", Column: "id", Alias: "avg_id"},
				},
				EstimatedRowCnt: 50,
				Cost_:           Cost{IOCost: 10.0, CPUCost: 30.0, TotalCost: 40.0},
			},
			expectedName: "Aggregate",
			expectedRows: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Name(); got != tt.expectedName {
				t.Errorf("Name() = %v, want %v", got, tt.expectedName)
			}
			if got := tt.op.EstimatedRows(); got != tt.expectedRows {
				t.Errorf("EstimatedRows() = %v, want %v", got, tt.expectedRows)
			}
			// Output columns should include grouping columns and aggregates
			cols := tt.op.OutputColumns()
			expectedColCount := len(tt.op.GroupByColumns) + len(tt.op.Aggregates)
			if len(cols) != expectedColCount {
				t.Errorf("OutputColumns() length = %v, want %v", len(cols), expectedColCount)
			}
		})
	}
}

// TestLimitOp tests the LimitOp operator.
func TestLimitOp(t *testing.T) {
	tests := []struct {
		name           string
		op             *LimitOp
		expectedName   string
		expectedRows   int64
	}{
		{
			name: "basic limit",
			op: &LimitOp{
				Child: &SeqScanOp{
					Schema:          "public",
					Table:           "users",
					Columns:         []string{"id", "name"},
					EstimatedRowCnt: 1000,
				},
				Limit:           10,
				Offset:          0,
				EstimatedRowCnt: 10,
				Cost_:           Cost{IOCost: 10.0, CPUCost: 5.0, TotalCost: 15.0},
			},
			expectedName: "Limit",
			expectedRows: 10,
		},
		{
			name: "limit with offset",
			op: &LimitOp{
				Child: &SeqScanOp{
					Schema:          "public",
					Table:           "users",
					Columns:         []string{"id", "name"},
					EstimatedRowCnt: 1000,
				},
				Limit:           10,
				Offset:          20,
				EstimatedRowCnt: 10,
				Cost_:           Cost{IOCost: 10.0, CPUCost: 5.0, TotalCost: 15.0},
			},
			expectedName: "Limit",
			expectedRows: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Name(); got != tt.expectedName {
				t.Errorf("Name() = %v, want %v", got, tt.expectedName)
			}
			if got := tt.op.EstimatedRows(); got != tt.expectedRows {
				t.Errorf("EstimatedRows() = %v, want %v", got, tt.expectedRows)
			}
			// Limit should inherit columns from child
			cols := tt.op.OutputColumns()
			if len(cols) == 0 {
				t.Errorf("OutputColumns() should inherit from child")
			}
		})
	}
}

// TestCostComparison tests cost comparison functions.
func TestCostComparison(t *testing.T) {
	tests := []struct {
		name     string
		c1       Cost
		c2       Cost
		expected int
	}{
		{
			name:     "c1 less than c2",
			c1:       Cost{IOCost: 5.0, CPUCost: 5.0, TotalCost: 10.0},
			c2:       Cost{IOCost: 10.0, CPUCost: 10.0, TotalCost: 20.0},
			expected: -1,
		},
		{
			name:     "c1 equal to c2",
			c1:       Cost{IOCost: 10.0, CPUCost: 10.0, TotalCost: 20.0},
			c2:       Cost{IOCost: 10.0, CPUCost: 10.0, TotalCost: 20.0},
			expected: 0,
		},
		{
			name:     "c1 greater than c2",
			c1:       Cost{IOCost: 15.0, CPUCost: 15.0, TotalCost: 30.0},
			c2:       Cost{IOCost: 10.0, CPUCost: 10.0, TotalCost: 20.0},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareCosts(tt.c1, tt.c2); got != tt.expected {
				t.Errorf("CompareCosts() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestAggregateCosts tests cost aggregation.
func TestAggregateCosts(t *testing.T) {
	tests := []struct {
		name     string
		costs    []Cost
		expected Cost
	}{
		{
			name: "single cost",
			costs: []Cost{
				{IOCost: 10.0, CPUCost: 5.0, TotalCost: 15.0},
			},
			expected: Cost{IOCost: 10.0, CPUCost: 5.0, TotalCost: 15.0},
		},
		{
			name: "multiple costs",
			costs: []Cost{
				{IOCost: 10.0, CPUCost: 5.0, TotalCost: 15.0},
				{IOCost: 20.0, CPUCost: 10.0, TotalCost: 30.0},
				{IOCost: 5.0, CPUCost: 2.5, TotalCost: 7.5},
			},
			expected: Cost{IOCost: 35.0, CPUCost: 17.5, TotalCost: 52.5},
		},
		{
			name:     "empty costs",
			costs:    []Cost{},
			expected: Cost{IOCost: 0, CPUCost: 0, TotalCost: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AggregateCosts(tt.costs...)
			if got.IOCost != tt.expected.IOCost || got.CPUCost != tt.expected.CPUCost || got.TotalCost != tt.expected.TotalCost {
				t.Errorf("AggregateCosts() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestNewCost tests cost creation with validation.
func TestNewCost(t *testing.T) {
	tests := []struct {
		name      string
		ioCost    float64
		cpuCost   float64
		shouldErr bool
	}{
		{
			name:      "valid cost",
			ioCost:    10.0,
			cpuCost:   5.0,
			shouldErr: false,
		},
		{
			name:      "zero cost",
			ioCost:    0,
			cpuCost:   0,
			shouldErr: false,
		},
		{
			name:      "negative io cost",
			ioCost:    -1.0,
			cpuCost:   5.0,
			shouldErr: true,
		},
		{
			name:      "negative cpu cost",
			ioCost:    10.0,
			cpuCost:   -1.0,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := NewCost(tt.ioCost, tt.cpuCost)
			if (err != nil) != tt.shouldErr {
				t.Errorf("NewCost() error = %v, shouldErr %v", err, tt.shouldErr)
			}
			if !tt.shouldErr {
				if cost.IOCost != tt.ioCost || cost.CPUCost != tt.cpuCost {
					t.Errorf("NewCost() = %v, want IOCost=%v, CPUCost=%v", cost, tt.ioCost, tt.cpuCost)
				}
				if cost.TotalCost != tt.ioCost+tt.cpuCost {
					t.Errorf("NewCost() TotalCost = %v, want %v", cost.TotalCost, tt.ioCost+tt.cpuCost)
				}
			}
		})
	}
}

// TestCardinalityBounds tests cardinality boundary enforcement.
func TestCardinalityBounds(t *testing.T) {
	tests := []struct {
		name      string
		estimated int64
		maxRows   int64
		expected  int64
	}{
		{
			name:      "within bounds",
			estimated: 500,
			maxRows:   1000,
			expected:  500,
		},
		{
			name:      "below minimum",
			estimated: 0,
			maxRows:   1000,
			expected:  1,
		},
		{
			name:      "above maximum",
			estimated: 2000,
			maxRows:   1000,
			expected:  1000,
		},
		{
			name:      "negative estimated",
			estimated: -100,
			maxRows:   1000,
			expected:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnsureCardinalityBounds(tt.estimated, tt.maxRows)
			if got != tt.expected {
				t.Errorf("EnsureCardinalityBounds() = %v, want %v", got, tt.expected)
			}
		})
	}
}
