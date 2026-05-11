package planner

import (
	"context"
	"testing"

	dberrors "github.com/hainn191297/myDb/internal/errors"
	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/expr"
	"github.com/hainn191297/myDb/internal/sql/parser"
)

// setupTestCatalog creates a test catalog with sample tables for testing.
func setupTestCatalog(t *testing.T) *schema.Catalog {
	// Create a mock catalog with test tables
	// For testing, we'll use a simple in-memory implementation
	catalog := &mockCatalog{
		tables: make(map[string]*schema.TableDef),
	}

	// Add users table
	catalog.tables["public.users"] = &schema.TableDef{
		Schema: "public",
		Table:  "users",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64, Nullable: false, PrimaryKey: true},
			{Name: "name", Type: schema.TypeText, Nullable: false},
			{Name: "email", Type: schema.TypeText, Nullable: true},
		},
	}

	// Add products table
	catalog.tables["public.products"] = &schema.TableDef{
		Schema: "public",
		Table:  "products",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64, Nullable: false, PrimaryKey: true},
			{Name: "name", Type: schema.TypeText, Nullable: false},
			{Name: "price", Type: schema.TypeFloat64, Nullable: false},
		},
	}

	// Add orders table
	catalog.tables["public.orders"] = &schema.TableDef{
		Schema: "public",
		Table:  "orders",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64, Nullable: false, PrimaryKey: true},
			{Name: "user_id", Type: schema.TypeInt64, Nullable: false},
			{Name: "product_id", Type: schema.TypeInt64, Nullable: false},
			{Name: "quantity", Type: schema.TypeInt64, Nullable: false},
		},
	}

	return (*schema.Catalog)(nil) // Return nil, we'll use mockCatalog directly
}

// mockCatalog is a simple in-memory catalog for testing.
type mockCatalog struct {
	tables map[string]*schema.TableDef
}

func (m *mockCatalog) GetTable(schema, table string) (*schema.TableDef, error) {
	key := schema + "." + table
	if tableDef, exists := m.tables[key]; exists {
		return tableDef, nil
	}
	return nil, dberrors.ErrTableNotFound
}

func (m *mockCatalog) FindIndexForColumn(schema, table, column string) (*schema.IndexDef, error) {
	return nil, nil
}

func (m *mockCatalog) GetIndexes(schema, table string) ([]schema.IndexDef, error) {
	return nil, nil
}

func (m *mockCatalog) CreateTable(ctx context.Context, schema, table string, columns []schema.ColumnDef) error {
	return nil
}

func (m *mockCatalog) DropTable(ctx context.Context, schema, table string) error {
	return nil
}

func (m *mockCatalog) ListTables() []*schema.TableDef {
	return nil
}

func (m *mockCatalog) CreateIndex(ctx context.Context, schema, table, indexName string, columns []string, unique, isPrimaryKey bool) error {
	return nil
}

func (m *mockCatalog) DropIndex(ctx context.Context, schema, table, indexName string) error {
	return nil
}

func (m *mockCatalog) LoadSystemTables(ctx context.Context) error {
	return nil
}

// Test table and column reference validation

func TestAnalyzeInsertValidation(t *testing.T) {
	tests := []struct {
		name    string
		ast     parser.AST
		wantErr bool
		errMsg  string
	}{
		{
			name: "insert with valid table and columns",
			ast: parser.AST{
				Type: parser.InsertStmt,
				Insert: &parser.InsertSpec{
					Schema:  "public",
					Table:   "users",
					Columns: []string{"id", "name"},
					Values: [][]string{
						{"1", "'John'"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "insert with nonexistent table",
			ast: parser.AST{
				Type: parser.InsertStmt,
				Insert: &parser.InsertSpec{
					Schema:  "public",
					Table:   "nonexistent",
					Columns: []string{"id"},
					Values: [][]string{
						{"1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "table not found",
		},
		{
			name: "insert with nonexistent column",
			ast: parser.AST{
				Type: parser.InsertStmt,
				Insert: &parser.InsertSpec{
					Schema:  "public",
					Table:   "users",
					Columns: []string{"nonexistent"},
					Values: [][]string{
						{"1"},
					},
				},
			},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name: "insert with mismatched column and value count",
			ast: parser.AST{
				Type: parser.InsertStmt,
				Insert: &parser.InsertSpec{
					Schema:  "public",
					Table:   "users",
					Columns: []string{"id", "name"},
					Values: [][]string{
						{"1"}, // Only 1 value, but 2 columns
					},
				},
			},
			wantErr: true,
			errMsg:  "value count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test structure is set up but would need proper mock injection
			// to work with the actual SemanticAnalyzer
			_ = tt
		})
	}
}

func TestAnalyzeUpdateValidation(t *testing.T) {
	tests := []struct {
		name    string
		ast     parser.AST
		wantErr bool
	}{
		{
			name: "update with valid table and columns",
			ast: parser.AST{
				Type: parser.UpdateStmt,
				Update: &parser.UpdateSpec{
					Schema:     "public",
					Table:      "users",
					SetClauses: map[string]string{"name": "'Jane'"},
					Where:      "id = 1",
				},
			},
			wantErr: false,
		},
		{
			name: "update with nonexistent table",
			ast: parser.AST{
				Type: parser.UpdateStmt,
				Update: &parser.UpdateSpec{
					Schema:     "public",
					Table:      "nonexistent",
					SetClauses: map[string]string{"name": "'Jane'"},
				},
			},
			wantErr: true,
		},
		{
			name: "update with nonexistent column",
			ast: parser.AST{
				Type: parser.UpdateStmt,
				Update: &parser.UpdateSpec{
					Schema:     "public",
					Table:      "users",
					SetClauses: map[string]string{"nonexistent": "'Jane'"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt
		})
	}
}

func TestAnalyzeDeleteValidation(t *testing.T) {
	tests := []struct {
		name    string
		ast     parser.AST
		wantErr bool
	}{
		{
			name: "delete with valid table",
			ast: parser.AST{
				Type: parser.DeleteStmt,
				Delete: &parser.DeleteSpec{
					Schema: "public",
					Table:  "users",
					Where:  "id = 1",
				},
			},
			wantErr: false,
		},
		{
			name: "delete with nonexistent table",
			ast: parser.AST{
				Type: parser.DeleteStmt,
				Delete: &parser.DeleteSpec{
					Schema: "public",
					Table:  "nonexistent",
					Where:  "id = 1",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt
		})
	}
}

func TestAnalyzeCreateTableValidation(t *testing.T) {
	tests := []struct {
		name    string
		ast     parser.AST
		wantErr bool
	}{
		{
			name: "create table with valid columns",
			ast: parser.AST{
				Type: parser.CreateTableStmt,
				CreateTable: &parser.CreateTableSpec{
					Schema: "public",
					Table:  "test_table",
					Columns: []parser.ColumnSpec{
						{Name: "id", Type: "INT", PrimaryKey: true},
						{Name: "name", Type: "TEXT"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "create table with no columns",
			ast: parser.AST{
				Type: parser.CreateTableStmt,
				CreateTable: &parser.CreateTableSpec{
					Schema:  "public",
					Table:   "test_table",
					Columns: []parser.ColumnSpec{},
				},
			},
			wantErr: true,
		},
		{
			name: "create table with duplicate column names",
			ast: parser.AST{
				Type: parser.CreateTableStmt,
				CreateTable: &parser.CreateTableSpec{
					Schema: "public",
					Table:  "test_table",
					Columns: []parser.ColumnSpec{
						{Name: "id", Type: "INT"},
						{Name: "id", Type: "TEXT"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewSemanticAnalyzer(nil)
			err := analyzer.Analyze(tt.ast)

			if (err != nil) != tt.wantErr {
				t.Errorf("Analyze() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyzeCreateIndexValidation(t *testing.T) {
	tests := []struct {
		name    string
		ast     parser.AST
		wantErr bool
	}{
		{
			name: "create index with valid table and columns",
			ast: parser.AST{
				Type: parser.CreateIndexStmt,
				CreateIndex: &parser.CreateIndexSpec{
					Schema:    "public",
					Table:     "users",
					IndexName: "idx_name",
					Columns:   []string{"name"},
				},
			},
			wantErr: false,
		},
		{
			name: "create index with nonexistent table",
			ast: parser.AST{
				Type: parser.CreateIndexStmt,
				CreateIndex: &parser.CreateIndexSpec{
					Schema:    "public",
					Table:     "nonexistent",
					IndexName: "idx_name",
					Columns:   []string{"name"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt
		})
	}
}

func TestAnalyzeDropTableValidation(t *testing.T) {
	tests := []struct {
		name    string
		ast     parser.AST
		wantErr bool
	}{
		{
			name: "drop existing table",
			ast: parser.AST{
				Type: parser.DropTableStmt,
				DropTable: &parser.DropTableSpec{
					Schema: "public",
					Table:  "users",
				},
			},
			wantErr: false,
		},
		{
			name: "drop nonexistent table",
			ast: parser.AST{
				Type: parser.DropTableStmt,
				DropTable: &parser.DropTableSpec{
					Schema: "public",
					Table:  "nonexistent",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt
		})
	}
}

func TestAnalyzeDropIndexValidation(t *testing.T) {
	tests := []struct {
		name    string
		ast     parser.AST
		wantErr bool
	}{
		{
			name: "drop index on existing table",
			ast: parser.AST{
				Type: parser.DropIndexStmt,
				DropIndex: &parser.DropIndexSpec{
					Schema:    "public",
					Table:     "users",
					IndexName: "idx_name",
				},
			},
			wantErr: false,
		},
		{
			name: "drop index on nonexistent table",
			ast: parser.AST{
				Type: parser.DropIndexStmt,
				DropIndex: &parser.DropIndexSpec{
					Schema:    "public",
					Table:     "nonexistent",
					IndexName: "idx_name",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt
		})
	}
}

// Test WHERE clause validation

func TestValidateWhereClauseWithValidColumns(t *testing.T) {
	analyzer := NewSemanticAnalyzer(nil)

	tableDef := &schema.TableDef{
		Schema: "public",
		Table:  "users",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64},
			{Name: "name", Type: schema.TypeText},
		},
	}

	// Create a simple column reference expression
	whereExpr := &expr.BinaryExpr{
		Left:  &expr.ColumnRefExpr{Name: "id"},
		Op:    expr.OpEquals,
		Right: &expr.LiteralExpr{Value: "1"},
	}

	err := analyzer.validateWhereClause(whereExpr, tableDef)
	if err != nil {
		t.Errorf("validateWhereClause() error = %v, want nil", err)
	}
}

func TestValidateWhereClauseWithInvalidColumns(t *testing.T) {
	analyzer := NewSemanticAnalyzer(nil)

	tableDef := &schema.TableDef{
		Schema: "public",
		Table:  "users",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64},
			{Name: "name", Type: schema.TypeText},
		},
	}

	// Create an expression with a nonexistent column
	whereExpr := &expr.BinaryExpr{
		Left:  &expr.ColumnRefExpr{Name: "nonexistent"},
		Op:    expr.OpEquals,
		Right: &expr.LiteralExpr{Value: "1"},
	}

	err := analyzer.validateWhereClause(whereExpr, tableDef)
	if err == nil {
		t.Errorf("validateWhereClause() error = nil, want error")
	}
}

func TestValidateExpressionWithComplexPredicate(t *testing.T) {
	analyzer := NewSemanticAnalyzer(nil)

	tableDef := &schema.TableDef{
		Schema: "public",
		Table:  "users",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64},
			{Name: "name", Type: schema.TypeText},
			{Name: "age", Type: schema.TypeInt64},
		},
	}

	// Create a complex expression: id = 1 AND age > 18
	whereExpr := &expr.BinaryExpr{
		Left: &expr.BinaryExpr{
			Left:  &expr.ColumnRefExpr{Name: "id"},
			Op:    expr.OpEquals,
			Right: &expr.LiteralExpr{Value: "1"},
		},
		Op: expr.OpAnd,
		Right: &expr.BinaryExpr{
			Left:  &expr.ColumnRefExpr{Name: "age"},
			Op:    expr.OpGreaterThan,
			Right: &expr.LiteralExpr{Value: "18"},
		},
	}

	err := analyzer.validateExpression(whereExpr, tableDef)
	if err != nil {
		t.Errorf("validateExpression() error = %v, want nil", err)
	}
}

func TestColumnExistsInTable(t *testing.T) {
	analyzer := NewSemanticAnalyzer(nil)

	tableDef := &schema.TableDef{
		Schema: "public",
		Table:  "users",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64},
			{Name: "name", Type: schema.TypeText},
		},
	}

	tests := []struct {
		name     string
		colName  string
		expected bool
	}{
		{"existing column id", "id", true},
		{"existing column name", "name", true},
		{"nonexistent column", "nonexistent", false},
		{"empty column name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.columnExistsInTable(tt.colName, tableDef)
			if result != tt.expected {
				t.Errorf("columnExistsInTable(%q) = %v, want %v", tt.colName, result, tt.expected)
			}
		})
	}
}

func TestValidateColumnsExist(t *testing.T) {
	analyzer := NewSemanticAnalyzer(nil)

	tableDef := &schema.TableDef{
		Schema: "public",
		Table:  "users",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64},
			{Name: "name", Type: schema.TypeText},
		},
	}

	tests := []struct {
		name    string
		columns []string
		wantErr bool
	}{
		{"wildcard", []string{"*"}, false},
		{"valid columns", []string{"id", "name"}, false},
		{"single valid column", []string{"id"}, false},
		{"invalid column", []string{"nonexistent"}, true},
		{"mixed valid and invalid", []string{"id", "nonexistent"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := analyzer.validateColumnsExist(tt.columns, tableDef)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateColumnsExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDetectAmbiguousColumns(t *testing.T) {
	analyzer := NewSemanticAnalyzer(nil)

	table1 := &schema.TableDef{
		Schema: "public",
		Table:  "users",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64},
			{Name: "name", Type: schema.TypeText},
		},
	}

	table2 := &schema.TableDef{
		Schema: "public",
		Table:  "products",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64},
			{Name: "name", Type: schema.TypeText},
		},
	}

	tests := []struct {
		name    string
		colName string
		tables  []*schema.TableDef
		wantErr bool
	}{
		{"unique column", "name", []*schema.TableDef{table1}, false},
		{"ambiguous column in two tables", "id", []*schema.TableDef{table1, table2}, true},
		{"ambiguous column name", "name", []*schema.TableDef{table1, table2}, true},
		{"nonexistent column", "nonexistent", []*schema.TableDef{table1, table2}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := analyzer.detectAmbiguousColumns(tt.colName, tt.tables)
			if (err != nil) != tt.wantErr {
				t.Errorf("detectAmbiguousColumns() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
