package parser

import (
	"testing"
)

func TestParseInsert(t *testing.T) {
	sql := "INSERT INTO users VALUES (1, 'alice', true)"
	ast, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse INSERT: %v", err)
	}
	if ast.Type != InsertStmt {
		t.Fatalf("expected INSERT type, got %s", ast.Type)
	}
	if ast.Insert == nil {
		t.Fatal("Insert spec is nil")
	}

	spec := ast.Insert
	if spec.Schema != "public" || spec.Table != "users" {
		t.Errorf("table mismatch: %s.%s", spec.Schema, spec.Table)
	}
	if len(spec.Values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(spec.Values))
	}
	if spec.Values[0] != "1" || spec.Values[1] != "'alice'" || spec.Values[2] != "true" {
		t.Errorf("values mismatch: %v", spec.Values)
	}
}

func TestParseInsertWithColumns(t *testing.T) {
	sql := "INSERT INTO users (id, name) VALUES (1, 'alice')"
	ast, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(ast.Insert.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(ast.Insert.Columns))
	}
	if ast.Insert.Columns[0] != "id" || ast.Insert.Columns[1] != "name" {
		t.Errorf("columns mismatch: %v", ast.Insert.Columns)
	}
}

func TestParseInsertWithSchema(t *testing.T) {
	sql := "INSERT INTO public.users VALUES (1)"
	ast, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.Insert.Schema != "public" || ast.Insert.Table != "users" {
		t.Errorf("table mismatch: %s.%s", ast.Insert.Schema, ast.Insert.Table)
	}
}

func TestParseUpdate(t *testing.T) {
	sql := "UPDATE users SET name = 'bob', active = false WHERE id = 1"
	ast, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse UPDATE: %v", err)
	}
	if ast.Type != UpdateStmt {
		t.Fatalf("expected UPDATE type, got %s", ast.Type)
	}
	if ast.Update == nil {
		t.Fatal("Update spec is nil")
	}

	spec := ast.Update
	if spec.Schema != "public" || spec.Table != "users" {
		t.Errorf("table mismatch: %s.%s", spec.Schema, spec.Table)
	}
	if len(spec.SetClauses) != 2 {
		t.Fatalf("expected 2 set clauses, got %d", len(spec.SetClauses))
	}
	if spec.SetClauses["name"] != "'bob'" {
		t.Errorf("set clause mismatch for name: %s", spec.SetClauses["name"])
	}
	if spec.Where != "id = 1" {
		t.Errorf("where clause mismatch: %s", spec.Where)
	}
}

func TestParseUpdateNoWhere(t *testing.T) {
	sql := "UPDATE users SET name = 'bob'"
	ast, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.Update.Where != "" {
		t.Errorf("expected empty WHERE, got %s", ast.Update.Where)
	}
}

func TestParseDelete(t *testing.T) {
	sql := "DELETE FROM users WHERE id = 1"
	ast, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse DELETE: %v", err)
	}
	if ast.Type != DeleteStmt {
		t.Fatalf("expected DELETE type, got %s", ast.Type)
	}
	if ast.Delete == nil {
		t.Fatal("Delete spec is nil")
	}

	spec := ast.Delete
	if spec.Schema != "public" || spec.Table != "users" {
		t.Errorf("table mismatch: %s.%s", spec.Schema, spec.Table)
	}
	if spec.Where != "id = 1" {
		t.Errorf("where clause mismatch: %s", spec.Where)
	}
}

func TestParseDeleteNoWhere(t *testing.T) {
	sql := "DELETE FROM users"
	ast, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.Delete.Where != "" {
		t.Errorf("expected empty WHERE, got %s", ast.Delete.Where)
	}
}

func TestParseDMLErrors(t *testing.T) {
	tests := []string{
		"INSERT INTO",              // Missing table
		"INSERT INTO users",        // Missing VALUES
		"INSERT INTO users VALUES", // Missing parentheses
		"UPDATE",                   // Missing table
		"UPDATE users",             // Missing SET
		"DELETE",                   // Missing FROM
	}

	for _, sql := range tests {
		_, err := Parse(sql)
		if err == nil {
			t.Errorf("expected error for %q, got nil", sql)
		}
	}
}
