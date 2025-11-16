package heap

import (
	"encoding/binary"
	"errors"

	"github.com/hainn191297/myDb/internal/storage/page"
)

const (
	headerSize   = 6 // slotCount(2) + slotDirEnd(2) + freeEnd(2)
	slotEntryLen = 4 // offset(2) + length(2)
)

type Slot struct {
	Offset uint16
	Length uint16
}

func initPage(p *page.Page) {
	writeHeader(p, 0, headerSize, page.PageSize)
}

func pageInitialized(p *page.Page) bool {
	_, slotDirEnd, freeEnd := readHeader(p)
	return slotDirEnd != 0 && freeEnd != 0
}

func readHeader(p *page.Page) (slots uint16, slotDirEnd uint16, freeEnd uint16) {
	slots = binary.LittleEndian.Uint16(p.Data[0:2])
	slotDirEnd = binary.LittleEndian.Uint16(p.Data[2:4])
	freeEnd = binary.LittleEndian.Uint16(p.Data[4:6])
	return
}

func writeHeader(p *page.Page, slots, slotDirEnd, freeEnd uint16) {
	binary.LittleEndian.PutUint16(p.Data[0:2], slots)
	binary.LittleEndian.PutUint16(p.Data[2:4], slotDirEnd)
	binary.LittleEndian.PutUint16(p.Data[4:6], freeEnd)
}

func slotOffset(idx uint16) int {
	return int(headerSize) + int(idx)*slotEntryLen
}

func getSlot(p *page.Page, index uint16) Slot {
	offset := slotOffset(index)
	return Slot{
		Offset: binary.LittleEndian.Uint16(p.Data[offset : offset+2]),
		Length: binary.LittleEndian.Uint16(p.Data[offset+2 : offset+4]),
	}
}

func setSlot(p *page.Page, index uint16, s Slot) {
	offset := slotOffset(index)
	binary.LittleEndian.PutUint16(p.Data[offset:offset+2], s.Offset)
	binary.LittleEndian.PutUint16(p.Data[offset+2:offset+4], s.Length)
}

func hasSpace(p *page.Page, payloadLen uint16) bool {
	slots, slotDirEnd, freeEnd := readHeader(p)
	if slotDirEnd == 0 && freeEnd == 0 {
		return false
	}
	if int(payloadLen) > page.PageSize {
		return false
	}

	dataPos := int(freeEnd) - int(payloadLen)
	if dataPos < 0 {
		return false
	}
	_, hasFree := findFreeSlot(p, slots)
	neededDir := int(slotDirEnd)
	if !hasFree {
		neededDir += slotEntryLen
	}
	return neededDir <= dataPos
}

func insertTuple(p *page.Page, payload []byte) (uint16, error) {
	payloadLen := uint16(len(payload))
	if int(payloadLen) > page.PageSize-headerSize-slotEntryLen {
		return 0, errTupleTooLarge
	}

	slots, slotDirEnd, freeEnd := readHeader(p)
	freeSlot, hasFree := findFreeSlot(p, slots)
	if !hasSpace(p, payloadLen) {
		return 0, errNoSpace
	}

	writePos := freeEnd - payloadLen
	copy(p.Data[writePos:writePos+payloadLen], payload)
	var slotIdx uint16
	if hasFree {
		slotIdx = freeSlot
	} else {
		slotIdx = slots
		slots++
		slotDirEnd += slotEntryLen
	}
	setSlot(p, slotIdx, Slot{Offset: writePos, Length: payloadLen})
	writeHeader(p, slots, slotDirEnd, writePos)
	return slotIdx, nil
}

func findFreeSlot(p *page.Page, slotCount uint16) (uint16, bool) {
	for i := uint16(0); i < slotCount; i++ {
		if getSlot(p, i).Length == 0 {
			return i, true
		}
	}
	return 0, false
}

func clearSlot(p *page.Page, slotIdx uint16) {
	setSlot(p, slotIdx, Slot{Offset: 0, Length: 0})
}

func iterateSlots(p *page.Page, fn func(slotID int, data []byte) bool) {
	slots, _, _ := readHeader(p)
	for i := 0; i < int(slots); i++ {
		slot := getSlot(p, uint16(i))
		if slot.Length == 0 {
			continue
		}
		if !fn(i, p.Data[slot.Offset:slot.Offset+slot.Length]) {
			return
		}
	}
}

var (
	errTupleTooLarge = errors.New("heap: tuple too large")
	errNoSpace       = errors.New("heap: no free space on page")
)
