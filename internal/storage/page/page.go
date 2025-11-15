package page

// default page size = 4KB
const PageSize = 4096

type PageID int64

// Page represents a fixed-size block of data.
type Page struct {
	ID   PageID
	Data []byte
}

// NewPage creates a new page with the given ID.
func NewPage(id PageID) *Page {
	return &Page{
		ID:   id,
		Data: make([]byte, PageSize),
	}
}
