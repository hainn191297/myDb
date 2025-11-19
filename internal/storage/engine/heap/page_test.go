package heap

import (
	"bytes"
	"testing"

	"github.com/hainn191297/myDb/internal/storage/page"
)

func TestHeapPageInsertAndIterate(t *testing.T) {
	hp := NewEmptyPage(page.PageID(1))

	freeBefore := hp.FreeSpace()
	tupleA := []byte("tuple-A")
	slotA, err := hp.InsertTuple(tupleA)
	if err != nil {
		t.Fatalf("insert A: %v", err)
	}
	if slotA != 0 {
		t.Fatalf("expected slot 0, got %d", slotA)
	}

	tupleB := []byte("tuple-BB")
	slotB, err := hp.InsertTuple(tupleB)
	if err != nil {
		t.Fatalf("insert B: %v", err)
	}
	if slotB != 1 {
		t.Fatalf("expected slot 1, got %d", slotB)
	}

	wantFree := freeBefore - (len(tupleA) + 2) - (len(tupleB) + 2)
	if hp.FreeSpace() != wantFree {
		t.Fatalf("free space mismatch: got %d want %d", hp.FreeSpace(), wantFree)
	}

	var visited []int
	hp.ForEach(func(slot int, off uint16) bool {
		visited = append(visited, slot)
		var expected []byte
		switch slot {
		case slotA:
			expected = tupleA
		case slotB:
			expected = tupleB
		default:
			t.Fatalf("unexpected slot %d", slot)
		}
		data := hp.raw.Data[off : int(off)+len(expected)]
		if !bytes.Equal(data, expected) {
			t.Fatalf("tuple mismatch slot %d: got %q want %q", slot, data, expected)
		}
		return true
	})

	if len(visited) != 2 {
		t.Fatalf("expected 2 live tuples, got %d", len(visited))
	}
}

func TestHeapPageDeleteTuple(t *testing.T) {
	hp := NewEmptyPage(page.PageID(2))

	slotA, err := hp.InsertTuple([]byte{1, 2, 3})
	if err != nil {
		t.Fatalf("insert A: %v", err)
	}
	slotB, err := hp.InsertTuple([]byte{4, 5, 6, 7})
	if err != nil {
		t.Fatalf("insert B: %v", err)
	}

	if err := hp.DeleteTuple(slotA); err != nil {
		t.Fatalf("DeleteTuple: %v", err)
	}
	if hp.IsSlotUsed(slotA) {
		t.Fatalf("expected slot A freed")
	}

	var slots []int
	hp.ForEach(func(slot int, off uint16) bool {
		slots = append(slots, slot)
		if slot == slotA {
			t.Fatalf("deleted slot visited")
		}
		buf := hp.raw.Data[off : int(off)+4]
		if slot == slotB && !bytes.Equal(buf, []byte{4, 5, 6, 7}) {
			t.Fatalf("remaining tuple corrupted: %v", buf)
		}
		return true
	})
	if len(slots) != 1 || slots[0] != slotB {
		t.Fatalf("expected only slotB visited, got %v", slots)
	}

	if err := hp.DeleteTuple(999); err == nil {
		t.Fatalf("expected error deleting invalid slot")
	}
}
