package buffer

import (
	"testing"

	"github.com/hainn191297/myDb/internal/storage/page"
)

func TestGlobalPoolUnpinMarksDirtyAndFlushes(t *testing.T) {
	pool := NewGlobalPool(1, nil)

	reader := func(pid page.PageID) (*page.Page, error) {
		return page.NewPage(pid), nil
	}

	var flushed []page.PageID
	writer := func(p *page.Page) error {
		flushed = append(flushed, p.ID)
		return nil
	}

	if _, err := pool.GetPage(10, 1, reader, writer); err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	pool.Unpin(10, 1, true)

	err := pool.FlushMatching(
		func(fid uint32) bool { return fid == 10 },
		func(fid uint32, p *page.Page) error {
			if fid != 10 {
				t.Fatalf("unexpected fileID %d", fid)
			}
			return writer(p)
		},
		func(uint32) error { return nil },
	)
	if err != nil {
		t.Fatalf("FlushMatching: %v", err)
	}

	if len(flushed) != 1 || flushed[0] != 1 {
		t.Fatalf("expected page 1 flushed once, got %v", flushed)
	}

	pool.mu.Lock()
	elem, ok := pool.frames[PageKey{FileID: 10, PID: 1}]
	if !ok {
		t.Fatalf("frame missing after flush")
	}
	frame := elem.Value.(*Frame)
	if frame.Dirty {
		t.Fatalf("expected frame clean after flush")
	}
	pool.mu.Unlock()
}

func TestGlobalPoolEvictsDirtyFrameByFlushing(t *testing.T) {
	pool := NewGlobalPool(1, nil)

	reader := func(pid page.PageID) (*page.Page, error) {
		return page.NewPage(pid), nil
	}

	var flushed []page.PageID
	writer := func(p *page.Page) error {
		flushed = append(flushed, p.ID)
		return nil
	}

	if _, err := pool.GetPage(42, 1, reader, writer); err != nil {
		t.Fatalf("GetPage pid1: %v", err)
	}
	pool.Unpin(42, 1, true)

	if _, err := pool.GetPage(42, 2, reader, writer); err != nil {
		t.Fatalf("GetPage pid2: %v", err)
	}

	if len(flushed) != 1 || flushed[0] != 1 {
		t.Fatalf("expected eviction flush of pid1, got %v", flushed)
	}

	pool.mu.Lock()
	if _, ok := pool.frames[PageKey{FileID: 42, PID: 1}]; ok {
		t.Fatalf("expected pid1 frame removed after eviction")
	}
	pool.mu.Unlock()
}

func TestFlushMatchingLeavesOtherDirtyFrames(t *testing.T) {
	pool := NewGlobalPool(2, nil)

	reader := func(pid page.PageID) (*page.Page, error) {
		return page.NewPage(pid), nil
	}

	if _, err := pool.GetPage(1, 1, reader, func(p *page.Page) error { return nil }); err != nil {
		t.Fatalf("GetPage file1: %v", err)
	}
	pool.Unpin(1, 1, true)

	if _, err := pool.GetPage(2, 1, reader, func(p *page.Page) error { return nil }); err != nil {
		t.Fatalf("GetPage file2: %v", err)
	}
	pool.Unpin(2, 1, true)

	var flushed []uint32
	err := pool.FlushMatching(
		func(fid uint32) bool { return fid == 1 },
		func(fid uint32, p *page.Page) error {
			flushed = append(flushed, fid)
			return nil
		},
		func(uint32) error { return nil },
	)
	if err != nil {
		t.Fatalf("FlushMatching: %v", err)
	}

	if len(flushed) != 1 || flushed[0] != 1 {
		t.Fatalf("expected only file 1 flushed, got %v", flushed)
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	frame, ok := pool.frames[PageKey{FileID: 2, PID: 1}]
	if !ok {
		t.Fatalf("missing frame for file 2")
	}
	if !frame.Value.(*Frame).Dirty {
		t.Fatalf("frame for file 2 should remain dirty")
	}
}
