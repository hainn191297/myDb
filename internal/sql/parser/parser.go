package parser

// AST represents the root of a parsed SQL statement. Only a handful of
// statements will be supported initially; keep the structure small and focused.
type AST struct {
    Type      StatementType
    TableName string
    Columns   []string
    Where     string
}

// StatementType enumerates supported SQL forms.
type StatementType string

const (
    SelectStmt StatementType = "SELECT"
    InsertStmt StatementType = "INSERT"
    UpdateStmt StatementType = "UPDATE"
    DeleteStmt StatementType = "DELETE"
)

// Parse converts SQL text into an AST. The implementation will grow from a
// basic tokenizer into a full parser with error recovery.
func Parse(sql string) (AST, error) {
    // TODO: wire actual lexer/parser. Returning explicit error avoids panics.
    return AST{}, ErrNotImplemented
}

var ErrNotImplemented = errPlaceholder("parser not implemented")

type errPlaceholder string

func (e errPlaceholder) Error() string { return string(e) }
