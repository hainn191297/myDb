package planner

import (
	"fmt"
	"math"
)

// Cost represents execution cost metrics for a query plan node.
// It tracks I/O operations, CPU operations, and total estimated cost.
type Cost struct {
	// IOCost represents the estimated cost of I/O operations (page reads/writes).
	// Measured in abstract units (typically page accesses).
	IOCost float64

	// CPUCost represents the estimated cost of CPU operations (comparisons, evaluations).
	// Measured in abstract units (typically number of operations).
	CPUCost float64

	// TotalCost is the sum of IOCost and CPUCost, representing total execution cost.
	TotalCost float64
}

// CostEstimator provides methods for estimating cardinality, I/O costs, and CPU costs
// for different query operations. It uses table statistics from the schema catalog
// to make informed estimates.
type CostEstimator interface {
	// EstimateCardinality estimates the number of rows produced by an operation.
	// Returns the estimated row count, which must be >= 1 and <= source table rows.
	EstimateCardinality(op Operator) int64

	// EstimateIOCost estimates the I/O cost for an operation.
	// Returns the estimated I/O cost in abstract units.
	EstimateIOCost(op Operator) float64

	// EstimateCPUCost estimates the CPU cost for an operation.
	// Returns the estimated CPU cost in abstract units.
	EstimateCPUCost(op Operator) float64

	// EstimateTotalCost estimates the total cost for an operation.
	// Returns a Cost struct with IOCost, CPUCost, and TotalCost populated.
	EstimateTotalCost(op Operator) Cost
}

// Cost comparison and aggregation helper functions

// CompareCosts returns -1 if c1 < c2, 0 if c1 == c2, 1 if c1 > c2.
// Comparison is based on TotalCost.
func CompareCosts(c1, c2 Cost) int {
	if c1.TotalCost < c2.TotalCost {
		return -1
	}
	if c1.TotalCost > c2.TotalCost {
		return 1
	}
	return 0
}

// AggregateCosts combines multiple costs into a single cost.
// Used when combining costs of sequential operations in a plan.
func AggregateCosts(costs ...Cost) Cost {
	if len(costs) == 0 {
		return Cost{IOCost: 0, CPUCost: 0, TotalCost: 0}
	}

	var totalIO, totalCPU float64
	for _, c := range costs {
		totalIO += c.IOCost
		totalCPU += c.CPUCost
	}

	return Cost{
		IOCost:    totalIO,
		CPUCost:   totalCPU,
		TotalCost: totalIO + totalCPU,
	}
}

// NewCost creates a new Cost struct with the given I/O and CPU costs.
// TotalCost is automatically calculated as IOCost + CPUCost.
// Returns an error if either cost is negative.
func NewCost(ioCost, cpuCost float64) (Cost, error) {
	if ioCost < 0 {
		return Cost{}, fmt.Errorf("cost: IOCost cannot be negative: %f", ioCost)
	}
	if cpuCost < 0 {
		return Cost{}, fmt.Errorf("cost: CPUCost cannot be negative: %f", cpuCost)
	}

	return Cost{
		IOCost:    ioCost,
		CPUCost:   cpuCost,
		TotalCost: ioCost + cpuCost,
	}, nil
}

// IsZero returns true if the cost is zero (no I/O or CPU cost).
func (c Cost) IsZero() bool {
	return c.IOCost == 0 && c.CPUCost == 0
}

// String returns a string representation of the cost.
func (c Cost) String() string {
	return fmt.Sprintf("Cost{IO: %.2f, CPU: %.2f, Total: %.2f}", c.IOCost, c.CPUCost, c.TotalCost)
}

// Cost estimation constants
// These represent abstract units and can be tuned based on actual system performance.

const (
	// SequentialReadCost is the cost of reading a page sequentially.
	// Sequential reads are cheaper than random reads due to disk locality.
	SequentialReadCost = 1.0

	// RandomReadCost is the cost of reading a page randomly.
	// Random reads are more expensive due to disk seek time.
	RandomReadCost = 5.0

	// PredicateEvaluationCost is the cost of evaluating a predicate for one row.
	PredicateEvaluationCost = 0.1

	// ColumnExtractionCost is the cost of extracting one column from a row.
	ColumnExtractionCost = 0.01

	// ComparisonCost is the cost of comparing two values.
	ComparisonCost = 0.05

	// DefaultRowCost is the default cost of processing one row.
	DefaultRowCost = 0.1

	// PageSize is the assumed page size in bytes (4KB).
	PageSize = 4096

	// AverageRowSize is the assumed average row size in bytes.
	AverageRowSize = 100

	// DefaultSelectivity is the default selectivity for unknown predicates.
	// Assumes 10% of rows match the predicate.
	DefaultSelectivity = 0.1
)

// EnsureCardinalityBounds ensures that estimated cardinality is within valid bounds.
// Cardinality must be >= 1 and <= maxRows.
// Returns the bounded cardinality value.
func EnsureCardinalityBounds(estimated, maxRows int64) int64 {
	if estimated < 1 {
		return 1
	}
	if estimated > maxRows {
		return maxRows
	}
	return estimated
}

// EstimateSelectivity estimates the selectivity factor for a predicate.
// Selectivity is a value between 0.0 and 1.0 representing the fraction of rows
// that satisfy the predicate.
// For unknown predicates, returns DefaultSelectivity (0.1).
func EstimateSelectivity(predicate string) float64 {
	// This is a simplified implementation.
	// A real implementation would analyze the predicate structure and use column statistics.
	// For now, return default selectivity.
	if predicate == "" {
		return 1.0 // No filter, all rows match
	}
	return DefaultSelectivity
}

// EstimateRowsInRange estimates the number of rows in a key range.
// Used for index range scans.
// This is a simplified implementation that assumes uniform distribution.
func EstimateRowsInRange(totalRows int64, rangeSelectivity float64) int64 {
	estimated := int64(math.Round(float64(totalRows) * rangeSelectivity))
	return EnsureCardinalityBounds(estimated, totalRows)
}

// EstimateIndexPages estimates the number of index pages to read for a range scan.
// Assumes a B-tree index with the given height and branching factor.
func EstimateIndexPages(indexHeight int, branchingFactor int, leafPages int64, rangeSelectivity float64) int64 {
	// Estimate leaf pages to scan
	leafPagesToScan := int64(math.Ceil(float64(leafPages) * rangeSelectivity))

	// Internal pages (one per level to reach the leaf)
	internalPages := int64(indexHeight - 1)

	return internalPages + leafPagesToScan
}

// EstimateDataPages estimates the number of data pages to read for a given number of rows.
// Assumes average row size and page size constants.
func EstimateDataPages(rowCount int64) int64 {
	if rowCount == 0 {
		return 0
	}
	return int64(math.Ceil(float64(rowCount*int64(AverageRowSize)) / float64(PageSize)))
}

// EstimateTablePages estimates the number of pages needed to store a table.
// Used for sequential scan cost estimation.
func EstimateTablePages(tableSizeBytes int64) int64 {
	if tableSizeBytes == 0 {
		return 1 // At least one page
	}
	return int64(math.Ceil(float64(tableSizeBytes) / float64(PageSize)))
}
