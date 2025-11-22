package schema

import (
	"testing"
)

func TestParseDataType(t *testing.T) {
	tests := []struct {
		input string
		want  DataType
		valid bool
	}{
		{"INT", TypeInt64, true},
		{"INTEGER", TypeInt64, true},
		{"BIGINT", TypeInt64, true},
		{"TEXT", TypeText, true},
		{"VARCHAR", TypeText, true},
		{"STRING", TypeText, true},
		{"BOOL", TypeBool, true},
		{"BOOLEAN", TypeBool, true},
		{"FLOAT", TypeFloat64, true},
		{"DOUBLE", TypeFloat64, true},
		{"REAL", TypeFloat64, true},
		{"UNKNOWN", TypeInt64, false},
		{"", TypeInt64, false},
	}

	for _, tt := range tests {
		got, err := ParseDataType(tt.input)
		if tt.valid && err != nil {
			t.Errorf("ParseDataType(%q) unexpected error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ParseDataType(%q) expected error, got nil", tt.input)
		}
		if tt.valid && got != tt.want {
			t.Errorf("ParseDataType(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestDataTypeString(t *testing.T) {
	tests := []struct {
		dt   DataType
		want string
	}{
		{TypeInt64, "INT"},
		{TypeText, "TEXT"},
		{TypeBool, "BOOL"},
		{TypeFloat64, "FLOAT"},
	}

	for _, tt := range tests {
		if got := tt.dt.String(); got != tt.want {
			t.Errorf("DataType(%d).String() = %q, want %q", tt.dt, got, tt.want)
		}
	}
}

func TestTypeValidation(t *testing.T) {
	tests := []struct {
		name  string
		dt    DataType
		value []byte
		valid bool
	}{
		// INT validation
		{"INT valid", TypeInt64, make([]byte, 8), true},
		{"INT too short", TypeInt64, make([]byte, 4), false},
		{"INT empty", TypeInt64, []byte{}, false},

		// TEXT validation
		{"TEXT valid", TypeText, []byte("hello"), true},
		{"TEXT empty", TypeText, []byte{}, false},

		// BOOL validation
		{"BOOL true", TypeBool, []byte{1}, true},
		{"BOOL false", TypeBool, []byte{0}, true},
		{"BOOL invalid value", TypeBool, []byte{2}, false},
		{"BOOL wrong size", TypeBool, make([]byte, 2), false},

		// FLOAT validation
		{"FLOAT valid", TypeFloat64, make([]byte, 8), true},
		{"FLOAT too short", TypeFloat64, make([]byte, 4), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dt.Validate(tt.value)
			if tt.valid && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Validate() expected error, got nil")
			}
		})
	}
}

func TestTypeEncodeDecode(t *testing.T) {
	tests := []struct {
		name  string
		dt    DataType
		value interface{}
		want  interface{}
	}{
		// INT
		{"INT from int64", TypeInt64, int64(42), int64(42)},
		{"INT from int", TypeInt64, int(100), int64(100)},
		{"INT from string", TypeInt64, "123", int64(123)},

		// TEXT
		{"TEXT from string", TypeText, "hello", "hello"},
		{"TEXT from bytes", TypeText, []byte("world"), "world"},

		// BOOL
		{"BOOL true", TypeBool, true, true},
		{"BOOL false", TypeBool, false, false},
		{"BOOL from int", TypeBool, 1, true},
		{"BOOL from string", TypeBool, "true", true},

		// FLOAT
		{"FLOAT from float64", TypeFloat64, float64(3.14), float64(3.14)},
		{"FLOAT from int", TypeFloat64, int(42), float64(42)},
		{"FLOAT from string", TypeFloat64, "2.71", float64(2.71)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := tt.dt.Encode(tt.value)
			if err != nil {
				t.Fatalf("Encode(%v) error: %v", tt.value, err)
			}

			// Validate
			if err := tt.dt.Validate(encoded); err != nil {
				t.Fatalf("Validate() error: %v", err)
			}

			// Decode
			decoded, err := tt.dt.Decode(encoded)
			if err != nil {
				t.Fatalf("Decode() error: %v", err)
			}

			// Compare (handle float precision)
			if tt.dt == TypeFloat64 {
				wantF := tt.want.(float64)
				gotF := decoded.(float64)
				if diff := wantF - gotF; diff < -0.0001 || diff > 0.0001 {
					t.Errorf("Decode() = %v, want %v", decoded, tt.want)
				}
			} else if decoded != tt.want {
				t.Errorf("Decode() = %v, want %v", decoded, tt.want)
			}
		})
	}
}

func TestTypeEncodeErrors(t *testing.T) {
	tests := []struct {
		name  string
		dt    DataType
		value interface{}
	}{
		{"INT from invalid string", TypeInt64, "not-a-number"},
		{"INT from unsupported type", TypeInt64, struct{}{}},
		{"TEXT from unsupported type", TypeText, 123},
		{"BOOL from invalid string", TypeBool, "maybe"},
		{"FLOAT from invalid string", TypeFloat64, "not-a-float"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.dt.Encode(tt.value)
			if err == nil {
				t.Errorf("Encode(%v) expected error, got nil", tt.value)
			}
		})
	}
}
