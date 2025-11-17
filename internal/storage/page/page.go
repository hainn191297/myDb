package page

// default page size = 4KB
const PageSize = 4096

type PageID int64

// Page represents a fixed-size 4KB block belonging to a given table file.
type Page struct {
	ID   PageID
	Data []byte
}

// NewPage creates an empty page buffer for the given table and ID.
func NewPage(id PageID) *Page {
	return &Page{
		ID:   id,
		Data: make([]byte, PageSize),
	}
}
