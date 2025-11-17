package heap

import (
	"encoding/binary"
	"fmt"

	"github.com/hainn191297/myDb/internal/storage/page"
)

/*
Heap page layout (slotted page):

  [0:2]  slotCount   (uint16)  → number of directory slots
  [2:4]  freeStart   (uint16)  → offset where free space currently begins
  [4:freeStart)      tuple data region (grows upward)
  [freeStart:freeEnd) free space
  [freeEnd:PageSize) slot directory (grows downward from the end)

Slot directory entry i (0-based):

  offset := PageSize - 2*(i+1)
  value  := uint16 tupleOffset OR 0xFFFF if slot is free

NOTE:
- We do NOT store tuple length here. Tuple encoding (len, columns, etc.)
  thuộc về layer tuple.go – ở đây chỉ quản lý vị trí trong page.
- Xoá slot chỉ đánh dấu slot offset = 0xFFFF; chưa compact free space.
*/

const (
	headerSize        = 4              // 2 bytes slotCount + 2 bytes freeStart
	invalidSlotOffset = uint16(0xFFFF) // đánh dấu slot đã xoá / trống
)

// Page là wrapper cho *page.Page chứa layout heap ở trên.
type Page struct {
	raw *page.Page
}

// WrapPage bọc một page vật lý thành heap.Page.
func WrapPage(p *page.Page) *Page {
	return &Page{raw: p}
}

// NewEmptyPage tạo một heap page mới với header khởi tạo.
func NewEmptyPage(pid page.PageID) *Page {
	p := page.NewPage(pid)
	hp := &Page{raw: p}
	hp.initHeader()
	return hp
}

// Raw trả về page vật lý bên dưới.
func (hp *Page) Raw() *page.Page {
	return hp.raw
}

// initHeader sets an empty header (no slots, freeStart after header).
func (hp *Page) initHeader() {
	data := hp.raw.Data
	binary.LittleEndian.PutUint16(data[0:2], 0)                  // slotCount
	binary.LittleEndian.PutUint16(data[2:4], uint16(headerSize)) // freeStart
}

// slotCount returns current number of slots (including free ones).
func (hp *Page) slotCount() uint16 {
	return binary.LittleEndian.Uint16(hp.raw.Data[0:2])
}

func (hp *Page) setSlotCount(n uint16) {
	binary.LittleEndian.PutUint16(hp.raw.Data[0:2], n)
}

// freeStart is the offset where tuple data region ends / free space begins.
func (hp *Page) freeStart() uint16 {
	return binary.LittleEndian.Uint16(hp.raw.Data[2:4])
}

func (hp *Page) setFreeStart(off uint16) {
	binary.LittleEndian.PutUint16(hp.raw.Data[2:4], off)
}

// freeEnd is where free space ends and slot directory begins.
func (hp *Page) freeEnd() int {
	return page.PageSize - int(hp.slotCount())*2
}

// FreeSpace returns size of contiguous free space.
func (hp *Page) FreeSpace() int {
	fs := int(hp.freeStart())
	fe := hp.freeEnd()
	if fe <= fs {
		return 0
	}
	return fe - fs
}

// NumSlots returns number of slots (including free ones).
func (hp *Page) NumSlots() int {
	return int(hp.slotCount())
}

// slotOffset returns tuple offset for slot i (or invalidSlotOffset).
func (hp *Page) slotOffset(i int) uint16 {
	if i < 0 || i >= int(hp.slotCount()) {
		return invalidSlotOffset
	}
	data := hp.raw.Data
	pos := page.PageSize - 2*(i+1)
	return binary.LittleEndian.Uint16(data[pos : pos+2])
}

func (hp *Page) setSlotOffset(i int, off uint16) {
	if i < 0 || i >= int(hp.slotCount()) {
		return
	}
	data := hp.raw.Data
	pos := page.PageSize - 2*(i+1)
	binary.LittleEndian.PutUint16(data[pos:pos+2], off)
}

// IsSlotUsed reports whether slot i currently points to a tuple.
func (hp *Page) IsSlotUsed(i int) bool {
	return hp.slotOffset(i) != invalidSlotOffset
}

/*
InsertTuple appends a new tuple into the free space and creates a new slot.

RETURN:
  - slot index (int) nếu thành công
  - error nếu không đủ chỗ

Ghi chú:
- Chỉ check contiguous free space (không reclaim các lỗ do xoá).
- tuple bytes đã được encode ở layer tuple.go.
*/
func (hp *Page) InsertTuple(tuple []byte) (int, error) {
	need := len(tuple) + 2 // tuple data + 1 slot entry (2 bytes)
	if need > hp.FreeSpace() {
		return -1, fmt.Errorf("heap: not enough space for tuple (%d bytes free, need %d)", hp.FreeSpace(), need)
	}

	data := hp.raw.Data
	slotCount := hp.slotCount()
	freeStart := hp.freeStart()

	// write tuple data at freeStart
	copy(data[freeStart:int(freeStart)+len(tuple)], tuple)

	// new slot index
	newSlot := int(slotCount)
	newSlotPos := page.PageSize - 2*(newSlot+1)
	binary.LittleEndian.PutUint16(data[newSlotPos:newSlotPos+2], freeStart)

	// update header
	hp.setSlotCount(slotCount + 1)
	hp.setFreeStart(freeStart + uint16(len(tuple)))

	return newSlot, nil
}

/*
DeleteTuple marks slot i as free (invalidSlotOffset).
Không compact heap ngay.
*/
func (hp *Page) DeleteTuple(slot int) error {
	if slot < 0 || slot >= int(hp.slotCount()) {
		return fmt.Errorf("heap: delete invalid slot %d", slot)
	}
	if !hp.IsSlotUsed(slot) {
		return nil // already free
	}
	hp.setSlotOffset(slot, invalidSlotOffset)
	return nil
}

/*
GetTuple returns a slice view of the tuple bytes for slot i.

IMPORTANT:
  - Slice này trỏ trực tiếp vào page.Data (no copy).
  - Tuple length phải do layer tuple.go giải mã (nếu có prefix length),
    ở đây chỉ trả từ offset đến trước vùng freeStart / next tuple (không an toàn lắm).
  - Hiện tại, ta chỉ support pattern: tuple encoder/decoder biết cách cắt.

Để tránh đoán length ở đây, ta thường để tuple.go nhận (pageData, offset)
và tự parse. Nên hàm này chỉ trả offset & raw tail.
*/
func (hp *Page) GetTupleRegion(slot int) (offset uint16, buf []byte, ok bool) {
	if slot < 0 || slot >= int(hp.slotCount()) {
		return 0, nil, false
	}
	off := hp.slotOffset(slot)
	if off == invalidSlotOffset {
		return 0, nil, false
	}
	data := hp.raw.Data
	return off, data[off:hp.freeStart()], true
}

/*
ForEach iterates over all USED slots and calls fn(slotIdx, offset).

fn trả về false để dừng sớm.
Layer khác (table/engine) sẽ dùng offset để giải mã tuple.
*/
func (hp *Page) ForEach(fn func(slot int, offset uint16) bool) {
	n := int(hp.slotCount())
	for i := 0; i < n; i++ {
		off := hp.slotOffset(i)
		if off == invalidSlotOffset {
			continue
		}
		if !fn(i, off) {
			return
		}
	}
}
