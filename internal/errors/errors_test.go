package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorIdentification(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		sentinel error
		want     bool
	}{
		{
			name:     "exact match",
			err:      ErrTableNotFound,
			sentinel: ErrTableNotFound,
			want:     true,
		},
		{
			name:     "wrapped error matches",
			err:      fmt.Errorf("wrapped: %w", ErrTableNotFound),
			sentinel: ErrTableNotFound,
			want:     true,
		},
		{
			name:     "different error",
			err:      ErrTableNotFound,
			sentinel: ErrTableExists,
			want:     false,
		},
		{
			name:     "nil error",
			err:      nil,
			sentinel: ErrTableNotFound,
			want:     false,
		},
		{
			name:     "double wrapped",
			err:      fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrKeyNotFound)),
			sentinel: ErrKeyNotFound,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errors.Is(tt.err, tt.sentinel)
			if got != tt.want {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.err, tt.sentinel, got, tt.want)
			}
		})
	}
}

func TestAllErrorsAreUnique(t *testing.T) {
	// Collect all error instances
	allErrors := []error{
		// Storage
		ErrKeyNotFound,
		ErrPageFull,
		ErrInvalidPageData,
		// Catalog
		ErrTableNotFound,
		ErrTableExists,
		ErrColumnNotFound,
		ErrIndexNotFound,
		ErrIndexExists,
		ErrEmptyTableName,
		ErrEmptyColumnList,
		// Transaction
		ErrTxnNotFound,
		ErrTxnAlreadyActive,
		ErrNoActiveTxn,
		// Parser
		ErrEmptyStatement,
		ErrInvalidSyntax,
		ErrMissingClause,
		// Executor
		ErrCatalogNotConfigured,
		ErrStorageNotConfigured,
		ErrTxnMgrNotConfigured,
		ErrUnsupportedOperator,
	}

	// Check that each error message is unique
	seen := make(map[string]bool)
	for _, err := range allErrors {
		msg := err.Error()
		if seen[msg] {
			t.Errorf("duplicate error message: %q", msg)
		}
		seen[msg] = true
	}

	// Verify we have the expected count
	expectedCount := 20
	if len(allErrors) != expectedCount {
		t.Errorf("expected %d errors, got %d", expectedCount, len(allErrors))
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		err     error
		wantMsg string
	}{
		{ErrTableNotFound, "table not found"},
		{ErrKeyNotFound, "key not found"},
		{ErrTxnAlreadyActive, "transaction already active"},
		{ErrCatalogNotConfigured, "catalog not configured"},
	}

	for _, tt := range tests {
		t.Run(tt.wantMsg, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("error message = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}
