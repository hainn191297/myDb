package btree

import (
	"bytes"
	"context"
	"fmt"

	dberrors "github.com/hainn191297/myDb/internal/errors"
	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/page"
)

// Engine is a simple page-backed B+Tree-like index with basic splitting to
// support growth beyond a single leaf. Delete does not rebalance but keeps
// search correctness.
type Engine struct {
	schema string
	table  string

	tm *engine.TableManager
	bm *engine.BufferManager

	root page.PageID
}

// New constructs an on-disk index bound to (schema, table) using the shared
// TableManager and BufferManager.
func New(schema, table string, tm *engine.TableManager, bm *engine.BufferManager) *Engine {
	return &Engine{
		schema: schema,
		table:  table,
		tm:     tm,
		bm:     bm,
		root:   1, // single-root leaf for now
	}
}

// Search returns the value for an exact key match.
func (e *Engine) Search(key []byte) ([]byte, bool, error) {
	fid, _, pg, err := e.loadRoot()
	if err != nil {
		return nil, false, err
	}
	// Unpin ancestors as we descend; keep leaf pinned until return.
	for !isLeaf(pg) {
		in, err := readInternal(pg)
		if err != nil {
			e.bm.Unpin(fid, pg.ID, false)
			return nil, false, err
		}
		child := in.chooseChild(key)
		parentFID, parentPID := fid, pg.ID
		fid, pg, err = e.loadPage(child)
		// release parent
		e.bm.Unpin(parentFID, parentPID, false)
		if err != nil {
			return nil, false, err
		}
	}
	defer e.bm.Unpin(fid, pg.ID, false)

	entries, err := readLeaf(pg)
	if err != nil {
		return nil, false, err
	}

	idx := findKey(entries, key)
	if idx < 0 {
		return nil, false, nil
	}
	return entries[idx].value, true, nil
}

// Insert adds a key/value pair. Duplicate keys are allowed; unique enforcement
// happens at higher layers.
func (e *Engine) Insert(key, value []byte) error {
	fid, fm, pg, err := e.loadRoot()
	if err != nil {
		return err
	}
	dirty := false
	defer func() {
		e.bm.Unpin(fid, pg.ID, dirty)
	}()

	path := []page.PageID{}
	node := pg
	curr := e.root
	currFID := fid
	for {
		path = append(path, curr)
		if isLeaf(node) {
			break
		}
		internal, err := readInternal(node)
		if err != nil {
			return err
		}
		child := internal.chooseChild(key)
		parentFID := currFID
		curr = child
		currFID, node, err = e.loadPage(child)
		if err != nil {
			return err
		}
		// release parent
		e.bm.Unpin(parentFID, path[len(path)-1], false)
	}

	leafEntries, err := readLeaf(node)
	if err != nil {
		return err
	}
	pos := insertPos(leafEntries, key)
	leafEntries = insertEntry(leafEntries, pos, kv{key: key, value: value})
	if err := writeLeaf(node, leafEntries, leafNext(node)); err == errPageFull {
		newKey, newPID, err := e.splitLeaf(fm, node, leafEntries)
		if err != nil {
			return err
		}
		if err := e.propagateSplit(path, newKey, newPID); err != nil {
			return err
		}
		dirty = true
	} else if err != nil {
		return err
	} else {
		dirty = true
	}
	return nil
}

// Delete removes the first occurrence of key if present.
func (e *Engine) Delete(key []byte) error {
	fid, _, pg, err := e.loadRoot()
	if err != nil {
		return err
	}
	dirty := false
	defer func() {
		e.bm.Unpin(fid, pg.ID, dirty)
	}()

	for !isLeaf(pg) {
		in, err := readInternal(pg)
		if err != nil {
			e.bm.Unpin(fid, pg.ID, false)
			return err
		}
		child := in.chooseChild(key)
		parentFID, parentPID := fid, pg.ID
		fid, pg, err = e.loadPage(child)
		// release parent while descending
		e.bm.Unpin(parentFID, parentPID, false)
		if err != nil {
			return err
		}
	}

	entries, err := readLeaf(pg)
	if err != nil {
		return err
	}

	idx := findKey(entries, key)
	if idx < 0 {
		return nil
	}
	entries = append(entries[:idx], entries[idx+1:]...)
	if err := writeLeaf(pg, entries, leafNext(pg)); err != nil {
		return err
	}
	// Page is already pinned by loadRoot; marking as dirty will trigger WAL + disk write on unpin
	dirty = true
	return nil
}

// RangeScan returns an iterator over [start, end).
func (e *Engine) RangeScan(ctx context.Context, start, end []byte) (engine.Iterator, error) {
	fid, _, pg, err := e.loadRoot()
	if err != nil {
		return nil, err
	}
	// descend to leaf containing start, unpinning ancestors
	for !isLeaf(pg) {
		in, err := readInternal(pg)
		if err != nil {
			e.bm.Unpin(fid, pg.ID, false)
			return nil, err
		}
		child := in.chooseChild(start)
		parentFID, parentPID := fid, pg.ID
		fid, pg, err = e.loadPage(child)
		e.bm.Unpin(parentFID, parentPID, false)
		if err != nil {
			return nil, err
		}
	}

	return &iter{
		ctx:  ctx,
		eng:  e,
		fid:  fid,
		leaf: pg,
		end:  end,
	}, nil
}

// loadRoot ensures the root page exists and returns it pinned.
func (e *Engine) loadRoot() (uint32, *page.FileManager, *page.Page, error) {
	_, fm, err := e.tm.OpenTable(e.schema, e.table)
	if err != nil {
		return 0, nil, nil, err
	}

	numPages, err := fm.NumPages()
	if err != nil {
		return 0, nil, nil, fmt.Errorf("btree: NumPages: %w", err)
	}

	if numPages == 0 {
		pid, err := fm.AllocatePage()
		if err != nil {
			return 0, nil, nil, err
		}
		if pid != e.root {
			e.root = pid
		}
		root := page.NewPage(pid)
		if err := initEmpty(root); err != nil {
			return 0, nil, nil, err
		}
		if err := fm.WritePage(root); err != nil {
			return 0, nil, nil, fmt.Errorf("btree: write new root: %w", err)
		}
	}

	fidRoot, pg, err := e.loadPagePinned(e.root)
	if err != nil {
		return 0, nil, nil, err
	}
	return fidRoot, fm, pg, nil
}

func (e *Engine) loadPagePinned(pid page.PageID) (uint32, *page.Page, error) {
	fid, pg, err := e.bm.GetPage(e.schema, e.table, pid)
	if err != nil {
		return 0, nil, fmt.Errorf("btree: GetPage(%d): %w", pid, err)
	}
	return fid, pg, nil
}

func (e *Engine) loadPage(pid page.PageID) (uint32, *page.Page, error) {
	fid, pg, err := e.bm.GetPage(e.schema, e.table, pid)
	if err != nil {
		return 0, nil, fmt.Errorf("btree: GetPage(%d): %w", pid, err)
	}
	return fid, pg, nil
}

const (
	nodeLeaf     byte = 1
	nodeInternal byte = 2
)

// Leaf layout:
// [0] nodeType
// [1:3] reserved
// [3:5] entryCount uint16 (offset 1)
// [5:13] next PageID int64
// entries: [keyLen uint16][valLen uint16][key][val]...
const leafHeader = 13

// Internal layout:
// [0] nodeType
// [1:3] reserved
// [3:5] keyCount uint16
// payload: firstChild int64, then repeated [keyLen uint16][key][child int64] * keyCount
const internalHeader = 5

type kv struct {
	key   []byte
	value []byte
}

type internalNode struct {
	keys     [][]byte
	children []page.PageID // len = len(keys)+1
}

var errPageFull = dberrors.ErrPageFull

func initEmpty(pg *page.Page) error {
	if len(pg.Data) != page.PageSize {
		return fmt.Errorf("btree: invalid page size %d", len(pg.Data))
	}
	for i := range pg.Data {
		pg.Data[i] = 0
	}
	pg.Data[0] = nodeLeaf
	return nil
}

func isLeaf(pg *page.Page) bool {
	return pg.Data[0] == nodeLeaf || pg.Data[0] == 0
}

func readLeaf(pg *page.Page) ([]kv, error) {
	data := pg.Data
	if len(data) < leafHeader {
		return nil, fmt.Errorf("btree: leaf too small")
	}
	count := int(uint16(data[3]) | uint16(data[4])<<8)
	entries := make([]kv, 0, count)

	offset := leafHeader
	for i := 0; i < count; i++ {
		if offset+4 > len(data) {
			return nil, fmt.Errorf("btree: corrupt leaf header")
		}
		klen := int(uint16(data[offset]) | uint16(data[offset+1])<<8)
		vlen := int(uint16(data[offset+2]) | uint16(data[offset+3])<<8)
		offset += 4
		if offset+klen+vlen > len(data) {
			return nil, fmt.Errorf("btree: corrupt leaf payload")
		}
		key := make([]byte, klen)
		copy(key, data[offset:offset+klen])
		offset += klen
		val := make([]byte, vlen)
		copy(val, data[offset:offset+vlen])
		offset += vlen
		entries = append(entries, kv{key: key, value: val})
	}
	return entries, nil
}

func writeLeaf(pg *page.Page, entries []kv, next page.PageID) error {
	data := pg.Data
	offset := leafHeader
	total := leafHeader
	for _, e := range entries {
		total += 4 + len(e.key) + len(e.value)
	}
	if total > len(data) {
		return errPageFull
	}
	for i := range data {
		data[i] = 0
	}
	data[0] = nodeLeaf
	data[3] = byte(len(entries))
	data[4] = byte(len(entries) >> 8)
	// next pointer
	putInt64(data[5:], int64(next))

	for _, e := range entries {
		klen := len(e.key)
		vlen := len(e.value)
		data[offset] = byte(klen)
		data[offset+1] = byte(klen >> 8)
		data[offset+2] = byte(vlen)
		data[offset+3] = byte(vlen >> 8)
		offset += 4
		copy(data[offset:offset+klen], e.key)
		offset += klen
		copy(data[offset:offset+vlen], e.value)
		offset += vlen
	}
	return nil
}

func leafNext(pg *page.Page) page.PageID {
	pid, _ := readInt64(pg.Data[5:])
	return pid
}

func readInternal(pg *page.Page) (*internalNode, error) {
	data := pg.Data
	if len(data) < internalHeader {
		return nil, fmt.Errorf("btree: internal too small")
	}
	count := int(uint16(data[3]) | uint16(data[4])<<8)
	if count == 0 {
		return &internalNode{}, nil
	}
	children := make([]page.PageID, count+1)
	offset := internalHeader

	firstChild, n := readInt64(data[offset:])
	if n == 0 {
		return nil, fmt.Errorf("btree: invalid child pointer")
	}
	children[0] = firstChild
	offset += n

	keys := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		if offset+2 > len(data) {
			return nil, fmt.Errorf("btree: corrupt internal key header")
		}
		klen := int(uint16(data[offset]) | uint16(data[offset+1])<<8)
		offset += 2
		if offset+klen > len(data) {
			return nil, fmt.Errorf("btree: corrupt internal key payload")
		}
		key := make([]byte, klen)
		copy(key, data[offset:offset+klen])
		offset += klen

		child, n := readInt64(data[offset:])
		if n == 0 {
			return nil, fmt.Errorf("btree: missing child pointer")
		}
		children[i+1] = child
		offset += n

		keys = append(keys, key)
	}
	return &internalNode{keys: keys, children: children}, nil
}

func writeInternal(pg *page.Page, in *internalNode) error {
	data := pg.Data
	offset := internalHeader
	total := internalHeader
	total += 8 // first child
	for _, k := range in.keys {
		total += 2 + len(k) + 8
	}
	if total > len(data) {
		return errPageFull
	}
	for i := range data {
		data[i] = 0
	}
	data[0] = nodeInternal
	data[3] = byte(len(in.keys))
	data[4] = byte(len(in.keys) >> 8)

	putInt64(data[offset:], int64(in.children[0]))
	offset += 8
	for i, k := range in.keys {
		data[offset] = byte(len(k))
		data[offset+1] = byte(len(k) >> 8)
		offset += 2
		copy(data[offset:offset+len(k)], k)
		offset += len(k)
		putInt64(data[offset:], int64(in.children[i+1]))
		offset += 8
	}
	return nil
}

func (in *internalNode) chooseChild(key []byte) page.PageID {
	for i, k := range in.keys {
		if bytes.Compare(key, k) < 0 {
			return in.children[i]
		}
	}
	return in.children[len(in.children)-1]
}

func putInt64(dst []byte, v int64) {
	_ = dst[7]
	dst[0] = byte(v)
	dst[1] = byte(v >> 8)
	dst[2] = byte(v >> 16)
	dst[3] = byte(v >> 24)
	dst[4] = byte(v >> 32)
	dst[5] = byte(v >> 40)
	dst[6] = byte(v >> 48)
	dst[7] = byte(v >> 56)
}

func readInt64(src []byte) (page.PageID, int) {
	if len(src) < 8 {
		return 0, 0
	}
	v := int64(src[0]) |
		int64(src[1])<<8 |
		int64(src[2])<<16 |
		int64(src[3])<<24 |
		int64(src[4])<<32 |
		int64(src[5])<<40 |
		int64(src[6])<<48 |
		int64(src[7])<<56
	return page.PageID(v), 8
}

// splitLeaf splits a full leaf into two, returning promoted key and new page id.
// The left page (pg) is already pinned by the caller and will be marked dirty.
// The new right page is allocated and managed through BufferManager for proper WAL logging.
func (e *Engine) splitLeaf(fm *page.FileManager, pg *page.Page, entries []kv) ([]byte, page.PageID, error) {
	mid := len(entries) / 2
	left := entries[:mid]
	right := entries[mid:]

	// Allocate new page ID
	newPID, err := fm.AllocatePage()
	if err != nil {
		return nil, 0, err
	}

	// Write left entries to existing page (pg)
	// Caller is responsible for unpinning with dirty=true
	if err := writeLeaf(pg, left, newPID); err != nil {
		return nil, 0, err
	}

	// Initialize new page on disk first (empty page)
	// This is safe as it's a brand new page with no prior data
	newPage := page.NewPage(newPID)
	if err := initEmpty(newPage); err != nil {
		return nil, 0, fmt.Errorf("btree: init new leaf: %w", err)
	}
	if err := fm.WritePage(newPage); err != nil {
		return nil, 0, fmt.Errorf("btree: write empty leaf: %w", err)
	}

	// Now load through BufferManager to ensure it's cached and tracked
	fid, rightPg, err := e.bm.GetPage(e.schema, e.table, newPID)
	if err != nil {
		return nil, 0, fmt.Errorf("btree: load new leaf: %w", err)
	}
	defer e.bm.Unpin(fid, newPID, true) // Mark as dirty

	// Write right entries to the new page
	if err := writeLeaf(rightPg, right, 0); err != nil {
		return nil, 0, err
	}

	return right[0].key, newPID, nil
}

// propagateSplit walks path from leaf to root inserting promoted keys.
func (e *Engine) propagateSplit(path []page.PageID, promoteKey []byte, newChild page.PageID) error {
	// path includes leaf; iterate bottom-up excluding leaf
	if len(path) == 1 {
		// root was leaf; create new root internal
		_, fm, err := e.tm.OpenTable(e.schema, e.table)
		if err != nil {
			return err
		}
		newRootID, err := fm.AllocatePage()
		if err != nil {
			return err
		}

		// Initialize empty internal page on disk
		newRootPage := page.NewPage(newRootID)
		if err := initEmpty(newRootPage); err != nil {
			return fmt.Errorf("btree: init new root: %w", err)
		}
		if err := fm.WritePage(newRootPage); err != nil {
			return fmt.Errorf("btree: write empty root: %w", err)
		}

		// Load through BufferManager
		fid, pg, err := e.bm.GetPage(e.schema, e.table, newRootID)
		if err != nil {
			return fmt.Errorf("btree: load new root: %w", err)
		}
		defer e.bm.Unpin(fid, newRootID, true)

		// Write internal node structure
		internal := &internalNode{
			keys:     [][]byte{promoteKey},
			children: []page.PageID{path[0], newChild},
		}
		if err := writeInternal(pg, internal); err != nil {
			return err
		}

		e.root = newRootID
		return nil
	}

	// traverse ancestors (excluding leaf) from bottom
	parentID := path[len(path)-2]
	parentFID, parentPg, err := e.loadPagePinned(parentID)
	if err != nil {
		return err
	}
	defer e.bm.Unpin(parentFID, parentPg.ID, true)

	if isLeaf(parentPg) {
		// parent is still leaf? should not happen
		return fmt.Errorf("btree: expected internal parent")
	}
	in, err := readInternal(parentPg)
	if err != nil {
		return err
	}
	pos := insertPosKV(in.keys, promoteKey)
	in.keys = insertKey(in.keys, pos, promoteKey)
	in.children = insertChild(in.children, pos+1, newChild)

	if err := writeInternal(parentPg, in); err == errPageFull {
		// split parent
		leftKeys, rightKeys, leftChildren, rightChildren, upKey := splitInternal(in)
		if err := writeInternal(parentPg, &internalNode{keys: leftKeys, children: leftChildren}); err != nil {
			return err
		}

		// Allocate new page for right half
		_, fm, err := e.tm.OpenTable(e.schema, e.table)
		if err != nil {
			return err
		}
		newPID, err := fm.AllocatePage()
		if err != nil {
			return err
		}

		// Initialize empty page
		newPage := page.NewPage(newPID)
		if err := initEmpty(newPage); err != nil {
			return fmt.Errorf("btree: init new internal: %w", err)
		}
		if err := fm.WritePage(newPage); err != nil {
			return fmt.Errorf("btree: write empty internal: %w", err)
		}

		// Load through BufferManager
		fid, newPg, err := e.bm.GetPage(e.schema, e.table, newPID)
		if err != nil {
			return fmt.Errorf("btree: load new internal: %w", err)
		}
		defer e.bm.Unpin(fid, newPID, true)

		// Write right half structure
		if err := writeInternal(newPg, &internalNode{keys: rightKeys, children: rightChildren}); err != nil {
			return err
		}

		// propagate further up
		return e.propagateSplit(path[:len(path)-1], upKey, newPID)
	} else if err != nil {
		return err
	}
	return nil
}

func insertPosKV(keys [][]byte, key []byte) int {
	lo, hi := 0, len(keys)
	for lo < hi {
		mid := (lo + hi) / 2
		if bytes.Compare(keys[mid], key) <= 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func insertKey(keys [][]byte, pos int, key []byte) [][]byte {
	keys = append(keys, nil)
	copy(keys[pos+1:], keys[pos:])
	keys[pos] = key
	return keys
}

func insertChild(children []page.PageID, pos int, child page.PageID) []page.PageID {
	children = append(children, 0)
	copy(children[pos+1:], children[pos:])
	children[pos] = child
	return children
}

func splitInternal(in *internalNode) (leftKeys, rightKeys [][]byte, leftChildren, rightChildren []page.PageID, upKey []byte) {
	mid := len(in.keys) / 2
	upKey = in.keys[mid]
	leftKeys = append([][]byte{}, in.keys[:mid]...)
	rightKeys = append([][]byte{}, in.keys[mid+1:]...)
	leftChildren = append([]page.PageID{}, in.children[:mid+1]...)
	rightChildren = append([]page.PageID{}, in.children[mid+1:]...)
	return
}

func findKey(entries []kv, key []byte) int {
	lo, hi := 0, len(entries)
	for lo < hi {
		mid := (lo + hi) / 2
		cmp := bytes.Compare(entries[mid].key, key)
		if cmp == 0 {
			return mid
		}
		if cmp < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return -1
}

func insertPos(entries []kv, key []byte) int {
	lo, hi := 0, len(entries)
	for lo < hi {
		mid := (lo + hi) / 2
		if bytes.Compare(entries[mid].key, key) <= 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func insertEntry(entries []kv, pos int, entry kv) []kv {
	entries = append(entries, kv{})
	copy(entries[pos+1:], entries[pos:])
	entries[pos] = entry
	return entries
}

type iter struct {
	ctx     context.Context
	eng     *Engine
	fid     uint32
	leaf    *page.Page
	entries []kv
	idx     int
	end     []byte
	err     error
}

func (it *iter) Next() bool {
	if it.err != nil {
		return false
	}
	if it.ctx != nil {
		select {
		case <-it.ctx.Done():
			it.err = it.ctx.Err()
			return false
		default:
		}
	}

	for {
		if it.entries == nil {
			entries, err := readLeaf(it.leaf)
			if err != nil {
				it.err = err
				return false
			}
			it.entries = entries
			it.idx = 0
		}
		if it.idx < len(it.entries) {
			entry := it.entries[it.idx]
			if it.end != nil && bytes.Compare(entry.key, it.end) >= 0 {
				it.eng.bm.Unpin(it.fid, it.leaf.ID, false)
				it.leaf = nil
				return false
			}
			it.idx++
			return true
		}
		// move to next leaf
		next := leafNext(it.leaf)
		it.eng.bm.Unpin(it.fid, it.leaf.ID, false)
		if next == 0 {
			return false
		}
		fid, pg, err := it.eng.loadPage(next)
		if err != nil {
			it.err = err
			return false
		}
		it.fid = fid
		it.leaf = pg
		it.entries = nil
	}
}

func (it *iter) Key() []byte {
	if it.err != nil || it.entries == nil || it.idx == 0 || it.idx > len(it.entries) {
		return nil
	}
	return it.entries[it.idx-1].key
}

func (it *iter) Value() []byte {
	if it.err != nil || it.entries == nil || it.idx == 0 || it.idx > len(it.entries) {
		return nil
	}
	return it.entries[it.idx-1].value
}

func (it *iter) Err() error { return it.err }

func (it *iter) Close() error {
	if it.leaf != nil {
		it.eng.bm.Unpin(it.fid, it.leaf.ID, false)
	}
	it.entries = nil
	return nil
}
