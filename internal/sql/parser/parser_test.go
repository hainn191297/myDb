package parser

import (
	"context"
	"testing"
)

func TestParseTxnStatements(t *testing.T) {
	tests := []struct {
		sql  string
		want StatementType
	}{
		{"BEGIN", BeginStmt},
		{"begin transaction", BeginStmt},
		{"COMMIT", CommitStmt},
		{"rollback", RollbackStmt},
	}
	for _, tt := range tests {
		ast, err := Parse(context.Background(), tt.sql)
		if err != nil {
			t.Fatalf("Parse(%q) unexpected error: %v", tt.sql, err)
		}
		if ast.Type != tt.want {
			t.Fatalf("Parse(%q) type mismatch: got %s want %s", tt.sql, ast.Type, tt.want)
		}
	}
}

func TestParseSelect(t *testing.T) {
	sql := "SELECT id, name FROM public.users WHERE id = 1"
	ast, err := Parse(context.Background(), sql)
	if err != nil {
		t.Fatalf("Parse SELECT: %v", err)
	}
	if ast.Type != SelectStmt {
		t.Fatalf("expected SELECT stmt type, got %s", ast.Type)
	}
	if ast.SchemaName != "public" || ast.TableName != "users" {
		t.Fatalf("schema/table mismatch: %s.%s", ast.SchemaName, ast.TableName)
	}
	if len(ast.Columns) != 2 || ast.Columns[0] != "id" || ast.Columns[1] != "name" {
		t.Fatalf("columns mismatch: %#v", ast.Columns)
	}
	if ast.Where != "id = 1" {
		t.Fatalf("where mismatch: %q", ast.Where)
	}
}

func TestParseErrors(t *testing.T) {
	_, err := Parse(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for empty statement")
	}

	_, err = Parse(context.Background(), "SELECT FROM")
	if err == nil {
		t.Fatalf("expected error for invalid select")
	}

	_, err = Parse(context.Background(), "UNSUPPORTED")
	if err == nil {
		t.Fatalf("expected error for unsupported statement")
	}
}

func TestParseSelectDefaultSchema(t *testing.T) {
	ast, err := Parse(context.Background(), "SELECT * FROM users")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.SchemaName != "public" || ast.TableName != "users" {
		t.Fatalf("expected default schema public, got %s.%s", ast.SchemaName, ast.TableName)
	}
}

func TestParseCreateTable(t *testing.T) {
	sql := "CREATE TABLE users (id INT, name TEXT, active BOOL)"
	ast, err := Parse(context.Background(), sql)

	// logx.Info(ast)
	if err != nil {
		t.Fatalf("Parse CREATE TABLE: %v", err)
	}
	if ast.Type != CreateTableStmt {
		t.Fatalf("expected CREATE_TABLE type, got %s", ast.Type)
	}
	if ast.CreateTable == nil {
		t.Fatal("CreateTable spec is nil")
	}

	spec := ast.CreateTable
	if spec.Schema != "public" || spec.Table != "users" {
		t.Errorf("table mismatch: got %s.%s, want public.users", spec.Schema, spec.Table)
	}
	if len(spec.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(spec.Columns))
	}

	// Verify column definitions
	if spec.Columns[0].Name != "id" || spec.Columns[0].Type != "INT" {
		t.Errorf("column 0 mismatch: %+v", spec.Columns[0])
	}
	if spec.Columns[1].Name != "name" || spec.Columns[1].Type != "TEXT" {
		t.Errorf("column 1 mismatch: %+v", spec.Columns[1])
	}
	if spec.Columns[2].Name != "active" || spec.Columns[2].Type != "BOOL" {
		t.Errorf("column 2 mismatch: %+v", spec.Columns[2])
	}
}

func TestParseCreateTableWithSchema(t *testing.T) {
	sql := "CREATE TABLE public.orders (order_id INT, total FLOAT)"
	ast, err := Parse(context.Background(), sql)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.CreateTable.Schema != "public" || ast.CreateTable.Table != "orders" {
		t.Errorf("table mismatch: %s.%s", ast.CreateTable.Schema, ast.CreateTable.Table)
	}
}

func TestParseCreateTableNotNull(t *testing.T) {
	sql := "CREATE TABLE users (id INT NOT NULL, name TEXT)"
	ast, err := Parse(context.Background(), sql)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.CreateTable.Columns[0].Nullable {
		t.Error("expected id column to be NOT NULL")
	}
	if !ast.CreateTable.Columns[1].Nullable {
		t.Error("expected name column to be nullable")
	}
}

func TestParseCreateTableErrors(t *testing.T) {
	tests := []string{
		"CREATE TABLE",               // Missing table name
		"CREATE TABLE users",         // Missing column definitions
		"CREATE TABLE users ()",      // Empty column list
		"CREATE TABLE users (id)",    // Incomplete column def
		"CREATE TABLE users (id INT", // Missing closing paren
	}

	for _, sql := range tests {
		_, err := Parse(context.Background(), sql)
		if err == nil {
			t.Errorf("expected error for %q, got nil", sql)
		}
	}
}

func TestParseDropTable(t *testing.T) {
	sql := "DROP TABLE users"
	ast, err := Parse(context.Background(), sql)
	if err != nil {
		t.Fatalf("Parse DROP TABLE: %v", err)
	}
	if ast.Type != DropTableStmt {
		t.Fatalf("expected DROP_TABLE type, got %s", ast.Type)
	}
	if ast.DropTable == nil {
		t.Fatal("DropTable spec is nil")
	}
	if ast.DropTable.Schema != "public" || ast.DropTable.Table != "users" {
		t.Errorf("table mismatch: %s.%s", ast.DropTable.Schema, ast.DropTable.Table)
	}
}

func TestParseDropTableWithSchema(t *testing.T) {
	sql := "DROP TABLE public.orders"
	ast, err := Parse(context.Background(), sql)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.DropTable.Schema != "public" || ast.DropTable.Table != "orders" {
		t.Errorf("table mismatch: %s.%s", ast.DropTable.Schema, ast.DropTable.Table)
	}
}

func TestParseDropTableErrors(t *testing.T) {
	_, err := Parse(context.Background(), "DROP TABLE")
	if err == nil {
		t.Error("expected error for DROP TABLE without table name")
	}
}
