//go:build !windows

package gameproc

import (
	"errors"
	"testing"
)

func TestStartOwnConsoleIsWindowsOnly(t *testing.T) {
	if _, err := StartOwnConsole(nil, "", "x", nil); !errors.Is(err, ErrNoConsole) {
		t.Fatalf("err = %v", err)
	}
}
