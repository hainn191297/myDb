package executor

import (
	"context"
	"testing"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/parser"
	"github.com/hainn191297/myDb/internal/sql/planner"
)

func TestInsertDuplicatePrimaryKeyRejected(t *testing.T) {
	ctx := context.Background()
	tableEng := newFakeEngineForCatalog()
	indexEng := newFakeEngineForCatalog()
	catalog := schema.NewCatalog(tableEng, indexEng)

	cols := []schema.ColumnDef{
		{Name: "id", Type: schema.TypeInt64, PrimaryKey: true},
		{Name: "email", Type: schema.TypeText},
	}
	if err := catalog.CreateTable(ctx, "public", "users", cols); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := catalog.CreateIndex(ctx, "public", "users", "__pk_users_id", []string{"id"}, true, true); err != nil {
		t.Fatalf("create pk index: %v", err)
	}

	provider := newFakeProvider()
	provider.engines["public.users"] = newFakeEngine()

	// first insert ok
	ast, _ := parser.Parse(context.Background(), "INSERT INTO users VALUES (1, 'a@example.com')")
	plan, err := planner.Build(context.Background(), ast, catalog)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	exec := New(plan, Options{Catalog: catalog, Provider: provider})
	if _, err := exec.Next(ctx); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// duplicate pk should fail
	ast, _ = parser.Parse(context.Background(), "INSERT INTO users VALUES (1, 'b@example.com')")
	plan, err = planner.Build(context.Background(), ast, catalog)
	if err != nil {
		t.Fatalf("plan dup: %v", err)
	}
	exec = New(plan, Options{Catalog: catalog, Provider: provider})
	if _, err := exec.Next(ctx); err == nil {
		t.Fatalf("expected duplicate primary key error")
	}
}

func TestUpdateUniqueIndexRejected(t *testing.T) {
	ctx := context.Background()
	tableEng := newFakeEngineForCatalog()
	indexEng := newFakeEngineForCatalog()
	catalog := schema.NewCatalog(tableEng, indexEng)

	cols := []schema.ColumnDef{
		{Name: "id", Type: schema.TypeInt64, PrimaryKey: true},
		{Name: "email", Type: schema.TypeText},
	}
	if err := catalog.CreateTable(ctx, "public", "users", cols); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := catalog.CreateIndex(ctx, "public", "users", "__pk_users_id", []string{"id"}, true, true); err != nil {
		t.Fatalf("create pk index: %v", err)
	}
	if err := catalog.CreateIndex(ctx, "public", "users", "idx_email", []string{"email"}, true, false); err != nil {
		t.Fatalf("create unique index: %v", err)
	}

	provider := newFakeProvider()
	provider.engines["public.users"] = newFakeEngine()

	// insert two rows
	for _, sql := range []string{
		"INSERT INTO users VALUES (1, 'a@example.com')",
		"INSERT INTO users VALUES (2, 'b@example.com')",
	} {
		ast, _ := parser.Parse(context.Background(), sql)
		plan, err := planner.Build(context.Background(), ast, catalog)
		if err != nil {
			t.Fatalf("plan insert: %v", err)
		}
		exec := New(plan, Options{Catalog: catalog, Provider: provider})
		if _, err := exec.Next(ctx); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	// update row 2 to conflicting email should fail
	ast, _ := parser.Parse(context.Background(), "UPDATE users SET email = 'a@example.com' WHERE id = 2")
	plan, err := planner.Build(context.Background(), ast, catalog)
	if err != nil {
		t.Fatalf("plan update: %v", err)
	}
	exec := New(plan, Options{Catalog: catalog, Provider: provider})
	if _, err := exec.Next(ctx); err == nil {
		t.Fatalf("expected duplicate email rejection")
	}
}
