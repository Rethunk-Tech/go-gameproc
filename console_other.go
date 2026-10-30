//go:build !windows

package gameproc

// StartOwnConsole starts a program in a new console that it reads and writes itself. That exists only on Windows;
// elsewhere it returns ErrNoConsole.
func StartOwnConsole([]string, string, string, []string) (<-chan error, error) {
	return nil, ErrNoConsole
}
