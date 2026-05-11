package planner

import (
	"context"
	"fmt"
	"strings"

	"github.com/hainn191297/myDb/internal/logging"
	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/expr"
	"github.com/hainn191297/myDb/internal/sql/parser"
)

// Plan describes a physical execution plan produced from an AST.
type Plan struct {
	Root Operator
}

// Operator is the core interface implemented by all plan nodes.
// It defines the contract for physical execution operators.
type Operator interface {
	// Name returns the operator type name (e.g., "SeqScan", "Filter", "Join").
	Name() string

	// EstimatedCost returns the estimated execution cost for this operator.
	// The cost includes I/O and CPU components.
	EstimatedCost() Cost

	// EstimatedRows returns the estimated number of rows produced by this operator.
	// Must be >= 1 and <= source table row count.
	EstimatedRows() int64

	// OutputColumns returns the names of columns produced by this operator.
	// Used for tracking column lineage through the plan tree.
	OutputColumns() []string
}

// SeqScanOp is a physical operator for sequential table scans.
// It reads all rows from a table, optionally applying a filter.
type SeqScanOp struct {
	Schema          string
	Table           string
	Columns         []string
	Filter          string    // Deprecated
	FilterExpr      expr.Expr // Structured filter
	Cost_           Cost
	EstimatedRowCnt int64
}

func (s *SeqScanOp) Name() string { return "SeqScan" }

func (s *SeqScanOp) EstimatedCost() Cost {
	return s.Cost_
}

func (s *SeqScanOp) EstimatedRows() int64 {
	return s.EstimatedRowCnt
}

func (s *SeqScanOp) OutputColumns() []string {
	if len(s.Columns) == 0 {
		return []string{"*"}
	}
	return s.Columns
}

// TxnAction enumerates transaction control operations.
type TxnAction string

const (
	TxnBegin    TxnAction = "BEGIN"
	TxnCommit   TxnAction = "COMMIT"
	TxnRollback TxnAction = "ROLLBACK"
)

// TxnOp represents BEGIN/COMMIT/ROLLBACK.
type TxnOp struct {
	Action TxnAction
}

func (t *TxnOp) Name() string { return string(t.Action) }

func (t *TxnOp) EstimatedCost() Cost {
	return Cost{IOCost: 0, CPUCost: 0, TotalCost: 0}
}

func (t *TxnOp) EstimatedRows() int64 {
	return 0
}

func (t *TxnOp) OutputColumns() []string {
	return []string{}
}

// CreateTableOp represents CREATE TABLE execution.
type CreateTableOp struct {
	Schema  string
	Table   string
	Columns []schema.ColumnDef
}

func (c *CreateTableOp) Name() string { return "CreateTable" }

func (c *CreateTableOp) EstimatedCost() Cost {
	return Cost{IOCost: 0, CPUCost: 0, TotalCost: 0}
}

func (c *CreateTableOp) EstimatedRows() int64 {
	return 0
}

func (c *CreateTableOp) OutputColumns() []string {
	return []string{}
}

// DropTableOp represents DROP TABLE execution.
type DropTableOp struct {
	Schema string
	Table  string
}

func (d *DropTableOp) Name() string { return "DropTable" }

func (d *DropTableOp) EstimatedCost() Cost {
	return Cost{IOCost: 0, CPUCost: 0, TotalCost: 0}
}

func (d *DropTableOp) EstimatedRows() int64 {
	return 0
}

func (d *DropTableOp) OutputColumns() []string {
	return []string{}
}

// InsertOp represents INSERT execution.
type InsertOp struct {
	Schema          string
	Table           string
	Columns         []string   // Column names (validated against schema)
	Values          [][][]byte // List of rows, each row is a list of encoded values
	Cost_           Cost
	EstimatedRowCnt int64
}

func (i *InsertOp) Name() string { return "Insert" }

func (i *InsertOp) EstimatedCost() Cost {
	return i.Cost_
}

func (i *InsertOp) EstimatedRows() int64 {
	return i.EstimatedRowCnt
}

func (i *InsertOp) OutputColumns() []string {
	return []string{}
}

// UpdateOp represents UPDATE execution.
type UpdateOp struct {
	Schema          string
	Table           string
	SetClauses      map[string][]byte // column -> encoded value
	Filter          string            // WHERE clause (string for MVP)
	FilterExpr      expr.Expr         // Structured filter
	Cost_           Cost
	EstimatedRowCnt int64
}

func (u *UpdateOp) Name() string { return "Update" }

func (u *UpdateOp) EstimatedCost() Cost {
	return u.Cost_
}

func (u *UpdateOp) EstimatedRows() int64 {
	return u.EstimatedRowCnt
}

func (u *UpdateOp) OutputColumns() []string {
	return []string{}
}

// DeleteOp represents DELETE execution.
type DeleteOp struct {
	Schema          string
	Table           string
	Filter          string    // WHERE clause
	FilterExpr      expr.Expr // Structured filter
	Cost_           Cost
	EstimatedRowCnt int64
}

func (de *DeleteOp) Name() string { return "Delete" }

func (de *DeleteOp) EstimatedCost() Cost {
	return de.Cost_
}

func (de *DeleteOp) EstimatedRows() int64 {
	return de.EstimatedRowCnt
}

func (de *DeleteOp) OutputColumns() []string {
	return []string{}
}

// CreateIndexOp represents CREATE INDEX execution.
type CreateIndexOp struct {
	Schema          string
	Table           string
	IndexName       string
	Columns         []string
	Unique          bool
	Cost_           Cost
	EstimatedRowCnt int64
}

func (c *CreateIndexOp) Name() string { return "CreateIndex" }

func (c *CreateIndexOp) EstimatedCost() Cost {
	return c.Cost_
}

func (c *CreateIndexOp) EstimatedRows() int64 {
	return c.EstimatedRowCnt
}

func (c *CreateIndexOp) OutputColumns() []string {
	return []string{}
}

// DropIndexOp represents DROP INDEX execution.
type DropIndexOp struct {
	Schema          string
	Table           string
	IndexName       string
	Cost_           Cost
	EstimatedRowCnt int64
}

func (d *DropIndexOp) Name() string { return "DropIndex" }

func (d *DropIndexOp) EstimatedCost() Cost {
	return d.Cost_
}

func (d *DropIndexOp) EstimatedRows() int64 {
	return d.EstimatedRowCnt
}

func (d *DropIndexOp) OutputColumns() []string {
	return []string{}
}

// IndexScanOp represents an index-based table scan.
type IndexScanOp struct {
	Schema          string
	Table           string
	IndexName       string
	Columns         []string
	Filter          string
	FilterExpr      expr.Expr
	StartKey        []byte // For range scans (future)
	EndKey          []byte // For range scans (future)
	Cost_           Cost
	EstimatedRowCnt int64
}

func (i *IndexScanOp) Name() string { return "IndexScan" }

func (i *IndexScanOp) EstimatedCost() Cost {
	return i.Cost_
}

func (i *IndexScanOp) EstimatedRows() int64 {
	return i.EstimatedRowCnt
}

func (i *IndexScanOp) OutputColumns() []string {
	if len(i.Columns) == 0 {
		return []string{"*"}
	}
	return i.Columns
}

// Build transforms the parser AST into an executable plan.
// The catalog is required for DML statements to validate schemas and encode values.
func Build(ctx context.Context, ast parser.AST, catalog *schema.Catalog) (Plan, error) {
	switch ast.Type {
	case parser.SelectStmt:
		return buildSelect(ctx, ast, catalog)
	case parser.BeginStmt:
		return Plan{Root: &TxnOp{Action: TxnBegin}}, nil
	case parser.CommitStmt:
		return Plan{Root: &TxnOp{Action: TxnCommit}}, nil
	case parser.RollbackStmt:
		return Plan{Root: &TxnOp{Action: TxnRollback}}, nil
	case parser.CreateTableStmt:
		return buildCreateTable(ast)
	case parser.DropTableStmt:
		return buildDropTable(ast)
	case parser.CreateIndexStmt:
		return buildCreateIndex(ast, catalog)
	case parser.DropIndexStmt:
		return buildDropIndex(ast, catalog)
	case parser.InsertStmt:
		return buildInsert(ctx, ast, catalog)
	case parser.UpdateStmt:
		return buildUpdate(ctx, ast, catalog)
	case parser.DeleteStmt:
		return buildDelete(ctx, ast, catalog)
	default:
		return Plan{}, fmt.Errorf("planner: unsupported statement %s", ast.Type)
	}
}

func buildSelect(ctx context.Context, ast parser.AST, catalog *schema.Catalog) (Plan, error) {
	if catalog == nil {
		return Plan{}, fmt.Errorf("planner: catalog required for SELECT")
	}

	if ast.TableName == "" {
		return Plan{}, fmt.Errorf("planner: select missing table")
	}

	// Check if we can use an index using the structured expression
	if ast.WhereExpr != nil {
		if indexScan := tryIndexScan(ast.WhereExpr, catalog, ast.SchemaName, ast.TableName); indexScan != nil {
			// Copy filter string for backward compatibility/logging
			indexScan.Filter = ast.Where
			return Plan{Root: indexScan}, nil
		}
	}

	// Fallback to string-based check if WhereExpr is nil but Where string exists (backward compat)
	if ast.Where != "" && ast.WhereExpr == nil {
		// Simple check for "col = val"
		// We split by "=" and check if the left side is an indexed column.
		parts := strings.Split(ast.Where, "=")
		if len(parts) == 2 {
			colName := strings.TrimSpace(parts[0])

			// Check if index exists for this column
			idx, err := catalog.FindIndexForColumn(ast.SchemaName, ast.TableName, colName)
			if err == nil && idx != nil {
				// Found an index! Use IndexScan.
				op := &IndexScanOp{
					Schema:    ast.SchemaName,
					Table:     ast.TableName,
					IndexName: idx.IndexName,
					Columns:   ast.Columns,
					Filter:    ast.Where,
					// StartKey/EndKey would be derived from parts[1] in a real implementation
				}
				return Plan{Root: op}, nil
			}
		}
	}

	op := &SeqScanOp{
		Schema:     ast.SchemaName,
		Table:      ast.TableName,
		Columns:    ast.Columns,
		Filter:     ast.Where,
		FilterExpr: ast.WhereExpr,
	}
	return Plan{Root: op}, nil
}

// tryIndexScan attempts to create an IndexScanOp from a WHERE expression.
// Currently supports simple equality checks: "col = val"
func tryIndexScan(whereExpr expr.Expr, catalog *schema.Catalog, schemaName, tableName string) *IndexScanOp {
	// Look for simple BinaryExpr with OpEquals
	if binExpr, ok := whereExpr.(*expr.BinaryExpr); ok {
		if binExpr.Op == expr.OpEquals {
			// Check if left side is a column reference
			if colRef, ok := binExpr.Left.(*expr.ColumnRefExpr); ok {
				colName := colRef.Name

				// Check if index exists for this column
				idx, err := catalog.FindIndexForColumn(schemaName, tableName, colName)
				if err == nil && idx != nil {
					// Found an index!
					// TODO: We should also extract the value from binExpr.Right to set StartKey/EndKey
					// For now, we just identify the index scan opportunity.
					return &IndexScanOp{
						Schema:     schemaName,
						Table:      tableName,
						IndexName:  idx.IndexName,
						Columns:    []string{"*"}, // Placeholder, will be refined
						FilterExpr: whereExpr,
					}
				}
			}
		}
	}
	return nil
}

func buildCreateTable(ast parser.AST) (Plan, error) {
	if ast.CreateTable == nil {
		return Plan{}, fmt.Errorf("planner: CREATE TABLE spec missing")
	}

	spec := ast.CreateTable
	if len(spec.Columns) == 0 {
		return Plan{}, fmt.Errorf("planner: CREATE TABLE requires at least one column")
	}

	// Convert parser column specs to schema column defs
	columns := make([]schema.ColumnDef, len(spec.Columns))
	for i, col := range spec.Columns {
		dataType, err := schema.ParseDataType(col.Type)
		if err != nil {
			return Plan{}, fmt.Errorf("planner: %w", err)
		}
		columns[i] = schema.ColumnDef{
			Name:       col.Name,
			Type:       dataType,
			Nullable:   col.Nullable,
			PrimaryKey: col.PrimaryKey,
		}
	}

	op := &CreateTableOp{
		Schema:  spec.Schema,
		Table:   spec.Table,
		Columns: columns,
	}
	return Plan{Root: op}, nil
}

func buildDropTable(ast parser.AST) (Plan, error) {
	if ast.DropTable == nil {
		return Plan{}, fmt.Errorf("planner: DROP TABLE spec missing")
	}

	spec := ast.DropTable
	op := &DropTableOp{
		Schema: spec.Schema,
		Table:  spec.Table,
	}
	return Plan{Root: op}, nil
}

func buildCreateIndex(ast parser.AST, catalog *schema.Catalog) (Plan, error) {
	if ast.CreateIndex == nil {
		return Plan{}, fmt.Errorf("planner: CREATE INDEX spec missing")
	}
	// TODO: Catalog check is optional for pure DDL build, but good for validation if we wanted to check table existence early.
	// For now, we'll let executor handle table existence check to keep planner simple.

	spec := ast.CreateIndex
	op := &CreateIndexOp{
		Schema:    spec.Schema,
		Table:     spec.Table,
		IndexName: spec.IndexName,
		Columns:   spec.Columns,
		Unique:    spec.Unique,
	}
	return Plan{Root: op}, nil
}

func buildDropIndex(ast parser.AST, catalog *schema.Catalog) (Plan, error) {
	if ast.DropIndex == nil {
		return Plan{}, fmt.Errorf("planner: DROP INDEX spec missing")
	}

	spec := ast.DropIndex
	op := &DropIndexOp{
		Schema:    spec.Schema,
		Table:     spec.Table,
		IndexName: spec.IndexName,
	}
	return Plan{Root: op}, nil
}

func buildInsert(ctx context.Context, ast parser.AST, catalog *schema.Catalog) (Plan, error) {
	if ast.Insert == nil {
		return Plan{}, fmt.Errorf("planner: INSERT spec missing")
	}
	if catalog == nil {
		return Plan{}, fmt.Errorf("planner: catalog required for INSERT")
	}

	spec := ast.Insert
	logging.DebugContext(ctx, "[Planner] Building INSERT plan for table %s.%s", spec.Schema, spec.Table)

	// Get table schema from catalog
	logging.DebugContext(ctx, "[Planner] Validating table schema for %s.%s", spec.Schema, spec.Table)
	tableDef, err := catalog.GetTable(spec.Schema, spec.Table)
	if err != nil {
		return Plan{}, fmt.Errorf("planner: table %s.%s not found: %w", spec.Schema, spec.Table, err)
	}
	logging.DebugContext(ctx, "[Planner] Table has %d columns defined", len(tableDef.Columns))

	// Determine columns (use all if not specified)
	columns := spec.Columns
	if len(columns) == 0 {
		for _, col := range tableDef.Columns {
			columns = append(columns, col.Name)
		}
	}

	// Encode all rows
	logging.DebugContext(ctx, "[Planner] Encoding %d rows for INSERT", len(spec.Values))
	var batchEncodedValues [][][]byte

	for rowIdx, rowValues := range spec.Values {
		// Validate column count
		if len(rowValues) != len(columns) {
			return Plan{}, fmt.Errorf("planner: INSERT value count (%d) for row %d doesn't match column count (%d)",
				len(rowValues), rowIdx, len(columns))
		}

		encodedRow := make([][]byte, len(columns))
		for i, colName := range columns {
			// Find column in table def
			var colDef *schema.ColumnDef
			for j := range tableDef.Columns {
				if tableDef.Columns[j].Name == colName {
					colDef = &tableDef.Columns[j]
					break
				}
			}
			if colDef == nil {
				return Plan{}, fmt.Errorf("planner: column %q not found in table %s.%s",
					colName, spec.Schema, spec.Table)
			}

			// Encode value using type system
			value := rowValues[i]
			// Strip quotes from string literals
			if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') {
				value = value[1 : len(value)-1]
			}

			encoded, err := colDef.Type.Encode(value)
			if err != nil {
				return Plan{}, fmt.Errorf("planner: cannot encode value %q for column %s: %w",
					rowValues[i], colName, err)
			}
			encodedRow[i] = encoded
		}
		batchEncodedValues = append(batchEncodedValues, encodedRow)
	}

	logging.DebugContext(ctx, "[Planner] Successfully encoded all values")

	op := &InsertOp{
		Schema:  spec.Schema,
		Table:   spec.Table,
		Columns: columns,
		Values:  batchEncodedValues,
	}
	logging.DebugContext(ctx, "[Planner] Created InsertOp for %s.%s with %d rows", spec.Schema, spec.Table, len(batchEncodedValues))
	return Plan{Root: op}, nil
}

func buildUpdate(ctx context.Context, ast parser.AST, catalog *schema.Catalog) (Plan, error) {
	if ast.Update == nil {
		return Plan{}, fmt.Errorf("planner: UPDATE spec missing")
	}
	if catalog == nil {
		return Plan{}, fmt.Errorf("planner: catalog required for UPDATE")
	}

	spec := ast.Update

	// Get table schema
	tableDef, err := catalog.GetTable(spec.Schema, spec.Table)
	if err != nil {
		return Plan{}, fmt.Errorf("planner: table %s.%s not found: %w", spec.Schema, spec.Table, err)
	}

	// Validate and encode SET clauses
	encodedSetClauses := make(map[string][]byte)
	for colName, valueStr := range spec.SetClauses {
		var colDef *schema.ColumnDef
		for i := range tableDef.Columns {
			if tableDef.Columns[i].Name == colName {
				colDef = &tableDef.Columns[i]
				break
			}
		}
		if colDef == nil {
			return Plan{}, fmt.Errorf("planner: column %q not found in table %s.%s",
				colName, spec.Schema, spec.Table)
		}

		// Strip quotes
		value := valueStr
		if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') {
			value = value[1 : len(value)-1]
		}

		encoded, err := colDef.Type.Encode(value)
		if err != nil {
			return Plan{}, fmt.Errorf("planner: cannot encode value %q for column %s: %w",
				valueStr, colName, err)
		}
		encodedSetClauses[colName] = encoded
	}

	op := &UpdateOp{
		Schema:     spec.Schema,
		Table:      spec.Table,
		SetClauses: encodedSetClauses,
		Filter:     spec.Where,
		FilterExpr: ast.WhereExpr,
	}
	return Plan{Root: op}, nil
}

func buildDelete(ctx context.Context, ast parser.AST, catalog *schema.Catalog) (Plan, error) {
	if ast.Delete == nil {
		return Plan{}, fmt.Errorf("planner: DELETE spec missing")
	}
	if catalog == nil {
		return Plan{}, fmt.Errorf("planner: catalog required for DELETE")
	}

	spec := ast.Delete

	// Validate table exists
	_, err := catalog.GetTable(spec.Schema, spec.Table)
	if err != nil {
		return Plan{}, fmt.Errorf("planner: table %s.%s not found: %w", spec.Schema, spec.Table, err)
	}

	op := &DeleteOp{
		Schema:     spec.Schema,
		Table:      spec.Table,
		Filter:     spec.Where,
		FilterExpr: ast.WhereExpr,
	}
	return Plan{Root: op}, nil
}
