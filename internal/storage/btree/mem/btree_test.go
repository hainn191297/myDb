package mem

import (
	"context"
	"fmt"
	"testing"
)

func TestBTreeInsertAndSearch(t *testing.T) {
	bt := New()

	// Insert some keys
	keys := []string{"5", "3", "7", "1", "9", "2", "8", "4", "6"}
	for _, k := range keys {
		err := bt.Insert([]byte(k), []byte("value_"+k))
		if err != nil {
			t.Fatalf("Insert(%s) failed: %v", k, err)
		}
	}

	// Search for each key
	for _, k := range keys {
		val, found, err := bt.Search([]byte(k))
		if err != nil {
			t.Fatalf("Search(%s) error: %v", k, err)
		}
		if !found {
			t.Errorf("Search(%s): key not found", k)
		}
		expected := "value_" + k
		if string(val) != expected {
			t.Errorf("Search(%s): got %s, want %s", k, val, expected)
		}
	}

	// Search for non-existent key
	_, found, _ := bt.Search([]byte("99"))
	if found {
		t.Error("Search(99): should not be found")
	}
}

func TestBTreeInsertDuplicate(t *testing.T) {
	bt := New()

	err := bt.Insert([]byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("First insert failed: %v", err)
	}

	err = bt.Insert([]byte("key1"), []byte("value2"))
	if err == nil {
		t.Error("Expected duplicate key error, got nil")
	}
}

func TestBTreeNodeSplitting(t *testing.T) {
	bt := New()

	// Insert enough keys to cause splits
	numKeys := 300
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key%05d", i)
		val := fmt.Sprintf("value%05d", i)
		err := bt.Insert([]byte(key), []byte(val))
		if err != nil {
			t.Fatalf("Insert(%s) failed: %v", key, err)
		}
	}

	// Verify all keys are still searchable
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key%05d", i)
		val, found, err := bt.Search([]byte(key))
		if err != nil {
			t.Fatalf("Search(%s) error: %v", key, err)
		}
		if !found {
			t.Errorf("Search(%s): key not found after splits", key)
		}
		expected := fmt.Sprintf("value%05d", i)
		if string(val) != expected {
			t.Errorf("Search(%s): got %s, want %s", key, val, expected)
		}
	}

	// Verify tree structure (root should not be leaf after splits)
	if bt.root.IsLeaf {
		t.Error("Root should not be leaf after inserting many keys")
	}
}

func TestBTreeDelete(t *testing.T) {
	bt := New()

	// Insert keys
	keys := []string{"1", "2", "3", "4", "5"}
	for _, k := range keys {
		bt.Insert([]byte(k), []byte("val_"+k))
	}

	// Delete a key
	err := bt.Delete([]byte("3"))
	if err != nil {
		t.Fatalf("Delete(3) failed: %v", err)
	}

	// Verify it's gone
	_, found, _ := bt.Search([]byte("3"))
	if found {
		t.Error("Key 3 should be deleted")
	}

	// Verify others still exist
	for _, k := range []string{"1", "2", "4", "5"} {
		_, found, _ := bt.Search([]byte(k))
		if !found {
			t.Errorf("Key %s should still exist", k)
		}
	}
}

func TestBTreeRangeScan(t *testing.T) {
	bt := New()
	ctx := context.Background()

	// Insert keys 0-9
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("%d", i)
		bt.Insert([]byte(key), []byte("value_"+key))
	}

	// Range scan [3, 7)
	iter, err := bt.RangeScan(ctx, []byte("3"), []byte("7"))
	if err != nil {
		t.Fatalf("RangeScan failed: %v", err)
	}
	defer iter.Close()

	expected := []string{"3", "4", "5", "6"}
	var results []string

	for iter.Next() {
		key := iter.Key()
		results = append(results, string(key))
		iter.Value() // Consume value
	}

	if iter.Err() != nil {
		t.Fatalf("Iterator error: %v", iter.Err())
	}

	if len(results) != len(expected) {
		t.Fatalf("Expected %d results, got %d: %v", len(expected), len(results), results)
	}

	for i, exp := range expected {
		if results[i] != exp {
			t.Errorf("Result[%d]: got %s, want %s", i, results[i], exp)
		}
	}
}

func TestBTreeRangeScanNoBounds(t *testing.T) {
	bt := New()
	ctx := context.Background()

	// Insert keys
	keys := []string{"1", "3", "5", "7", "9"}
	for _, k := range keys {
		bt.Insert([]byte(k), []byte("val_"+k))
	}

	// Scan all (nil start/end)
	iter, _ := bt.RangeScan(ctx, nil, nil)
	defer iter.Close()

	count := 0
	for iter.Next() {
		iter.Value()
		count++
	}

	if count != len(keys) {
		t.Errorf("Expected %d keys, got %d", len(keys), count)
	}
}

func TestBTreeDeleteRebalanceBorrow(t *testing.T) {
	bt := newWithOrder(2) // small order to trigger split/borrow easily

	keys := []string{"1", "2", "3", "4"}
	for _, k := range keys {
		if err := bt.Insert([]byte(k), []byte("value_"+k)); err != nil {
			t.Fatalf("Insert(%s): %v", k, err)
		}
	}

	// Delete two keys from the left leaf to force underflow and borrowing
	for _, k := range []string{"1", "2"} {
		if err := bt.Delete([]byte(k)); err != nil {
			t.Fatalf("Delete(%s): %v", k, err)
		}
	}

	// Remaining keys should still be present
	for _, k := range []string{"3", "4"} {
		val, found, err := bt.Search([]byte(k))
		if err != nil || !found || string(val) != "value_"+k {
			t.Fatalf("Search(%s) after rebalance: found=%v err=%v val=%s", k, found, err, val)
		}
	}

	// Deleted keys should be absent
	for _, k := range []string{"1", "2"} {
		if _, found, _ := bt.Search([]byte(k)); found {
			t.Errorf("Expected %s to be deleted", k)
		}
	}

	if bt.root == nil {
		t.Fatalf("root should not be nil after borrow rebalance")
	}
}

func TestBTreeDeleteRebalanceMerge(t *testing.T) {
	bt := newWithOrder(2)

	keys := []string{"1", "2", "3", "4"}
	for _, k := range keys {
		if err := bt.Insert([]byte(k), []byte("value_"+k)); err != nil {
			t.Fatalf("Insert(%s): %v", k, err)
		}
	}

	// Delete three keys to force a merge (siblings at minimum)
	for _, k := range []string{"1", "2", "3"} {
		if err := bt.Delete([]byte(k)); err != nil {
			t.Fatalf("Delete(%s): %v", k, err)
		}
	}

	// Only "4" should remain
	val, found, err := bt.Search([]byte("4"))
	if err != nil || !found || string(val) != "value_4" {
		t.Fatalf("Search(4) after merge: found=%v err=%v val=%s", found, err, val)
	}

	for _, k := range []string{"1", "2", "3"} {
		if _, found, _ := bt.Search([]byte(k)); found {
			t.Errorf("Expected %s to be deleted after merge", k)
		}
	}

	if bt.root == nil || !bt.root.IsLeaf {
		t.Fatalf("root should collapse to leaf after merges")
	}
}

func TestBTreeEmpty(t *testing.T) {
	bt := New()

	// Search in empty tree
	_, found, err := bt.Search([]byte("key"))
	if err != nil {
		t.Errorf("Search in empty tree: %v", err)
	}
	if found {
		t.Error("Should not find key in empty tree")
	}

	// Range scan empty tree
	iter, err := bt.RangeScan(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("RangeScan empty tree: %v", err)
	}
	defer iter.Close()

	if iter.Next() {
		t.Error("Empty tree should have no elements")
	}
}
