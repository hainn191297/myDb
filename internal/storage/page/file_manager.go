package page

import (
	"fmt"
	"os"
	"sync"
)

type FileManager struct {
	dir      string
	file     *os.File
	mu       sync.Mutex
	nextPage PageID
}

func NewFileManager(dir string) (*FileManager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	filePath := fmt.Sprintf("%s/main.db", dir)

	// 0644 = rw-r--r--
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	next := PageID(stat.Size()/PageSize) + 1

	return &FileManager{
		dir:      dir,
		file:     file,
		nextPage: next,
	}, nil
}

// Allocate a new page and return its ID
func (fm *FileManager) AllocatePage() (PageID, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	pageID := fm.nextPage
	fm.nextPage++
	return pageID, nil
}

func (fm *FileManager) ReadPage(pageID PageID) (*Page, error) {
	data := make([]byte, PageSize)
	offset := int64(pageID-1) * PageSize

	_, err := fm.file.ReadAt(data, offset)
	if err != nil {
		return nil, err
	}
	return &Page{ID: pageID, Data: data}, nil
}

func (fm *FileManager) WritePage(page *Page) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	offset := int64(page.ID-1) * PageSize
	_, err := fm.file.WriteAt(page.Data, offset)
	return err
}

func (fm *FileManager) Sync() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return fm.file.Sync()
}

func (fm *FileManager) NumPages() (int64, error) {
	stat, err := fm.file.Stat()
	if err != nil {
		return 0, err
	}
	return stat.Size() / PageSize, nil
}

func (fm *FileManager) Close() error {
	return fm.file.Close()
}
