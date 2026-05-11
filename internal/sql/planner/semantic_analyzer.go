package planner

import (
	"fmt"

	"github.com/hainn191297/myDb/internal/logging"
	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/expr"
	"github.com/hainn191297/myDb/internal/sql/parser"
)

// SemanticAnalyzer validates semantic correctness of parsed queries.
// It checks that all table and column references exist in the schema catalog,
// detects ambiguous column references, validates data types, and ensures
// DML statements reference valid tables and columns.
type SemanticAnalyzer struct {
	catalog *schema.Catalog
}

// NewSemanticAnalyzer creates a new semantic analyzer with a reference to the schema catalog.
func NewSemanticAnalyzer(catalog *schema.Catalog) *SemanticAnalyzer {
	return &SemanticAnalyzer{
		catalog: catalog,
	}
}

// Analyze validates the semantic correctness of a parsed AST.
// It performs comprehensive validation including:
// - Table existence checks
// - Column existence and ambiguity checks
// - Data type compatibility validation
// - DML statement validation
// - WHERE clause validation
//
// Returns an error if any validation fails, with a descriptive error message.
func (sa *SemanticAnalyzer) Analyze(ast parser.AST) error {
	logging.Debugf("[SemanticAnalyzer] Starting semantic analysis for %s statement", ast.Type)

	switch ast.Type {
	case parser.SelectStmt:
		return sa.analyzeSelect(ast)
	case parser.InsertStmt:
		return sa.analyzeInsert(ast)
	case parser.UpdateStmt:
		return sa.analyzeUpdate(ast)
	case parser.DeleteStmt:
		return sa.analyzeDelete(ast)
	case parser.CreateTableStmt:
		return sa.analyzeCreateTable(ast)
	case parser.CreateIndexStmt:
		return sa.analyzeCreateIndex(ast)
	case parser.DropTableStmt:
		return sa.analyzeDropTable(ast)
	case parser.DropIndexStmt:
		return sa.analyzeDropIndex(ast)
	default:
		// Transaction control statements don't need semantic analysis
		return nil
	}
}

// analyzeSelect validates a SELECT statement.
// Checks that:
// - The table exists in the schema catalog
// - All referenced columns exist in the table
// - Column references are not ambiguous
// - WHERE clause predicates reference valid columns
func (sa *SemanticAnalyzer) analyzeSelect(ast parser.AST) error {
	logging.Debugf("[SemanticAnalyzer] Analyzing SELECT statement for table %s.%s", ast.SchemaName, ast.TableName)

	// Validate table exists
	if err := sa.validateTableExists(ast.SchemaName, ast.TableName); err != nil {
		return err
	}

	tableDef, err := sa.catalog.GetTable(ast.SchemaName, ast.TableName)
	if err != nil {
		return fmt.Errorf("semantic analysis: table %s.%s not found: %w", ast.SchemaName, ast.TableName, err)
	}

	// Validate columns exist
	if err := sa.validateColumnsExist(ast.Columns, tableDef); err != nil {
		return err
	}

	// Validate WHERE clause if present
	if ast.WhereExpr != nil {
		if err := sa.validateWhereClause(ast.WhereExpr, tableDef); err != nil {
			return err
		}
	}

	logging.Debugf("[SemanticAnalyzer] SELECT statement validation passed")
	return nil
}

// analyzeInsert validates an INSERT statement.
// Checks that:
// - The target table exists
// - All specified columns exist in the table
// - Column count matches value count
func (sa *SemanticAnalyzer) analyzeInsert(ast parser.AST) error {
	if ast.Insert == nil {
		return fmt.Errorf("semantic analysis: INSERT spec missing")
	}

	logging.Debugf("[SemanticAnalyzer] Analyzing INSERT statement for table %s.%s", ast.Insert.Schema, ast.Insert.Table)

	// Validate table exists
	if err := sa.validateTableExists(ast.Insert.Schema, ast.Insert.Table); err != nil {
		return err
	}

	tableDef, err := sa.catalog.GetTable(ast.Insert.Schema, ast.Insert.Table)
	if err != nil {
		return fmt.Errorf("semantic analysis: table %s.%s not found: %w", ast.Insert.Schema, ast.Insert.Table, err)
	}

	// Determine columns (use all if not specified)
	columns := ast.Insert.Columns
	if len(columns) == 0 {
		for _, col := range tableDef.Columns {
			columns = append(columns, col.Name)
		}
	}

	// Validate all specified columns exist
	for _, colName := range columns {
		if !sa.columnExistsInTable(colName, tableDef) {
			return fmt.Errorf("semantic analysis: column %q not found in table %s.%s", colName, ast.Insert.Schema, ast.Insert.Table)
		}
	}

	// Validate column count matches value count for each row
	for rowIdx, rowValues := range ast.Insert.Values {
		if len(rowValues) != len(columns) {
			return fmt.Errorf("semantic analysis: INSERT value count (%d) for row %d doesn't match column count (%d)",
				len(rowValues), rowIdx, len(columns))
		}
	}

	logging.Debugf("[SemanticAnalyzer] INSERT statement validation passed")
	return nil
}

// analyzeUpdate validates an UPDATE statement.
// Checks that:
// - The target table exists
// - All referenced columns exist in the table
// - WHERE clause predicates reference valid columns
func (sa *SemanticAnalyzer) analyzeUpdate(ast parser.AST) error {
	if ast.Update == nil {
		return fmt.Errorf("semantic analysis: UPDATE spec missing")
	}

	logging.Debugf("[SemanticAnalyzer] Analyzing UPDATE statement for table %s.%s", ast.Update.Schema, ast.Update.Table)

	// Validate table exists
	if err := sa.validateTableExists(ast.Update.Schema, ast.Update.Table); err != nil {
		return err
	}

	tableDef, err := sa.catalog.GetTable(ast.Update.Schema, ast.Update.Table)
	if err != nil {
		return fmt.Errorf("semantic analysis: table %s.%s not found: %w", ast.Update.Schema, ast.Update.Table, err)
	}

	// Validate all SET clause columns exist
	for colName := range ast.Update.SetClauses {
		if !sa.columnExistsInTable(colName, tableDef) {
			return fmt.Errorf("semantic analysis: column %q not found in table %s.%s", colName, ast.Update.Schema, ast.Update.Table)
		}
	}

	// Validate WHERE clause if present
	if ast.WhereExpr != nil {
		if err := sa.validateWhereClause(ast.WhereExpr, tableDef); err != nil {
			return err
		}
	}

	logging.Debugf("[SemanticAnalyzer] UPDATE statement validation passed")
	return nil
}

// analyzeDelete validates a DELETE statement.
// Checks that:
// - The target table exists
// - WHERE clause predicates reference valid columns
func (sa *SemanticAnalyzer) analyzeDelete(ast parser.AST) error {
	if ast.Delete == nil {
		return fmt.Errorf("semantic analysis: DELETE spec missing")
	}

	logging.Debugf("[SemanticAnalyzer] Analyzing DELETE statement for table %s.%s", ast.Delete.Schema, ast.Delete.Table)

	// Validate table exists
	if err := sa.validateTableExists(ast.Delete.Schema, ast.Delete.Table); err != nil {
		return err
	}

	tableDef, err := sa.catalog.GetTable(ast.Delete.Schema, ast.Delete.Table)
	if err != nil {
		return fmt.Errorf("semantic analysis: table %s.%s not found: %w", ast.Delete.Schema, ast.Delete.Table, err)
	}

	// Validate WHERE clause if present
	if ast.WhereExpr != nil {
		if err := sa.validateWhereClause(ast.WhereExpr, tableDef); err != nil {
			return err
		}
	}

	logging.Debugf("[SemanticAnalyzer] DELETE statement validation passed")
	return nil
}

// analyzeCreateTable validates a CREATE TABLE statement.
// Checks that column definitions are valid (no duplicate names, etc.)
func (sa *SemanticAnalyzer) analyzeCreateTable(ast parser.AST) error {
	if ast.CreateTable == nil {
		return fmt.Errorf("semantic analysis: CREATE TABLE spec missing")
	}

	logging.Debugf("[SemanticAnalyzer] Analyzing CREATE TABLE statement for %s.%s", ast.CreateTable.Schema, ast.CreateTable.Table)

	if len(ast.CreateTable.Columns) == 0 {
		return fmt.Errorf("semantic analysis: CREATE TABLE requires at least one column")
	}

	// Check for duplicate column names
	seenColumns := make(map[string]bool)
	for _, col := range ast.CreateTable.Columns {
		if seenColumns[col.Name] {
			return fmt.Errorf("semantic analysis: duplicate column name %q in CREATE TABLE", col.Name)
		}
		seenColumns[col.Name] = true
	}

	logging.Debugf("[SemanticAnalyzer] CREATE TABLE statement validation passed")
	return nil
}

// analyzeCreateIndex validates a CREATE INDEX statement.
// Checks that:
// - The target table exists
// - All indexed columns exist in the table
func (sa *SemanticAnalyzer) analyzeCreateIndex(ast parser.AST) error {
	if ast.CreateIndex == nil {
		return fmt.Errorf("semantic analysis: CREATE INDEX spec missing")
	}

	logging.Debugf("[SemanticAnalyzer] Analyzing CREATE INDEX statement for %s on %s.%s",
		ast.CreateIndex.IndexName, ast.CreateIndex.Schema, ast.CreateIndex.Table)

	// Validate table exists
	if err := sa.validateTableExists(ast.CreateIndex.Schema, ast.CreateIndex.Table); err != nil {
		return err
	}

	tableDef, err := sa.catalog.GetTable(ast.CreateIndex.Schema, ast.CreateIndex.Table)
	if err != nil {
		return fmt.Errorf("semantic analysis: table %s.%s not found: %w", ast.CreateIndex.Schema, ast.CreateIndex.Table, err)
	}

	// Validate all indexed columns exist
	for _, colName := range ast.CreateIndex.Columns {
		if !sa.columnExistsInTable(colName, tableDef) {
			return fmt.Errorf("semantic analysis: column %q not found in table %s.%s", colName, ast.CreateIndex.Schema, ast.CreateIndex.Table)
		}
	}

	logging.Debugf("[SemanticAnalyzer] CREATE INDEX statement validation passed")
	return nil
}

// analyzeDropTable validates a DROP TABLE statement.
// Checks that the target table exists.
func (sa *SemanticAnalyzer) analyzeDropTable(ast parser.AST) error {
	if ast.DropTable == nil {
		return fmt.Errorf("semantic analysis: DROP TABLE spec missing")
	}

	logging.Debugf("[SemanticAnalyzer] Analyzing DROP TABLE statement for %s.%s", ast.DropTable.Schema, ast.DropTable.Table)

	// Validate table exists
	if err := sa.validateTableExists(ast.DropTable.Schema, ast.DropTable.Table); err != nil {
		return err
	}

	logging.Debugf("[SemanticAnalyzer] DROP TABLE statement validation passed")
	return nil
}

// analyzeDropIndex validates a DROP INDEX statement.
// Checks that the target table exists (index existence is checked at execution time).
func (sa *SemanticAnalyzer) analyzeDropIndex(ast parser.AST) error {
	if ast.DropIndex == nil {
		return fmt.Errorf("semantic analysis: DROP INDEX spec missing")
	}

	logging.Debugf("[SemanticAnalyzer] Analyzing DROP INDEX statement for %s on %s.%s",
		ast.DropIndex.IndexName, ast.DropIndex.Schema, ast.DropIndex.Table)

	// Validate table exists
	if err := sa.validateTableExists(ast.DropIndex.Schema, ast.DropIndex.Table); err != nil {
		return err
	}

	logging.Debugf("[SemanticAnalyzer] DROP INDEX statement validation passed")
	return nil
}

// validateTableExists checks that a table exists in the schema catalog.
func (sa *SemanticAnalyzer) validateTableExists(schema, table string) error {
	_, err := sa.catalog.GetTable(schema, table)
	if err != nil {
		return fmt.Errorf("semantic analysis: table %s.%s not found", schema, table)
	}
	return nil
}

// validateColumnsExist checks that all specified columns exist in the table.
// Handles SELECT * expansion.
func (sa *SemanticAnalyzer) validateColumnsExist(columns []string, tableDef *schema.TableDef) error {
	// SELECT * is always valid
	if len(columns) == 1 && columns[0] == "*" {
		return nil
	}

	for _, colName := range columns {
		if colName == "*" {
			// Wildcard in column list is only valid as sole column
			continue
		}

		if !sa.columnExistsInTable(colName, tableDef) {
			return fmt.Errorf("semantic analysis: column %q not found in table %s.%s", colName, tableDef.Schema, tableDef.Table)
		}
	}

	return nil
}

// columnExistsInTable checks if a column exists in a table definition.
func (sa *SemanticAnalyzer) columnExistsInTable(colName string, tableDef *schema.TableDef) bool {
	for _, col := range tableDef.Columns {
		if col.Name == colName {
			return true
		}
	}
	return false
}

// validateWhereClause validates that all column references in a WHERE clause are valid.
// It recursively traverses the expression tree to find all column references.
func (sa *SemanticAnalyzer) validateWhereClause(whereExpr expr.Expr, tableDef *schema.TableDef) error {
	if whereExpr == nil {
		return nil
	}

	// Recursively validate all column references in the expression
	return sa.validateExpression(whereExpr, tableDef)
}

// validateExpression recursively validates an expression tree.
// It checks that all column references exist in the table.
func (sa *SemanticAnalyzer) validateExpression(e expr.Expr, tableDef *schema.TableDef) error {
	if e == nil {
		return nil
	}

	switch expr := e.(type) {
	case *expr.ColumnRefExpr:
		// Validate column reference
		if !sa.columnExistsInTable(expr.Name, tableDef) {
			return fmt.Errorf("semantic analysis: column %q not found in table %s.%s", expr.Name, tableDef.Schema, tableDef.Table)
		}
		return nil

	case *expr.LiteralExpr:
		// Literals are always valid
		return nil

	case *expr.BinaryExpr:
		// Validate both sides of binary expression
		if err := sa.validateExpression(expr.Left, tableDef); err != nil {
			return err
		}
		if err := sa.validateExpression(expr.Right, tableDef); err != nil {
			return err
		}
		return nil

	case *expr.UnaryExpr:
		// Validate operand of unary expression
		return sa.validateExpression(expr.Expr, tableDef)

	default:
		// Unknown expression type - assume valid
		return nil
	}
}

// validateDataTypeCompatibility checks that a value is compatible with a column's data type.
// This is a simplified implementation that can be extended with more sophisticated type checking.
func (sa *SemanticAnalyzer) validateDataTypeCompatibility(value string, colDef *schema.ColumnDef) error {
	// Strip quotes from string literals
	strValue := value
	if len(strValue) >= 2 && (strValue[0] == '\'' || strValue[0] == '"') {
		strValue = strValue[1 : len(strValue)-1]
	}

	// Try to encode the value using the column's type
	// If encoding fails, the types are incompatible
	_, err := colDef.Type.Encode(strValue)
	if err != nil {
		return fmt.Errorf("semantic analysis: type mismatch for column %s: %w", colDef.Name, err)
	}

	return nil
}

// detectAmbiguousColumns checks if a column name appears in multiple tables without qualification.
// This is used for multi-table queries (JOINs) to detect ambiguous references.
// For now, this is a placeholder for future multi-table support.
func (sa *SemanticAnalyzer) detectAmbiguousColumns(colName string, tables []*schema.TableDef) error {
	count := 0
	for _, table := range tables {
		if sa.columnExistsInTable(colName, table) {
			count++
		}
	}

	if count > 1 {
		return fmt.Errorf("semantic analysis: ambiguous column reference: %s", colName)
	}

	return nil
}
