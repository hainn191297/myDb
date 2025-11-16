package page

import (
	"fmt"
	"os"
	"sync"
)

/*
FileManager

FileManager is responsible for ALL low-level disk I/O.

It represents ONE physical table file (similar to a `.ibd` file in MySQL),
and provides the following operations:

  - AllocatePage()   → assigns a new PageID (append-only model)
  - ReadPage()       → load exactly 1 page from disk (4KB)
  - WritePage()      → write exactly 1 page to disk
  - Sync()           → fsync() for durability
  - NumPages()       → physical page count
  - Close()          → close file handle

IMPORTANT:
  - FileManager does NOT know anything about BufferPool, WAL, transactions.
  - It only provides raw fixed-offset page reads and writes.
  - Page layout, free list, B+Tree, etc. are in higher layers.
*/
type FileManager struct {
	filePath string       // full path of this table file, e.g. "data/product.db"
	file     *os.File     // opened file descriptor
	mu       sync.RWMutex // change to RWMutex to allow concurrent reads
	nextPage PageID       // ID to assign for the next allocated page
}

/*
NewFileManager

Open or create a table file.

This function also computes nextPage by inspecting file size:

	file size = N bytes
	PageSize   = 4096 bytes
	→ numPages = N / 4096
	→ nextPage = numPages + 1

In other words, pages are numbered starting at 1:

	Page 1 → offset 0 * PageSize
	Page 2 → offset 1 * PageSize
	Page 3 → offset 2 * PageSize
	...
*/
func NewFileManager(filePath string) (*FileManager, error) {

	// 0644 = rw-r--r--
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("fm: cannot open file %s: %w", filePath, err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("fm: stat failed on %s: %w", filePath, err)
	}

	// compute how many pages already exist
	numPages := stat.Size() / int64(PageSize) // change to int64
	next := PageID(numPages + 1)

	return &FileManager{
		filePath: filePath,
		file:     f,
		nextPage: next,
	}, nil
}

/*
AllocatePage

Allocate a new PageID.

NOTE:
  - This function does NOT write anything to disk
  - Only reserves a logical ID
  - Caller must write actual content using WritePage()

Equivalent to InnoDB allocating a new page within a tablespace.
*/
func (fm *FileManager) AllocatePage() (PageID, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	id := fm.nextPage
	fm.nextPage++
	return id, nil
}

/*
ReadPage

Read exactly one page from disk.

Procedure:

	offset = (pageID - 1) * PageSize
	ReadAt 4096 bytes into memory

If the page does not physically exist (offset beyond EOF),
ReadAt returns an error.

This matches the behavior of real DBMS engines.
*/
func (fm *FileManager) ReadPage(id PageID) (*Page, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	offset := int64(id-1) * PageSize
	data := make([]byte, PageSize)

	_, err := fm.file.ReadAt(data, offset)
	if err != nil {
		return nil, fmt.Errorf("fm: read page %d failed: %w", id, err)
	}

	return &Page{
		ID:   id,
		Data: data,
	}, nil
}

/*
WritePage

Write exactly one page to its disk position.

If the page ID goes beyond current file size,
WriteAt will automatically extend the file.

This is how most DBMS implement append-only page allocation.
*/
func (fm *FileManager) WritePage(p *Page) error {
	if len(p.Data) != PageSize {
		return fmt.Errorf(
			"fm: invalid page size %d for page %d (expected %d)",
			len(p.Data), p.ID, PageSize,
		)
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	offset := (int64(p.ID) - 1) * int64(PageSize)

	_, err := fm.file.WriteAt(p.Data, offset)
	if err != nil {
		return fmt.Errorf("fm: write page %d failed: %w", p.ID, err)
	}

	return nil
}

/*
Sync (fsync)

Flush kernel page cache buffers to disk.

This is required for durability guarantees,
especially when used together with Write-Ahead Logging (WAL).
*/
func (fm *FileManager) Sync() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	return fm.file.Sync()
}

/*
NumPages

	Return the number of pages currently stored on disk.

	This is simply:
		fileSize / PageSize

	Useful for validation, debugging, or storage engine metadata.
*/
func (fm *FileManager) NumPages() (int64, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	stat, err := fm.file.Stat()
	if err != nil {
		return 0, fmt.Errorf("fm: stat failed: %w", err)
	}
	return stat.Size() / int64(PageSize), nil
}

/*
Close

Close the underlying OS file descriptor.
*/
func (fm *FileManager) Close() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return fm.file.Close()
}
