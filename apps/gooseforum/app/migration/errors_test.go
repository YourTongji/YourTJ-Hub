package migration

import (
	"errors"
	"fmt"
	"testing"
)

func TestDeferredClassifiesMigrationOutcomes(t *testing.T) {
	hard := errors.New("app migration v25 admin secret plaintext encryption: 3 failed")
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "hard failure", err: hard, want: false},
		{name: "generic error", err: errors.New("boom"), want: false},
		{name: "lock unavailable", err: fmt.Errorf("%w: sqlite lock still held", ErrLockUnavailable), want: true},
		{name: "retry later", err: fmt.Errorf("%w: v13 aggregate search indexes", ErrRetryLater), want: true},
		{name: "wrapped hard failure", err: fmt.Errorf("run migrations: %w", hard), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Deferred(tt.err); got != tt.want {
				t.Fatalf("Deferred(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
