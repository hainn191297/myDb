package parser

import (
	"context"
	"testing"
)

func TestParseCreateIndex(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    *CreateIndexSpec
		wantErr bool
	}{
		{
			name: "Basic Create Index",
			sql:  "CREATE INDEX idx_email ON users (email)",
			want: &CreateIndexSpec{
				Schema:    "public",
				Table:     "users",
				IndexName: "idx_email",
				Columns:   []string{"email"},
				Unique:    false,
			},
			wantErr: false,
		},
		{
			name: "Create Unique Index",
			sql:  "CREATE UNIQUE INDEX idx_email ON users (email)",
			want: &CreateIndexSpec{
				Schema:    "public",
				Table:     "users",
				IndexName: "idx_email",
				Columns:   []string{"email"},
				Unique:    true,
			},
			wantErr: false,
		},
		{
			name: "Create Index with Schema",
			sql:  "CREATE INDEX idx_email ON public.users (email)",
			want: &CreateIndexSpec{
				Schema:    "public",
				Table:     "users",
				IndexName: "idx_email",
				Columns:   []string{"email"},
				Unique:    false,
			},
			wantErr: false,
		},
		{
			name:    "Missing ON",
			sql:     "CREATE INDEX idx_email users (email)",
			wantErr: true,
		},
		{
			name:    "Missing Columns",
			sql:     "CREATE INDEX idx_email ON users",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := Parse(context.Background(), tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if ast.Type != CreateIndexStmt {
					t.Errorf("expected CreateIndexStmt, got %v", ast.Type)
				}
				got := ast.CreateIndex
				if got.Schema != tt.want.Schema {
					t.Errorf("Schema = %v, want %v", got.Schema, tt.want.Schema)
				}
				if got.Table != tt.want.Table {
					t.Errorf("Table = %v, want %v", got.Table, tt.want.Table)
				}
				if got.IndexName != tt.want.IndexName {
					t.Errorf("IndexName = %v, want %v", got.IndexName, tt.want.IndexName)
				}
				if got.Unique != tt.want.Unique {
					t.Errorf("Unique = %v, want %v", got.Unique, tt.want.Unique)
				}
				if len(got.Columns) != len(tt.want.Columns) {
					t.Errorf("Columns length = %v, want %v", len(got.Columns), len(tt.want.Columns))
				} else {
					for i := range got.Columns {
						if got.Columns[i] != tt.want.Columns[i] {
							t.Errorf("Columns[%d] = %v, want %v", i, got.Columns[i], tt.want.Columns[i])
						}
					}
				}
			}
		})
	}
}

func TestParseDropIndex(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    *DropIndexSpec
		wantErr bool
	}{
		{
			name: "Basic Drop Index",
			sql:  "DROP INDEX idx_email ON users",
			want: &DropIndexSpec{
				Schema:    "public",
				Table:     "users",
				IndexName: "idx_email",
			},
			wantErr: false,
		},
		{
			name: "Drop Index with Schema",
			sql:  "DROP INDEX idx_email ON public.users",
			want: &DropIndexSpec{
				Schema:    "public",
				Table:     "users",
				IndexName: "idx_email",
			},
			wantErr: false,
		},
		{
			name:    "Missing ON",
			sql:     "DROP INDEX idx_email users",
			wantErr: true,
		},
		{
			name:    "Missing Table",
			sql:     "DROP INDEX idx_email ON",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := Parse(context.Background(), tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if ast.Type != DropIndexStmt {
					t.Errorf("expected DropIndexStmt, got %v", ast.Type)
				}
				got := ast.DropIndex
				if got.Schema != tt.want.Schema {
					t.Errorf("Schema = %v, want %v", got.Schema, tt.want.Schema)
				}
				if got.Table != tt.want.Table {
					t.Errorf("Table = %v, want %v", got.Table, tt.want.Table)
				}
				if got.IndexName != tt.want.IndexName {
					t.Errorf("IndexName = %v, want %v", got.IndexName, tt.want.IndexName)
				}
			}
		})
	}
}

func TestParsePrimaryKey(t *testing.T) {
	sql := "CREATE TABLE users (id INT PRIMARY KEY, email TEXT NOT NULL)"
	ast, err := Parse(context.Background(), sql)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ast.Type != CreateTableStmt {
		t.Fatalf("expected CreateTableStmt, got %v", ast.Type)
	}

	cols := ast.CreateTable.Columns
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}

	// Check ID column
	if cols[0].Name != "id" {
		t.Errorf("expected id column, got %s", cols[0].Name)
	}
	if !cols[0].PrimaryKey {
		t.Error("expected id to be PRIMARY KEY")
	}
	if cols[0].Nullable {
		t.Error("expected PRIMARY KEY to imply NOT NULL")
	}

	// Check Email column
	if cols[1].Name != "email" {
		t.Errorf("expected email column, got %s", cols[1].Name)
	}
	if cols[1].PrimaryKey {
		t.Error("expected email NOT to be PRIMARY KEY")
	}
	if cols[1].Nullable {
		t.Error("expected email to be NOT NULL")
	}
}
