package parser

import (
	"context"
	"fmt"
	"strings"

	dberrors "github.com/hainn191297/myDb/internal/errors"
	"github.com/hainn191297/myDb/internal/logging"
	"github.com/hainn191297/myDb/internal/sql/expr"
)

// AST represents the root of a parsed SQL statement.
type AST struct {
	Type       StatementType
	SchemaName string
	TableName  string
	Columns    []string

	// WHERE clause (dual support for migration)
	Where     string    // Deprecated: use WhereExpr instead
	WhereExpr expr.Expr // Structured WHERE clause (nil if empty)

	// DDL-specific fields
	CreateTable *CreateTableSpec
	DropTable   *DropTableSpec
	CreateIndex *CreateIndexSpec
	DropIndex   *DropIndexSpec

	// DML-specific fields
	Insert *InsertSpec
	Update *UpdateSpec
	Delete *DeleteSpec
}

// CreateTableSpec holds CREATE TABLE statement details.
type CreateTableSpec struct {
	Schema  string
	Table   string
	Columns []ColumnSpec
}

// ColumnSpec represents a column definition in CREATE TABLE.
type ColumnSpec struct {
	Name         string
	Type         string // "INT", "TEXT", "BOOL", "FLOAT"
	Nullable     bool
	PrimaryKey   bool
	DefaultValue string
}

// DropTableSpec holds DROP TABLE statement details.
type DropTableSpec struct {
	Schema string
	Table  string
}

// CreateIndexSpec holds CREATE INDEX statement details.
type CreateIndexSpec struct {
	Schema    string
	Table     string
	IndexName string
	Columns   []string
	Unique    bool
}

// DropIndexSpec holds DROP INDEX statement details.
type DropIndexSpec struct {
	Schema    string
	Table     string
	IndexName string
}

// InsertSpec holds INSERT INTO statement details.
type InsertSpec struct {
	Schema  string
	Table   string
	Columns []string   // Optional: if empty, insert to all columns in order
	Values  [][]string // List of rows, each row is a list of raw string values
}

// UpdateSpec holds UPDATE statement details.
type UpdateSpec struct {
	Schema     string
	Table      string
	SetClauses map[string]string // column -> value
	Where      string            // String-based for MVP
}

// DeleteSpec holds DELETE statement details.
type DeleteSpec struct {
	Schema string
	Table  string
	Where  string
}

// StatementType enumerates supported SQL forms.
type StatementType string

const (
	SelectStmt      StatementType = "SELECT"
	InsertStmt      StatementType = "INSERT"
	UpdateStmt      StatementType = "UPDATE"
	DeleteStmt      StatementType = "DELETE"
	BeginStmt       StatementType = "BEGIN"
	CommitStmt      StatementType = "COMMIT"
	RollbackStmt    StatementType = "ROLLBACK"
	CreateTableStmt StatementType = "CREATE_TABLE"
	DropTableStmt   StatementType = "DROP_TABLE"
	CreateIndexStmt StatementType = "CREATE_INDEX"
	DropIndexStmt   StatementType = "DROP_INDEX"
)

// Parse converts SQL text into an AST for a limited subset of statements. The
// goal is to provide a structured representation that future planners/executors
// can build upon while more sophisticated parsing is implemented later.
func Parse(ctx context.Context, sql string) (AST, error) {
	trimmed := strings.TrimSpace(sql)
	trimmed = strings.TrimSuffix(trimmed, ";")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return AST{}, ErrEmptyStatement
	}

	upper := strings.ToUpper(trimmed)

	switch {
	case strings.HasPrefix(upper, "BEGIN"):
		return AST{Type: BeginStmt}, nil
	case strings.HasPrefix(upper, "COMMIT"):
		return AST{Type: CommitStmt}, nil
	case strings.HasPrefix(upper, "ROLLBACK"):
		return AST{Type: RollbackStmt}, nil
	case strings.HasPrefix(upper, "SELECT"):
		return parseSelect(trimmed)
	case strings.HasPrefix(upper, "INSERT INTO"):
		return parseInsert(ctx, trimmed)
	case strings.HasPrefix(upper, "UPDATE"):
		return parseUpdate(trimmed)
	case strings.HasPrefix(upper, "DELETE FROM"):
		return parseDelete(trimmed)
	case strings.HasPrefix(upper, "CREATE TABLE"):
		return parseCreateTable(trimmed)
	case strings.HasPrefix(upper, "DROP TABLE"):
		return parseDropTable(trimmed)
	case strings.HasPrefix(upper, "CREATE INDEX") || strings.HasPrefix(upper, "CREATE UNIQUE INDEX"):
		return parseCreateIndex(trimmed)
	case strings.HasPrefix(upper, "DROP INDEX"):
		return parseDropIndex(trimmed)
	default:
		return AST{}, fmt.Errorf("parser: unsupported statement %q", trimmed)
	}
}

func parseSelect(sql string) (AST, error) {
	upper := strings.ToUpper(sql)
	if !strings.HasPrefix(upper, "SELECT ") {
		return AST{}, fmt.Errorf("parser: invalid SELECT syntax")
	}

	fromIdx := strings.Index(upper, " FROM ")
	if fromIdx == -1 {
		return AST{}, fmt.Errorf("parser: SELECT missing FROM clause")
	}

	colsPart := strings.TrimSpace(sql[len("SELECT "):fromIdx])
	if colsPart == "" {
		return AST{}, fmt.Errorf("parser: SELECT missing column list")
	}

	rest := strings.TrimSpace(sql[fromIdx+len(" FROM "):])
	if rest == "" {
		return AST{}, fmt.Errorf("parser: SELECT missing table name")
	}

	whereIdx := strings.Index(strings.ToUpper(rest), " WHERE ")
	var tablePart, wherePart string
	if whereIdx >= 0 {
		tablePart = strings.TrimSpace(rest[:whereIdx])
		wherePart = strings.TrimSpace(rest[whereIdx+len(" WHERE "):])
	} else {
		tablePart = strings.TrimSpace(rest)
	}

	if tablePart == "" {
		return AST{}, fmt.Errorf("parser: SELECT missing table name")
	}

	schema, table := splitSchemaTable(tablePart)

	cols := splitColumns(colsPart)
	if len(cols) == 0 {
		return AST{}, fmt.Errorf("parser: invalid column list")
	}

	ast := AST{
		Type:       SelectStmt,
		SchemaName: schema,
		TableName:  table,
		Columns:    cols,
		Where:      wherePart, // Keep for backward compat
	}

	// Parse WHERE clause into structured expression
	if wherePart != "" {
		whereExpr, err := expr.ParseExpr(wherePart)
		if err != nil {
			return AST{}, fmt.Errorf("parser: invalid WHERE clause: %w", err)
		}
		ast.WhereExpr = whereExpr
	}

	return ast, nil
}

func splitColumns(part string) []string {
	raw := strings.Split(part, ",")
	var cols []string
	for _, c := range raw {
		name := strings.TrimSpace(c)
		if name == "" {
			continue
		}
		cols = append(cols, name)
	}
	return cols
}

func splitSchemaTable(name string) (string, string) {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", name
}

// parseCreateTable parses CREATE TABLE statements.
// Supported syntax:
//
//	CREATE TABLE users (id INT, name TEXT)
//	CREATE TABLE public.users (id INT NOT NULL, active BOOL)
func parseCreateTable(sql string) (AST, error) {
	upper := strings.ToUpper(sql)
	if !strings.HasPrefix(upper, "CREATE TABLE ") {
		return AST{}, fmt.Errorf("parser: invalid CREATE TABLE syntax")
	}

	// Skip "CREATE TABLE "
	rest := strings.TrimSpace(sql[len("CREATE TABLE "):])

	// Find opening parenthesis
	openParen := strings.Index(rest, "(")
	if openParen == -1 {
		return AST{}, fmt.Errorf("parser: CREATE TABLE missing column definitions")
	}

	tablePart := strings.TrimSpace(rest[:openParen])
	if tablePart == "" {
		return AST{}, fmt.Errorf("parser: CREATE TABLE missing table name")
	}

	// Find closing parenthesis
	closeParen := strings.LastIndex(rest, ")")
	if closeParen == -1 || closeParen < openParen {
		return AST{}, fmt.Errorf("parser: CREATE TABLE missing closing parenthesis")
	}

	columnsPart := strings.TrimSpace(rest[openParen+1 : closeParen])
	if columnsPart == "" {
		return AST{}, fmt.Errorf("parser: CREATE TABLE must have at least one column")
	}

	schema, table := splitSchemaTable(tablePart)

	// Parse column definitions
	columns, err := parseColumnDefs(columnsPart)
	if err != nil {
		return AST{}, fmt.Errorf("parser: %w", err)
	}

	return AST{
		Type: CreateTableStmt,
		CreateTable: &CreateTableSpec{
			Schema:  schema,
			Table:   table,
			Columns: columns,
		},
	}, nil
}

// parseDropTable parses DROP TABLE statements.
// Supported syntax:
//
//	DROP TABLE users
//	DROP TABLE public.users
func parseDropTable(sql string) (AST, error) {
	upper := strings.ToUpper(sql)
	if !strings.HasPrefix(upper, "DROP TABLE ") {
		return AST{}, fmt.Errorf("parser: invalid DROP TABLE syntax")
	}

	tablePart := strings.TrimSpace(sql[len("DROP TABLE "):])
	if tablePart == "" {
		return AST{}, fmt.Errorf("parser: DROP TABLE missing table name")
	}

	schema, table := splitSchemaTable(tablePart)

	return AST{
		Type: DropTableStmt,
		DropTable: &DropTableSpec{
			Schema: schema,
			Table:  table,
		},
	}, nil
}

// parseCreateIndex parses CREATE INDEX statements.
// Supported syntax:
//
//	CREATE INDEX idx_name ON users (email)
//	CREATE UNIQUE INDEX idx_name ON users (email)
func parseCreateIndex(sql string) (AST, error) {
	upper := strings.ToUpper(sql)
	unique := false
	var rest string

	if strings.HasPrefix(upper, "CREATE UNIQUE INDEX ") {
		unique = true
		rest = strings.TrimSpace(sql[len("CREATE UNIQUE INDEX "):])
	} else if strings.HasPrefix(upper, "CREATE INDEX ") {
		rest = strings.TrimSpace(sql[len("CREATE INDEX "):])
	} else {
		return AST{}, fmt.Errorf("parser: invalid CREATE INDEX syntax")
	}

	// Parse "idx_name ON table (col)"
	onIdx := strings.Index(strings.ToUpper(rest), " ON ")
	if onIdx == -1 {
		return AST{}, fmt.Errorf("parser: CREATE INDEX missing ON clause")
	}

	indexName := strings.TrimSpace(rest[:onIdx])
	if indexName == "" {
		return AST{}, fmt.Errorf("parser: CREATE INDEX missing index name")
	}

	afterOn := strings.TrimSpace(rest[onIdx+len(" ON "):])

	// Parse "table (col)"
	openParen := strings.Index(afterOn, "(")
	if openParen == -1 {
		return AST{}, fmt.Errorf("parser: CREATE INDEX missing column list")
	}

	tablePart := strings.TrimSpace(afterOn[:openParen])
	if tablePart == "" {
		return AST{}, fmt.Errorf("parser: CREATE INDEX missing table name")
	}

	closeParen := strings.LastIndex(afterOn, ")")
	if closeParen == -1 || closeParen < openParen {
		return AST{}, fmt.Errorf("parser: CREATE INDEX missing closing parenthesis")
	}

	columnsPart := strings.TrimSpace(afterOn[openParen+1 : closeParen])
	if columnsPart == "" {
		return AST{}, fmt.Errorf("parser: CREATE INDEX must have at least one column")
	}

	schema, table := splitSchemaTable(tablePart)
	columns := splitColumns(columnsPart)

	return AST{
		Type: CreateIndexStmt,
		CreateIndex: &CreateIndexSpec{
			Schema:    schema,
			Table:     table,
			IndexName: indexName,
			Columns:   columns,
			Unique:    unique,
		},
	}, nil
}

// parseDropIndex parses DROP INDEX statements.
// Supported syntax:
//
//	DROP INDEX idx_name ON users
func parseDropIndex(sql string) (AST, error) {
	upper := strings.ToUpper(sql)
	if !strings.HasPrefix(upper, "DROP INDEX ") {
		return AST{}, fmt.Errorf("parser: invalid DROP INDEX syntax")
	}

	rest := strings.TrimSpace(sql[len("DROP INDEX "):])

	onIdx := strings.Index(strings.ToUpper(rest), " ON ")
	if onIdx == -1 {
		return AST{}, fmt.Errorf("parser: DROP INDEX missing ON clause")
	}

	indexName := strings.TrimSpace(rest[:onIdx])
	if indexName == "" {
		return AST{}, fmt.Errorf("parser: DROP INDEX missing index name")
	}

	tablePart := strings.TrimSpace(rest[onIdx+len(" ON "):])
	if tablePart == "" {
		return AST{}, fmt.Errorf("parser: DROP INDEX missing table name")
	}

	schema, table := splitSchemaTable(tablePart)

	return AST{
		Type: DropIndexStmt,
		DropIndex: &DropIndexSpec{
			Schema:    schema,
			Table:     table,
			IndexName: indexName,
		},
	}, nil
}

// parseColumnDefs parses column definitions from CREATE TABLE.
// Example: "id INT, name TEXT NOT NULL, active BOOL"
func parseColumnDefs(columnsPart string) ([]ColumnSpec, error) {
	// Split by comma
	parts := strings.Split(columnsPart, ",")
	if len(parts) == 0 {
		return nil, fmt.Errorf("no column definitions found")
	}

	var columns []ColumnSpec
	for _, part := range parts {
		col, err := parseColumnDef(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}

	return columns, nil
}

// parseColumnDef parses a single column definition.
// Example: "id INT", "name TEXT NOT NULL", "active BOOL", "id INT PRIMARY KEY"
func parseColumnDef(def string) (ColumnSpec, error) {
	tokens := strings.Fields(def)
	if len(tokens) < 2 {
		return ColumnSpec{}, fmt.Errorf("invalid column definition %q", def)
	}

	name := tokens[0]
	typeName := strings.ToUpper(tokens[1])

	// Default to nullable, check for NOT NULL and PRIMARY KEY
	nullable := true
	primaryKey := false
	var defaultValue string

	for i := 2; i < len(tokens); i++ {
		token := strings.ToUpper(tokens[i])
		if token == "NOT" && i+1 < len(tokens) && strings.ToUpper(tokens[i+1]) == "NULL" {
			nullable = false
			i++ // Skip NULL
		} else if token == "PRIMARY" && i+1 < len(tokens) && strings.ToUpper(tokens[i+1]) == "KEY" {
			primaryKey = true
			nullable = false // PRIMARY KEY implies NOT NULL
			i++              // Skip KEY
		} else if token == "DEFAULT" && i+1 < len(tokens) {
			defaultValue = tokens[i+1]
			i++ // Skip value
		}
	}

	return ColumnSpec{
		Name:         name,
		Type:         typeName,
		Nullable:     nullable,
		PrimaryKey:   primaryKey,
		DefaultValue: defaultValue,
	}, nil
}

// parseInsert parses INSERT INTO statements.
// Supported syntax:
//
//	INSERT INTO users VALUES (1, 'alice', true)
//	INSERT INTO users (id, name) VALUES (1, 'alice'), (2, 'bob')
func parseInsert(ctx context.Context, sql string) (AST, error) {
	logging.DebugContext(ctx, "[Parser] Parsing INSERT statement")

	upper := strings.ToUpper(sql)
	if !strings.HasPrefix(upper, "INSERT INTO ") {
		return AST{}, fmt.Errorf("parser: invalid INSERT syntax")
	}

	rest := strings.TrimSpace(sql[len("INSERT INTO "):])

	// Find VALUES keyword
	valuesIdx := strings.Index(strings.ToUpper(rest), " VALUES ")
	if valuesIdx == -1 {
		return AST{}, fmt.Errorf("parser: INSERT missing VALUES clause")
	}

	tablePart := strings.TrimSpace(rest[:valuesIdx])
	valuesPart := strings.TrimSpace(rest[valuesIdx+len(" VALUES "):])

	// Parse table name and optional column list
	var schema, table string
	var columns []string

	if strings.Contains(tablePart, "(") {
		// INSERT INTO users (col1, col2) VALUES ...
		openParen := strings.Index(tablePart, "(")
		closeParen := strings.LastIndex(tablePart, ")")
		if closeParen == -1 || closeParen < openParen {
			return AST{}, fmt.Errorf("parser: INSERT invalid column list")
		}

		tableNamePart := strings.TrimSpace(tablePart[:openParen])
		schema, table = splitSchemaTable(tableNamePart)

		columnsPart := strings.TrimSpace(tablePart[openParen+1 : closeParen])
		columns = splitColumns(columnsPart)
	} else {
		// INSERT INTO users VALUES ...
		schema, table = splitSchemaTable(tablePart)
	}

	// Parse VALUES (v1, v2), (v3, v4)
	var rows [][]string
	var currentRowStr strings.Builder
	inParens := false
	parenDepth := 0
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range valuesPart {
		if ch == '\'' || ch == '"' {
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuotes = false
			}
			currentRowStr.WriteRune(ch)
		} else if ch == '(' && !inQuotes {
			if parenDepth == 0 {
				inParens = true
				currentRowStr.Reset()
			} else {
				currentRowStr.WriteRune(ch)
			}
			parenDepth++
		} else if ch == ')' && !inQuotes {
			parenDepth--
			if parenDepth == 0 {
				inParens = false
				rowValues := parseValues(currentRowStr.String())
				rows = append(rows, rowValues)
				currentRowStr.Reset()
			} else {
				currentRowStr.WriteRune(ch)
			}
		} else if ch == ',' && !inQuotes && !inParens {
			continue
		} else {
			if inParens {
				currentRowStr.WriteRune(ch)
			}
		}
	}

	if parenDepth != 0 {
		return AST{}, fmt.Errorf("parser: unbalanced parentheses in VALUES clause")
	}
	if len(rows) == 0 {
		return AST{}, fmt.Errorf("parser: no values found in INSERT statement")
	}

	logging.DebugContext(ctx, "[Parser] Successfully parsed INSERT into %s.%s with %d rows, %d columns",
		schema, table, len(rows), len(columns))

	return AST{
		Type: InsertStmt,
		Insert: &InsertSpec{
			Schema:  schema,
			Table:   table,
			Columns: columns,
			Values:  rows,
		},
	}, nil
}

// parseUpdate parses UPDATE statements.
// Supported syntax:
//
//	UPDATE users SET name = 'bob', active = false WHERE id = 1
//	UPDATE users SET name = 'bob'
func parseUpdate(sql string) (AST, error) {
	upper := strings.ToUpper(sql)
	if !strings.HasPrefix(upper, "UPDATE ") {
		return AST{}, fmt.Errorf("parser: invalid UPDATE syntax")
	}

	rest := strings.TrimSpace(sql[len("UPDATE "):])

	// Find SET keyword
	setIdx := strings.Index(strings.ToUpper(rest), " SET ")
	if setIdx == -1 {
		return AST{}, fmt.Errorf("parser: UPDATE missing SET clause")
	}

	tablePart := strings.TrimSpace(rest[:setIdx])
	schema, table := splitSchemaTable(tablePart)

	afterSet := strings.TrimSpace(rest[setIdx+len(" SET "):])

	// Find WHERE clause
	var setClauses map[string]string
	var wherePart string

	whereIdx := strings.Index(strings.ToUpper(afterSet), " WHERE ")
	if whereIdx >= 0 {
		setClausePart := strings.TrimSpace(afterSet[:whereIdx])
		wherePart = strings.TrimSpace(afterSet[whereIdx+len(" WHERE "):])
		setClauses = parseSetClauses(setClausePart)
	} else {
		setClauses = parseSetClauses(afterSet)
	}

	spec := &UpdateSpec{
		Schema:     schema,
		Table:      table,
		SetClauses: setClauses,
		Where:      wherePart, // Backward compat
	}

	// Parse WHERE clause into structured expression
	var whereExpr expr.Expr
	if wherePart != "" {
		var err error
		whereExpr, err = expr.ParseExpr(wherePart)
		if err != nil {
			return AST{}, fmt.Errorf("parser: invalid WHERE clause in UPDATE: %w", err)
		}
	}

	return AST{
		Type:      UpdateStmt,
		Update:    spec,
		WhereExpr: whereExpr,
	}, nil
}

// parseDelete parses DELETE FROM statements.
// Supported syntax:
//
//	DELETE FROM users WHERE id = 1
//	DELETE FROM users
func parseDelete(sql string) (AST, error) {
	upper := strings.ToUpper(sql)
	if !strings.HasPrefix(upper, "DELETE FROM ") {
		return AST{}, fmt.Errorf("parser: invalid DELETE syntax")
	}

	rest := strings.TrimSpace(sql[len("DELETE FROM "):])

	// Find WHERE clause
	whereIdx := strings.Index(strings.ToUpper(rest), " WHERE ")
	var tablePart, wherePart string

	if whereIdx >= 0 {
		tablePart = strings.TrimSpace(rest[:whereIdx])
		wherePart = strings.TrimSpace(rest[whereIdx+len(" WHERE "):])
	} else {
		tablePart = strings.TrimSpace(rest)
	}

	schema, table := splitSchemaTable(tablePart)

	spec := &DeleteSpec{
		Schema: schema,
		Table:  table,
		Where:  wherePart, // Backward compat
	}

	// Parse WHERE clause into structured expression
	var whereExpr expr.Expr
	if wherePart != "" {
		var err error
		whereExpr, err = expr.ParseExpr(wherePart)
		if err != nil {
			return AST{}, fmt.Errorf("parser: invalid WHERE clause in DELETE: %w", err)
		}
	}

	return AST{
		Type:      DeleteStmt,
		Delete:    spec,
		WhereExpr: whereExpr,
	}, nil
}

// parseValues splits comma-separated values from INSERT statement.
// Example: "1, 'alice', true" -> ["1", "'alice'", "true"]
func parseValues(valuesStr string) []string {
	var values []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range valuesStr {
		if ch == '\'' || ch == '"' {
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuotes = false
			}
			current.WriteRune(ch)
		} else if ch == ',' && !inQuotes {
			values = append(values, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		values = append(values, strings.TrimSpace(current.String()))
	}

	return values
}

// parseSetClauses parses SET clause from UPDATE statement.
// Example: "name = 'bob', active = false" -> map[name:'bob' active:false]
func parseSetClauses(setClauseStr string) map[string]string {
	setClauses := make(map[string]string)
	parts := strings.Split(setClauseStr, ",")

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			setClauses[key] = value
		}
	}

	return setClauses
}

var (
	// ErrEmptyStatement is returned when parsing an empty SQL string.
	// Deprecated: Use dberrors.ErrEmptyStatement instead.
	ErrEmptyStatement = dberrors.ErrEmptyStatement
)
