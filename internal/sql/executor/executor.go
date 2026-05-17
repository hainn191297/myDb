package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hainn191297/myDb/internal/logging"
	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/planner"
	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/txn"
)

// Row represents one projected row emitted by the executor.
type Row struct {
	Columns []string
	Values  [][]byte
}

// EngineProvider resolves a storage engine for (schema, table).
type EngineProvider interface {
	Engine(schema, table string) (engine.Engine, error)
	Index(schema, table, indexName string) (engine.IndexEngine, error)
}

// SessionTxn tracks the transaction bound to a session.
type SessionTxn struct {
	Current *txn.State
}

// Options configures Executor dependencies.
type Options struct {
	TxnManager *txn.Manager
	SessionTxn *SessionTxn
	Provider   EngineProvider
	Catalog    *schema.Catalog // NEW: schema catalog
}

// Executor walks a plan tree and produces row streams.
type Executor struct {
	plan       planner.Plan
	txnManager *txn.Manager
	sessionTxn *SessionTxn
	provider   EngineProvider
	catalog    *schema.Catalog // NEW

	dataIter engine.Iterator
	rowSpecs []columnSpec
	tableDef *schema.TableDef // Cache table definition for decoding
	current  Row
	started  bool
	closed   bool
	isPKScan bool // NEW: track if current index scan is on PK
}

type columnKind int

const (
	columnKey columnKind = iota
	columnValue
)

type columnSpec struct {
	name  string
	kind  columnKind
	index int
}

// New builds an executor for the supplied plan.
func New(plan planner.Plan, opts Options) *Executor {
	return &Executor{
		plan:       plan,
		txnManager: opts.TxnManager,
		sessionTxn: opts.SessionTxn,
		provider:   opts.Provider,
		catalog:    opts.Catalog,
	}
}

// Next executes the plan and returns whether more rows exist.
func (e *Executor) Next(ctx context.Context) (bool, error) {
	switch op := e.plan.Root.(type) {
	case *planner.TxnOp:
		if e.started {
			return false, nil
		}
		e.started = true
		return false, e.executeTxn(ctx, op)
	case *planner.SeqScanOp:
		return e.nextSeqScan(ctx, op)
	case *planner.IndexScanOp:
		return e.nextIndexScan(ctx, op)
	case *planner.CreateTableOp:
		if e.started {
			return false, nil
		}
		e.started = true
		return false, e.executeCreateTable(ctx, op)
	case *planner.DropTableOp:
		if e.started {
			return false, nil
		}
		e.started = true
		return false, e.executeDropTable(ctx, op)
	case *planner.CreateIndexOp:
		if e.started {
			return false, nil
		}
		e.started = true
		return false, e.executeCreateIndex(ctx, op)
	case *planner.DropIndexOp:
		if e.started {
			return false, nil
		}
		e.started = true
		return false, e.executeDropIndex(ctx, op)
	case *planner.InsertOp:
		if e.started {
			return false, nil
		}
		e.started = true
		return false, e.executeInsert(ctx, op)
	case *planner.UpdateOp:
		if e.started {
			return false, nil
		}
		e.started = true
		return false, e.executeUpdate(ctx, op)
	case *planner.DeleteOp:
		if e.started {
			return false, nil
		}
		e.started = true
		return false, e.executeDelete(ctx, op)
	default:
		return false, fmt.Errorf("executor: unsupported operator %T", e.plan.Root)
	}
}

// acquireLock is a helper to acquire a row-level lock via the transaction manager.
func (e *Executor) acquireLock(ctx context.Context, schemaName, table string, key []byte, lockType txn.LockType) error {
	if e.txnManager == nil || e.sessionTxn == nil || e.sessionTxn.Current == nil {
		return nil
	}
	txnID := e.sessionTxn.Current.ID
	// Key construction: pass full table identifier and row key string.
	// Manager will join them as "schema.table.key".
	return e.txnManager.AcquireLock(ctx, txnID, schemaName+"."+table, string(key), lockType)
}

func (e *Executor) nextSeqScan(ctx context.Context, op *planner.SeqScanOp) (bool, error) {
	if e.dataIter == nil {
		if err := e.initSeqScan(ctx, op); err != nil {
			return false, err
		}
	}
	if e.dataIter == nil {
		return false, nil
	}

	// Loop until we find a row that matches filter
	for {
		if !e.dataIter.Next() {
			err := e.dataIter.Err()
			e.closeIter()
			if err != nil {
				return false, err
			}
			return false, nil
		}

		key := append([]byte(nil), e.dataIter.Key()...)

		// LOCKING: Acquire Read Lock on the row
		if err := e.acquireLock(ctx, op.Schema, op.Table, key, txn.ReadLock); err != nil {
			return false, fmt.Errorf("executor: acquire read lock: %w", err)
		}

		val := append([]byte(nil), e.dataIter.Value()...)

		// Decode row if needed
		var decodedValues [][]byte
		if e.tableDef != nil {
			var err error
			decodedValues, err = decodeRow(val, len(e.tableDef.Columns))
			if err != nil {
				return false, fmt.Errorf("executor: decode row: %w", err)
			}
		}

		// Apply filter using FilterExpr if available, otherwise fall back to string filter
		if op.FilterExpr != nil {
			// Construct row with ALL columns for filtering
			var filterRow Row
			if e.tableDef != nil && decodedValues != nil {
				allCols := make([]string, len(e.tableDef.Columns))
				for i, col := range e.tableDef.Columns {
					allCols[i] = col.Name
				}
				filterRow = Row{Columns: allCols, Values: decodedValues}
			} else {
				// Legacy case or no table def - best effort using projected columns
				// This might fail if filter refers to non-projected columns
				tempValues := make([][]byte, len(e.rowSpecs))
				tempCols := make([]string, len(e.rowSpecs))
				for i, spec := range e.rowSpecs {
					tempCols[i] = spec.name
					if spec.kind == columnKey {
						tempValues[i] = key
					} else {
						tempValues[i] = val
					}
				}
				filterRow = Row{Columns: tempCols, Values: tempValues}
			}

			match, err := evaluateFilter(ctx, op.FilterExpr, filterRow, e.tableDef)
			if err != nil {
				return false, fmt.Errorf("executor: evaluate filter: %w", err)
			}
			if !match {
				continue
			}
		} else if op.Filter != "" && !e.matchesFilter(op.Schema, op.Table, key, val, op.Filter) {
			continue
		}

		// Build the projected row
		values := make([][]byte, len(e.rowSpecs))
		cols := make([]string, len(e.rowSpecs))
		for i, spec := range e.rowSpecs {
			cols[i] = spec.name
			switch spec.kind {
			case columnKey:
				values[i] = key
			case columnValue:
				if spec.index >= 0 && decodedValues != nil {
					if spec.index < len(decodedValues) {
						values[i] = decodedValues[spec.index]
					} else {
						values[i] = nil
					}
				} else {
					values[i] = val
				}
			}
		}

		e.current = Row{Columns: cols, Values: values}
		return true, nil
	}
}

func (e *Executor) initSeqScan(ctx context.Context, op *planner.SeqScanOp) error {
	if e.provider == nil {
		return errors.New("executor: storage provider not configured")
	}

	eng, err := e.provider.Engine(op.Schema, op.Table)
	if err != nil {
		return fmt.Errorf("executor: resolve engine: %w", err)
	}

	iter, err := eng.Scan(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("executor: scan %s.%s: %w", op.Schema, op.Table, err)
	}

	if e.catalog != nil {
		tableDef, err := e.catalog.GetTable(op.Schema, op.Table)
		if err != nil {
			iter.Close()
			return fmt.Errorf("executor: get table definition for %s.%s: %w", op.Schema, op.Table, err)
		}
		e.tableDef = tableDef
	}

	specs, err := buildColumnSpecs(e.tableDef, op.Columns)
	if err != nil {
		iter.Close()
		return err
	}

	e.dataIter = iter
	e.rowSpecs = specs
	return nil
}

// buildColumnSpecs creates projection specs from requested columns.
func buildColumnSpecs(tableDef *schema.TableDef, columns []string) ([]columnSpec, error) {
	if tableDef == nil {
		// Legacy/Raw mode: only allow "key" and "value"
		specs := make([]columnSpec, 0, len(columns))
		for _, col := range columns {
			switch strings.ToLower(col) {
			case "key":
				specs = append(specs, columnSpec{name: col, kind: columnKey})
			case "value":
				specs = append(specs, columnSpec{name: col, kind: columnValue, index: -1})
			default:
				return nil, fmt.Errorf("executor: column %q requires table definition", col)
			}
		}
		return specs, nil
	}

	if len(columns) == 0 || (len(columns) == 1 && columns[0] == "*") {
		// Expand * to all columns
		specs := make([]columnSpec, len(tableDef.Columns))
		for i, col := range tableDef.Columns {
			specs[i] = columnSpec{name: col.Name, kind: columnValue, index: i}
		}
		return specs, nil
	}

	specs := make([]columnSpec, 0, len(columns))
	for _, colName := range columns {
		// Check for special "key" column (if we want to expose raw key)
		// For now, let's treat "key" as a special column if it doesn't exist in table.
		// But usually we just want table columns.

		// Find column in table
		found := false
		for i, tableCol := range tableDef.Columns {
			if tableCol.Name == colName {
				specs = append(specs, columnSpec{name: colName, kind: columnValue, index: i})
				found = true
				break
			}
		}

		if !found {
			// Check for "key" or "value" raw access (for debugging/legacy tests)
			if strings.ToLower(colName) == "key" {
				specs = append(specs, columnSpec{name: "key", kind: columnKey})
				found = true
			} else if strings.ToLower(colName) == "value" {
				// Raw value access
				specs = append(specs, columnSpec{name: "value", kind: columnValue, index: -1})
				found = true
			}
		}

		if !found {
			return nil, fmt.Errorf("executor: column %q not found in table %s", colName, tableDef.Table)
		}
	}
	return specs, nil
}

// Row returns the most recent row produced by Next.
func (e *Executor) Row() Row {
	return e.current
}

// Close releases any open iterators.
func (e *Executor) Close() error {
	if e.closed {
		return nil
	}
	e.closed = true
	return e.closeIter()
}

func (e *Executor) closeIter() error {
	if e.dataIter == nil {
		return nil
	}
	err := e.dataIter.Close()
	e.dataIter = nil
	return err
}

func (e *Executor) executeTxn(ctx context.Context, op *planner.TxnOp) error {
	if e.txnManager == nil || e.sessionTxn == nil {
		return errors.New("executor: transaction manager not configured")
	}

	switch op.Action {
	case planner.TxnBegin:
		if st := e.sessionTxn.Current; st != nil && st.Status == txn.StatusActive {
			return ErrTxnAlreadyActive
		}
		state, err := e.txnManager.Begin(ctx)
		if err != nil {
			return err
		}
		e.sessionTxn.Current = &state
		return nil
	case planner.TxnCommit:
		if e.sessionTxn.Current == nil {
			return ErrNoActiveTxn
		}
		if err := e.txnManager.Commit(e.sessionTxn.Current.ID); err != nil {
			return err
		}
		e.sessionTxn.Current = nil
		return nil
	case planner.TxnRollback:
		if e.sessionTxn.Current == nil {
			return ErrNoActiveTxn
		}
		if err := e.txnManager.Rollback(e.sessionTxn.Current.ID); err != nil {
			return err
		}
		e.sessionTxn.Current = nil
		return nil
	default:
		return fmt.Errorf("executor: unknown txn action %s", op.Action)
	}
}

func (e *Executor) executeCreateTable(ctx context.Context, op *planner.CreateTableOp) error {
	if e.catalog == nil {
		return errors.New("executor: catalog not configured")
	}

	// Check if table already exists
	if _, err := e.catalog.GetTable(op.Schema, op.Table); err == nil {
		return fmt.Errorf("executor: table %s.%s already exists", op.Schema, op.Table)
	}

	// Create table in catalog
	if err := e.catalog.CreateTable(ctx, op.Schema, op.Table, op.Columns); err != nil {
		return fmt.Errorf("executor: create table %s.%s: %w", op.Schema, op.Table, err)
	}

	// Handle PRIMARY KEY
	for _, col := range op.Columns {
		if col.PrimaryKey {
			// Auto-create unique index for PRIMARY KEY
			indexName := fmt.Sprintf("__pk_%s_%s", op.Table, col.Name)
			// We use the internal CreateIndex logic (add to catalog + create engine)
			// But we can't call executeCreateIndex directly because we don't have a plan op.
			// We can duplicate logic or create a helper.
			// Duplicating logic for now as it's short.

			// 1. Add to catalog
			if err := e.catalog.CreateIndex(ctx, op.Schema, op.Table, indexName, []string{col.Name}, true, true); err != nil {
				return fmt.Errorf("executor: create pk index metadata: %w", err)
			}

			// 2. Create Index Engine
			if _, err := e.provider.Index(op.Schema, op.Table, indexName); err != nil {
				return fmt.Errorf("executor: create pk index engine: %w", err)
			}
		}
	}

	// CRITICAL: Flush catalog changes to disk immediately.
	// DDL operations bypass transactions, so we must manually flush buffer pool
	// and sync WAL to ensure catalog metadata is durable.
	if flusher, ok := e.provider.(interface{ FlushAll() error }); ok {
		if err := flusher.FlushAll(); err != nil {
			return fmt.Errorf("executor: flush catalog: %w", err)
		}
	}
	if syncer, ok := e.provider.(interface{ SyncAll() error }); ok {
		if err := syncer.SyncAll(); err != nil {
			return fmt.Errorf("executor: sync WAL: %w", err)
		}
	}

	return nil
}

func (e *Executor) executeDropTable(ctx context.Context, op *planner.DropTableOp) error {
	if e.catalog == nil {
		return errors.New("executor: catalog not configured")
	}

	// Remove from catalog
	if err := e.catalog.DropTable(ctx, op.Schema, op.Table); err != nil {
		return fmt.Errorf("executor: drop table %s.%s: %w", op.Schema, op.Table, err)
	}

	// Flush catalog changes to disk
	if flusher, ok := e.provider.(interface{ FlushAll() error }); ok {
		if err := flusher.FlushAll(); err != nil {
			return fmt.Errorf("executor: flush catalog: %w", err)
		}
	}
	if syncer, ok := e.provider.(interface{ SyncAll() error }); ok {
		if err := syncer.SyncAll(); err != nil {
			return fmt.Errorf("executor: sync WAL: %w", err)
		}
	}

	return nil
}

func (e *Executor) executeInsert(ctx context.Context, op *planner.InsertOp) error {
	logging.DebugContext(ctx, "[Executor] Starting INSERT execution for %s.%s with %d rows", op.Schema, op.Table, len(op.Values))

	if e.provider == nil {
		return errors.New("executor: storage provider not configured")
	}

	// Get storage engine
	eng, err := e.provider.Engine(op.Schema, op.Table)
	if err != nil {
		return fmt.Errorf("executor: resolve engine for %s.%s: %w", op.Schema, op.Table, err)
	}
	logging.DebugContext(ctx, "[Executor] Resolved storage engine for %s.%s", op.Schema, op.Table)

	// Get indexes once
	indexes, err := e.catalog.GetIndexes(op.Schema, op.Table)
	if err != nil {
		return fmt.Errorf("executor: get indexes: %w", err)
	}

	// Iterate over all rows in the batch
	for i, rowValues := range op.Values {
		// Encode row: [len1][val1][len2][val2]...
		var rowData []byte
		for _, value := range rowValues {
			// Write length prefix (4 bytes)
			lenBytes := make([]byte, 4)
			lenBytes[0] = byte(len(value))
			lenBytes[1] = byte(len(value) >> 8)
			lenBytes[2] = byte(len(value) >> 16)
			lenBytes[3] = byte(len(value) >> 24)
			rowData = append(rowData, lenBytes...)
			rowData = append(rowData, value...)
		}

		// Use first column as key (simple strategy for MVP)
		key := rowValues[0]

		// LOCKING: Acquire Write Lock on the row key
		if err := e.acquireLock(ctx, op.Schema, op.Table, key, txn.WriteLock); err != nil {
			return fmt.Errorf("executor: acquire write lock for row %d: %w", i, err)
		}

		// Check for duplicate primary key
		if _, err := eng.Get(ctx, key); err == nil {
			logging.WarnContext(ctx, "[Executor] Duplicate primary key detected: %x (row %d)", key, i)
			return fmt.Errorf("executor: duplicate primary key at row %d: %v", i, key)
		} else if !errors.Is(err, engine.ErrKeyNotFound) {
			return fmt.Errorf("executor: check duplicate key at row %d: %w", i, err)
		}

		// Store row
		if err := eng.Put(ctx, key, rowData); err != nil {
			return fmt.Errorf("executor: insert row %d into %s.%s: %w", i, op.Schema, op.Table, err)
		}

		// Maintain Indexes
		for _, idx := range indexes {
			if idx.IsPrimaryKey {
				continue // HeapEngine handles PK index maintenance
			}

			idxEng, err := e.provider.Index(op.Schema, op.Table, idx.IndexName)
			if err != nil {
				return fmt.Errorf("executor: open index %s: %w", idx.IndexName, err)
			}

			// Extract index key from current row values
			indexKey, err := extractIndexKeyFromOp(rowValues, op.Columns, idx.Columns)
			if err != nil {
				return fmt.Errorf("executor: extract key for index %s: %w", idx.IndexName, err)
			}

			// Check uniqueness if required
			if idx.Unique {
				_, found, err := idxEng.Search(indexKey)
				if err != nil {
					return fmt.Errorf("executor: check unique index %s: %w", idx.IndexName, err)
				}
				if found {
					return fmt.Errorf("duplicate value for unique index %s", idx.IndexName)
				}
			}

			if err := idxEng.Insert(indexKey, key); err != nil {
				return fmt.Errorf("executor: insert into index %s: %w", idx.IndexName, err)
			}
		}
	}

	logging.DebugContext(ctx, "[Executor] All %d rows inserted into storage engine", len(op.Values))

	// Flush data to disk for durability
	if flusher, ok := e.provider.(interface{ FlushAll() error }); ok {
		if err := flusher.FlushAll(); err != nil {
			return fmt.Errorf("executor: flush data: %w", err)
		}
	}
	if syncer, ok := e.provider.(interface{ SyncAll() error }); ok {
		if err := syncer.SyncAll(); err != nil {
			return fmt.Errorf("executor: sync WAL: %w", err)
		}
	}

	logging.InfoContext(ctx, "[Executor] INSERT completed successfully for %s.%s", op.Schema, op.Table)
	return nil
}

func (e *Executor) executeUpdate(ctx context.Context, op *planner.UpdateOp) error {
	if e.provider == nil {
		return errors.New("executor: storage provider not configured")
	}
	if e.catalog == nil {
		return errors.New("executor: catalog not configured")
	}

	// Get table schema
	tableDef, err := e.catalog.GetTable(op.Schema, op.Table)
	if err != nil {
		return fmt.Errorf("executor: table %s.%s not found: %w", op.Schema, op.Table, err)
	}

	// Get storage engine
	eng, err := e.provider.Engine(op.Schema, op.Table)
	if err != nil {
		return fmt.Errorf("executor: resolve engine for %s.%s: %w", op.Schema, op.Table, err)
	}

	// Scan table
	iter, err := eng.Scan(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("executor: scan %s.%s: %w", op.Schema, op.Table, err)
	}
	defer iter.Close()

	// Update matching rows
	for iter.Next() {
		key := iter.Key()

		// LOCKING: Acquire Write Lock on key
		// NOTE: Strictly speaking, we should acquire ReadLock for Scan, then upgrade to WriteLock if matches.
		// But Scan returns keys, so we can just grab WriteLock.
		// If we want to be more concurrency friendly, we check match first, but match requires reading value.
		// So: Read Value -> Check Match -> Acquire Write -> Put.
		// But reading value without lock is unsafe (dirty read).
		// So: Acquire Read (implied by 2PL during scan?) -> Read Value -> Check -> Upgrade to Write.
		// Since we iterate sequentially, acquiring Write lock here is safe and correct (Pessimistic).
		if err := e.acquireLock(ctx, op.Schema, op.Table, key, txn.WriteLock); err != nil {
			return fmt.Errorf("executor: acquire lock for update: %w", err)
		}

		rowData := iter.Value()

		// Decode row
		oldValues, err := decodeRow(rowData, len(tableDef.Columns))
		if err != nil {
			return fmt.Errorf("executor: decode row: %w", err)
		}

		// Evaluate filter using FilterExpr if available
		if op.FilterExpr != nil {
			// Build row with column names
			cols := make([]string, len(tableDef.Columns))
			for i, col := range tableDef.Columns {
				cols[i] = col.Name
			}

			tempRow := Row{Columns: cols, Values: oldValues}
			match, err := evaluateFilter(ctx, op.FilterExpr, tempRow, tableDef)
			if err != nil {
				return fmt.Errorf("executor: evaluate filter: %w", err)
			}
			if !match {
				continue
			}
		} else if op.Filter != "" && !e.matchesFilter(op.Schema, op.Table, key, rowData, op.Filter) {
			continue
		}

		// Create new values (copy)
		newValues := make([][]byte, len(oldValues))
		copy(newValues, oldValues)

		// Apply SET clauses
		for i, col := range tableDef.Columns {
			if newValue, exists := op.SetClauses[col.Name]; exists {
				newValues[i] = newValue
			}
		}

		// Re-encode row
		var newRowData []byte
		for _, value := range newValues {
			lenBytes := make([]byte, 4)
			lenBytes[0] = byte(len(value))
			lenBytes[1] = byte(len(value) >> 8)
			lenBytes[2] = byte(len(value) >> 16)
			lenBytes[3] = byte(len(value) >> 24)
			newRowData = append(newRowData, lenBytes...)
			newRowData = append(newRowData, value...)
		}

		// Update row
		if err := eng.Put(ctx, key, newRowData); err != nil {
			return fmt.Errorf("executor: update row: %w", err)
		}

		// Maintain Indexes
		indexes, err := e.catalog.GetIndexes(op.Schema, op.Table)
		if err != nil {
			return fmt.Errorf("executor: get indexes: %w", err)
		}

		for _, idx := range indexes {
			if idx.IsPrimaryKey {
				continue // HeapEngine handles PK index maintenance
			}

			idxEng, err := e.provider.Index(op.Schema, op.Table, idx.IndexName)
			if err != nil {
				return fmt.Errorf("executor: open index %s: %w", idx.IndexName, err)
			}

			oldKey, err := extractIndexKey(oldValues, idx.Columns, tableDef)
			if err != nil {
				return fmt.Errorf("executor: extract old key for index %s: %w", idx.IndexName, err)
			}

			newKey, err := extractIndexKey(newValues, idx.Columns, tableDef)
			if err != nil {
				return fmt.Errorf("executor: extract new key for index %s: %w", idx.IndexName, err)
			}

			if !bytes.Equal(oldKey, newKey) {
				if err := idxEng.Delete(oldKey); err != nil {
					// Ignore key not found?
				}

				// Check uniqueness if required
				if idx.Unique {
					_, found, err := idxEng.Search(newKey)
					if err != nil {
						return fmt.Errorf("executor: check unique index %s: %w", idx.IndexName, err)
					}
					if found {
						return fmt.Errorf("duplicate value for unique index %s", idx.IndexName)
					}
				}

				if err := idxEng.Insert(newKey, key); err != nil {
					return fmt.Errorf("executor: update index %s: %w", idx.IndexName, err)
				}
			}
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("executor: iterator error: %w", err)
	}

	// Flush data to disk for durability
	if flusher, ok := e.provider.(interface{ FlushAll() error }); ok {
		if err := flusher.FlushAll(); err != nil {
			return fmt.Errorf("executor: flush data: %w", err)
		}
	}
	if syncer, ok := e.provider.(interface{ SyncAll() error }); ok {
		if err := syncer.SyncAll(); err != nil {
			return fmt.Errorf("executor: sync WAL: %w", err)
		}
	}

	return nil
}

func (e *Executor) executeDelete(ctx context.Context, op *planner.DeleteOp) error {
	if e.provider == nil {
		return errors.New("executor: storage provider not configured")
	}

	// Get storage engine
	eng, err := e.provider.Engine(op.Schema, op.Table)
	if err != nil {
		return fmt.Errorf("executor: resolve engine for %s.%s: %w", op.Schema, op.Table, err)
	}

	// Scan table
	iter, err := eng.Scan(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("executor: scan %s.%s: %w", op.Schema, op.Table, err)
	}
	defer iter.Close()

	// Scan table and get table definition
	tableDef, err := e.catalog.GetTable(op.Schema, op.Table)
	if err != nil {
		return fmt.Errorf("executor: get table definition: %w", err)
	}

	// Collect keys to delete (can't delete while iterating)
	var keysToDelete [][]byte
	for iter.Next() {
		key := iter.Key()

		// LOCKING: Acquire Write Lock
		if err := e.acquireLock(ctx, op.Schema, op.Table, key, txn.WriteLock); err != nil {
			return fmt.Errorf("executor: acquire lock for delete: %w", err)
		}

		rowData := iter.Value()

		// Evaluate filter using FilterExpr if available
		if op.FilterExpr != nil {
			// Decode row for filtering
			values, err := decodeRow(rowData, len(tableDef.Columns))
			if err != nil {
				return fmt.Errorf("executor: decode row for filter: %w", err)
			}

			// Build row with column names
			cols := make([]string, len(tableDef.Columns))
			for i, col := range tableDef.Columns {
				cols[i] = col.Name
			}

			tempRow := Row{Columns: cols, Values: values}
			match, err := evaluateFilter(ctx, op.FilterExpr, tempRow, tableDef)
			if err != nil {
				return fmt.Errorf("executor: evaluate filter: %w", err)
			}
			if !match {
				continue
			}
		} else if op.Filter != "" && !e.matchesFilter(op.Schema, op.Table, key, rowData, op.Filter) {
			continue
		}

		keysToDelete = append(keysToDelete, append([]byte(nil), key...))
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("executor: iterator error: %w", err)
	}

	// Delete collected keys
	for _, key := range keysToDelete {
		// Maintain Indexes: Need to delete from indexes.
		// We need the row data to know the index keys.
		// I should have collected rowData too?
		// Or fetch it again? Fetching again is safer but slower.
		// keysToDelete only has keys.

		rowData, err := eng.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("executor: get row for delete: %w", err)
		}

		// Decode row
		tableDef, err := e.catalog.GetTable(op.Schema, op.Table)
		if err != nil {
			return err
		}
		values, err := decodeRow(rowData, len(tableDef.Columns))
		if err != nil {
			return err
		}

		indexes, err := e.catalog.GetIndexes(op.Schema, op.Table)
		if err != nil {
			return err
		}

		for _, idx := range indexes {
			if idx.IsPrimaryKey {
				continue // HeapEngine handles PK index maintenance
			}

			idxEng, err := e.provider.Index(op.Schema, op.Table, idx.IndexName)
			if err != nil {
				return err
			}

			indexKey, err := extractIndexKey(values, idx.Columns, tableDef)
			if err != nil {
				return err
			}

			if err := idxEng.Delete(indexKey); err != nil {
				// Ignore key not found?
				// return err
			}
		}

		if err := eng.Delete(ctx, key); err != nil {
			return fmt.Errorf("executor: delete row: %w", err)
		}
	}

	// Flush data to disk for durability
	if flusher, ok := e.provider.(interface{ FlushAll() error }); ok {
		if err := flusher.FlushAll(); err != nil {
			return fmt.Errorf("executor: flush data: %w", err)
		}
	}
	if syncer, ok := e.provider.(interface{ SyncAll() error }); ok {
		if err := syncer.SyncAll(); err != nil {
			return fmt.Errorf("executor: sync WAL: %w", err)
		}
	}

	return nil
}

// matchesFilter performs simple string-based WHERE clause matching (MVP).
// For example, "id = 1" checks if the column "id" has value "1".
func (e *Executor) matchesFilter(schemaName, table string, key, rowData []byte, filter string) bool {
	if filter == "" {
		return true
	}

	// Parse "col = val"
	parts := strings.Split(filter, "=")
	if len(parts) != 2 {
		// Fallback to naive check if not simple equality
		keyStr := string(key)
		rowStr := string(rowData)
		combined := keyStr + " " + rowStr
		return strings.Contains(combined, filter)
	}

	colName := strings.TrimSpace(parts[0])
	valStr := strings.TrimSpace(parts[1])
	// Remove quotes
	if len(valStr) >= 2 && (valStr[0] == '\'' || valStr[0] == '"') {
		valStr = valStr[1 : len(valStr)-1]
	}

	// Get table def
	tableDef, err := e.catalog.GetTable(schemaName, table)
	if err != nil {
		return false // Fail safe
	}

	// Decode row
	values, err := decodeRow(rowData, len(tableDef.Columns))
	if err != nil {
		return false
	}

	// Find column
	idx := -1
	var colDef *schema.ColumnDef
	for i, col := range tableDef.Columns {
		if col.Name == colName {
			idx = i
			colDef = &col
			break
		}
	}

	if idx == -1 {
		// Column not found, maybe it's "key"?
		// For MVP, ignore.
		return false
	}

	// Compare values
	// We need to encode valStr to compare with stored bytes
	encodedVal, err := colDef.Type.Encode(valStr)
	if err != nil {
		return false
	}

	return string(values[idx]) == string(encodedVal)
}

// decodeRow decodes a row from storage format.
func decodeRow(data []byte, numColumns int) ([][]byte, error) {
	values := make([][]byte, 0, numColumns)
	offset := 0

	for i := 0; i < numColumns; i++ {
		if offset+4 > len(data) {
			return nil, fmt.Errorf("executor: corrupt row data (missing length prefix)")
		}

		// Read length (little-endian)
		length := int(data[offset]) | int(data[offset+1])<<8 | int(data[offset+2])<<16 | int(data[offset+3])<<24
		offset += 4

		if offset+length > len(data) {
			return nil, fmt.Errorf("executor: corrupt row data (truncated value)")
		}

		value := make([]byte, length)
		copy(value, data[offset:offset+length])
		values = append(values, value)
		offset += length
	}

	return values, nil
}

func (e *Executor) executeCreateIndex(ctx context.Context, op *planner.CreateIndexOp) error {
	if e.catalog == nil {
		return errors.New("executor: catalog not configured")
	}
	if e.provider == nil {
		return errors.New("executor: storage provider not configured")
	}

	// 1. Add index to catalog
	// Note: We pass isPrimaryKey=false here. PRIMARY KEYs are handled in CreateTable.
	// If this was a "CREATE UNIQUE INDEX", unique=true.
	if err := e.catalog.CreateIndex(ctx, op.Schema, op.Table, op.IndexName, op.Columns, op.Unique, false); err != nil {
		return fmt.Errorf("executor: create index metadata: %w", err)
	}

	// 2. Get Index Engine (creates the file if needed)
	idxEng, err := e.provider.Index(op.Schema, op.Table, op.IndexName)
	if err != nil {
		return fmt.Errorf("executor: open index engine: %w", err)
	}

	// 3. Populate Index (Bulk Load)
	// Scan the table and insert every row into the index.
	tableEng, err := e.provider.Engine(op.Schema, op.Table)
	if err != nil {
		return fmt.Errorf("executor: open table engine: %w", err)
	}

	iter, err := tableEng.Scan(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("executor: scan table: %w", err)
	}
	defer iter.Close()

	tableDef, err := e.catalog.GetTable(op.Schema, op.Table)
	if err != nil {
		return fmt.Errorf("executor: get table def: %w", err)
	}

	for iter.Next() {
		rowKey := iter.Key()
		rowData := iter.Value()

		// Decode row to get column values
		values, err := decodeRow(rowData, len(tableDef.Columns))
		if err != nil {
			return fmt.Errorf("executor: decode row: %w", err)
		}

		// Extract index key
		indexKey, err := extractIndexKey(values, op.Columns, tableDef)
		if err != nil {
			return fmt.Errorf("executor: extract index key: %w", err)
		}

		// Insert into index
		// Value in index is the rowKey (pointer to heap)
		if err := idxEng.Insert(indexKey, rowKey); err != nil {
			return fmt.Errorf("executor: insert into index: %w", err)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("executor: iterator error: %w", err)
	}

	// Flush index data to disk
	if flusher, ok := e.provider.(interface{ FlushAll() error }); ok {
		if err := flusher.FlushAll(); err != nil {
			return fmt.Errorf("executor: flush index: %w", err)
		}
	}
	if syncer, ok := e.provider.(interface{ SyncAll() error }); ok {
		if err := syncer.SyncAll(); err != nil {
			return fmt.Errorf("executor: sync WAL: %w", err)
		}
	}

	return nil
}

func (e *Executor) executeDropIndex(ctx context.Context, op *planner.DropIndexOp) error {
	if e.catalog == nil {
		return errors.New("executor: catalog not configured")
	}

	// Remove from catalog
	if err := e.catalog.DropIndex(ctx, op.Schema, op.Table, op.IndexName); err != nil {
		return fmt.Errorf("executor: drop index: %w", err)
	}

	// Note: Physical file deletion is not yet implemented in EngineProvider interface.
	// For MVP, leaving the file is acceptable (or we could extend interface).

	// Flush catalog changes to disk
	if flusher, ok := e.provider.(interface{ FlushAll() error }); ok {
		if err := flusher.FlushAll(); err != nil {
			return fmt.Errorf("executor: flush catalog: %w", err)
		}
	}
	if syncer, ok := e.provider.(interface{ SyncAll() error }); ok {
		if err := syncer.SyncAll(); err != nil {
			return fmt.Errorf("executor: sync WAL: %w", err)
		}
	}

	return nil
}

func (e *Executor) nextIndexScan(ctx context.Context, op *planner.IndexScanOp) (bool, error) {
	if e.dataIter == nil {
		if err := e.initIndexScan(ctx, op); err != nil {
			return false, err
		}
	}
	if e.dataIter == nil {
		return false, nil
	}

	// Use loop instead of recursion to prevent stack overflow on long scans
	for {
		// Iterate over index
		if !e.dataIter.Next() {
			err := e.dataIter.Err()
			e.closeIter()
			if err != nil {
				return false, err
			}
			return false, nil
		}

		// Index Key (columns) -> Index Value (Row Key)
		rowKey := e.dataIter.Value()

		// Fetch actual row from table heap
		// We need the table engine for this.
		// Optimization: Cache table engine in Executor? For now, resolve it.
		tableEng, err := e.provider.Engine(op.Schema, op.Table)
		if err != nil {
			return false, fmt.Errorf("executor: resolve table engine: %w", err)
		}

		var rowData []byte
		if e.isPKScan {
			// PK index stores Location, not User Key
			if heapEng, ok := tableEng.(interface {
				GetByLocation(context.Context, []byte) ([]byte, error)
			}); ok {
				rowData, err = heapEng.GetByLocation(ctx, rowKey)
			} else {
				return false, fmt.Errorf("executor: engine does not support GetByLocation")
			}
		} else {
			rowData, err = tableEng.Get(ctx, rowKey)
		}

		if err != nil {
			return false, fmt.Errorf("executor: fetch row from heap: %w", err)
		}

		// LOCKING: Acquire Read Lock on the row
		if err := e.acquireLock(ctx, op.Schema, op.Table, rowKey, txn.ReadLock); err != nil {
			return false, fmt.Errorf("executor: acquire read lock: %w", err)
		}

		// We have the row data, now project it.
		// Note: We are reusing the same projection logic as SeqScan, but we need to
		// ensure rowSpecs are set up.
		// Wait, SeqScan sets up rowSpecs. IndexScan needs to do the same.

		// Decode row if needed
		var decodedValues [][]byte
		if e.tableDef != nil {
			var err error
			decodedValues, err = decodeRow(rowData, len(e.tableDef.Columns))
			if err != nil {
				return false, fmt.Errorf("executor: decode row: %w", err)
			}
		}

		key := append([]byte(nil), rowKey...)
		val := append([]byte(nil), rowData...)

		// Apply filter using FilterExpr if available
		if op.FilterExpr != nil {
			// Construct row with ALL columns for filtering
			var filterRow Row
			if e.tableDef != nil && decodedValues != nil {
				allCols := make([]string, len(e.tableDef.Columns))
				for i, col := range e.tableDef.Columns {
					allCols[i] = col.Name
				}
				filterRow = Row{Columns: allCols, Values: decodedValues}
			} else {
				// Fallback
				tempCols := []string{"key", "value"}
				tempValues := [][]byte{key, val}
				filterRow = Row{Columns: tempCols, Values: tempValues}
			}

			match, err := evaluateFilter(ctx, op.FilterExpr, filterRow, e.tableDef)
			if err != nil {
				return false, fmt.Errorf("executor: evaluate filter: %w", err)
			}
			if !match {
				continue
			}
		} else if op.Filter != "" && !e.matchesFilter(op.Schema, op.Table, rowKey, rowData, op.Filter) {
			continue
		}

		// Build projected row
		values := make([][]byte, len(e.rowSpecs))
		cols := make([]string, len(e.rowSpecs))
		for i, spec := range e.rowSpecs {
			cols[i] = spec.name
			switch spec.kind {
			case columnKey:
				values[i] = key
			case columnValue:
				if spec.index >= 0 && decodedValues != nil {
					if spec.index < len(decodedValues) {
						values[i] = decodedValues[spec.index]
					} else {
						values[i] = nil
					}
				} else {
					values[i] = val
				}
			}
		}
		e.current = Row{Columns: cols, Values: values}
		return true, nil
	}
}

func (e *Executor) initIndexScan(ctx context.Context, op *planner.IndexScanOp) error {
	if e.provider == nil {
		return errors.New("executor: storage provider not configured")
	}

	// Get Index Engine
	idxEng, err := e.provider.Index(op.Schema, op.Table, op.IndexName)
	if err != nil {
		return fmt.Errorf("executor: resolve index engine: %w", err)
	}

	// Get Table Def first
	tableDef, err := e.catalog.GetTable(op.Schema, op.Table)
	if err != nil {
		return err
	}
	e.tableDef = tableDef

	// Check if this is a PK index scan
	indexes, err := e.catalog.GetIndexes(op.Schema, op.Table)
	if err != nil {
		return err
	}
	for _, idx := range indexes {
		if idx.IndexName == op.IndexName {
			e.isPKScan = idx.IsPrimaryKey
			break
		}
	}

	// Determine scan range
	// For "col = val", start = val, end = val (or val + 1 for exclusive)
	// The current planner heuristic puts "col = val" in Filter.
	// We need to parse the filter to get the key.
	// This is a bit hacky for MVP. Real planner should pass StartKey/EndKey.
	// Let's try to extract it from Filter if StartKey is empty.
	var startKey, endKey []byte
	if len(op.StartKey) == 0 && op.Filter != "" {
		parts := strings.Split(op.Filter, "=")
		if len(parts) == 2 {
			valStr := strings.TrimSpace(parts[1])
			// Remove quotes
			if len(valStr) >= 2 && (valStr[0] == '\'' || valStr[0] == '"') {
				valStr = valStr[1 : len(valStr)-1]
			}

			// We need to encode this value to match the index key format.
			// We need the column type.
			// tableDef is already fetched above.

			// Find the indexed column
			// Assuming single column index for now
			// colName := op.Columns[0] // Wait, op.Columns is projection.
			// We need the index definition to know which column is indexed.
			indexes, err := e.catalog.GetIndexes(op.Schema, op.Table)
			if err != nil {
				return err
			}
			var targetIdx *schema.IndexDef
			for _, idx := range indexes {
				if idx.IndexName == op.IndexName {
					targetIdx = &idx
					break
				}
			}
			if targetIdx == nil {
				return fmt.Errorf("index %s not found", op.IndexName)
			}

			// Find column def
			var colDef *schema.ColumnDef
			for _, col := range tableDef.Columns {
				if col.Name == targetIdx.Columns[0] {
					colDef = &col
					break
				}
			}

			if colDef != nil {
				encoded, err := colDef.Type.Encode(valStr)
				if err != nil {
					return err
				}
				startKey = encoded
				endKey = encoded // Point lookup
				// Note: RangeScan usually expects [start, end).
				// If we want exact match, we might need a different method or handle end key carefully.
				// BTree RangeScan implementation:
				// if end != nil && bytes.Compare(key, end) >= 0 { return false }
				// So if start == end, it returns nothing.
				// We need end to be slightly larger than start, or use a specific "Prefix" scan or "Exact" scan.
				// Or we can just use Search() if it's a point lookup!

				// Let's use Search() if it's an exact match?
				// But Iterator interface is nice.
				// Let's append a 0xFF byte to endKey to make it inclusive?
				// Or just use Search if startKey == endKey.

				// Actually, let's assume RangeScan handles [start, end] or we adjust.
				// The BTree implementation I saw:
				// if it.end != nil && bytes.Compare(key, it.end) >= 0 { return false }
				// So it is [start, end).
				// To match "val", we need end to be > val.
				// We can append 0x00 to endKey? No, that might be smaller depending on encoding.
				// For strings, "val" < "val\0".
				// For integers, it depends on encoding.

				// Hack: append a byte to endKey.
				// Ensure we don't modify startKey if they share backing array
				endKey = append(append([]byte(nil), encoded...), 0xFF)

				// fmt.Printf("IndexScan: Filter=%s, Start=%s, End=%x\n", op.Filter, string(startKey), endKey)
			}
		}
	}

	iter, err := idxEng.RangeScan(ctx, startKey, endKey)
	if err != nil {
		return fmt.Errorf("executor: index scan: %w", err)
	}

	specs, err := buildColumnSpecs(tableDef, op.Columns)
	if err != nil {
		iter.Close()
		return err
	}

	e.dataIter = iter
	e.rowSpecs = specs
	return nil
}

// extractIndexKey extracts and encodes the key for an index from a row of values.
func extractIndexKey(values [][]byte, indexCols []string, tableDef *schema.TableDef) ([]byte, error) {
	// MVP: Only support single-column indexes
	if len(indexCols) == 0 {
		return nil, errors.New("empty index columns")
	}
	colName := indexCols[0]

	// Find column index in table
	idx := -1
	for i, col := range tableDef.Columns {
		if col.Name == colName {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, fmt.Errorf("column %s not found in table", colName)
	}

	if idx >= len(values) {
		return nil, fmt.Errorf("row values missing column %s", colName)
	}

	return values[idx], nil
}

// extractIndexKeyFromOp extracts index key from INSERT values (which align with op.Columns).
func extractIndexKeyFromOp(values [][]byte, opColumns []string, indexCols []string) ([]byte, error) {
	if len(indexCols) == 0 {
		return nil, errors.New("empty index columns")
	}
	colName := indexCols[0]

	// Find column index in op.Columns
	idx := -1
	for i, name := range opColumns {
		if name == colName {
			idx = i
			break
		}
	}
	if idx == -1 {
		// If column not in INSERT list, it must be default/null?
		// For now, assume it's required or we fail.
		return nil, fmt.Errorf("index column %s not in insert list", colName)
	}

	return values[idx], nil
}

var (
	ErrTxnAlreadyActive = errors.New("executor: transaction already active")
	ErrNoActiveTxn      = errors.New("executor: no active transaction")
)
