package schema

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
)

// DataType represents a column data type in the database.
type DataType int

const (
	// TypeInt64 represents a 64-bit signed integer.
	TypeInt64 DataType = iota
	// TypeText represents a variable-length UTF-8 string.
	TypeText
	// TypeBool represents a boolean value (true/false).
	TypeBool
	// TypeFloat64 represents a 64-bit floating point number.
	TypeFloat64
)

// String returns the string representation of the data type.
func (dt DataType) String() string {
	switch dt {
	case TypeInt64:
		return "INT"
	case TypeText:
		return "TEXT"
	case TypeBool:
		return "BOOL"
	case TypeFloat64:
		return "FLOAT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", dt)
	}
}

// ParseDataType converts a SQL type string to a DataType.
func ParseDataType(s string) (DataType, error) {
	switch s {
	case "INT", "INTEGER", "BIGINT":
		return TypeInt64, nil
	case "TEXT", "VARCHAR", "STRING":
		return TypeText, nil
	case "BOOL", "BOOLEAN":
		return TypeBool, nil
	case "FLOAT", "DOUBLE", "REAL":
		return TypeFloat64, nil
	default:
		return 0, fmt.Errorf("schema: unsupported data type %q", s)
	}
}

// Validate checks if the given byte slice is a valid value for this type.
// This performs basic structural validation, not domain-specific constraints.
func (dt DataType) Validate(value []byte) error {
	if len(value) == 0 {
		return fmt.Errorf("schema: value cannot be empty for type %s", dt)
	}

	switch dt {
	case TypeInt64:
		if len(value) != 8 {
			return fmt.Errorf("schema: INT requires exactly 8 bytes, got %d", len(value))
		}
		return nil
	case TypeText:
		// Any byte sequence is valid for text (UTF-8 validation could be added)
		return nil
	case TypeBool:
		if len(value) != 1 {
			return fmt.Errorf("schema: BOOL requires exactly 1 byte, got %d", len(value))
		}
		if value[0] != 0 && value[0] != 1 {
			return fmt.Errorf("schema: BOOL must be 0 or 1, got %d", value[0])
		}
		return nil
	case TypeFloat64:
		if len(value) != 8 {
			return fmt.Errorf("schema: FLOAT requires exactly 8 bytes, got %d", len(value))
		}
		return nil
	default:
		return fmt.Errorf("schema: unknown type %d", dt)
	}
}

// Encode converts a Go value to the binary representation for this type.
func (dt DataType) Encode(value interface{}) ([]byte, error) {
	switch dt {
	case TypeInt64:
		var v int64
		switch val := value.(type) {
		case int64:
			v = val
		case int:
			v = int64(val)
		case int32:
			v = int64(val)
		case string:
			parsed, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("schema: cannot parse %q as INT: %w", val, err)
			}
			v = parsed
		default:
			return nil, fmt.Errorf("schema: cannot encode %T as INT", value)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(v))
		return buf, nil

	case TypeText:
		var s string
		switch val := value.(type) {
		case string:
			s = val
		case []byte:
			s = string(val)
		default:
			return nil, fmt.Errorf("schema: cannot encode %T as TEXT", value)
		}
		return []byte(s), nil

	case TypeBool:
		var b bool
		switch val := value.(type) {
		case bool:
			b = val
		case int:
			b = val != 0
		case string:
			parsed, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("schema: cannot parse %q as BOOL: %w", val, err)
			}
			b = parsed
		default:
			return nil, fmt.Errorf("schema: cannot encode %T as BOOL", value)
		}
		if b {
			return []byte{1}, nil
		}
		return []byte{0}, nil

	case TypeFloat64:
		var f float64
		switch val := value.(type) {
		case float64:
			f = val
		case float32:
			f = float64(val)
		case int:
			f = float64(val)
		case int64:
			f = float64(val)
		case string:
			parsed, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, fmt.Errorf("schema: cannot parse %q as FLOAT: %w", val, err)
			}
			f = parsed
		default:
			return nil, fmt.Errorf("schema: cannot encode %T as FLOAT", value)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, math.Float64bits(f))
		return buf, nil

	default:
		return nil, fmt.Errorf("schema: unknown type %d", dt)
	}
}

// Decode converts the binary representation to a Go value.
func (dt DataType) Decode(data []byte) (interface{}, error) {
	if err := dt.Validate(data); err != nil {
		return nil, err
	}

	switch dt {
	case TypeInt64:
		v := binary.LittleEndian.Uint64(data)
		return int64(v), nil

	case TypeText:
		return string(data), nil

	case TypeBool:
		return data[0] == 1, nil

	case TypeFloat64:
		bits := binary.LittleEndian.Uint64(data)
		return math.Float64frombits(bits), nil

	default:
		return nil, fmt.Errorf("schema: unknown type %d", dt)
	}
}
