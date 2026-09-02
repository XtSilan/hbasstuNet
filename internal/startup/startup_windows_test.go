//go:build windows

package startup

import (
	"errors"
	"syscall"
	"testing"
)

func TestIgnoreMissingValue(t *testing.T) {
	if err := ignoreMissingValue(syscall.ERROR_FILE_NOT_FOUND); err != nil {
		t.Fatalf("missing value error was not ignored: %v", err)
	}
	want := errors.New("other error")
	if got := ignoreMissingValue(want); !errors.Is(got, want) {
		t.Fatalf("other error = %v, want %v", got, want)
	}
}
