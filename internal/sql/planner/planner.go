package planner

import (
	"fmt"
	"strings"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/parser"
)

// Plan describes a physical execution plan produced from an AST.
type Plan struct {
	Root Operator
}

// Operator is the core interface implemented by executors.
type Operator interface {
	Name() string
}

// SeqScanOp is a placeholder physical operator for table scans.
type SeqScanOp struct {
	Schema  string
	Table   string
	Columns []string
	Filter  string
}

func (s *SeqScanOp) Name() string { return "SeqScan" }

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

// CreateTableOp represents CREATE TABLE execution.
type CreateTableOp struct {
	Schema  string
	Table   string
	Columns []schema.ColumnDef
}

func (c *CreateTableOp) Name() string { return "CreateTable" }

// DropTableOp represents DROP TABLE execution.
type DropTableOp struct {
	Schema string
	Table  string
}

func (d *DropTableOp) Name() string { return "DropTable" }

// InsertOp represents INSERT execution.
type InsertOp struct {
	Schema  string
	Table   string
	Columns []string // Column names (validated against schema)
	Values  [][]byte // Encoded values (typed according to schema)
}

func (i *InsertOp) Name() string { return "Insert" }

// UpdateOp represents UPDATE execution.
type UpdateOp struct {
	Schema     string
	Table      string
	SetClauses map[string][]byte // column -> encoded value
	Filter     string            // WHERE clause (string for MVP)
}

func (u *UpdateOp) Name() string { return "Update" }

// DeleteOp represents DELETE execution.
type DeleteOp struct {
	Schema string
	Table  string
	Filter string // WHERE clause
}

func (de *DeleteOp) Name() string { return "Delete" }

// CreateIndexOp represents CREATE INDEX execution.
type CreateIndexOp struct {
	Schema    string
	Table     string
	IndexName string
	Columns   []string
	Unique    bool
}

func (c *CreateIndexOp) Name() string { return "CreateIndex" }

// DropIndexOp represents DROP INDEX execution.
type DropIndexOp struct {
	Schema    string
	Table     string
	IndexName string
}

func (d *DropIndexOp) Name() string { return "DropIndex" }

// IndexScanOp represents an index-based table scan.
type IndexScanOp struct {
	Schema    string
	Table     string
	IndexName string
	Columns   []string
	Filter    string
	StartKey  []byte // For range scans (future)
	EndKey    []byte // For range scans (future)
}

func (i *IndexScanOp) Name() string { return "IndexScan" }

// Build transforms the parser AST into an executable plan.
// The catalog is required for DML statements to validate schemas and encode values.
func Build(ast parser.AST, catalog *schema.Catalog) (Plan, error) {
	switch ast.Type {
	case parser.SelectStmt:
		return buildSelect(ast, catalog)
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
		return buildInsert(ast, catalog)
	case parser.UpdateStmt:
		return buildUpdate(ast, catalog)
	case parser.DeleteStmt:
		return buildDelete(ast, catalog)
	default:
		return Plan{}, fmt.Errorf("planner: unsupported statement %s", ast.Type)
	}
}

func buildSelect(ast parser.AST, catalog *schema.Catalog) (Plan, error) {
	if ast.TableName == "" {
		return Plan{}, fmt.Errorf("planner: select missing table")
	}

	// Basic heuristic: if catalog is available and we have a WHERE clause,
	// check if we can use an index.
	if catalog != nil && ast.Where != "" {
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
		Schema:  ast.SchemaName,
		Table:   ast.TableName,
		Columns: ast.Columns,
		Filter:  ast.Where,
	}
	return Plan{Root: op}, nil
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

func buildInsert(ast parser.AST, catalog *schema.Catalog) (Plan, error) {
	if ast.Insert == nil {
		return Plan{}, fmt.Errorf("planner: INSERT spec missing")
	}
	if catalog == nil {
		return Plan{}, fmt.Errorf("planner: catalog required for INSERT")
	}

	spec := ast.Insert

	// Get table schema from catalog
	tableDef, err := catalog.GetTable(spec.Schema, spec.Table)
	if err != nil {
		return Plan{}, fmt.Errorf("planner: table %s.%s not found: %w", spec.Schema, spec.Table, err)
	}

	// Determine columns (use all if not specified)
	columns := spec.Columns
	if len(columns) == 0 {
		for _, col := range tableDef.Columns {
			columns = append(columns, col.Name)
		}
	}

	// Validate column count
	if len(spec.Values) != len(columns) {
		return Plan{}, fmt.Errorf("planner: INSERT value count (%d) doesn't match column count (%d)",
			len(spec.Values), len(columns))
	}

	// Validate columns exist and encode values
	encodedValues := make([][]byte, len(columns))
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
		value := spec.Values[i]
		// Strip quotes from string literals
		if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') {
			value = value[1 : len(value)-1]
		}

		encoded, err := colDef.Type.Encode(value)
		if err != nil {
			return Plan{}, fmt.Errorf("planner: cannot encode value %q for column %s: %w",
				spec.Values[i], colName, err)
		}
		encodedValues[i] = encoded
	}

	op := &InsertOp{
		Schema:  spec.Schema,
		Table:   spec.Table,
		Columns: columns,
		Values:  encodedValues,
	}
	return Plan{Root: op}, nil
}

func buildUpdate(ast parser.AST, catalog *schema.Catalog) (Plan, error) {
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
	}
	return Plan{Root: op}, nil
}

func buildDelete(ast parser.AST, catalog *schema.Catalog) (Plan, error) {
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
		Schema: spec.Schema,
		Table:  spec.Table,
		Filter: spec.Where,
	}
	return Plan{Root: op}, nil
}
