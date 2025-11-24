package errors

import "errors"

// Storage Engine Errors
// These errors are returned by the storage layer when accessing data.
var (
	// ErrKeyNotFound is returned when a requested key does not exist.
	ErrKeyNotFound = errors.New("key not found")

	// ErrPageFull is returned when attempting to insert into a full page.
	ErrPageFull = errors.New("page is full")

	// ErrInvalidPageData is returned when page data is corrupted or malformed.
	ErrInvalidPageData = errors.New("invalid page data")
)

// Catalog Errors
// These errors are returned by the schema catalog when managing metadata.
var (
	// ErrTableNotFound is returned when a table does not exist.
	ErrTableNotFound = errors.New("table not found")

	// ErrTableExists is returned when attempting to create a table that already exists.
	ErrTableExists = errors.New("table already exists")

	// ErrColumnNotFound is returned when a column does not exist in the schema.
	ErrColumnNotFound = errors.New("column not found")

	// ErrIndexNotFound is returned when an index does not exist.
	ErrIndexNotFound = errors.New("index not found")

	// ErrIndexExists is returned when attempting to create an index that already exists.
	ErrIndexExists = errors.New("index already exists")

	// ErrEmptyTableName is returned when table or schema name is empty.
	ErrEmptyTableName = errors.New("table name cannot be empty")

	// ErrEmptyColumnList is returned when creating a table with no columns.
	ErrEmptyColumnList = errors.New("table must have at least one column")
)

// Transaction Errors
// These errors are returned by the transaction manager.
var (
	// ErrTxnNotFound is returned when a transaction ID does not exist.
	ErrTxnNotFound = errors.New("transaction not found")

	// ErrTxnAlreadyActive is returned when attempting to start a transaction while one is already active.
	ErrTxnAlreadyActive = errors.New("transaction already active")

	// ErrNoActiveTxn is returned when attempting transaction operations without an active transaction.
	ErrNoActiveTxn = errors.New("no active transaction")

	// ErrLockTimeout is returned when a lock acquisition times out.
	ErrLockTimeout = errors.New("lock acquisition timeout")

	// ErrLockConflict is returned when a lock cannot be acquired due to conflict.
	ErrLockConflict = errors.New("lock conflict")
)

// Parser Errors
// These errors are returned by the SQL parser.
var (
	// ErrEmptyStatement is returned when parsing an empty SQL string.
	ErrEmptyStatement = errors.New("empty SQL statement")

	// ErrInvalidSyntax is returned when SQL syntax is malformed.
	ErrInvalidSyntax = errors.New("invalid SQL syntax")

	// ErrMissingClause is returned when a required SQL clause is missing.
	ErrMissingClause = errors.New("missing required SQL clause")
)

// Executor Errors
// These errors are returned by the SQL executor.
var (
	// ErrCatalogNotConfigured is returned when executor is used without a catalog.
	ErrCatalogNotConfigured = errors.New("catalog not configured")

	// ErrStorageNotConfigured is returned when executor is used without storage.
	ErrStorageNotConfigured = errors.New("storage provider not configured")

	// ErrTxnMgrNotConfigured is returned when executor is used without transaction manager.
	ErrTxnMgrNotConfigured = errors.New("transaction manager not configured")

	// ErrUnsupportedOperator is returned when an unknown operator type is encountered.
	ErrUnsupportedOperator = errors.New("unsupported operator type")
)
