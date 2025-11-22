// Package mem implements a pure in-memory B+ Tree used for algorithm
// exploration and tests. It omits persistence, buffer/WAL integration,
// and full delete underflow handling.
package mem

import (
	"bytes"
	"context"
	"errors"
	"fmt"
)

// Order defines the default maximum number of children per node.
// For order M, each node can have M-1 to 2M-1 keys.
const Order = 128

// BTree is a B+ Tree index structure for efficient key-value lookups.
type BTree struct {
	root  *Node
	order int
}

// Node represents a node in the B+ Tree.
type Node struct {
	IsLeaf   bool
	Keys     [][]byte // Sorted keys
	Children []*Node  // For internal nodes (len = len(Keys) + 1)
	Values   [][]byte // For leaf nodes (len = len(Keys))
	Next     *Node    // Leaf linked list for range scans
	Parent   *Node    // Parent pointer for splits
}

// New creates a new empty B+ Tree.
func New() *BTree {
	return &BTree{
		order: Order,
		root: &Node{
			IsLeaf: true,
			Keys:   make([][]byte, 0),
			Values: make([][]byte, 0),
		},
	}
}

// newWithOrder constructs a B+Tree with a custom order (used in tests).
func newWithOrder(order int) *BTree {
	if order < 2 {
		order = 2
	}
	return &BTree{
		order: order,
		root: &Node{
			IsLeaf: true,
			Keys:   make([][]byte, 0),
			Values: make([][]byte, 0),
		},
	}
}

// Search searches for a key in the B+ Tree.
// Returns the value and true if found, nil and false otherwise.
func (bt *BTree) Search(key []byte) ([]byte, bool, error) {
	if bt.root == nil {
		return nil, false, nil
	}

	node := bt.root

	// Navigate to leaf
	for !node.IsLeaf {
		idx := findKeyIndex(node.Keys, key)
		if idx < len(node.Children) {
			node = node.Children[idx]
		} else {
			return nil, false, fmt.Errorf("btree: invalid tree structure")
		}
	}

	// Search in leaf
	idx := findExactKey(node.Keys, key)
	if idx >= 0 {
		return node.Values[idx], true, nil
	}

	return nil, false, nil
}

// Insert inserts a key-value pair into the B+ Tree.
func (bt *BTree) Insert(key, value []byte) error {
	if bt.root == nil {
		bt.root = &Node{
			IsLeaf: true,
			Keys:   [][]byte{key},
			Values: [][]byte{value},
		}
		return nil
	}

	// Find the leaf node
	leaf := bt.findLeaf(key)

	// Check for duplicate key
	idx := findExactKey(leaf.Keys, key)
	if idx >= 0 {
		return fmt.Errorf("btree: duplicate key")
	}

	// Insert into leaf
	insertIdx := findInsertPosition(leaf.Keys, key)
	leaf.Keys = insertAt(leaf.Keys, insertIdx, key)
	leaf.Values = insertAt(leaf.Values, insertIdx, value)

	// Check if leaf needs splitting
	if len(leaf.Keys) >= 2*bt.order {
		return bt.splitLeaf(leaf)
	}

	return nil
}

// Delete removes a key from the B+ Tree.
func (bt *BTree) Delete(key []byte) error {
	if bt.root == nil {
		return errors.New("btree: tree is empty")
	}

	leaf := bt.findLeaf(key)
	idx := findExactKey(leaf.Keys, key)

	if idx < 0 {
		return errors.New("btree: key not found")
	}

	// Remove from leaf
	leaf.Keys = removeAt(leaf.Keys, idx)
	leaf.Values = removeAt(leaf.Values, idx)

	// Rebalance if leaf underflows
	bt.rebalanceLeaf(leaf)

	// If root became empty internal node, collapse height
	if bt.root != nil && !bt.root.IsLeaf && len(bt.root.Keys) == 0 && len(bt.root.Children) == 1 {
		bt.root = bt.root.Children[0]
		bt.root.Parent = nil
	}

	// If root leaf lost all keys, tree is now empty
	if bt.root != nil && bt.root.IsLeaf && len(bt.root.Keys) == 0 {
		bt.root = nil
	}

	return nil
}

// RangeScan returns an iterator for keys in [start, end).
func (bt *BTree) RangeScan(ctx context.Context, start, end []byte) (Iterator, error) {
	if bt.root == nil {
		return &rangeIterator{}, nil
	}

	// Find starting leaf
	var startLeaf *Node
	if start == nil {
		// Find leftmost leaf
		startLeaf = bt.root
		for !startLeaf.IsLeaf {
			startLeaf = startLeaf.Children[0]
		}
	} else {
		startLeaf = bt.findLeaf(start)
	}

	return &rangeIterator{
		ctx:     ctx,
		current: startLeaf,
		start:   start,
		end:     end,
		index:   0,
	}, nil
}

// findLeaf navigates from root to the appropriate leaf for the given key.
func (bt *BTree) findLeaf(key []byte) *Node {
	node := bt.root

	for !node.IsLeaf {
		idx := findKeyIndex(node.Keys, key)
		if idx < len(node.Children) {
			node = node.Children[idx]
		} else {
			// Shouldn't happen in a valid tree
			break
		}
	}

	return node
}

// splitLeaf splits a full leaf node.
func (bt *BTree) splitLeaf(leaf *Node) error {
	mid := len(leaf.Keys) / 2

	// Create new right leaf
	right := &Node{
		IsLeaf: true,
		Keys:   append([][]byte{}, leaf.Keys[mid:]...),
		Values: append([][]byte{}, leaf.Values[mid:]...),
		Next:   leaf.Next,
		Parent: leaf.Parent,
	}

	// Update left leaf
	leaf.Keys = leaf.Keys[:mid]
	leaf.Values = leaf.Values[:mid]
	leaf.Next = right

	// Promote middle key to parent
	promoteKey := right.Keys[0]

	if leaf.Parent == nil {
		// Create new root
		bt.root = &Node{
			IsLeaf:   false,
			Keys:     [][]byte{promoteKey},
			Children: []*Node{leaf, right},
		}
		leaf.Parent = bt.root
		right.Parent = bt.root
		return nil
	}

	// Insert into parent
	return bt.insertIntoParent(leaf.Parent, promoteKey, right)
}

// insertIntoParent inserts a key and child pointer into an internal node.
func (bt *BTree) insertIntoParent(parent *Node, key []byte, right *Node) error {
	idx := findInsertPosition(parent.Keys, key)

	parent.Keys = insertAt(parent.Keys, idx, key)
	parent.Children = insertAt(parent.Children, idx+1, right)

	// Check if parent needs splitting
	if len(parent.Keys) >= 2*bt.order {
		return bt.splitInternal(parent)
	}

	return nil
}

// splitInternal splits a full internal node.
func (bt *BTree) splitInternal(node *Node) error {
	mid := len(node.Keys) / 2
	promoteKey := node.Keys[mid]

	// Create new right node
	right := &Node{
		IsLeaf:   false,
		Keys:     append([][]byte{}, node.Keys[mid+1:]...),
		Children: append([]*Node{}, node.Children[mid+1:]...),
		Parent:   node.Parent,
	}

	// Update children's parent pointers
	for _, child := range right.Children {
		child.Parent = right
	}

	// Update left node
	node.Keys = node.Keys[:mid]
	node.Children = node.Children[:mid+1]

	if node.Parent == nil {
		// Create new root
		bt.root = &Node{
			IsLeaf:   false,
			Keys:     [][]byte{promoteKey},
			Children: []*Node{node, right},
		}
		node.Parent = bt.root
		right.Parent = bt.root
		return nil
	}

	// Insert into parent
	return bt.insertIntoParent(node.Parent, promoteKey, right)
}

// rebalanceLeaf fixes underflow for a leaf by borrowing from siblings or merging.
func (bt *BTree) rebalanceLeaf(leaf *Node) {
	if leaf == nil || leaf.Parent == nil {
		return
	}
	if len(leaf.Keys) >= bt.minKeys(leaf) {
		return
	}

	parent := leaf.Parent
	idx := bt.childIndex(parent, leaf)

	// Try borrow from left sibling
	if idx > 0 {
		left := parent.Children[idx-1]
		if len(left.Keys) > bt.minKeys(left) {
			// Move last key/value from left to front of leaf
			leaf.Keys = insertAt(leaf.Keys, 0, left.Keys[len(left.Keys)-1])
			leaf.Values = insertAt(leaf.Values, 0, left.Values[len(left.Values)-1])
			left.Keys = left.Keys[:len(left.Keys)-1]
			left.Values = left.Values[:len(left.Values)-1]
			parent.Keys[idx-1] = leaf.Keys[0]
			return
		}
	}

	// Try borrow from right sibling
	if idx+1 < len(parent.Children) {
		right := parent.Children[idx+1]
		if len(right.Keys) > bt.minKeys(right) {
			leaf.Keys = append(leaf.Keys, right.Keys[0])
			leaf.Values = append(leaf.Values, right.Values[0])
			right.Keys = right.Keys[1:]
			right.Values = right.Values[1:]
			if len(right.Keys) > 0 {
				parent.Keys[idx] = right.Keys[0]
			} else {
				// right became empty; use leaf's last as separator
				parent.Keys[idx] = leaf.Keys[len(leaf.Keys)-1]
			}
			return
		}
	}

	// Merge with sibling
	if idx > 0 {
		// Merge into left sibling
		left := parent.Children[idx-1]
		left.Keys = append(left.Keys, leaf.Keys...)
		left.Values = append(left.Values, leaf.Values...)
		left.Next = leaf.Next
		parent.Keys = removeAt(parent.Keys, idx-1)
		parent.Children = removeAt(parent.Children, idx)
		bt.rebalanceInternal(parent)
		return
	}

	// Merge with right sibling (idx == 0)
	if idx+1 < len(parent.Children) {
		right := parent.Children[idx+1]
		leaf.Keys = append(leaf.Keys, right.Keys...)
		leaf.Values = append(leaf.Values, right.Values...)
		leaf.Next = right.Next
		parent.Keys = removeAt(parent.Keys, idx)
		parent.Children = removeAt(parent.Children, idx+1)
		bt.rebalanceInternal(parent)
	}
}

// rebalanceInternal fixes underflow for an internal node.
func (bt *BTree) rebalanceInternal(node *Node) {
	if node == nil {
		return
	}

	// Root can be smaller; handled by caller after merges.
	if node.Parent == nil {
		if len(node.Keys) == 0 && len(node.Children) == 1 {
			bt.root = node.Children[0]
			bt.root.Parent = nil
		}
		return
	}

	if len(node.Keys) >= bt.minKeys(node) {
		return
	}

	parent := node.Parent
	idx := bt.childIndex(parent, node)

	// Borrow from left sibling
	if idx > 0 {
		left := parent.Children[idx-1]
		if len(left.Keys) > bt.minKeys(left) {
			separator := parent.Keys[idx-1]

			// Move last child/key from left
			borrowedKey := left.Keys[len(left.Keys)-1]
			borrowedChild := left.Children[len(left.Children)-1]

			left.Keys = left.Keys[:len(left.Keys)-1]
			left.Children = left.Children[:len(left.Children)-1]

			// Prepend separator to node
			node.Keys = insertAt(node.Keys, 0, separator)
			node.Children = insertAt(node.Children, 0, borrowedChild)
			borrowedChild.Parent = node

			// Update parent separator
			parent.Keys[idx-1] = borrowedKey
			return
		}
	}

	// Borrow from right sibling
	if idx+1 < len(parent.Children) {
		right := parent.Children[idx+1]
		if len(right.Keys) > bt.minKeys(right) {
			separator := parent.Keys[idx]

			borrowedKey := right.Keys[0]
			borrowedChild := right.Children[0]

			right.Keys = right.Keys[1:]
			right.Children = right.Children[1:]

			node.Keys = append(node.Keys, separator)
			node.Children = append(node.Children, borrowedChild)
			borrowedChild.Parent = node

			parent.Keys[idx] = borrowedKey
			return
		}
	}

	// Merge with sibling
	if idx > 0 {
		left := parent.Children[idx-1]
		separator := parent.Keys[idx-1]

		left.Keys = append(left.Keys, separator)
		left.Keys = append(left.Keys, node.Keys...)
		left.Children = append(left.Children, node.Children...)
		for _, child := range node.Children {
			child.Parent = left
		}

		parent.Keys = removeAt(parent.Keys, idx-1)
		parent.Children = removeAt(parent.Children, idx)
		bt.rebalanceInternal(parent)
		return
	}

	// Merge with right sibling (idx == 0)
	if idx+1 < len(parent.Children) {
		right := parent.Children[idx+1]
		separator := parent.Keys[idx]

		node.Keys = append(node.Keys, separator)
		node.Keys = append(node.Keys, right.Keys...)
		node.Children = append(node.Children, right.Children...)
		for _, child := range right.Children {
			child.Parent = node
		}

		parent.Keys = removeAt(parent.Keys, idx)
		parent.Children = removeAt(parent.Children, idx+1)
		bt.rebalanceInternal(parent)
	}
}

// minKeys returns the minimum number of keys a node should have.
func (bt *BTree) minKeys(node *Node) int {
	if node == bt.root {
		if node.IsLeaf {
			return 0
		}
		return 1
	}
	return bt.order - 1
}

// childIndex finds the index of child in parent.Children.
func (bt *BTree) childIndex(parent *Node, child *Node) int {
	for i, c := range parent.Children {
		if c == child {
			return i
		}
	}
	return -1
}

// Helper functions

// findKeyIndex returns the index of the child to follow for the given key.
func findKeyIndex(keys [][]byte, key []byte) int {
	for i, k := range keys {
		if bytes.Compare(key, k) < 0 {
			return i
		}
	}
	return len(keys)
}

// findExactKey returns the index of the exact key match, or -1 if not found.
func findExactKey(keys [][]byte, key []byte) int {
	for i, k := range keys {
		if bytes.Equal(k, key) {
			return i
		}
	}
	return -1
}

// findInsertPosition returns the index where key should be inserted to maintain sort order.
func findInsertPosition(keys [][]byte, key []byte) int {
	for i, k := range keys {
		if bytes.Compare(key, k) < 0 {
			return i
		}
	}
	return len(keys)
}

// insertAt inserts an element at the specified index in a slice.
func insertAt[T any](slice []T, index int, value T) []T {
	slice = append(slice, value) // Expand slice
	copy(slice[index+1:], slice[index:])
	slice[index] = value
	return slice
}

// removeAt removes an element at the specified index from a slice.
func removeAt[T any](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}

// Iterator interface for range scans.
type Iterator interface {
	Next() bool
	Key() []byte
	Value() []byte
	Err() error
	Close() error
}

// rangeIterator implements Iterator for B+ Tree range scans.
type rangeIterator struct {
	ctx     context.Context
	current *Node
	start   []byte
	end     []byte
	index   int
	err     error
}

func (it *rangeIterator) Next() bool {
	if it.current == nil {
		return false
	}

	// Check context cancellation
	if it.ctx != nil {
		select {
		case <-it.ctx.Done():
			it.err = it.ctx.Err()
			return false
		default:
		}
	}

	// Skip keys before start
	for it.index < len(it.current.Keys) {
		key := it.current.Keys[it.index]

		// Check if before start
		if it.start != nil && bytes.Compare(key, it.start) < 0 {
			it.index++
			continue
		}

		// Check if past end
		if it.end != nil && bytes.Compare(key, it.end) >= 0 {
			it.current = nil
			return false
		}

		return true
	}

	// Move to next leaf
	it.current = it.current.Next
	it.index = 0

	return it.Next()
}

func (it *rangeIterator) Key() []byte {
	if it.current == nil || it.index >= len(it.current.Keys) {
		return nil
	}
	return it.current.Keys[it.index]
}

func (it *rangeIterator) Value() []byte {
	if it.current == nil || it.index >= len(it.current.Values) {
		return nil
	}
	value := it.current.Values[it.index]
	it.index++ // Advance for next call
	return value
}

func (it *rangeIterator) Err() error {
	return it.err
}

func (it *rangeIterator) Close() error {
	it.current = nil
	return nil
}
