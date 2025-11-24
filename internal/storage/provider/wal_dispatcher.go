package provider

import (
	"fmt"

	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/page"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

// walDispatcher adapts wal.Manager to wal.Logger by routing writes based on FileID.
type walDispatcher struct {
	tm  *engine.TableManager
	mgr *wal.Manager
}

func newWalDispatcher(tm *engine.TableManager, mgr *wal.Manager) *walDispatcher {
	return &walDispatcher{tm: tm, mgr: mgr}
}

func (w *walDispatcher) Append(fileID uint32, pid page.PageID, data []byte) error {
	schema, table, ok := w.tm.LookupName(fileID)
	if !ok {
		return fmt.Errorf("wal: unknown fileID %d", fileID)
	}
	logger, err := w.mgr.Open(schema, table)
	if err != nil {
		return fmt.Errorf("wal: open logger for %s.%s: %w", schema, table, err)
	}
	return logger.Append(fileID, pid, data)
}

func (w *walDispatcher) Sync() error {
	return w.mgr.SyncAll()
}

func (w *walDispatcher) Close() error {
	return w.mgr.Close()
}
