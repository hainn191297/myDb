package heap

import (
	"encoding/binary"
	"fmt"
	"math"
)

// encodeTuple serializes key/value into [keyLen][valLen][key][value].
func encodeTuple(key, value []byte) []byte {
	const maxTupleSize = 64 * 1024 * 1024 // 64 MB, adjust as reasonable for your app
	tupleLen := 8 + len(key) + len(value)
	if len(key) < 0 || len(value) < 0 || tupleLen < 0 || tupleLen > maxTupleSize || tupleLen > math.MaxInt {
		panic(fmt.Sprintf("encodeTuple: tuple too large (key=%d, value=%d, total=%d)", len(key), len(value), tupleLen))
	}
	buf := make([]byte, tupleLen)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(key)))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(value)))
	copy(buf[8:8+len(key)], key)
	copy(buf[8+len(key):], value)
	return buf
}

// TODO: optimize allocations
func decodeTuple(data []byte) ([]byte, []byte, error) {
	if len(data) < 8 {
		return nil, nil, fmt.Errorf("heap engine: corrupt tuple")
	}
	keyLen := binary.LittleEndian.Uint32(data[0:4])
	valLen := binary.LittleEndian.Uint32(data[4:8])
	if int(8+keyLen+valLen) > len(data) {
		return nil, nil, fmt.Errorf("heap engine: truncated tuple")
	}
	key := append([]byte(nil), data[8:8+keyLen]...)
	val := append([]byte(nil), data[8+keyLen:8+keyLen+valLen]...)
	return key, val, nil
}
